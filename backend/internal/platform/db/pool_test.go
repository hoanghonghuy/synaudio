package db_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/synaudio/synaudio/backend/internal/platform/config"
	"github.com/synaudio/synaudio/backend/internal/platform/db"
)

func TestNewPoolAppliesExplicitMaxConns(t *testing.T) {
	settings := config.DatabasePoolSettings{
		MaxConns:          7,
		MinConns:          1,
		MaxConnLifetime:   time.Hour,
		MaxConnIdleTime:   30 * time.Minute,
		HealthCheckPeriod: time.Minute,
		AcquireTimeout:    30 * time.Second,
	}

	pool, err := db.NewPool(context.Background(), "postgres://invalid:invalid@127.0.0.1:1/invalid?connect_timeout=1", settings)
	if err != nil {
		t.Fatalf("expected pool construction to succeed before connect, got %v", err)
	}
	defer pool.Close()

	if pool.Config().MaxConns != 7 {
		t.Fatalf("expected max conns 7, got %d", pool.Config().MaxConns)
	}
	if pool.Config().MinConns != 1 {
		t.Fatalf("expected min conns 1, got %d", pool.Config().MinConns)
	}
}

func TestNewPoolRejectsInvalidDatabaseURL(t *testing.T) {
	settings := config.DatabasePoolSettings{
		MaxConns:          4,
		MaxConnLifetime:   time.Hour,
		MaxConnIdleTime:   30 * time.Minute,
		HealthCheckPeriod: time.Minute,
		AcquireTimeout:    30 * time.Second,
	}

	_, err := db.NewPool(context.Background(), "not-a-url", settings)
	if err == nil || !strings.Contains(err.Error(), "parse database pool config") {
		t.Fatalf("expected parse failure, got %v", err)
	}
}
