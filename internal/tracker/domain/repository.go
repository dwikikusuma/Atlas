package domain

import "context"

type LocationRepository interface {
	// UpdatePosition updates the geospatial location of a user (driver).
	UpdatePosition(ctx context.Context, userID string, lat float64, lon float64) error
}
