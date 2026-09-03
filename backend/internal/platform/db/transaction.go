package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// InTransaction starts a transaction (or a nested savepoint when executor is
// Contextual and parent already contains a transaction), runs the callback with
// that transaction in context, and commits only on success.
func InTransaction(parent context.Context, executor DBTX, run func(context.Context) error) error {
	beginner, ok := executor.(interface {
		Begin(context.Context) (pgx.Tx, error)
	})
	if !ok {
		return errors.New("database transaction support unavailable")
	}
	tx, err := beginner.Begin(parent)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(parent) }()
	if err := run(WithTx(parent, tx)); err != nil {
		return err
	}
	return tx.Commit(parent)
}
