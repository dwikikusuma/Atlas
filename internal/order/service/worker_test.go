package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	dispatchModel "github.com/dwikikusuma/atlas/internal/dispatch/model"
	"github.com/dwikikusuma/atlas/internal/order/db"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/mock"
)

// --- Mocks ---

type MockConsumer struct {
	mock.Mock
}

func (m *MockConsumer) FetchMessage(ctx context.Context) (kafka.Message, error) {
	args := m.Called(ctx)
	return args.Get(0).(kafka.Message), args.Error(1)
}

func (m *MockConsumer) CommitMessages(ctx context.Context, msgs ...kafka.Message) error {
	args := m.Called(ctx, msgs)
	return args.Error(0)
}

func (m *MockConsumer) Close() error { return nil }

type MockStore struct {
	db.Querier // Embed the interface to skip implementing all methods
	mock.Mock
}

func (m *MockStore) UpdateOrderDriver(ctx context.Context, arg db.UpdateOrderDriverParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

// --- Test ---

func TestOrderWorker_ProcessMatch(t *testing.T) {
	mockConsumer := new(MockConsumer)
	mockStore := new(MockStore)
	worker := NewOrderWorker(mockConsumer, mockStore)

	// 1. Setup Data
	orderID := "550e8400-e29b-41d4-a716-446655440000" // Valid UUID
	driverID := "driver-123"

	event := dispatchModel.RideDispatchedEvent{
		RideID:   orderID,
		DriverID: driverID,
	}
	eventBytes, _ := json.Marshal(event)

	msg := kafka.Message{
		Key:   []byte(orderID),
		Value: eventBytes,
	}

	// 2. Expectations
	// Expect FetchMessage to be called once and return our msg
	mockConsumer.On("FetchMessage", mock.Anything).Return(msg, nil).Once()

	// Expect DB Update with correct UUID and DriverID
	mockStore.On("UpdateOrderDriver", mock.Anything, mock.MatchedBy(func(arg db.UpdateOrderDriverParams) bool {
		// Verify UUID conversion worked
		validID := arg.ID.Bytes != [16]byte{}
		validDriver := arg.DriverID.String == driverID
		return validID && validDriver
	})).Return(nil)

	// Expect Commit
	mockConsumer.On("CommitMessages", mock.Anything, mock.Anything).Return(nil)

	// Expect second fetch to block/cancel (to stop the loop in test)
	mockConsumer.On("FetchMessage", mock.Anything).After(1*time.Second).Return(kafka.Message{}, context.Canceled)

	// 3. Execute (Run for a short time)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_ = worker.Start(ctx)

	// 4. Verify
	mockStore.AssertExpectations(t)
	mockConsumer.AssertExpectations(t)
}
