package repository

import (
	"context"
	"log"

	"github.com/dwikikusuma/atlas/internal/tracker/domain"
	"github.com/dwikikusuma/atlas/internal/tracker/model"
	"github.com/redis/go-redis/v9"
)

type RedisClientRepo struct {
	client *redis.Client
}

func NewRedisClientRepo(client *redis.Client) domain.LocationRepository {
	return &RedisClientRepo{
		client: client,
	}
}

func (r *RedisClientRepo) UpdatePosition(ctx context.Context, userID string, lat float64, lon float64) error {
	const key = "atlas:tracker:positions"

	_, err := r.client.GeoAdd(ctx, key, &redis.GeoLocation{
		Name:      userID,
		Longitude: lon,
		Latitude:  lat,
	}).Result()

	if err != nil {
		log.Printf("redis geoAdd failed: %v", err)
		return err
	}

	return err
}

func (r *RedisClientRepo) GetNearbyDrivers(ctx context.Context, lat float64, lon float64, radius float64) ([]model.LocationEvent, error) {
	const key = "atlas:tracker:positions"

	res, err := r.client.GeoSearchLocation(ctx, key, &redis.GeoSearchLocationQuery{
		GeoSearchQuery: redis.GeoSearchQuery{
			Longitude:  lon,
			Latitude:   lat,
			Radius:     radius,
			RadiusUnit: "km",
			Count:      10,
			Sort:       "ASC",
		},
		WithCoord: true,
	}).Result()

	if err != nil {
		log.Printf("redis geoSearch failed: %v", err)
		return nil, err
	}

	var drivers []model.LocationEvent
	for _, loc := range res {
		drivers = append(drivers, model.LocationEvent{
			UserID:    loc.Name,
			Longitude: loc.Longitude,
			Latitude:  loc.Latitude,
		})
	}

	return drivers, nil
}
