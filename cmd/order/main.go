package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"

	"github.com/dwikikusuma/atlas/internal/order/db"
	"github.com/dwikikusuma/atlas/internal/order/service"
	"github.com/dwikikusuma/atlas/pkg/database"
	"github.com/dwikikusuma/atlas/pkg/kafka"
	"github.com/dwikikusuma/atlas/pkg/pb/order"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	postgresURI   = "postgres://atlas:atlaspassword@localhost:5432/atlas_db?sslmode=disable"
	grpcPort      = ":50052"
	kafkaBroker   = "localhost:9092"
	dispatchTopic = "ride-dispatch"
	dispatchGroup = "order-service-group"
)

func main() {
	ctx := context.Background()

	dbConfig := database.PostgresConfig{
		ConnectionURL: postgresURI,
		MaxConn:       10,
		MinConn:       2,
		MaxIdleTime:   300,
		HealthCheck:   true,
	}

	connPool, err := database.NewPostgresPool(ctx, dbConfig)
	if err != nil {
		log.Fatalf("❌ cannot connect to Postgres: %v", err)
	}
	defer connPool.Close()
	log.Println("✅ Connected to Postgres")

	sqlcDB := db.New(connPool)
	svc := service.NewOrderService(sqlcDB)

	grpcServer := grpc.NewServer()
	order.RegisterOrderServiceServer(grpcServer, svc)
	reflection.Register(grpcServer)

	dispatchConsumer := kafka.NewConsumer([]string{kafkaBroker}, dispatchGroup, dispatchTopic)
	setDriverConsumer := service.NewOrderWorker(dispatchConsumer, sqlcDB)
	go func() {
		if err = setDriverConsumer.Start(ctx); err != nil {
			log.Fatalf("❌ order worker failed: %v", err)
		}
	}()

	listener, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("❌ cannot create listener: %v", err)
	}
	go func() {
		log.Printf("🚀 Order gRPC server listening on %s", listener.Addr().String())
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
	log.Println("✅ gRPC server stopped")

	connPool.Close()
	log.Println("✅ Postgres connection closed")
}
