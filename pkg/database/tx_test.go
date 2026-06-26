package database

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestInTx_NilDB(t *testing.T) {
	err := InTx(context.Background(), nil, func(ctx context.Context, tx pgx.Tx) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error for nil db")
	}
	if err != ErrNoTx {
		t.Errorf("got %v, want %v", err, ErrNoTx)
	}
}

func TestErrNoTx(t *testing.T) {
	err := ErrNoTx
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if err.Error() != "database: nil transaction" {
		t.Errorf("error message: got %q, want %q", err.Error(), "database: nil transaction")
	}
}

func TestInTx_BeginError(t *testing.T) {
	// A nil TxRunner is caught by the nil check,
	// but a non-nil interface that returns an error from BeginTx
	// would propagate. We test the nil check path here.
	err := InTx(context.Background(), nil, func(ctx context.Context, tx pgx.Tx) error {
		t.Fatal("fn should not be called")
		return nil
	})
	if err != ErrNoTx {
		t.Errorf("got %v, want %v", err, ErrNoTx)
	}
}

func TestWithTx(t *testing.T) {
	// WithTx just calls fn, so it should return whatever fn returns
	called := false
	err := WithTx(nil, func(tx pgx.Tx) error {
		called = true
		return nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("fn was not called")
	}
}

func TestWithTx_PassesError(t *testing.T) {
	sentinelErr := context.DeadlineExceeded
	err := WithTx(nil, func(tx pgx.Tx) error {
		return sentinelErr
	})
	if err != sentinelErr {
		t.Errorf("got %v, want %v", err, sentinelErr)
	}
}
