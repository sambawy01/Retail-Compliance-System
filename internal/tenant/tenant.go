// Package tenant provides context-based tenant (organization) isolation.
// The org ID is propagated through the request context and used to scope
// database access via Row Level Security (RLS).
package tenant

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// ctxKey is an unexported type so the context key is unique to this package.
type ctxKey struct{}

// ErrNoOrgID is returned when an org ID is not present in the context.
var ErrNoOrgID = errors.New("tenant: org_id not found in context")

// WithOrgID returns a copy of ctx that carries the given org ID. An empty or
// zero UUID is treated as missing and panics to surface programmer error early.
func WithOrgID(ctx context.Context, orgID uuid.UUID) context.Context {
	return context.WithValue(ctx, ctxKey{}, orgID)
}

// WithOrgIDString is a convenience wrapper that parses a string org ID.
func WithOrgIDString(ctx context.Context, orgID string) (context.Context, error) {
	id, err := uuid.Parse(orgID)
	if err != nil {
		return ctx, fmt.Errorf("tenant: invalid org_id %q: %w", orgID, err)
	}
	return WithOrgID(ctx, id), nil
}

// OrgIDFrom extracts the org ID from the context. Returns ErrNoOrgID when it
// is absent.
func OrgIDFrom(ctx context.Context) (uuid.UUID, error) {
	v, ok := ctx.Value(ctxKey{}).(uuid.UUID)
	if !ok || v == uuid.Nil {
		return uuid.Nil, ErrNoOrgID
	}
	return v, nil
}

// MustOrgIDFrom is like OrgIDFrom but panics on error. Use only where the org
// ID is guaranteed to be present (e.g. inside a TenantTx after RLS setup).
func MustOrgIDFrom(ctx context.Context) uuid.UUID {
	id, err := OrgIDFrom(ctx)
	if err != nil {
		panic(err)
	}
	return id
}