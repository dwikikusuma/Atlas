package repository

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/dwikikusuma/atlas/internal/tracker/domain"
	"github.com/dwikikusuma/atlas/pkg/model"
	"github.com/redis/go-redis/v9"
)

const (
	keyDriverPositions = "atlas:tracker:positions"
	keyDriverLastSeen  = "atlas:tracker:last_seen"
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

	_, err := r.client.GeoAdd(ctx, keyDriverPositions, &redis.GeoLocation{
		Name:      userID,
		Longitude: lon,
		Latitude:  lat,
	}).Result()

	if err != nil {
		log.Printf("redis geoAdd failed: %v", err)
		return err
	}

	err = r.client.ZAdd(ctx, keyDriverLastSeen, redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: userID,
	}).Err()

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

func (r *RedisClientRepo) GetDriverLocation(ctx context.Context, driverID string) (*model.LocationEvent, error) {
	const key = "atlas:tracker:positions"
	res, err := r.client.GeoPos(ctx, key, driverID).Result()
	if err != nil {
		log.Printf("redis geoPos failed: %v", err)
		return nil, err
	}

	if len(res) == 0 || res[0] == nil {
		return nil, errors.New("no driver found")
	}

	return &model.LocationEvent{
		UserID:    driverID,
		Longitude: res[0].Longitude,
		Latitude:  res[0].Latitude,
	}, nil
}

func (r *RedisClientRepo) RemoveStaleDrivers(ctx context.Context, ttl time.Duration) error {
	limit := time.Now().Add(-ttl).Unix()
	staleDrivers, err := r.client.ZRangeByScore(ctx, keyDriverLastSeen, &redis.ZRangeBy{
		Min: "-inf",
		Max: fmt.Sprintf("%d", limit),
	}).Result()

	if err != nil {
		log.Printf("redis ZRangeByScore failed: %v", err)
		return err
	}

	if len(staleDrivers) == 0 {
		return nil
	}

	members := make([]interface{}, len(staleDrivers))
	for i, d := range staleDrivers {
		members[i] = d
	}

	pipe := r.client.Pipeline()
	pipe.ZRem(ctx, keyDriverLastSeen, members...)
	pipe.ZRem(ctx, keyDriverPositions, members...)

	_, err = pipe.Exec(ctx)
	if err != nil {
		log.Printf("redis pipeline exec failed: %v", err)
		return err
	}

	return nil
}
