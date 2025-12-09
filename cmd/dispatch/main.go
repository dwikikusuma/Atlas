package main

import (
	"log"
	"net"

	"github.com/dwikikusuma/atlas/internal/dispatch/service"
	"github.com/dwikikusuma/atlas/pkg/kafka"
	"github.com/dwikikusuma/atlas/pkg/pb/dispatch"
	"github.com/dwikikusuma/atlas/pkg/pb/tracker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

const (
	grpcPort    = ":50052"
	trackerAddr = "localhost:50051"
	kafkaBroker = "localhost:9092"
)

func main() {
	conn, err := grpc.NewClient(trackerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("could not connect to tracker: %v", err)

	}
	defer func(conn *grpc.ClientConn) {
		err := conn.Close()
		if err != nil {
			log.Fatalf("could not close connection to tracker: %v", err)
		}
	}(conn)

	producer := kafka.NewProducer([]string{kafkaBroker})
	defer func() {
		if err := producer.Close(); err != nil {
			log.Fatalf("could not close kafka producer: %v", err)
		}
	}()

	trackerClient := tracker.NewTrackerServiceClient(conn)
	srv := service.NewDispatchService(trackerClient, producer)

	grpcServer := grpc.NewServer()
	dispatch.RegisterDispatchServiceServer(grpcServer, srv)
	reflection.Register(grpcServer)

	listener, err := net.Listen("tcp", grpcPort)
	if err != nil {
		panic(err)
	}
	err = grpcServer.Serve(listener)
	if err != nil {
		panic(err)
	}
}
