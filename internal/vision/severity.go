// Package vision — severity.go maps event subjects to default Severity and
// provides ResolveSeverity for callers without an explicit severity.
package vision

// defaultSeverity maps each event subject to its default Severity. Critical
// subjects surface incidents; warning subjects flag operational issues; info
// subjects are telemetry/observations.
var defaultSeverity = map[string]Severity{
	EventSafetySlipFall:               SeverityCritical,
	EventTheftCashDrawer:              SeverityCritical,
	EventAccessAfterHours:             SeverityCritical,
	EventLaborBuddyPunch:              SeverityCritical,
	EventComplianceBlockedExit:         SeverityCritical,

	EventComplianceUniformViolation:    SeverityWarning,
	EventComplianceHygieneViolation:     SeverityWarning,
	EventCompliancePhoneUsage:           SeverityWarning,
	EventComplianceCleanlinessAlert:     SeverityWarning,
	EventOperationsCheckoutBottleneck:   SeverityWarning,
	EventInventoryStockroomAnomaly:       SeverityWarning,
	EventSecurityLoitering:              SeverityWarning,
	EventCameraDegraded:                 SeverityWarning,

	EventCustomerLoyaltyRecognized: SeverityInfo,
	EventOccupancyUpdate:           SeverityInfo,
	EventActivityUpdate:            SeverityInfo,
}

// ResolveSeverity returns the default Severity for a subject. If the subject is
// unknown it falls back to SeverityInfo.
func ResolveSeverity(subject string) Severity {
	if s, ok := defaultSeverity[subject]; ok {
		return s
	}
	return SeverityInfo
}

// DefaultSeverityMap returns a copy of the default severity map, useful for
// documentation and configuration endpoints.
func DefaultSeverityMap() map[string]Severity {
	cp := make(map[string]Severity, len(defaultSeverity))
	for k, v := range defaultSeverity {
		cp[k] = v
	}
	return cp
}