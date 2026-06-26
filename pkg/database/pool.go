// Package database wraps the pgxpool connection pool and provides
// tenant-scoped transaction helpers that configure Postgres Row Level
// Security (RLS) before running application code.
package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sambawy01/Retail-Compliance-System/internal/tenant"
)

// Pool is a type alias for *pgxpool.Pool for ergonomic use in service
// constructors (e.g. identity.New(pool *database.Pool, ...)).
type Pool = pgxpool.Pool

// PoolConfig overrides pgxpool defaults if non-zero.
type PoolConfig struct {
	MaxConns int32
	MinConns int32
}

// NewPool creates and configures a pgxpool.Pool connected to databaseURL.
// The pool is already started; callers should Close it on shutdown.
func NewPool(ctx context.Context, databaseURL string, opts ...PoolConfig) (*pgxpool.Pool, error) {
	if databaseURL == "" {
		return nil, errors.New("database: databaseURL is required")
	}

	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("database: parse config: %w", err)
	}

	for _, o := range opts {
		if o.MaxConns > 0 {
			cfg.MaxConns = o.MaxConns
		}
		if o.MinConns > 0 {
			cfg.MinConns = o.MinConns
		}
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("database: new pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database: ping: %w", err)
	}

	return pool, nil
}

// TenantTx runs fn inside a transaction with the RLS session variable
// app.current_org_id set from the context. The transaction is committed if
// fn returns nil and rolled back on error. The RLS setting is RESET before
// commit so the connection returns to the pool in a clean state.
func TenantTx(ctx context.Context, pool *pgxpool.Pool, fn func(ctx context.Context, tx pgx.Tx) error) error {
	orgID, err := tenant.OrgIDFrom(ctx)
	if err != nil {
		return fmt.Errorf("database: tenant scope: %w", err)
	}

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("database: begin tx: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	// Set the RLS session variable for this transaction's connection.
	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_org_id', $1, true)", orgID.String()); err != nil {
		return fmt.Errorf("database: set rls config: %w", err)
	}

	if err := fn(ctx, tx); err != nil {
		return err
	}

	// Reset RLS before returning the connection to the pool.
	if _, err := tx.Exec(ctx, "RESET app.current_org_id"); err != nil {
		// Non-fatal: log via ctx? Keep simple — ignore reset errors on commit path.
		_ = err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("database: commit: %w", err)
	}
	committed = true
	return nil
}

// Querier is the minimal query surface shared by *pgxpool.Pool and pgx.Tx,
// so repository code can accept either without code changes.
type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// WithTx runs fn with the given tx. Kept for manual tx control.
func WithTx(tx pgx.Tx, fn func(tx pgx.Tx) error) error {
	return fn(tx)
}