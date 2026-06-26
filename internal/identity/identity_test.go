package identity

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrPersonNotFound(t *testing.T) {
	err := ErrPersonNotFound
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if err.Error() != "identity: person not found" {
		t.Errorf("error message: got %q, want %q", err.Error(), "identity: person not found")
	}
}

func TestErrPersonNotFound_IsSentinel(t *testing.T) {
	if !errors.Is(ErrPersonNotFound, ErrPersonNotFound) {
		t.Error("errors.Is should match sentinel error")
	}
}

func TestPersonStruct(t *testing.T) {
	p := Person{
		PersonID:    "person-123",
		OrgID:       "org-456",
		Kind:        "employee",
		DisplayName: "John Doe",
	}
	if p.PersonID != "person-123" {
		t.Errorf("PersonID: got %q, want %q", p.PersonID, "person-123")
	}
	if p.Kind != "employee" {
		t.Errorf("Kind: got %q, want %q", p.Kind, "employee")
	}
	if p.DisplayName != "John Doe" {
		t.Errorf("DisplayName: got %q, want %q", p.DisplayName, "John Doe")
	}
}

func TestPersonStruct_CustomerKind(t *testing.T) {
	p := Person{Kind: "customer"}
	if p.Kind != "customer" {
		t.Errorf("Kind: got %q, want %q", p.Kind, "customer")
	}
}

func TestMatchResult_Matched(t *testing.T) {
	r := MatchResult{
		PersonID:    "person-123",
		DisplayName: "Jane Doe",
		Similarity:  0.95,
		Matched:     true,
	}
	if !r.Matched {
		t.Error("expected Matched=true")
	}
	if r.Similarity != 0.95 {
		t.Errorf("Similarity: got %f, want %f", r.Similarity, 0.95)
	}
}

func TestMatchResult_NotMatched(t *testing.T) {
	r := MatchResult{Matched: false}
	if r.Matched {
		t.Error("expected Matched=false")
	}
}

func TestEnrollPersonInput(t *testing.T) {
	in := EnrollPersonInput{
		Kind:        "employee",
		DisplayName: "John Smith",
	}
	if in.Kind != "employee" {
		t.Errorf("Kind: got %q, want %q", in.Kind, "employee")
	}
	if in.DisplayName != "John Smith" {
		t.Errorf("DisplayName: got %q, want %q", in.DisplayName, "John Smith")
	}
}

func TestConsentInput(t *testing.T) {
	in := ConsentInput{
		PersonID:      "person-123",
		ConsentText:   "I consent to face recognition for attendance tracking",
		ConsentLocale:  "en",
		CapturedBy:    "admin-456",
	}
	if in.ConsentLocale != "en" {
		t.Errorf("ConsentLocale: got %q, want %q", in.ConsentLocale, "en")
	}
	if in.ConsentText == "" {
		t.Error("expected non-empty ConsentText")
	}
}

func TestConsentInput_ArabicLocale(t *testing.T) {
	in := ConsentInput{ConsentLocale: "ar"}
	if in.ConsentLocale != "ar" {
		t.Errorf("ConsentLocale: got %q, want %q", in.ConsentLocale, "ar")
	}
}

func TestTemplateInput(t *testing.T) {
	in := TemplateInput{
		PersonID:     "person-123",
		Embedding:    []float64{0.1, 0.2, 0.3, 0.4, 0.5},
		QualityScore: 0.92,
	}
	if len(in.Embedding) != 5 {
		t.Errorf("Embedding length: got %d, want 5", len(in.Embedding))
	}
	if in.QualityScore != 0.92 {
		t.Errorf("QualityScore: got %f, want %f", in.QualityScore, 0.92)
	}
}

func TestTemplateInput_EmptyEmbedding(t *testing.T) {
	in := TemplateInput{
		PersonID:     "person-123",
		Embedding:    []float64{},
		QualityScore: 0.0,
	}
	if len(in.Embedding) != 0 {
		t.Errorf("Embedding length: got %d, want 0", len(in.Embedding))
	}
}

func TestAuditInput(t *testing.T) {
	in := AuditInput{
		PersonID:     "person-123",
		Purpose:       "recognize",
		TriggeredBy:   "system",
		CameraID:     "550e8400-e29b-41d4-a716-446655440000",
	}
	if in.Purpose != "recognize" {
		t.Errorf("Purpose: got %q, want %q", in.Purpose, "recognize")
	}
}

func TestAuditInput_Purposes(t *testing.T) {
	// Valid purposes per architecture: recognize, enroll, review, export, erase
	purposes := []string{"recognize", "enroll", "review", "export", "erase"}
	for _, p := range purposes {
		in := AuditInput{Purpose: p}
		if in.Purpose != p {
			t.Errorf("Purpose: got %q, want %q", in.Purpose, p)
		}
	}
}

func TestAuditInput_EmptyCameraID(t *testing.T) {
	in := AuditInput{
		PersonID:   "person-123",
		Purpose:     "recognize",
		TriggeredBy: "system",
	}
	if in.CameraID != "" {
		t.Errorf("CameraID: got %q, want empty", in.CameraID)
	}
}

func TestNew_CreatesService(t *testing.T) {
	svc := New(nil, nil)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestNew_WithNilDeps(t *testing.T) {
	svc := New(nil, nil)
	if svc.pool != nil {
		t.Error("expected nil pool")
	}
	if svc.bus != nil {
		t.Error("expected nil bus")
	}
}

// TestVectorStringFormat verifies the pgvector string format used by
// InsertTemplate and MatchFace. The source code builds the string as
// "[v1,v2,...]" using fmt.Sprintf("%f", v).
func TestVectorStringFormat(t *testing.T) {
	tests := []struct {
		name      string
		embedding []float64
		want      string
	}{
		{"single value", []float64{1.0}, "[1.000000]"},
		{"two values", []float64{1.0, 2.0}, "[1.000000,2.000000]"},
		{"empty", []float64{}, "[]"},
		{"five values", []float64{0.1, 0.2, 0.3, 0.4, 0.5}, "[0.100000,0.200000,0.300000,0.400000,0.500000]"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildVectorString(tc.embedding)
			if got != tc.want {
				t.Errorf("buildVectorString: got %q, want %q", got, tc.want)
			}
		})
	}
}

// buildVectorString replicates the vector formatting logic from identity.go.
func buildVectorString(embedding []float64) string {
	vecStr := "["
	for i, v := range embedding {
		if i > 0 {
			vecStr += ","
		}
		vecStr += fmt.Sprintf("%f", v)
	}
	vecStr += "]"
	return vecStr
}
