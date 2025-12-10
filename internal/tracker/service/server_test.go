package service

import (
	"context"
	"errors"
	"testing"

	"github.com/dwikikusuma/atlas/internal/tracker/model"
	"github.com/dwikikusuma/atlas/pkg/pb/tracker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// =============================================================================
// MOCKS (Same as before)
// =============================================================================

type MockEventProducer struct {
	mock.Mock
}

func (m *MockEventProducer) Publish(ctx context.Context, topic string, key string, value []byte) error {
	args := m.Called(ctx, topic, key, value)
	return args.Error(0)
}

func (m *MockEventProducer) Close() error {
	args := m.Called()
	return args.Error(0)
}

type MockLocationRepository struct {
	mock.Mock
}

func (m *MockLocationRepository) UpdatePosition(ctx context.Context, userID string, lat float64, lon float64) error {
	args := m.Called(ctx, userID, lat, lon)
	return args.Error(0)
}

func (m *MockLocationRepository) GetNearbyDrivers(ctx context.Context, lat float64, lon float64, radius float64) ([]model.LocationEvent, error) {
	args := m.Called(ctx, lat, lon, radius)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.LocationEvent), args.Error(1)
}

func (m *MockLocationRepository) GetDriverLocation(ctx context.Context, driverID string) (*model.LocationEvent, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.LocationEvent), args.Error(1)
}

// =============================================================================
// TESTS WITH LOGGING
// =============================================================================

func TestUpdateLocation(t *testing.T) {
	mockProducer := new(MockEventProducer)
	mockRepo := new(MockLocationRepository)
	server := NewServer(mockProducer, mockRepo)
	ctx := context.Background()

	req := &tracker.UpdateLocationRequest{
		UserId:    "user-123",
		Latitude:  -6.2088,
		Longitude: 106.8456,
		Timestamp: "2023-10-27T10:00:00Z",
	}

	t.Run("Success", func(t *testing.T) {
		t.Logf("🧪 [SCENARIO]: Driver Sends Location Update")
		t.Logf("📝 INPUT: UserID=%s Lat=%f Long=%f", req.UserId, req.Latitude, req.Longitude)

		// Expectation
		mockProducer.On("Publish", ctx, "driver-gps", "user-123", mock.Anything).Return(nil).Once()

		// Execution
		resp, err := server.UpdateLocation(ctx, req)

		// Verification
		if err != nil {
			t.Logf("❌ RESULT: FAILED with error: %v", err)
		} else {
			t.Logf("✅ RESULT: Success=%v", resp.Success)
		}

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.Success)
		mockProducer.AssertExpectations(t)
	})

	t.Run("Kafka Failure", func(t *testing.T) {
		t.Logf("🧪 [SCENARIO]: Kafka Broker is Down")
		t.Logf("📝 INPUT: UserID=%s", req.UserId)

		mockProducer.On("Publish", ctx, "driver-gps", "user-123", mock.Anything).Return(errors.New("kafka error")).Once()

		resp, err := server.UpdateLocation(ctx, req)

		t.Logf("⚠️ EXPECTED ERROR received: %v", err)

		assert.Error(t, err)
		assert.Nil(t, resp)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.Internal, st.Code())
		mockProducer.AssertExpectations(t)
	})
}

func TestGetNearbyDrivers(t *testing.T) {
	mockProducer := new(MockEventProducer)
	mockRepo := new(MockLocationRepository)
	server := NewServer(mockProducer, mockRepo)
	ctx := context.Background()

	req := &tracker.GetNearbyDriverRequest{
		Latitude:  -6.2,
		Longitude: 106.8,
		Radius:    5.0,
	}

	t.Run("Success", func(t *testing.T) {
		t.Logf("🧪 [SCENARIO]: Search Nearby Drivers")
		t.Logf("📝 INPUT: Center=(%f, %f) Radius=%f km", req.Latitude, req.Longitude, req.Radius)

		mockData := []model.LocationEvent{
			{UserID: "driver-1", Latitude: -6.21, Longitude: 106.81},
			{UserID: "driver-2", Latitude: -6.22, Longitude: 106.82},
		}

		mockRepo.On("GetNearbyDrivers", ctx, req.Latitude, req.Longitude, req.Radius).Return(mockData, nil).Once()

		resp, err := server.GetNearbyDrivers(ctx, req)

		t.Logf("✅ RESULT: Found %d drivers", len(resp.Drivers))
		for i, d := range resp.Drivers {
			t.Logf("   📍 [%d] DriverID: %s at (%f, %f)", i+1, d.DriverId, d.Latitude, d.Longitude)
		}

		assert.NoError(t, err)
		assert.Len(t, resp.Drivers, 2)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Repo Failure", func(t *testing.T) {
		t.Logf("🧪 [SCENARIO]: Redis GeoSearch Fails")

		mockRepo.On("GetNearbyDrivers", ctx, req.Latitude, req.Longitude, req.Radius).Return(nil, errors.New("redis error")).Once()

		resp, err := server.GetNearbyDrivers(ctx, req)

		t.Logf("⚠️ EXPECTED ERROR: %v", err)

		assert.Error(t, err)
		assert.Nil(t, resp)
		mockRepo.AssertExpectations(t)
	})
}

func TestGetDriverLocation(t *testing.T) {
	mockProducer := new(MockEventProducer)
	mockRepo := new(MockLocationRepository)
	server := NewServer(mockProducer, mockRepo)
	ctx := context.Background()

	req := &tracker.GetDriverLocationRequest{DriverId: "driver-99"}

	t.Run("Success", func(t *testing.T) {
		t.Logf("🧪 [SCENARIO]: Get Specific Driver Location")
		t.Logf("📝 INPUT: DriverID=%s", req.DriverId)

		mockData := &model.LocationEvent{UserID: "driver-99", Latitude: -6.5, Longitude: 106.5}
		mockRepo.On("GetDriverLocation", ctx, "driver-99").Return(mockData, nil).Once()

		resp, err := server.GetDriverLocation(ctx, req)

		t.Logf("✅ RESULT: Driver found at (%f, %f)", resp.Latitude, resp.Longitude)

		assert.NoError(t, err)
		assert.Equal(t, "driver-99", resp.DriverId)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Driver Not Found", func(t *testing.T) {
		t.Logf("🧪 [SCENARIO]: Driver Does Not Exist")
		t.Logf("📝 INPUT: DriverID=%s", req.DriverId)

		mockRepo.On("GetDriverLocation", ctx, "driver-99").Return(nil, nil).Once()

		_, err := server.GetDriverLocation(ctx, req)

		t.Logf("⚠️ EXPECTED ERROR (404 Not Found): %v", err)

		assert.Error(t, err)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.NotFound, st.Code())
		mockRepo.AssertExpectations(t)
	})
}
