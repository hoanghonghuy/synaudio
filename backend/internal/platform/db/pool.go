package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/synaudio/synaudio/backend/internal/platform/config"
)

// NewPool constructs a PostgreSQL pool using the shared Synaudio pool contract.
func NewPool(ctx context.Context, databaseURL string, settings config.DatabasePoolSettings) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database pool config: %w", err)
	}

	poolConfig.MaxConns = settings.EffectiveMaxConns()
	poolConfig.MinConns = settings.MinConns
	poolConfig.MaxConnLifetime = settings.MaxConnLifetime
	poolConfig.MaxConnIdleTime = settings.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = settings.HealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}
	return pool, nil
}
