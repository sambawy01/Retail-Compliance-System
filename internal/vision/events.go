// Package vision — events.go defines the retail event subjects and payload
// structs published on the in-process event bus.
package vision

import (
	"time"

	"github.com/google/uuid"
)

// Event subjects (NATS-compatible). 15 named events plus the generic
// vision.activity.update for low-frequency telemetry.
const (
	// Critical
	EventSafetySlipFall       = "vision.safety.slip_fall"
	EventTheftCashDrawer      = "vision.theft.cash_drawer"
	EventAccessAfterHours     = "vision.access.after_hours"
	EventLaborBuddyPunch      = "vision.labor.buddy_punch"
	EventComplianceBlockedExit = "vision.compliance.blocked_exit"

	// Warning
	EventComplianceUniformViolation  = "vision.compliance.uniform_violation"
	EventComplianceHygieneViolation  = "vision.compliance.hygiene_violation"
	EventCompliancePhoneUsage        = "vision.compliance.phone_usage"
	EventComplianceCleanlinessAlert = "vision.compliance.cleanliness_alert"
	EventOperationsCheckoutBottleneck = "vision.operations.checkout_bottleneck"
	EventInventoryStockroomAnomaly   = "vision.inventory.stockroom_anomaly"
	EventSecurityLoitering           = "vision.security.loitering"
	EventCameraDegraded              = "vision.camera.degraded"

	// Info
	EventCustomerLoyaltyRecognized = "vision.customer.loyalty_recognized"
	EventOccupancyUpdate            = "vision.occupancy.update"
	EventActivityUpdate             = "vision.activity.update"
)

// allSubjects is the canonical list returned by AllSubjects.
var allSubjects = []string{
	EventSafetySlipFall,
	EventTheftCashDrawer,
	EventAccessAfterHours,
	EventLaborBuddyPunch,
	EventComplianceBlockedExit,
	EventComplianceUniformViolation,
	EventComplianceHygieneViolation,
	EventCompliancePhoneUsage,
	EventComplianceCleanlinessAlert,
	EventOperationsCheckoutBottleneck,
	EventInventoryStockroomAnomaly,
	EventSecurityLoitering,
	EventCameraDegraded,
	EventCustomerLoyaltyRecognized,
	EventOccupancyUpdate,
	EventActivityUpdate,
}

// AllSubjects returns all 16 subjects published by the vision pipeline.
func AllSubjects() []string {
	cp := make([]string, len(allSubjects))
	copy(cp, allSubjects)
	return cp
}

// OccupancyPayload reports current headcount for a zone.
type OccupancyPayload struct {
	ZoneID     uuid.UUID `json:"zone_id"`
	Count      int       `json:"count"`
	Capacity   int       `json:"capacity,omitempty"`
	ChangeDelta int      `json:"change_delta,omitempty"`
}

// CompliancePayload covers uniform, hygiene, phone-usage, blocked-exit and
// cleanliness events.
type CompliancePayload struct {
	ViolationType string  `json:"violation_type"`
	Description   string  `json:"description,omitempty"`
	StaffID       *uuid.UUID `json:"staff_id,omitempty"`
	ZoneID        *uuid.UUID `json:"zone_id,omitempty"`
	Bbox          []float64  `json:"bbox,omitempty"` // [x1,y1,x2,y2] normalized
}

// AnomalyPayload covers stockroom anomalies and checkout bottlenecks.
type AnomalyPayload struct {
	AnomalyKind string    `json:"anomaly_kind"`
	Metric      string    `json:"metric"`
	Value       float64   `json:"value"`
	Threshold   float64   `json:"threshold,omitempty"`
	ZoneID      *uuid.UUID `json:"zone_id,omitempty"`
}

// CustomerRecognitionPayload covers loyalty recognition.
type CustomerRecognitionPayload struct {
	CustomerRef string    `json:"customer_ref"`
	LoyaltyTier string    `json:"loyalty_tier,omitempty"`
	VisitCount  int       `json:"visit_count,omitempty"`
	ZoneID      *uuid.UUID `json:"zone_id,omitempty"`
}

// RetailEventPayload is a generic payload for phone_usage, cleanliness_alert,
// loitering and other subjects without a dedicated struct.
type RetailEventPayload struct {
	Subject     string         `json:"subject"`
	Description string         `json:"description,omitempty"`
	CameraID    uuid.UUID      `json:"camera_id,omitempty"`
	ZoneID      *uuid.UUID     `json:"zone_id,omitempty"`
	Bbox        []float64      `json:"bbox,omitempty"`
	Extra       map[string]any `json:"extra,omitempty"`
	At          time.Time      `json:"at,omitempty"`
}