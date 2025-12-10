package service

import (
	"context"
	"math"
	"time"

	"github.com/dwikikusuma/atlas/internal/order/db"
	"github.com/dwikikusuma/atlas/pkg/pb/order"
	"github.com/jackc/pgx/v5/pgtype"
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

func (s *Service) GetOrder(ctx context.Context, req *order.GetOrderRequest) (*order.GetOrderResponse, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var orderID pgtype.UUID
	err := orderID.Scan(req.OrderId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order ID")
	}

	orderDetail, err := s.store.GetOrder(dbCtx, orderID)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, status.Errorf(codes.NotFound, "order not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get order: %v", err)
	}

	return &order.GetOrderResponse{
		OrderId:     orderDetail.ID.String(),
		PassengerId: orderDetail.PassengerID,
		DriverId:    orderDetail.DriverID,
		PickupLat:   orderDetail.PickupLat,
		PickupLong:  orderDetail.PickupLong,
		DropoffLat:  orderDetail.DropoffLat,
		DropoffLong: orderDetail.DropoffLong,
		Status:      orderDetail.Status,
		Price:       orderDetail.Price,
		CreatedAt:   orderDetail.CreatedAt.Time.String(),
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
