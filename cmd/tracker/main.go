package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/dwikikusuma/atlas/internal/tracker/repository"
	"github.com/dwikikusuma/atlas/internal/tracker/service"
	"github.com/dwikikusuma/atlas/pkg/database"
	"github.com/dwikikusuma/atlas/pkg/kafka"
	"github.com/dwikikusuma/atlas/pkg/pb/tracker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	redisAddr   = "localhost:6379"
	kafkaBroker = "localhost:9092"
	kafkaTopic  = "driver-gps"
	kafkaGroup  = "tracker-group"
	serverPort  = ":50051"
)

func main() {
	// Create cancellable context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Redis
	redisClient, err := database.NewRedisClient(database.Config{
		Addr: redisAddr,
	})
	if err != nil {
		log.Fatalf("❌ failed to initialize redis client: %v", err)
	}
	defer redisClient.Close()
	log.Println("✅ Connected to Redis")

	locationRepo := repository.NewRedisClientRepo(redisClient)

	// Initialize Kafka Producer
	producer := kafka.NewProducer([]string{kafkaBroker})
	defer func() {
		if err := producer.Close(); err != nil {
			log.Printf("⚠️ failed to close kafka producer: %v", err)
		} else {
			log.Println("✅ Kafka producer closed")
		}
	}()

	// Initialize Kafka Consumer
	consumer := kafka.NewConsumer([]string{kafkaBroker}, kafkaGroup, kafkaTopic)
	defer func() {
		if err := consumer.Close(); err != nil {
			log.Printf("⚠️ failed to close kafka consumer: %v", err)
		} else {
			log.Println("✅ Kafka consumer closed")
		}
	}()

	// WaitGroup to track goroutines
	var wg sync.WaitGroup

	// Start Kafka ingestion worker
	worker := service.NewIngestionWorker(consumer, locationRepo)
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("🚀 Starting ingestion worker...")
		worker.Run(ctx)
		log.Println("✅ Ingestion worker stopped")
	}()

	// Initialize gRPC server
	srv := service.NewServer(producer, locationRepo)
	grpcServer := grpc.NewServer()
	tracker.RegisterTrackerServiceServer(grpcServer, srv)
	reflection.Register(grpcServer)

	listener, err := net.Listen("tcp", serverPort)
	if err != nil {
		log.Fatalf("❌ cannot create listener: %v", err)
	}

	// Start gRPC server
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("🚀 Tracker gRPC server listening on %s", listener.Addr().String())
		if err := grpcServer.Serve(listener); err != nil {
			log.Printf("gRPC server stopped: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutdown signal received, starting graceful shutdown...")

	// Cancel context to stop worker
	cancel()

	// Gracefully stop gRPC server with timeout
	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		log.Println("✅ gRPC server stopped gracefully")
	case <-time.After(10 * time.Second):
		log.Println("⚠️ Forcing gRPC server shutdown after timeout")
		grpcServer.Stop()
	}

	// Wait for all goroutines to finish
	log.Println("⏳ Waiting for worker to finish...")
	wg.Wait()

	log.Println("✅ All services stopped")
	log.Println("👋 Bye!")
}
