package service

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/dwikikusuma/atlas/internal/dispatch/model"
	"github.com/dwikikusuma/atlas/pkg/kafka"
	"github.com/dwikikusuma/atlas/pkg/pb/dispatch"
	"github.com/dwikikusuma/atlas/pkg/pb/tracker"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	dispatchTopic = "ride-dispatch"
)

type DispatchService struct {
	dispatch.UnimplementedDispatchServiceServer
	trackerClient tracker.TrackerServiceClient
	producer      *kafka.Producer
}

func NewDispatchService(trackerClient tracker.TrackerServiceClient, producer *kafka.Producer) *DispatchService {
	return &DispatchService{
		trackerClient: trackerClient,
		producer:      producer,
	}
}

func (s *DispatchService) RequestRide(ctx context.Context, req *dispatch.RequestRideRequest) (*dispatch.RequestRideResponse, error) {
	res, err := s.trackerClient.GetNearbyDrivers(ctx, &tracker.GetNearbyDriverRequest{
		Longitude: req.PickupLong,
		Latitude:  req.PickupLat,
		Radius:    5, // 5 km radius
	})

	if err != nil {
		log.Printf("❌ Failed to query tracker: %v", err)
		return nil, status.Errorf(codes.Unavailable, "failed to query tracker: %v", err)
	}

	var selectedDriverId string
	searchStatus := "DRIVERS_NOT_FOUND"
	if len(res.Drivers) > 0 {
		searchStatus = "DRIVERS_FOUND"
		selectedDriverId = res.Drivers[0].DriverId

		msg := model.RideDispatchedEvent{
			RideID:      req.PassengerId,
			PassengerID: req.PassengerId,
			DriverID:    selectedDriverId,
			PickupLat:   req.PickupLat,
			PickupLong:  req.PickupLong,
			Timestamp:   time.Now().Unix(),
		}

		payload, _ := json.Marshal(&msg)

		err = s.producer.Publish(ctx, dispatchTopic, selectedDriverId, payload)
		if err != nil {
			log.Printf("❌ Failed to publish dispatch event: %v", err)
			return nil, status.Errorf(codes.Internal, "failed to publish dispatch event: %v", err)
		} else {
			log.Printf("✅ Event Published: Passenger %s -> Driver %s", req.PassengerId, selectedDriverId)
		}
	}

	return &dispatch.RequestRideResponse{
		Status:   searchStatus,
		RideId:   req.PassengerId,
		DriverId: selectedDriverId,
	}, nil
}
