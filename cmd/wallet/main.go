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

	"github.com/dwikikusuma/atlas/internal/wallet/service"
	"github.com/dwikikusuma/atlas/pkg/database"
	"github.com/dwikikusuma/atlas/pkg/kafka"
	"github.com/dwikikusuma/atlas/pkg/pb/wallet"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	postgresURI = "postgres://atlas:atlaspassword@localhost:5432/atlas_db?sslmode=disable"
	grpcPort    = ":50054"
	kafkaBroker = "localhost:9092"
)

func main() {
	// Create cancellable context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to database
	conn, err := connectDB(ctx)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer conn.Close()

	// Initialize service
	svc := service.NewPostgresWalletService(conn)

	// WaitGroup to track running goroutines
	var wg sync.WaitGroup

	// Start Kafka consumer worker
	wg.Add(1)
	go func() {
		defer wg.Done()
		startConsumer(ctx, svc)
	}()

	// Start gRPC server
	grpcServer := grpc.NewServer()
	wg.Add(1)
	go func() {
		defer wg.Done()
		startGRPCServer(grpcServer, svc)
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutdown signal received, starting graceful shutdown...")

	// Cancel context to stop all workers
	cancel()

	// Stop gRPC server gracefully
	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	// Wait for graceful stop with timeout
	select {
	case <-stopped:
		log.Println("✅ gRPC server stopped gracefully")
	case <-time.After(10 * time.Second):
		log.Println("⚠️ Forcing gRPC server shutdown after timeout")
		grpcServer.Stop()
	}

	// Wait for all goroutines to finish
	wg.Wait()
	log.Println("✅ All services stopped, exiting...")
}

func connectDB(ctx context.Context) (*pgxpool.Pool, error) {
	conn, err := database.NewPostgresPool(ctx, database.PostgresConfig{
		ConnectionURL: postgresURI,
		MaxConn:       10,
		MinConn:       2,
		MaxIdleTime:   300,
		HealthCheck:   true,
	})
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
		return nil, err
	}
	return conn, nil
}

func startGRPCServer(grpcServer *grpc.Server, svc wallet.WalletServiceServer) {
	wallet.RegisterWalletServiceServer(grpcServer, svc)
	reflection.Register(grpcServer)

	listener, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	log.Printf("🚀 gRPC wallet service starting on %s", grpcPort)
	if err := grpcServer.Serve(listener); err != nil {
		log.Printf("gRPC server stopped: %v", err)
	}
}

func startConsumer(ctx context.Context, svc service.WalletService) {
	consumer := kafka.NewConsumer([]string{kafkaBroker}, "wallet-group", "wallet-transactions")
	worker := service.NewWalletWorker(consumer, svc)
	go func() {
		if err := worker.Start(ctx); err != nil {
			log.Printf("❌ Wallet worker stopped with error: %v", err)
		} else {
			log.Println("✅ Wallet worker stopped gracefully")
		}
	}()
}
