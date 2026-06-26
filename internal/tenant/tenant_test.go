package tenant

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestWithOrgID(t *testing.T) {
	orgID := uuid.New()
	ctx := WithOrgID(context.Background(), orgID)

	got, err := OrgIDFrom(ctx)
	if err != nil {
		t.Fatalf("OrgIDFrom: %v", err)
	}
	if got != orgID {
		t.Errorf("got %v, want %v", got, orgID)
	}
}

func TestWithOrgIDString(t *testing.T) {
	orgID := uuid.New()
	ctx, err := WithOrgIDString(context.Background(), orgID.String())
	if err != nil {
		t.Fatalf("WithOrgIDString: %v", err)
	}

	got, err := OrgIDFrom(ctx)
	if err != nil {
		t.Fatalf("OrgIDFrom: %v", err)
	}
	if got != orgID {
		t.Errorf("got %v, want %v", got, orgID)
	}
}

func TestWithOrgIDString_InvalidUUID(t *testing.T) {
	tests := []struct {
		name string
		id   string
	}{
		{"empty string", ""},
		{"not a uuid", "not-a-uuid"},
		{"partial uuid", "550e8400-e29b-41d4"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := WithOrgIDString(context.Background(), tc.id)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), "invalid org_id") {
				t.Errorf("expected 'invalid org_id' error, got %q", err.Error())
			}
		})
	}
}

func TestOrgIDFrom_MissingFromContext(t *testing.T) {
	_, err := OrgIDFrom(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != ErrNoOrgID {
		t.Errorf("got %v, want %v", err, ErrNoOrgID)
	}
}

func TestOrgIDFrom_NilUUID(t *testing.T) {
	ctx := WithOrgID(context.Background(), uuid.Nil)
	_, err := OrgIDFrom(ctx)
	if err == nil {
		t.Fatal("expected error for nil UUID")
	}
	if err != ErrNoOrgID {
		t.Errorf("got %v, want %v", err, ErrNoOrgID)
	}
}

func TestOrgIDFrom_WrongValueType(t *testing.T) {
	type otherKey struct{}
	ctx := context.WithValue(context.Background(), otherKey{}, "not-a-uuid")
	_, err := OrgIDFrom(ctx)
	if err == nil {
		t.Fatal("expected error for wrong value type")
	}
	if err != ErrNoOrgID {
		t.Errorf("got %v, want %v", err, ErrNoOrgID)
	}
}

func TestMustOrgIDFrom_Present(t *testing.T) {
	orgID := uuid.New()
	ctx := WithOrgID(context.Background(), orgID)
	got := MustOrgIDFrom(ctx)
	if got != orgID {
		t.Errorf("got %v, want %v", got, orgID)
	}
}

func TestMustOrgIDFrom_Missing(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for missing org ID")
		}
	}()
	MustOrgIDFrom(context.Background())
}

func TestOrgIDFrom_RoundTrip(t *testing.T) {
	// Test multiple round trips to ensure consistency
	for i := 0; i < 10; i++ {
		orgID := uuid.New()
		ctx := WithOrgID(context.Background(), orgID)
		got, err := OrgIDFrom(ctx)
		if err != nil {
			t.Fatalf("OrgIDFrom iteration %d: %v", i, err)
		}
		if got != orgID {
			t.Errorf("iteration %d: got %v, want %v", i, got, orgID)
		}
	}
}

func TestWithOrgID_DoesNotModifyParent(t *testing.T) {
	parent := context.Background()
	orgID := uuid.New()
	_ = WithOrgID(parent, orgID)

	// Parent should not have the value
	_, err := OrgIDFrom(parent)
	if err != ErrNoOrgID {
		t.Errorf("parent context should not have org ID, got %v", err)
	}
}

func TestWithOrgIDString_RoundTrip(t *testing.T) {
	orgID := uuid.New()
	ctx, err := WithOrgIDString(context.Background(), orgID.String())
	if err != nil {
		t.Fatalf("WithOrgIDString: %v", err)
	}
	got, err := OrgIDFrom(ctx)
	if err != nil {
		t.Fatalf("OrgIDFrom: %v", err)
	}
	if got != orgID {
		t.Errorf("got %v, want %v", got, orgID)
	}
}
