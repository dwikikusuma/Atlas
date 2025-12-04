package service

import (
	"context"
	"encoding/json"
	"log"

	"github.com/dwikikusuma/atlas/internal/tracker/model"
	"github.com/dwikikusuma/atlas/pkg/kafka"
	"github.com/dwikikusuma/atlas/pkg/pb/tracker"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	tracker.UnimplementedTrackerServiceServer
	producer kafka.EventProducer
}

func NewServer(producer kafka.EventProducer) *Server {
	return &Server{
		producer: producer,
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

	err = s.producer.Publish(ctx, "location-updates", req.UserId, eventByte)
	if err != nil {
		log.Printf("failed to publish event: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to publish event: %v", err)
	}
	return &tracker.UpdateLocationResponse{
		Success: true,
	}, nil
}
