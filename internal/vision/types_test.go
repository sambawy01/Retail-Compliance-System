package vision

import (
	"testing"

	"github.com/google/uuid"
)

func TestSeverityConstants(t *testing.T) {
	tests := []struct {
		name string
		sev  Severity
		want string
	}{
		{"critical", SeverityCritical, "critical"},
		{"warning", SeverityWarning, "warning"},
		{"info", SeverityInfo, "info"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if string(tc.sev) != tc.want {
				t.Errorf("got %q, want %q", string(tc.sev), tc.want)
			}
		})
	}
}

func TestCameraStatusConstants(t *testing.T) {
	tests := []struct {
		name string
		s    CameraStatus
		want string
	}{
		{"pending", CameraStatusPending, "pending"},
		{"online", CameraStatusOnline, "online"},
		{"offline", CameraStatusOffline, "offline"},
		{"degraded", CameraStatusDegraded, "degraded"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if string(tc.s) != tc.want {
				t.Errorf("got %q, want %q", string(tc.s), tc.want)
			}
		})
	}
}

func TestZoneKindConstants(t *testing.T) {
	tests := []struct {
		name string
		k    ZoneKind
		want string
	}{
		{"checkout", ZoneKindCheckout, "checkout"},
		{"aisles", ZoneKindAisles, "aisles"},
		{"stockroom", ZoneKindStockroom, "stockroom"},
		{"back_office", ZoneKindBackOffice, "back_office"},
		{"entrance", ZoneKindEntrance, "entrance"},
		{"restroom", ZoneKindRestroom, "restroom"},
		{"restricted", ZoneKindRestricted, "restricted"},
		{"privacy_mask", ZoneKindPrivacyMask, "privacy_mask"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if string(tc.k) != tc.want {
				t.Errorf("got %q, want %q", string(tc.k), tc.want)
			}
		})
	}
}

func TestZoneKind_Count(t *testing.T) {
	// Architecture specifies 8 zone types
	zones := []ZoneKind{
		ZoneKindCheckout, ZoneKindAisles, ZoneKindStockroom,
		ZoneKindBackOffice, ZoneKindEntrance, ZoneKindRestroom,
		ZoneKindRestricted, ZoneKindPrivacyMask,
	}
	if len(zones) != 8 {
		t.Errorf("expected 8 zone kinds, got %d", len(zones))
	}
}

func TestRetentionTierConstants(t *testing.T) {
	tests := []struct {
		name string
		r    RetentionTier
		want string
	}{
		{"hot", RetentionHot, "hot"},
		{"warm", RetentionWarm, "warm"},
		{"glacier", RetentionGlacier, "glacier"},
		{"deleted", RetentionDeleted, "deleted"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if string(tc.r) != tc.want {
				t.Errorf("got %q, want %q", string(tc.r), tc.want)
			}
		})
	}
}

func TestCameraStruct(t *testing.T) {
	cam := Camera{
		CameraID:    uuid.New(),
		OrgID:       uuid.New(),
		LocationID:  uuid.New(),
		Name:        "Front Door",
		RTSPURL:     "rtsp://example.com/stream",
		Status:      CameraStatusOnline,
	}
	if cam.Name != "Front Door" {
		t.Errorf("Name: got %q, want %q", cam.Name, "Front Door")
	}
	if cam.Status != CameraStatusOnline {
		t.Errorf("Status: got %q, want %q", cam.Status, CameraStatusOnline)
	}
}

func TestZoneStruct(t *testing.T) {
	capacity := 10
	zone := Zone{
		ZoneID:   uuid.New(),
		OrgID:     uuid.New(),
		CameraID:  uuid.New(),
		Name:      "Checkout Area 1",
		Kind:      ZoneKindCheckout,
		Capacity:  &capacity,
	}
	if zone.Kind != ZoneKindCheckout {
		t.Errorf("Kind: got %q, want %q", zone.Kind, ZoneKindCheckout)
	}
	if zone.Capacity == nil || *zone.Capacity != 10 {
		t.Errorf("Capacity: got %v", zone.Capacity)
	}
}

func TestDetectionStruct(t *testing.T) {
	det := Detection{
		DetectionID: uuid.New(),
		EventType:   EventCompliancePhoneUsage,
		Severity:     SeverityWarning,
		Confidence:   0.95,
	}
	if det.EventType != EventCompliancePhoneUsage {
		t.Errorf("EventType: got %q, want %q", det.EventType, EventCompliancePhoneUsage)
	}
	if det.Severity != SeverityWarning {
		t.Errorf("Severity: got %q, want %q", det.Severity, SeverityWarning)
	}
	if det.Confidence != 0.95 {
		t.Errorf("Confidence: got %f, want %f", det.Confidence, 0.95)
	}
}

func TestClipStruct(t *testing.T) {
	clip := Clip{
		ClipID:          uuid.New(),
		S3Bucket:        "mybucket",
		S3Key:           "clips/abc123.mp4",
		DurationSeconds: 30.5,
		RetentionTier:   RetentionHot,
	}
	if clip.S3Bucket != "mybucket" {
		t.Errorf("S3Bucket: got %q, want %q", clip.S3Bucket, "mybucket")
	}
	if clip.RetentionTier != RetentionHot {
		t.Errorf("RetentionTier: got %q, want %q", clip.RetentionTier, RetentionHot)
	}
}
