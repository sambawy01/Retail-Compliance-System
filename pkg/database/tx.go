// Package database — tx.go provides a thin transaction abstraction for use
// outside of TenantTx (e.g. admin/maintenance paths that don't need RLS).
package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNoTx is returned when a nil tx is passed to a helper.
var ErrNoTx = errors.New("database: nil transaction")

// TxRunner abstracts BeginTx for tests. *pgxpool.Pool satisfies it.
type TxRunner interface {
	BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error)
}

var _ TxRunner = (*pgxpool.Pool)(nil)

// InTx runs fn inside a transaction (no RLS). Commits on nil error, rolls
// back otherwise. Use TenantTx for tenant-scoped work.
func InTx(ctx context.Context, db TxRunner, fn func(ctx context.Context, tx pgx.Tx) error) error {
	if db == nil {
		return ErrNoTx
	}
	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("database: begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	if err := fn(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("database: commit: %w", err)
	}
	committed = true
	return nil
}