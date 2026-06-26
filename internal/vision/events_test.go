package vision

import (
	"github.com/google/uuid"
	"testing"
)

func TestAllSubjects(t *testing.T) {
	subs := AllSubjects()
	if len(subs) != 16 {
		t.Errorf("expected 16 subjects, got %d", len(subs))
	}
}

func TestAllSubjects_ReturnsCopy(t *testing.T) {
	subs1 := AllSubjects()
	subs2 := AllSubjects()
	if len(subs1) != len(subs2) {
		t.Fatalf("lengths differ: %d vs %d", len(subs1), len(subs2))
	}
	// Modify copy, ensure original is unchanged
	subs1[0] = "modified.subject"
	subs3 := AllSubjects()
	if subs3[0] == "modified.subject" {
		t.Error("AllSubjects should return a copy")
	}
}

func TestAllSubjects_NoDuplicates(t *testing.T) {
	subs := AllSubjects()
	seen := make(map[string]bool)
	for _, s := range subs {
		if seen[s] {
			t.Errorf("duplicate subject: %s", s)
		}
		seen[s] = true
	}
}

func TestEventConstants(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		// Critical events
		{"EventSafetySlipFall", EventSafetySlipFall, "vision.safety.slip_fall"},
		{"EventTheftCashDrawer", EventTheftCashDrawer, "vision.theft.cash_drawer"},
		{"EventAccessAfterHours", EventAccessAfterHours, "vision.access.after_hours"},
		{"EventLaborBuddyPunch", EventLaborBuddyPunch, "vision.labor.buddy_punch"},
		{"EventComplianceBlockedExit", EventComplianceBlockedExit, "vision.compliance.blocked_exit"},

		// Warning events
		{"EventComplianceUniformViolation", EventComplianceUniformViolation, "vision.compliance.uniform_violation"},
		{"EventComplianceHygieneViolation", EventComplianceHygieneViolation, "vision.compliance.hygiene_violation"},
		{"EventCompliancePhoneUsage", EventCompliancePhoneUsage, "vision.compliance.phone_usage"},
		{"EventComplianceCleanlinessAlert", EventComplianceCleanlinessAlert, "vision.compliance.cleanliness_alert"},
		{"EventOperationsCheckoutBottleneck", EventOperationsCheckoutBottleneck, "vision.operations.checkout_bottleneck"},
		{"EventInventoryStockroomAnomaly", EventInventoryStockroomAnomaly, "vision.inventory.stockroom_anomaly"},
		{"EventSecurityLoitering", EventSecurityLoitering, "vision.security.loitering"},
		{"EventCameraDegraded", EventCameraDegraded, "vision.camera.degraded"},

		// Info events
		{"EventCustomerLoyaltyRecognized", EventCustomerLoyaltyRecognized, "vision.customer.loyalty_recognized"},
		{"EventOccupancyUpdate", EventOccupancyUpdate, "vision.occupancy.update"},
		{"EventActivityUpdate", EventActivityUpdate, "vision.activity.update"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.value != tc.want {
				t.Errorf("got %q, want %q", tc.value, tc.want)
			}
		})
	}
}

func TestEventSubjects_Count(t *testing.T) {
	// 15 named retail events + 1 activity update = 16 total
	if len(allSubjects) != 16 {
		t.Errorf("expected 16 total subjects, got %d", len(allSubjects))
	}
}

func TestOccupancyPayload(t *testing.T) {
	p := OccupancyPayload{
		ZoneID:      uuid.New(),
		Count:       42,
		Capacity:    100,
		ChangeDelta: 3,
	}
	if p.Count != 42 {
		t.Errorf("Count: got %d, want %d", p.Count, 42)
	}
	if p.Capacity != 100 {
		t.Errorf("Capacity: got %d, want %d", p.Capacity, 100)
	}
	if p.ChangeDelta != 3 {
		t.Errorf("ChangeDelta: got %d, want %d", p.ChangeDelta, 3)
	}
}

func TestCompliancePayload(t *testing.T) {
	staffID := uuid.New()
	zoneID := uuid.New()
	p := CompliancePayload{
		ViolationType: "phone_usage",
		Description:   "Staff using phone at checkout",
		StaffID:       &staffID,
		ZoneID:        &zoneID,
		Bbox:          []float64{0.1, 0.2, 0.3, 0.4},
	}
	if p.ViolationType != "phone_usage" {
		t.Errorf("ViolationType: got %q, want %q", p.ViolationType, "phone_usage")
	}
	if p.StaffID == nil || *p.StaffID != staffID {
		t.Errorf("StaffID mismatch")
	}
	if len(p.Bbox) != 4 {
		t.Errorf("Bbox length: got %d, want 4", len(p.Bbox))
	}
}

func TestAnomalyPayload(t *testing.T) {
	p := AnomalyPayload{
		AnomalyKind: "stockout",
		Metric:      "quantity",
		Value:       0,
		Threshold:   5,
	}
	if p.AnomalyKind != "stockout" {
		t.Errorf("AnomalyKind: got %q, want %q", p.AnomalyKind, "stockout")
	}
	if p.Value != 0 {
		t.Errorf("Value: got %f, want %f", p.Value, 0.0)
	}
}

func TestCustomerRecognitionPayload(t *testing.T) {
	p := CustomerRecognitionPayload{
		CustomerRef: "cust-123",
		LoyaltyTier:  "gold",
		VisitCount:   15,
	}
	if p.CustomerRef != "cust-123" {
		t.Errorf("CustomerRef: got %q, want %q", p.CustomerRef, "cust-123")
	}
	if p.LoyaltyTier != "gold" {
		t.Errorf("LoyaltyTier: got %q, want %q", p.LoyaltyTier, "gold")
	}
}

func TestRetailEventPayload(t *testing.T) {
	p := RetailEventPayload{
		Subject:     EventSecurityLoitering,
		Description: "Person loitering near entrance",
		Bbox:        []float64{0.1, 0.2, 0.3, 0.4},
		Extra:       map[string]any{"duration": 300},
	}
	if p.Subject != EventSecurityLoitering {
		t.Errorf("Subject: got %q, want %q", p.Subject, EventSecurityLoitering)
	}
	if p.Extra["duration"] != 300 {
		t.Errorf("Extra[duration]: got %v, want %v", p.Extra["duration"], 300)
	}
}
