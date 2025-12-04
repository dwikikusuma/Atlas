package service

import (
	"context"
	"errors"
	"testing"

	"github.com/dwikikusuma/atlas/pkg/pb/tracker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockEventProducer is a mock implementation of kafka.EventProducer
type MockEventProducer struct {
	mock.Mock
}

func (m *MockEventProducer) Publish(ctx context.Context, topic string, key string, value []byte) error {
	// Captures arguments and returns values defined in the test
	args := m.Called(ctx, topic, key, value)
	return args.Error(0)
}

func (m *MockEventProducer) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestUpdateLocation(t *testing.T) {
	// 1. Setup
	mockProducer := new(MockEventProducer)
	server := NewServer(mockProducer)
	ctx := context.Background()

	req := &tracker.UpdateLocationRequest{
		UserId:    "user-123",
		Latitude:  -6.2088,
		Longitude: 106.8456,
		Timestamp: "2023-10-27T10:00:00Z",
	}

	t.Run("Success", func(t *testing.T) {
		// Expect Publish to be called with specific arguments
		// mock.Anything is used for the []byte payload since marshaled JSON might vary slightly in spacing
		mockProducer.On("Publish", ctx, "location-updates", "user-123", mock.Anything).Return(nil).Once()

		// 2. Execute
		resp, err := server.UpdateLocation(ctx, req)

		// 3. Assert
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.Success)
		mockProducer.AssertExpectations(t)
	})

	t.Run("Kafka Publish Error", func(t *testing.T) {
		// Simulate Kafka being down or returning an error
		expectedErr := errors.New("kafka connection refused")
		mockProducer.On("Publish", ctx, "location-updates", "user-123", mock.Anything).Return(expectedErr).Once()

		resp, err := server.UpdateLocation(ctx, req)

		// Assert that we get an error and nil response
		assert.Error(t, err)
		assert.Nil(t, resp)

		// Optional: Check if it's a gRPC status error
		// st, ok := status.FromError(err)
		// assert.True(t, ok)
		// assert.Equal(t, codes.Internal, st.Code())

		mockProducer.AssertExpectations(t)
	})
}
