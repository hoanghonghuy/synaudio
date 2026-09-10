package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPoolExecFailsWhenAcquireTimeoutExpires(t *testing.T) {
	originalAcquire := acquirePoolConn
	t.Cleanup(func() { acquirePoolConn = originalAcquire })

	acquirePoolConn = func(_ *pgxpool.Pool, ctx context.Context) (*pgxpool.Conn, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}

	pool := &Pool{acquireTimeout: 20 * time.Millisecond}
	started := time.Now()
	_, err := pool.Exec(context.Background(), "SELECT 1")
	elapsed := time.Since(started)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected acquire deadline exceeded, got %v", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("acquire timeout was not bounded: elapsed=%s", elapsed)
	}
}

func TestPoolQueryRowSurfacesAcquireTimeoutOnScan(t *testing.T) {
	originalAcquire := acquirePoolConn
	t.Cleanup(func() { acquirePoolConn = originalAcquire })

	acquirePoolConn = func(_ *pgxpool.Pool, ctx context.Context) (*pgxpool.Conn, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}

	pool := &Pool{acquireTimeout: 15 * time.Millisecond}
	err := pool.QueryRow(context.Background(), "SELECT 1").Scan(new(int))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected acquire deadline exceeded from QueryRow.Scan, got %v", err)
	}
}

func TestPoolBeginUsesAcquireTimeout(t *testing.T) {
	originalBegin := beginPoolTx
	t.Cleanup(func() { beginPoolTx = originalBegin })

	beginPoolTx = func(_ *pgxpool.Pool, ctx context.Context) (pgx.Tx, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}

	pool := &Pool{acquireTimeout: 20 * time.Millisecond}
	_, err := pool.Begin(context.Background())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected transaction acquire deadline exceeded, got %v", err)
	}
}

func TestPoolAcquireRespectsEarlierCallerDeadline(t *testing.T) {
	originalAcquire := acquirePoolConn
	t.Cleanup(func() { acquirePoolConn = originalAcquire })

	acquirePoolConn = func(_ *pgxpool.Pool, ctx context.Context) (*pgxpool.Conn, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}

	pool := &Pool{acquireTimeout: time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Millisecond)
	defer cancel()

	started := time.Now()
	_, err := pool.Exec(ctx, "SELECT 1")
	elapsed := time.Since(started)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected caller deadline exceeded, got %v", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("caller deadline was not preserved: elapsed=%s", elapsed)
	}
}
