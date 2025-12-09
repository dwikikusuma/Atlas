package service

import (
	"context"
	"log"

	"github.com/dwikikusuma/atlas/pkg/pb/dispatch"
	"github.com/dwikikusuma/atlas/pkg/pb/tracker"
)

type DispatchService struct {
	dispatch.UnimplementedDispatchServiceServer
	trackerClient tracker.TrackerServiceClient
}

func NewDispatchService(trackerClient tracker.TrackerServiceClient) *DispatchService {
	return &DispatchService{
		trackerClient: trackerClient,
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
		// For now, fail gracefully
		return &dispatch.RequestRideResponse{
			Status: "ERROR_TRACKER_DOWN",
		}, nil
	}

	status := "DRIVERS_NOT_FOUND"
	if len(res.Drivers) > 0 {
		status = "DRIVERS_FOUND"
	}

	return &dispatch.RequestRideResponse{
		Status: status,
		RideId: "RIDE" + req.PassengerId,
	}, nil
}
