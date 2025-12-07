package repository

import (
	"context"
	"log"

	"github.com/dwikikusuma/atlas/internal/tracker/domain"
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
