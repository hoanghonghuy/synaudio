package db

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/synaudio/synaudio/backend/internal/platform/config"
)

var acquirePoolConn = func(pool *pgxpool.Pool, ctx context.Context) (*pgxpool.Conn, error) {
	return pool.Acquire(ctx)
}

var beginPoolTx = func(pool *pgxpool.Pool, ctx context.Context) (pgx.Tx, error) {
	return pool.Begin(ctx)
}

// Pool wraps pgxpool with an acquire-only timeout while preserving the normal
// pgx execution context after a connection has been obtained. This prevents
// saturation from waiting forever without turning DATABASE_POOL_ACQUIRE_TIMEOUT
// into a query-duration timeout.
type Pool struct {
	*pgxpool.Pool
	acquireTimeout config.DatabasePoolSettings
}

// NewPool constructs a PostgreSQL pool using the shared Synaudio pool contract.
func NewPool(ctx context.Context, databaseURL string, settings config.DatabasePoolSettings) (*Pool, error) {
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
	return &Pool{Pool: pool, acquireTimeout: settings}, nil
}

func (p *Pool) acquire(ctx context.Context) (*pgxpool.Conn, error) {
	acquireCtx, cancel := context.WithTimeout(ctx, p.acquireTimeout.AcquireTimeout)
	defer cancel()
	conn, err := acquirePoolConn(p.Pool, acquireCtx)
	if err != nil {
		return nil, fmt.Errorf("acquire database connection: %w", err)
	}
	return conn, nil
}

func (p *Pool) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	conn, err := p.acquire(ctx)
	if err != nil {
		return pgconn.CommandTag{}, err
	}
	defer conn.Release()
	return conn.Exec(ctx, query, args...)
}

func (p *Pool) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	conn, err := p.acquire(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := conn.Query(ctx, query, args...)
	if err != nil {
		conn.Release()
		return nil, err
	}
	return &releasingRows{Rows: rows, release: conn.Release}, nil
}

func (p *Pool) QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
	conn, err := p.acquire(ctx)
	if err != nil {
		return errorRow{err: err}
	}
	return &releasingRow{Row: conn.QueryRow(ctx, query, args...), release: conn.Release}
}

// Begin applies the same bounded acquire contract to transaction establishment.
// The returned pgxpool transaction retains ownership of its acquired connection
// until Commit/Rollback, so canceling the acquisition context here is safe.
func (p *Pool) Begin(ctx context.Context) (pgx.Tx, error) {
	acquireCtx, cancel := context.WithTimeout(ctx, p.acquireTimeout.AcquireTimeout)
	defer cancel()
	tx, err := beginPoolTx(p.Pool, acquireCtx)
	if err != nil {
		return nil, fmt.Errorf("begin database transaction: %w", err)
	}
	return tx, nil
}

type releasingRows struct {
	pgx.Rows
	release func()
	once    sync.Once
}

func (r *releasingRows) releaseOnce() {
	r.once.Do(r.release)
}

func (r *releasingRows) Close() {
	r.Rows.Close()
	r.releaseOnce()
}

func (r *releasingRows) Next() bool {
	ok := r.Rows.Next()
	if !ok {
		r.releaseOnce()
	}
	return ok
}

func (r *releasingRows) Scan(dest ...any) error {
	err := r.Rows.Scan(dest...)
	if err != nil {
		r.releaseOnce()
	}
	return err
}

func (r *releasingRows) Values() ([]any, error) {
	values, err := r.Rows.Values()
	if err != nil {
		r.releaseOnce()
	}
	return values, err
}

type releasingRow struct {
	pgx.Row
	release func()
	once    sync.Once
}

func (r *releasingRow) Scan(dest ...any) error {
	defer r.once.Do(r.release)
	return r.Row.Scan(dest...)
}

type errorRow struct {
	err error
}

func (r errorRow) Scan(...any) error {
	return r.err
}
