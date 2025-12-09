package service

import (
	"context"
	"encoding/json"
	"log"

	"github.com/dwikikusuma/atlas/internal/tracker/domain"
	"github.com/dwikikusuma/atlas/internal/tracker/model"
	"github.com/dwikikusuma/atlas/pkg/kafka"
	"github.com/dwikikusuma/atlas/pkg/pb/tracker"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	tracker.UnimplementedTrackerServiceServer
	producer kafka.EventProducer
	repo     domain.LocationRepository
}

func NewServer(producer kafka.EventProducer, repo domain.LocationRepository) *Server {
	return &Server{
		producer: producer,
		repo:     repo,
	}
}

func (s *Server) UpdateLocation(ctx context.Context, req *tracker.UpdateLocationRequest) (*tracker.UpdateLocationResponse, error) {
	event := model.LocationEvent{
		UserID:    req.UserId,
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
		Timestamp: req.Timestamp,
	}

	eventByte, err := json.Marshal(event)
	if err != nil {
		log.Printf("failed to marshal event: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to marshal event: %v", err)
	}

	err = s.producer.Publish(ctx, "driver-gps", req.UserId, eventByte)
	if err != nil {
		log.Printf("failed to publish event: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to publish event: %v", err)
	}
	return &tracker.UpdateLocationResponse{
		Success: true,
	}, nil
}

func (s *Server) GetNearbyDrivers(ctx context.Context, req *tracker.GetNearbyDriverRequest) (*tracker.GetNearbyDriverResponse, error) {
	location, err := s.repo.GetNearbyDrivers(ctx, req.Latitude, req.Longitude, req.Radius)
	if err != nil {
		log.Printf("failed to get nearby drivers: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to get nearby drivers: %v", err)
	}

	var res []*tracker.Driver
	for _, loc := range location {
		res = append(res, &tracker.Driver{
			DriverId:  loc.UserID,
			Longitude: loc.Longitude,
			Latitude:  loc.Latitude,
		})
	}

	return &tracker.GetNearbyDriverResponse{Drivers: res}, nil
}

func (s *Server) GetDriverLocation(ctx context.Context, req *tracker.GetDriverLocationRequest) (*tracker.GetDriverLocationResponse, error) {
	location, err := s.repo.GetDriverLocation(ctx, req.DriverId)
	if err != nil {
		log.Printf("failed to get driver location: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to get driver location: %v", err)
	}

	if location == nil {
		return nil, status.Errorf(codes.NotFound, "driver location not found")
	}

	return &tracker.GetDriverLocationResponse{
		DriverId:  location.UserID,
		Longitude: location.Longitude,
		Latitude:  location.Latitude,
	}, nil
}
