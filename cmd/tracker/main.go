package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
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

	ctx, cancel := context.WithCancel(context.Background())

	redisClient, err := database.NewRedisClient(database.Config{
		Addr: redisAddr,
	})

	if err != nil {
		log.Fatalf("failed to initialize redis client: %v", err)
	}

	locationRepo := repository.NewRedisClientRepo(redisClient)

	producer := kafka.NewProducer([]string{kafkaBroker})
	defer func(producer *kafka.Producer) {
		err := producer.Close()
		if err != nil {
			log.Printf("failed to close kafka producer: %v", err)
		}
	}(producer)

	consumer := kafka.NewConsumer([]string{kafkaBroker}, kafkaGroup, kafkaTopic)
	defer func(consumer *kafka.Consumer) {
		err := consumer.Close()
		if err != nil {
			log.Printf("failed to close kafka consumer: %v", err)
		}
	}(consumer)

	worker := service.NewIngestionWorker(consumer, locationRepo)
	go func() {
		worker.Run(ctx)
	}()

	srv := service.NewServer(producer, locationRepo)

	grpcServer := grpc.NewServer()
	tracker.RegisterTrackerServiceServer(grpcServer, srv)
	reflection.Register(grpcServer)

	listener, err := net.Listen("tcp", serverPort)
	if err != nil {
		log.Fatalf("❌ cannot create listener: %v", err)
	}

	go func() {
		log.Printf("🚀 Tracker gRPC server listening on %s", listener.Addr().String())
		err = grpcServer.Serve(listener)
		if err != nil {
			log.Fatalf("❌ cannot start grpc server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Println("🛑 Shutting down server...")
	grpcServer.GracefulStop()

	cancel()
	log.Println("✅ Server stopped")

	time.Sleep(1 * time.Second)
	log.Println("✅ Worker stopped")

	log.Println("👋 Bye!")
}
