package database

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresConfig struct {
	ConnectionURL string
	MaxConn       int32
	MinConn       int32
	MaxIdleTime   int32
	HealthCheck   bool
}

func NewPostgresPool(ctx context.Context, cfg PostgresConfig) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.ConnectionURL)
	if err != nil {
		log.Fatalf("❌ unable to parse database URL: %v", err)
		return nil, err
	}

	if cfg.MaxConn > 0 {
		poolConfig.MaxConns = cfg.MaxConn
	}

	if cfg.MinConn > 0 {
		poolConfig.MinConns = cfg.MinConn
	}

	if cfg.MaxIdleTime > 0 {
		poolConfig.MaxConnIdleTime = time.Duration(cfg.MaxIdleTime) * time.Second
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		log.Fatalf("❌ unable to create connection pool: %v", err)
		return nil, err
	}

	if cfg.HealthCheck {
		err = pool.Ping(ctx)
		if err != nil {
			pool.Close()
			log.Fatalf("❌ unable to connect to database: %v", err)
			return nil, err
		}
	}

	return pool, nil
}
