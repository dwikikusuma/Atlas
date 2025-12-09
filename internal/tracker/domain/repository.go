package domain

import (
	"context"

	"github.com/dwikikusuma/atlas/internal/tracker/model"
)

type LocationRepository interface {
	// UpdatePosition updates the geospatial location of a user (driver).
	UpdatePosition(ctx context.Context, userID string, lat float64, lon float64) error

	GetNearbyDrivers(ctx context.Context, lat float64, lon float64, radius float64) ([]model.LocationEvent, error)

	GetDriverLocation(ctx context.Context, driverID string) (*model.LocationEvent, error)
}
