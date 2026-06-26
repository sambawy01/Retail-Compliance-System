package vision

import (
	"testing"
)

func TestResolveSeverity(t *testing.T) {
	tests := []struct {
		name    string
		subject string
		want    Severity
	}{
		// Critical events
		{"slip_fall", EventSafetySlipFall, SeverityCritical},
		{"cash_drawer", EventTheftCashDrawer, SeverityCritical},
		{"after_hours", EventAccessAfterHours, SeverityCritical},
		{"buddy_punch", EventLaborBuddyPunch, SeverityCritical},
		{"blocked_exit", EventComplianceBlockedExit, SeverityCritical},

		// Warning events
		{"uniform_violation", EventComplianceUniformViolation, SeverityWarning},
		{"hygiene_violation", EventComplianceHygieneViolation, SeverityWarning},
		{"phone_usage", EventCompliancePhoneUsage, SeverityWarning},
		{"cleanliness_alert", EventComplianceCleanlinessAlert, SeverityWarning},
		{"checkout_bottleneck", EventOperationsCheckoutBottleneck, SeverityWarning},
		{"stockroom_anomaly", EventInventoryStockroomAnomaly, SeverityWarning},
		{"loitering", EventSecurityLoitering, SeverityWarning},
		{"camera_degraded", EventCameraDegraded, SeverityWarning},

		// Info events
		{"loyalty_recognized", EventCustomerLoyaltyRecognized, SeverityInfo},
		{"occupancy_update", EventOccupancyUpdate, SeverityInfo},
		{"activity_update", EventActivityUpdate, SeverityInfo},

		// Unknown subjects default to info
		{"unknown subject", "vision.unknown.event", SeverityInfo},
		{"empty string", "", SeverityInfo},
		{"non-vision subject", "alerts.telegram.sent", SeverityInfo},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveSeverity(tc.subject)
			if got != tc.want {
				t.Errorf("ResolveSeverity(%q) = %q, want %q", tc.subject, got, tc.want)
			}
		})
	}
}

func TestResolveSeverity_AllSubjects(t *testing.T) {
	// Every subject in AllSubjects should have a defined severity (not fallback)
	for _, subject := range AllSubjects() {
		sev := ResolveSeverity(subject)
		if sev != SeverityCritical && sev != SeverityWarning && sev != SeverityInfo {
			t.Errorf("subject %q returned unknown severity %q", subject, sev)
		}
	}
}

func TestResolveSeverity_CriticalCount(t *testing.T) {
	// Architecture specifies 5 critical event types
	criticalCount := 0
	for _, subject := range AllSubjects() {
		if ResolveSeverity(subject) == SeverityCritical {
			criticalCount++
		}
	}
	if criticalCount != 5 {
		t.Errorf("expected 5 critical subjects, got %d", criticalCount)
	}
}

func TestResolveSeverity_WarningCount(t *testing.T) {
	// Architecture specifies 8 warning event types
	warningCount := 0
	for _, subject := range AllSubjects() {
		if ResolveSeverity(subject) == SeverityWarning {
			warningCount++
		}
	}
	if warningCount != 8 {
		t.Errorf("expected 8 warning subjects, got %d", warningCount)
	}
}

func TestResolveSeverity_InfoCount(t *testing.T) {
	// 3 info events
	infoCount := 0
	for _, subject := range AllSubjects() {
		if ResolveSeverity(subject) == SeverityInfo {
			infoCount++
		}
	}
	if infoCount != 3 {
		t.Errorf("expected 3 info subjects, got %d", infoCount)
	}
}

func TestDefaultSeverityMap(t *testing.T) {
	m := DefaultSeverityMap()
	if len(m) != len(allSubjects) {
		t.Errorf("map length: got %d, want %d", len(m), len(allSubjects))
	}
}

func TestDefaultSeverityMap_ReturnsCopy(t *testing.T) {
	m1 := DefaultSeverityMap()
	m2 := DefaultSeverityMap()
	if len(m1) != len(m2) {
		t.Fatalf("map lengths differ: %d vs %d", len(m1), len(m2))
	}
	// Modify copy
	m1[EventSafetySlipFall] = SeverityInfo
	// Original should be unchanged
	if DefaultSeverityMap()[EventSafetySlipFall] != SeverityCritical {
		t.Error("DefaultSeverityMap should return a copy")
	}
}

func TestDefaultSeverityMap_AllSubjectsPresent(t *testing.T) {
	m := DefaultSeverityMap()
	for _, subject := range AllSubjects() {
		if _, ok := m[subject]; !ok {
			t.Errorf("subject %q not in severity map", subject)
		}
	}
}

func TestDefaultSeverityMap_CorrectValues(t *testing.T) {
	m := DefaultSeverityMap()
	for subject, sev := range m {
		if sev != ResolveSeverity(subject) {
			t.Errorf("map[%q] = %q but ResolveSeverity returned different value", subject, sev)
		}
	}
}
