-- 001_init_schema.sql — Watch Dog canonical schema migration
-- This file creates the ENTIRE schema from scratch with all fixes applied.
-- Safe to run on a fresh database. Uses IF NOT EXISTS for idempotency.

-- =========================================================================
-- EXTENSIONS
-- =========================================================================
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "vector";

-- =========================================================================
-- ROLES
-- =========================================================================
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'watchdog_owner') THEN
        CREATE ROLE watchdog_owner NOLOGIN NOBYPASSRLS;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'watchdog_app') THEN
        CREATE ROLE watchdog_app NOLOGIN NOBYPASSRLS;
    END IF;
END $$;
GRANT watchdog_app TO postgres;

-- =========================================================================
-- ORGANIZATIONS
-- =========================================================================
CREATE TABLE IF NOT EXISTS organizations (
    org_id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    status      TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'cancelled')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
ALTER TABLE organizations ENABLE ROW LEVEL SECURITY;
ALTER TABLE organizations FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON organizations;
CREATE POLICY org_isolation ON organizations
    USING (org_id = current_setting('app.current_org_id', true)::UUID)
    WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- =========================================================================
-- LOCATIONS
-- =========================================================================
CREATE TABLE IF NOT EXISTS locations (
    location_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES organizations(org_id),
    name        TEXT NOT NULL,
    address     TEXT,
    timezone    TEXT NOT NULL DEFAULT 'Africa/Cairo',
    status      TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'onboarding')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_locations_org_id ON locations(org_id);
CREATE INDEX IF NOT EXISTS idx_locations_org_status ON locations(org_id, status);
ALTER TABLE locations ENABLE ROW LEVEL SECURITY;
ALTER TABLE locations FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON locations;
CREATE POLICY org_isolation ON locations
    USING (org_id = current_setting('app.current_org_id', true)::UUID)
    WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- =========================================================================
-- USERS
-- =========================================================================
CREATE TABLE IF NOT EXISTS users (
    user_id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        UUID NOT NULL REFERENCES organizations(org_id),
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    display_name  TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'staff' CHECK (role IN ('owner', 'admin', 'manager', 'staff', 'read_only')),
    status        TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'locked')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_users_org_id ON users(org_id);
CREATE INDEX IF NOT EXISTS idx_users_org_status ON users(org_id, status);
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE users FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON users;
CREATE POLICY org_isolation ON users
    USING (org_id = current_setting('app.current_org_id', true)::UUID)
    WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- =========================================================================
-- LOGIN LOOKUP FUNCTION (SECURITY DEFINER — bypasses RLS for auth)
-- =========================================================================
CREATE OR REPLACE FUNCTION fn_login_lookup(p_email TEXT)
RETURNS TABLE (user_id UUID, org_id UUID, role TEXT, display_name TEXT, password_hash TEXT)
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, pg_temp
AS $$
BEGIN
    RETURN QUERY
    SELECT u.user_id, u.org_id, u.role, u.display_name, u.password_hash
    FROM users u WHERE u.email = p_email AND u.status = 'active' LIMIT 1;
END;
$$;
REVOKE EXECUTE ON FUNCTION fn_login_lookup(TEXT) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION fn_login_lookup(TEXT) TO watchdog_app;

-- =========================================================================
-- EMPLOYEES
-- =========================================================================
CREATE TABLE IF NOT EXISTS employees (
    employee_id  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id       UUID NOT NULL REFERENCES organizations(org_id),
    location_id  UUID NOT NULL REFERENCES locations(location_id),
    user_id      UUID REFERENCES users(user_id),
    display_name TEXT NOT NULL,
    pin_hash     TEXT,
    role         TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'terminated')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_employees_org_id ON employees(org_id);
CREATE INDEX IF NOT EXISTS idx_employees_org_status ON employees(org_id, status);
CREATE INDEX IF NOT EXISTS idx_employees_org_location ON employees(org_id, location_id);
CREATE INDEX IF NOT EXISTS idx_employees_user_id ON employees(user_id);
ALTER TABLE employees ENABLE ROW LEVEL SECURITY;
ALTER TABLE employees FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON employees;
CREATE POLICY org_isolation ON employees
    USING (org_id = current_setting('app.current_org_id', true)::UUID)
    WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- =========================================================================
