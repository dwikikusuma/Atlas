package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"

	"github.com/dwikikusuma/atlas/internal/wallet/service"
	"github.com/dwikikusuma/atlas/pkg/database"
	"github.com/dwikikusuma/atlas/pkg/pb/wallet"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	postgresURI = "postgres://atlas:atlaspassword@localhost:5432/atlas_db?sslmode=disable"
	grpcPort    = ":50054"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, err := database.NewPostgresPool(ctx, database.PostgresConfig{
		ConnectionURL: postgresURI,
		MaxConn:       10,
		MinConn:       2,
		MaxIdleTime:   300,
		HealthCheck:   true,
	})
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer conn.Close()

	svc := service.NewPostgresWalletService(conn)
	grpcServer := grpc.NewServer()
	wallet.RegisterWalletServiceServer(grpcServer, svc)
	reflection.Register(grpcServer)

	listener, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	go func() {
		log.Println("grpc wallet service is running on port", grpcPort)
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("failed to serve grpc server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Println("shutting down server...")
	grpcServer.GracefulStop()
}
