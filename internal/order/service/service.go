package service

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"github.com/dwikikusuma/atlas/internal/order/db"
	"github.com/dwikikusuma/atlas/pkg/kafka"
	orderModel "github.com/dwikikusuma/atlas/pkg/model"
	"github.com/dwikikusuma/atlas/pkg/pb/order"
	wallet "github.com/dwikikusuma/atlas/pkg/pb/wallet"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	order.UnimplementedOrderServiceServer
	store        db.Querier
	producer     *kafka.Producer
	walletClient wallet.WalletServiceClient
}

func NewOrderService(store db.Querier, producer *kafka.Producer, walletClient wallet.WalletServiceClient) *Service {
	return &Service{
		store:        store,
		producer:     producer,
		walletClient: walletClient,
	}
}

func (s *Service) CreateOrder(ctx context.Context, req *order.CreateOrderRequest) (*order.CreateOrderResponse, error) {
	price := calculatePrice(req.PickupLat, req.PickupLong, req.DropoffLat, req.DropoffLong)
	balance, err := s.walletClient.GetBalance(ctx, &wallet.GetBalanceRequest{UserId: req.UserId})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user balance: %v", err)
	}

	if balance.Balance < price {
		return nil, status.Errorf(codes.FailedPrecondition, "insufficient balance: %v", balance.Balance)
	}

	createOrderParams := db.CreateOrderParams{
		PassengerID: req.UserId,
		PickupLong:  req.PickupLong,
		PickupLat:   req.PickupLat,
		DropoffLat:  req.DropoffLat,
		DropoffLong: req.DropoffLong,
		Status:      "CREATED",
		Price:       price,
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
		DriverId:    orderDetail.DriverID.String,
		PickupLat:   orderDetail.PickupLat,
		PickupLong:  orderDetail.PickupLong,
		DropoffLat:  orderDetail.DropoffLat,
		DropoffLong: orderDetail.DropoffLong,
		Status:      orderDetail.Status,
		Price:       orderDetail.Price,
		CreatedAt:   orderDetail.CreatedAt.Time.String(),
	}, nil
}

func (s *Service) UpdateOrderStatus(ctx context.Context, req *order.UpdateOrderStatusRequest) (*order.UpdateOrderStatusResponse, error) {
	if req.Status != "STARTED" && req.Status != "FINISHED" {
		return nil, status.Errorf(codes.InvalidArgument, "invalid status: %s", req.Status)
	}

	var orderID pgtype.UUID
	if err := orderID.Scan(req.OrderId); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order ID")
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	args := db.UpdateOrderStatusParams{
		ID:     orderID,
		Status: req.Status,
	}
	if err := s.store.UpdateOrderStatus(dbCtx, args); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if req.Status == "FINISHED" {
		if err := s.ProcessPayment(dbCtx, orderID); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to process payment: %v", err)
		}
	}

	return &order.UpdateOrderStatusResponse{
		OrderId:   orderID.String(),
		Status:    req.Status,
		UpdatedAt: time.Now().UTC().String(),
	}, nil
}

func (s *Service) ProcessPayment(dbCtx context.Context, orderID pgtype.UUID) error {
	orderDetail, err := s.store.GetOrder(dbCtx, orderID)
	if err != nil {
		return err
	}

	orderString := orderID.String()
	debitEvent := orderModel.DebitBalanceEvent{
		Amount:    orderDetail.Price,
		UserID:    orderDetail.PassengerID,
		Reference: orderString,
	}

	debitByte, err := json.Marshal(&debitEvent)
	if err != nil {
		return err
	}

	if err = s.producer.Publish(dbCtx, "wallet-transactions", orderString, debitByte); err != nil {
		return err
	}
	return nil
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