-- USER LOCATION ACCESS
-- =========================================================================
CREATE TABLE IF NOT EXISTS user_location_access (
    user_id     UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    location_id UUID NOT NULL REFERENCES locations(location_id) ON DELETE CASCADE,
    org_id      UUID NOT NULL REFERENCES organizations(org_id),
    PRIMARY KEY (user_id, location_id)
);
CREATE INDEX IF NOT EXISTS idx_ula_org_id ON user_location_access(org_id);
CREATE INDEX IF NOT EXISTS idx_ula_location_id ON user_location_access(location_id);
ALTER TABLE user_location_access ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_location_access FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON user_location_access;
CREATE POLICY org_isolation ON user_location_access
    USING (org_id = current_setting('app.current_org_id', true)::UUID)
    WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- =========================================================================
-- AUDIT LOG
-- =========================================================================
CREATE TABLE IF NOT EXISTS audit_log (
    log_id        BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    org_id        UUID NOT NULL REFERENCES organizations(org_id),
    user_id       UUID REFERENCES users(user_id),
    action        TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id   TEXT,
    detail        JSONB,
    ip_address    INET,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_audit_org_id ON audit_log(org_id);
CREATE INDEX IF NOT EXISTS idx_audit_org_time ON audit_log(org_id, created_at);
CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_log(org_id, action, created_at);
ALTER TABLE audit_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_log FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON audit_log;
CREATE POLICY org_isolation ON audit_log
    USING (org_id = current_setting('app.current_org_id', true)::UUID)
    WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- =========================================================================
-- VISION CAMERAS
-- =========================================================================
CREATE TABLE IF NOT EXISTS vision_cameras (
    camera_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id            UUID NOT NULL REFERENCES organizations(org_id),
    location_id       UUID REFERENCES locations(location_id),
    name              TEXT NOT NULL,
    rtsp_url          TEXT NOT NULL,
    local_agent_id    TEXT,
    status            TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('online', 'offline', 'degraded', 'pending')),
    last_heartbeat_at TIMESTAMPTZ,
    feature_flags     JSONB,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_cameras_org_id ON vision_cameras(org_id);
CREATE INDEX IF NOT EXISTS idx_cameras_org_location ON vision_cameras(org_id, location_id);
CREATE INDEX IF NOT EXISTS idx_cameras_org_status ON vision_cameras(org_id, status);
ALTER TABLE vision_cameras ENABLE ROW LEVEL SECURITY;
ALTER TABLE vision_cameras FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON vision_cameras;
CREATE POLICY org_isolation ON vision_cameras
    USING (org_id = current_setting('app.current_org_id', true)::UUID)
    WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- =========================================================================
-- VISION ZONES
-- =========================================================================
CREATE TABLE IF NOT EXISTS vision_zones (
    zone_id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     UUID NOT NULL REFERENCES organizations(org_id),
    camera_id  UUID NOT NULL REFERENCES vision_cameras(camera_id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    kind       TEXT NOT NULL CHECK (kind IN ('checkout','aisles','stockroom','back_office','entrance','restroom','restricted','privacy_mask')),
    polygon    JSONB,
    capacity   INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_zones_org_id ON vision_zones(org_id);
CREATE INDEX IF NOT EXISTS idx_zones_camera_id ON vision_zones(camera_id);
ALTER TABLE vision_zones ENABLE ROW LEVEL SECURITY;
ALTER TABLE vision_zones FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON vision_zones;
CREATE POLICY org_isolation ON vision_zones
    USING (org_id = current_setting('app.current_org_id', true)::UUID)
    WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- =========================================================================
-- VISION DETECTIONS (renamed from detections for Go code alignment)
-- =========================================================================
CREATE TABLE IF NOT EXISTS vision_detections (
    detection_id  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        UUID NOT NULL REFERENCES organizations(org_id),
    location_id   UUID REFERENCES locations(location_id),
    camera_id     UUID REFERENCES vision_cameras(camera_id) ON DELETE SET NULL,
    zone_id       UUID REFERENCES vision_zones(zone_id),
    event_type    TEXT NOT NULL,
    severity      TEXT NOT NULL DEFAULT 'info' CHECK (severity IN ('critical','warning','info')),
    confidence    NUMERIC,
    payload       JSONB,
    clip_id       UUID,
    detected_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    retention_tier TEXT NOT NULL DEFAULT 'hot' CHECK (retention_tier IN ('hot', 'warm', 'deleted')),
    expires_at    TIMESTAMPTZ,
    retention_changed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_detections_org_id ON vision_detections(org_id);
CREATE INDEX IF NOT EXISTS idx_detections_org_camera ON vision_detections(org_id, camera_id);
CREATE INDEX IF NOT EXISTS idx_detections_org_time ON vision_detections(org_id, detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_detections_org_event ON vision_detections(org_id, event_type);
CREATE INDEX IF NOT EXISTS idx_detections_org_severity ON vision_detections(org_id, severity);
CREATE INDEX IF NOT EXISTS idx_detections_retention ON vision_detections(org_id, retention_tier, expires_at);
-- GIN index for staff performance queries that search payload->>'person_id'
CREATE INDEX IF NOT EXISTS idx_detections_payload_person ON vision_detections USING GIN (payload jsonb_path_ops);
ALTER TABLE vision_detections ENABLE ROW LEVEL SECURITY;
ALTER TABLE vision_detections FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON vision_detections;
CREATE POLICY org_isolation ON vision_detections
    USING (org_id = current_setting('app.current_org_id', true)::UUID)
    WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- =========================================================================
-- CLIPS (table name matches Go code — clips, not vision_clips)
-- =========================================================================
CREATE TABLE IF NOT EXISTS clips (
    clip_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id           UUID NOT NULL REFERENCES organizations(org_id),
    location_id      UUID REFERENCES locations(location_id),
    camera_id        UUID REFERENCES vision_cameras(camera_id) ON DELETE SET NULL,
    detection_id     UUID,
    s3_bucket        TEXT,
    s3_key           TEXT,
    duration_seconds FLOAT8,
    starts_at        TIMESTAMPTZ,
    retention_tier   TEXT NOT NULL DEFAULT 'hot' CHECK (retention_tier IN ('hot', 'warm', 'deleted')),
    expires_at       TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_clips_org_id ON clips(org_id);
CREATE INDEX IF NOT EXISTS idx_clips_org_camera ON clips(org_id, camera_id);
CREATE INDEX IF NOT EXISTS idx_clips_detection_id ON clips(detection_id);
CREATE INDEX IF NOT EXISTS idx_clips_retention ON clips(org_id, retention_tier, expires_at);
ALTER TABLE clips ENABLE ROW LEVEL SECURITY;
ALTER TABLE clips FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON clips;
CREATE POLICY org_isolation ON clips
    USING (org_id = current_setting('app.current_org_id', true)::UUID)
    WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- =========================================================================
-- VISION SEVERITY OVERRIDES
-- =========================================================================
CREATE TABLE IF NOT EXISTS vision_severity_overrides (
    override_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES organizations(org_id),
    event_type  TEXT NOT NULL,
    severity    TEXT NOT NULL CHECK (severity IN ('critical','warning','info')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(org_id, event_type)
);
ALTER TABLE vision_severity_overrides ENABLE ROW LEVEL SECURITY;
ALTER TABLE vision_severity_overrides FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON vision_severity_overrides;
CREATE POLICY org_isolation ON vision_severity_overrides
    USING (org_id = current_setting('app.current_org_id', true)::UUID)
    WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- =========================================================================
-- IDENTITY PERSONS
-- =========================================================================
CREATE TABLE IF NOT EXISTS identity_persons (
    person_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id           UUID NOT NULL REFERENCES organizations(org_id),
    kind             TEXT NOT NULL CHECK (kind IN ('employee', 'customer')),
    employee_id      UUID REFERENCES employees(employee_id),
    customer_id      UUID,
    display_name     TEXT NOT NULL,
    phone            TEXT,
    job_role         TEXT,
    photo_url        TEXT,
    employee_id_code TEXT,
    department       TEXT,
    shift_start      TEXT,
    shift_end        TEXT,
    hire_date        DATE,
    status           TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','inactive','on_leave','terminated')),
    enrolled_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at       TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_identity_persons_org_id ON identity_persons(org_id);
CREATE INDEX IF NOT EXISTS idx_identity_persons_org_kind ON identity_persons(org_id, kind);
CREATE INDEX IF NOT EXISTS idx_identity_persons_employee ON identity_persons(employee_id);
ALTER TABLE identity_persons ENABLE ROW LEVEL SECURITY;
ALTER TABLE identity_persons FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON identity_persons;
CREATE POLICY org_isolation ON identity_persons
    USING (org_id = current_setting('app.current_org_id', true)::UUID)
    WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- =========================================================================
-- IDENTITY TEMPLATES (encrypted biometric embeddings)
-- =========================================================================
CREATE TABLE IF NOT EXISTS identity_templates (
    template_id     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES organizations(org_id),
    person_id       UUID NOT NULL REFERENCES identity_persons(person_id) ON DELETE CASCADE,
    embedding       VECTOR(512) NOT NULL,
    embedding_ct    BYTEA,
    embedding_nonce  BYTEA,
    kms_dek_id      TEXT NOT NULL,
    quality_score   NUMERIC(4,3),
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_identity_templates_org_id ON identity_templates(org_id);
CREATE INDEX IF NOT EXISTS idx_identity_templates_person_id ON identity_templates(person_id);
CREATE INDEX IF NOT EXISTS idx_identity_templates_expiry ON identity_templates(org_id, expires_at);
CREATE INDEX IF NOT EXISTS idx_identity_templates_embedding_hnsw ON identity_templates USING hnsw (embedding vector_cosine_ops);
ALTER TABLE identity_templates ENABLE ROW LEVEL SECURITY;
ALTER TABLE identity_templates FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON identity_templates;
CREATE POLICY org_isolation ON identity_templates
    USING (org_id = current_setting('app.current_org_id', true)::UUID)
    WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- =========================================================================
-- IDENTITY CONSENTS (PDP Law)
-- =========================================================================
CREATE TABLE IF NOT EXISTS identity_consents (
    consent_id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id           UUID NOT NULL REFERENCES organizations(org_id),
    person_id        UUID NOT NULL REFERENCES identity_persons(person_id) ON DELETE CASCADE,
    captured_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    consent_text     TEXT NOT NULL,
    consent_locale   TEXT NOT NULL CHECK (consent_locale IN ('en', 'ar')),
    signature_sha256 BYTEA NOT NULL,
    captured_by      TEXT NOT NULL,
    captured_ip      INET,
    revoked          BOOLEAN NOT NULL DEFAULT false,
    revoked_at       TIMESTAMPTZ,
    revoked_reason   TEXT,
    lawful_basis     TEXT
);
CREATE INDEX IF NOT EXISTS idx_identity_consents_org_id ON identity_consents(org_id);
CREATE INDEX IF NOT EXISTS idx_identity_consents_person_id ON identity_consents(person_id);
CREATE INDEX IF NOT EXISTS idx_identity_consents_revoked ON identity_consents(org_id, revoked);
ALTER TABLE identity_consents ENABLE ROW LEVEL SECURITY;
ALTER TABLE identity_consents FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON identity_consents;
CREATE POLICY org_isolation ON identity_consents
    USING (org_id = current_setting('app.current_org_id', true)::UUID)
    WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- =========================================================================
-- IDENTITY ACCESS AUDIT
-- =========================================================================
CREATE TABLE IF NOT EXISTS identity_access_audit (
    audit_id     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id       UUID NOT NULL REFERENCES organizations(org_id),
    person_id    UUID,
    accessed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    purpose      TEXT NOT NULL CHECK (purpose IN ('recognize', 'enroll', 'review', 'export', 'erase')),
    triggered_by TEXT NOT NULL,
    camera_id    UUID,
    detection_id UUID,
    notes        TEXT
);
CREATE INDEX IF NOT EXISTS idx_identity_access_audit_org_id ON identity_access_audit(org_id);
CREATE INDEX IF NOT EXISTS idx_identity_access_audit_person ON identity_access_audit(person_id);
CREATE INDEX IF NOT EXISTS idx_identity_access_audit_time ON identity_access_audit(org_id, accessed_at);
ALTER TABLE identity_access_audit ENABLE ROW LEVEL SECURITY;
ALTER TABLE identity_access_audit FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON identity_access_audit;
CREATE POLICY org_isolation ON identity_access_audit
    USING (org_id = current_setting('app.current_org_id', true)::UUID)
    WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- =========================================================================
-- IDENTITY DEK (encryption key envelope)
-- =========================================================================
CREATE TABLE IF NOT EXISTS identity_dek (
    dek_id        TEXT PRIMARY KEY,
    org_id        UUID NOT NULL REFERENCES organizations(org_id),
    wrapped_key   BYTEA NOT NULL,
    kms_key_arn   TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    rotated_out_at TIMESTAMPTZ,
    UNIQUE (org_id, dek_id)
);
ALTER TABLE identity_dek ENABLE ROW LEVEL SECURITY;
ALTER TABLE identity_dek FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON identity_dek;
CREATE POLICY org_isolation ON identity_dek
    USING (org_id = current_setting('app.current_org_id', true)::UUID)
    WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- =========================================================================
-- FACE ERASURE LOG
-- =========================================================================
CREATE TABLE IF NOT EXISTS face_erasure_log (
    erasure_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id             UUID NOT NULL REFERENCES organizations(org_id),
    person_id          UUID NOT NULL,
    erased_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    reason             TEXT NOT NULL,
    requested_by       TEXT NOT NULL,
    templates_deleted  INT NOT NULL DEFAULT 0,
    detections_cleared INT NOT NULL DEFAULT 0,
    consents_revoked   INT NOT NULL DEFAULT 0
);
ALTER TABLE face_erasure_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE face_erasure_log FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON face_erasure_log;
CREATE POLICY org_isolation ON face_erasure_log
    USING (org_id = current_setting('app.current_org_id', true)::UUID)
    WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- =========================================================================
-- WEBRTC SESSIONS
-- =========================================================================
CREATE TABLE IF NOT EXISTS webrtc_sessions (
    session_id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id       UUID NOT NULL REFERENCES organizations(org_id),
    camera_id    UUID REFERENCES vision_cameras(camera_id) ON DELETE SET NULL,
    viewer_id    UUID REFERENCES users(user_id),
    sdp_offer    TEXT NOT NULL,
    sdp_answer   TEXT,
    status       TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','connected','closed','failed')),
    ended_at     TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_webrtc_org_id ON webrtc_sessions(org_id);
CREATE INDEX IF NOT EXISTS idx_webrtc_camera_id ON webrtc_sessions(camera_id);
CREATE INDEX IF NOT EXISTS idx_webrtc_cleanup ON webrtc_sessions(org_id, status, ended_at);
ALTER TABLE webrtc_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE webrtc_sessions FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON webrtc_sessions;
CREATE POLICY org_isolation ON webrtc_sessions
    USING (org_id = current_setting('app.current_org_id', true)::UUID)
    WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- =========================================================================
-- WEBRTC TURN CREDENTIALS
-- =========================================================================
CREATE TABLE IF NOT EXISTS webrtc_turn_credentials (
    credential_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        UUID NOT NULL REFERENCES organizations(org_id),
    username      TEXT NOT NULL,
    credential    TEXT NOT NULL,
    expires_at    TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
ALTER TABLE webrtc_turn_credentials ENABLE ROW LEVEL SECURITY;
ALTER TABLE webrtc_turn_credentials FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON webrtc_turn_credentials;
CREATE POLICY org_isolation ON webrtc_turn_credentials
    USING (org_id = current_setting('app.current_org_id', true)::UUID)
    WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- =========================================================================
-- NOTIFICATION RULES
-- =========================================================================
CREATE TABLE IF NOT EXISTS notification_rules (
    rule_id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     UUID NOT NULL REFERENCES organizations(org_id),
    event_type TEXT NOT NULL,
    severity   TEXT NOT NULL CHECK (severity IN ('critical', 'warning', 'info')),
    channel    TEXT NOT NULL CHECK (channel IN ('telegram', 'email', 'sms', 'dashboard')),
    target     TEXT NOT NULL,
    enabled    BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_notif_rules_org_id ON notification_rules(org_id);
CREATE INDEX IF NOT EXISTS idx_notif_rules_org_event ON notification_rules(org_id, event_type);
CREATE INDEX IF NOT EXISTS idx_notif_rules_org_severity ON notification_rules(org_id, severity);
CREATE INDEX IF NOT EXISTS idx_notif_rules_org_enabled ON notification_rules(org_id, enabled);
ALTER TABLE notification_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification_rules FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON notification_rules;
CREATE POLICY org_isolation ON notification_rules
    USING (org_id = current_setting('app.current_org_id', true)::UUID)
    WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- =========================================================================
-- NOTIFICATION LOG
-- =========================================================================
CREATE TABLE IF NOT EXISTS notification_log (
    log_id     BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    org_id     UUID NOT NULL REFERENCES organizations(org_id),
    rule_id    UUID REFERENCES notification_rules(rule_id),
    event_type TEXT NOT NULL,
    severity   TEXT NOT NULL CHECK (severity IN ('critical', 'warning', 'info')),
    channel    TEXT NOT NULL CHECK (channel IN ('telegram', 'email', 'sms', 'dashboard')),
    target     TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('sent', 'failed', 'pending')),
    error      TEXT,
    sent_at    TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_notif_log_org_id ON notification_log(org_id);
CREATE INDEX IF NOT EXISTS idx_notif_log_org_time ON notification_log(org_id, created_at);
CREATE INDEX IF NOT EXISTS idx_notif_log_org_status ON notification_log(org_id, status);
CREATE INDEX IF NOT EXISTS idx_notif_log_rule_id ON notification_log(rule_id);
ALTER TABLE notification_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification_log FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON notification_log;
CREATE POLICY org_isolation ON notification_log
    USING (org_id = current_setting('app.current_org_id', true)::UUID)
    WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- =========================================================================
-- STAFF ATTENDANCE
-- =========================================================================
CREATE TABLE IF NOT EXISTS staff_attendance (
    attendance_id     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id            UUID NOT NULL REFERENCES organizations(org_id),
    person_id         UUID NOT NULL REFERENCES identity_persons(person_id) ON DELETE CASCADE,
    date              DATE NOT NULL,
    check_in          TIMESTAMPTZ,
    check_out         TIMESTAMPTZ,
    status            TEXT NOT NULL DEFAULT 'present' CHECK (status IN ('present','late','absent','half_day','remote')),
    late_minutes      INTEGER DEFAULT 0,
    overtime_minutes  INTEGER DEFAULT 0,
    notes             TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(person_id, date)
);
CREATE INDEX IF NOT EXISTS idx_attendance_org_id ON staff_attendance(org_id);
CREATE INDEX IF NOT EXISTS idx_attendance_person ON staff_attendance(person_id, date DESC);
CREATE INDEX IF NOT EXISTS idx_attendance_date ON staff_attendance(date);
ALTER TABLE staff_attendance ENABLE ROW LEVEL SECURITY;
ALTER TABLE staff_attendance FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON staff_attendance;
CREATE POLICY org_isolation ON staff_attendance
    USING (org_id = current_setting('app.current_org_id', true)::UUID)
    WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- =========================================================================
-- STAFF HOLIDAYS / LEAVE
-- =========================================================================
CREATE TABLE IF NOT EXISTS staff_holidays (
    holiday_id  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES organizations(org_id),
    person_id   UUID NOT NULL REFERENCES identity_persons(person_id) ON DELETE CASCADE,
    start_date  DATE NOT NULL,
    end_date    DATE NOT NULL,
    type        TEXT NOT NULL CHECK (type IN ('annual','sick','personal','unpaid','public_holiday')),
    status      TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','approved','rejected')),
    reason      TEXT,
    approved_by UUID REFERENCES users(user_id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_holidays_org_id ON staff_holidays(org_id);
CREATE INDEX IF NOT EXISTS idx_holidays_person ON staff_holidays(person_id, start_date DESC);
ALTER TABLE staff_holidays ENABLE ROW LEVEL SECURITY;
ALTER TABLE staff_holidays FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON staff_holidays;
CREATE POLICY org_isolation ON staff_holidays
    USING (org_id = current_setting('app.current_org_id', true)::UUID)
    WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- =========================================================================
-- GRANTS
-- =========================================================================
GRANT SELECT, INSERT, UPDATE, DELETE ON
    organizations, locations, users, user_location_access, employees,
    vision_cameras, vision_zones, vision_detections, clips,
    identity_persons, identity_templates, identity_consents, identity_dek,
    webrtc_sessions, webrtc_turn_credentials,
    notification_rules, notification_log,
    staff_attendance, staff_holidays, vision_severity_overrides
TO watchdog_app;

GRANT SELECT, INSERT ON audit_log, identity_access_audit, face_erasure_log TO watchdog_app;

GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO watchdog_app;

-- =========================================================================
-- SEED DATA (idempotent)
-- =========================================================================
INSERT INTO organizations (name, slug)
SELECT 'Easy Mart', 'easy-mart'
WHERE NOT EXISTS (SELECT 1 FROM organizations WHERE slug = 'easy-mart');

INSERT INTO users (org_id, email, password_hash, display_name, role)
SELECT
    (SELECT org_id FROM organizations WHERE slug = 'easy-mart'),
    'owner@easymart.com',
    '$2b$10$ho9B7dz.3U/3ZNnyc8x..e99d1f8koQe2lxhmYLZFD8m18gIK1GWK',
    'Store Owner',
    'owner'
WHERE NOT EXISTS (SELECT 1 FROM users WHERE email = 'owner@easymart.com');