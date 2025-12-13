package main

import (
	"log"
	"net/http"

	"github.com/dwikikusuma/atlas/internal/gateaway"
	"github.com/dwikikusuma/atlas/pkg/pb/dispatch"
	"github.com/dwikikusuma/atlas/pkg/pb/order"
	"github.com/dwikikusuma/atlas/pkg/pb/tracker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	trackerAddr  = "localhost:50051"
	orderAddr    = "localhost:50052"
	dispatchAddr = "localhost:50053"
	httpPort     = ":8085"
)

func main() {

	trackerConn, err := grpc.NewClient(trackerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	trackerClient := tracker.NewTrackerServiceClient(trackerConn)
	defer trackerConn.Close()

	orderConn, err := grpc.NewClient(orderAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	orderClient := order.NewOrderServiceClient(orderConn)
	defer orderConn.Close()

	dispatchConn, err := grpc.NewClient(dispatchAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	dispatchClient := dispatch.NewDispatchServiceClient(dispatchConn)
	defer dispatchConn.Close()

	custHandler := gateaway.NewCustomerHandler(orderClient, dispatchClient)
	driverHandler := gateaway.NewDriverHandler(trackerClient, orderClient)

	mux := http.NewServeMux()
	custHandler.RegisterRoutes(mux)
	driverHandler.RegisterRoutes(mux)

	log.Printf("🚀 Gateway listening on HTTP %s", httpPort)
	if err = http.ListenAndServe(httpPort, mux); err != nil {
		log.Fatalf("❌ HTTP Server failed: %v", err)
	}
}
