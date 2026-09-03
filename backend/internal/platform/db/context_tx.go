package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type contextTxKey struct{}

// Contextual returns a DBTX that transparently routes sqlc queries through the
// request transaction stored in context, falling back to the base executor when
// no transaction is active. This lets existing stores participate in an HTTP
// transaction without changing every store interface.
func Contextual(base DBTX) DBTX {
	return &contextualDBTX{base: base}
}

// WithTx attaches a pgx transaction to a request context for Contextual DBTX.
func WithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, contextTxKey{}, tx)
}

func txFromContext(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(contextTxKey{}).(pgx.Tx)
	return tx, ok && tx != nil
}

type contextualDBTX struct {
	base DBTX
}

func (c *contextualDBTX) executor(ctx context.Context) DBTX {
	if tx, ok := txFromContext(ctx); ok {
		return tx
	}
	return c.base
}

func (c *contextualDBTX) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	return c.executor(ctx).Exec(ctx, query, args...)
}

func (c *contextualDBTX) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	return c.executor(ctx).Query(ctx, query, args...)
}

func (c *contextualDBTX) QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
	return c.executor(ctx).QueryRow(ctx, query, args...)
}

// Begin preserves transaction support expected by persistence adapters such as
// WRITER batch establishment. Inside an audited request it starts a nested pgx
// transaction/savepoint; otherwise it delegates to the underlying pool.
func (c *contextualDBTX) Begin(ctx context.Context) (pgx.Tx, error) {
	beginner, ok := c.executor(ctx).(interface {
		Begin(context.Context) (pgx.Tx, error)
	})
	if !ok {
		return nil, errors.New("database transaction support unavailable")
	}
	return beginner.Begin(ctx)
}
