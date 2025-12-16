package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"

	"github.com/dwikikusuma/atlas/internal/order/db"
	"github.com/dwikikusuma/atlas/internal/order/service"
	"github.com/dwikikusuma/atlas/pkg/database"
	"github.com/dwikikusuma/atlas/pkg/kafka"
	"github.com/dwikikusuma/atlas/pkg/pb/order"
	wallet "github.com/dwikikusuma/atlas/pkg/pb/wallet"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

const (
	postgresURI   = "postgres://atlas:atlaspassword@localhost:5432/atlas_db?sslmode=disable"
	grpcPort      = ":50052"
	kafkaBroker   = "localhost:9092"
	dispatchTopic = "ride-dispatch"
	dispatchGroup = "order-service-group"
	wallerPort    = ":50054"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	connPool, err := connectDB(ctx)
	if err != nil {
		log.Fatalf("❌ cannot connect to Postgres: %v", err)
	}
	defer connPool.Close()

	producer := kafka.NewProducer([]string{kafkaBroker})
	log.Println("✅ Connecting to Producer...")

	walletConn, err := grpc.NewClient(wallerPort, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("❌ cannot connect to Wallet Service: %v", err)
	}
	walletClient := wallet.NewWalletServiceClient(walletConn)

	var wg sync.WaitGroup

	sqlcDB := db.New(connPool)
	svc := service.NewOrderService(sqlcDB, producer, walletClient)

	wg.Add(1)
	go func() {
		defer wg.Done()
		startConsumer(ctx, sqlcDB)
	}()

	grpcServer := grpc.NewServer()
	wg.Add(1)
	go func() {
		defer wg.Done()
		startGRPCServer(grpcServer, svc)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Println("🛑 Shutting down server...")

	grpcServer.GracefulStop()
	log.Println("✅ gRPC server stopped")

	connPool.Close()
	log.Println("✅ Postgres connection closed")
}

func connectDB(ctx context.Context) (*pgxpool.Pool, error) {
	dbConfig := database.PostgresConfig{
		ConnectionURL: postgresURI,
		MaxConn:       10,
		MinConn:       2,
		MaxIdleTime:   300,
		HealthCheck:   true,
	}

	connPool, err := database.NewPostgresPool(ctx, dbConfig)
	if err != nil {
		return nil, err
	}
	log.Println("✅ Connected to Postgres")
	return connPool, nil
}

func startConsumer(ctx context.Context, sqlcDB *db.Queries) {
	dispatchConsumer := kafka.NewConsumer([]string{kafkaBroker}, dispatchGroup, dispatchTopic)
	setDriverConsumer := service.NewOrderWorker(dispatchConsumer, sqlcDB)
	if err := setDriverConsumer.Start(ctx); err != nil {
		log.Fatalf("❌ order worker failed: %v", err)
	}
	log.Println("✅ Order worker started")
}

func startGRPCServer(grpcServer *grpc.Server, svc *service.Service) {

	order.RegisterOrderServiceServer(grpcServer, svc)
	reflection.Register(grpcServer)

	listener, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("❌ cannot create listener: %v", err)
	}
	log.Printf("🚀 Order gRPC server listening on %s", listener.Addr().String())
	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatalf("❌ cannot start grpc server: %v", err)
	}
}
