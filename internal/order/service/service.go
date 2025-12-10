package service

import (
	"context"
	"math"
	"time"

	"github.com/dwikikusuma/atlas/internal/order/db"
	"github.com/dwikikusuma/atlas/pkg/pb/order"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	order.UnimplementedOrderServiceServer
	store db.Querier
}

func NewOrderService(store db.Querier) *Service {
	return &Service{
		store: store,
	}
}

func (s *Service) CreateOrder(ctx context.Context, req *order.CreateOrderRequest) (*order.CreateOrderResponse, error) {

	createOrderParams := db.CreateOrderParams{
		PassengerID: req.UserId,
		PickupLong:  req.PickupLong,
		PickupLat:   req.PickupLat,
		DropoffLat:  req.DropoffLat,
		DropoffLong: req.DropoffLong,
		Status:      "CREATED",
		Price:       calculatePrice(req.PickupLat, req.PickupLong, req.DropoffLat, req.DropoffLong),
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	orderDetail, err := s.store.CreateOrder(dbCtx, createOrderParams)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &order.CreateOrderResponse{
		OrderId: orderDetail.ID.String(),
		Status:  orderDetail.Status,
		Price:   orderDetail.Price,
	}, nil
}

func calculatePrice(lat1, lon1, lat2, lon2 float64) float64 {
	// 1. Calculate Distance (Euclidean approximation for short distances)
	// In production, use the Haversine formula for better accuracy.
	// 1 degree of latitude ~= 111km
	x := lat2 - lat1
	y := lon2 - lon1
	distanceKm := math.Sqrt(x*x+y*y) * 111.32

	// 2. Pricing Rules
	baseFare := 10000.0  // IDR
	pricePerKm := 3000.0 // IDR

	price := baseFare + (distanceKm * pricePerKm)

	// Round to nearest whole number for clean display
	return math.Round(price)
}
