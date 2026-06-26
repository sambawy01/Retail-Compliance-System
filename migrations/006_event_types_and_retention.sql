-- 006_event_types_and_retention.sql
-- Watch Dog retail compliance system — event type CHECK constraint + retention automation.

-- -----------------------------------------------------------------------
-- 1. Add CHECK constraint on detections.event_type for the 15+ event types
-- -----------------------------------------------------------------------
ALTER TABLE detections
ADD CONSTRAINT detections_event_type_check CHECK (event_type IN (
    'vision.safety.slip_fall',
    'vision.theft.cash_drawer',
    'vision.access.after_hours',
    'vision.labor.buddy_punch',
    'vision.compliance.blocked_exit',
    'vision.compliance.uniform_violation',
    'vision.compliance.hygiene_violation',
    'vision.compliance.phone_usage',
    'vision.compliance.cleanliness_alert',
    'vision.operations.checkout_bottleneck',
    'vision.inventory.stockroom_anomaly',
    'vision.security.loitering',
    'vision.camera.degraded',
    'vision.customer.loyalty_recognized',
    'vision.occupancy.update',
    'vision.activity.update'
));

-- Same CHECK on notification_rules.event_type
ALTER TABLE notification_rules
ADD CONSTRAINT notification_rules_event_type_check CHECK (event_type IN (
    'vision.safety.slip_fall',
    'vision.theft.cash_drawer',
    'vision.access.after_hours',
    'vision.labor.buddy_punch',
    'vision.compliance.blocked_exit',
    'vision.compliance.uniform_violation',
    'vision.compliance.hygiene_violation',
    'vision.compliance.phone_usage',
    'vision.compliance.cleanliness_alert',
    'vision.operations.checkout_bottleneck',
    'vision.inventory.stockroom_anomaly',
    'vision.security.loitering',
    'vision.camera.degraded',
    'vision.customer.loyalty_recognized',
    'vision.occupancy.update',
    'vision.activity.update'
));

-- Same CHECK on notification_log.event_type
ALTER TABLE notification_log
ADD CONSTRAINT notification_log_event_type_check CHECK (event_type IN (
    'vision.safety.slip_fall',
    'vision.theft.cash_drawer',
    'vision.access.after_hours',
    'vision.labor.buddy_punch',
    'vision.compliance.blocked_exit',
    'vision.compliance.uniform_violation',
    'vision.compliance.hygiene_violation',
    'vision.compliance.phone_usage',
    'vision.compliance.cleanliness_alert',
    'vision.operations.checkout_bottleneck',
    'vision.inventory.stockroom_anomaly',
    'vision.security.loitering',
    'vision.camera.degraded',
    'vision.customer.loyalty_recognized',
    'vision.occupancy.update',
    'vision.activity.update'
));

-- -----------------------------------------------------------------------
-- 2. Add retention/expires_at columns for automated purge sweeps
-- -----------------------------------------------------------------------

-- Clips: add expires_at for the 7d→30d→90d lifecycle
ALTER TABLE clips
ADD COLUMN expires_at TIMESTAMPTZ,
ADD COLUMN retention_changed_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE INDEX idx_clips_retention_sweep ON clips(org_id, retention_tier, expires_at);

-- Detections: add retention_tier + expires_at (high-volume, must purge)
ALTER TABLE detections
ADD COLUMN retention_tier TEXT NOT NULL DEFAULT 'hot'
    CHECK (retention_tier IN ('hot', 'warm', 'deleted')),
ADD COLUMN expires_at TIMESTAMPTZ,
ADD COLUMN retention_changed_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE INDEX idx_detections_retention_sweep ON detections(org_id, retention_tier, expires_at);

-- Identity templates: add expires_at for biometric retention
ALTER TABLE identity_templates
ADD COLUMN expires_at TIMESTAMPTZ;

CREATE INDEX idx_identity_templates_expiry ON identity_templates(org_id, expires_at);

-- WebRTC sessions: add ended_at for cleanup
ALTER TABLE webrtc_sessions
ADD COLUMN ended_at TIMESTAMPTZ;

CREATE INDEX idx_webrtc_sessions_cleanup ON webrtc_sessions(org_id, status, ended_at);

-- Notification log: add expires_at for purge
ALTER TABLE notification_log
ADD COLUMN expires_at TIMESTAMPTZ;

CREATE INDEX idx_notif_log_retention_sweep ON notification_log(org_id, expires_at);

-- -----------------------------------------------------------------------
-- 3. Align clips.retention_tier CHECK to the spec (hot, warm, deleted)
--    Drop the old CHECK that included 'glacier' (not in the spec)
-- -----------------------------------------------------------------------
ALTER TABLE clips DROP CONSTRAINT IF EXISTS clips_retention_tier_check;
ALTER TABLE clips
ADD CONSTRAINT clips_retention_tier_check CHECK (retention_tier IN ('hot', 'warm', 'deleted'));

-- -----------------------------------------------------------------------
-- 4. Notification rules: add uniqueness to prevent duplicate alerts
-- -----------------------------------------------------------------------
ALTER TABLE notification_rules
ADD CONSTRAINT notification_rules_unique UNIQUE (org_id, event_type, severity, channel, target);
