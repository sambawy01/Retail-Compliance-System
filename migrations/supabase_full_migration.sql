-- Watch Dog: Full schema migration for Supabase
-- Run this in the Supabase Dashboard > SQL Editor
-- This creates roles, tables, RLS policies, and seed data

-- =========================================================================
-- 1. ROLES (Supabase doesn't allow CREATE ROLE via pooler, run here instead)
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

-- Grant watchdog_app to postgres (so the backend can SET ROLE)
GRANT watchdog_app TO postgres;

-- =========================================================================
-- 2. EXTENSIONS
-- =========================================================================
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- =========================================================================
-- 3. CORE SCHEMA (001)
-- =========================================================================
CREATE TABLE IF NOT EXISTS organizations (
    org_id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    status      TEXT NOT NULL DEFAULT 'active'
                CHECK (status IN ('active', 'suspended', 'cancelled')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
ALTER TABLE organizations ENABLE ROW LEVEL SECURITY;
ALTER TABLE organizations FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON organizations;
CREATE POLICY org_isolation ON organizations
    USING (org_id = current_setting('app.current_org_id', true)::UUID) WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

CREATE TABLE IF NOT EXISTS locations (
    location_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES organizations(org_id),
    name        TEXT NOT NULL,
    address     TEXT,
    timezone    TEXT NOT NULL DEFAULT 'Africa/Cairo',
    status      TEXT NOT NULL DEFAULT 'active'
                CHECK (status IN ('active', 'inactive', 'onboarding')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_locations_org_id       ON locations(org_id);
CREATE INDEX IF NOT EXISTS idx_locations_org_status    ON locations(org_id, status);
ALTER TABLE locations ENABLE ROW LEVEL SECURITY;
ALTER TABLE locations FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON locations;
CREATE POLICY org_isolation ON locations
    USING (org_id = current_setting('app.current_org_id', true)::UUID) WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

CREATE TABLE IF NOT EXISTS users (
    user_id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        UUID NOT NULL REFERENCES organizations(org_id),
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    display_name  TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'staff'
                  CHECK (role IN ('owner', 'admin', 'manager', 'staff', 'read_only')),
    status        TEXT NOT NULL DEFAULT 'active'
                  CHECK (status IN ('active', 'inactive', 'locked')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_users_org_id    ON users(org_id);
CREATE INDEX IF NOT EXISTS idx_users_org_status ON users(org_id, status);
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE users FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON users;
CREATE POLICY org_isolation ON users
    USING (org_id = current_setting('app.current_org_id', true)::UUID) WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

CREATE TABLE IF NOT EXISTS user_location_access (
    user_id     UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    location_id UUID NOT NULL REFERENCES locations(location_id) ON DELETE CASCADE,
    org_id      UUID NOT NULL REFERENCES organizations(org_id),
    PRIMARY KEY (user_id, location_id)
);
CREATE INDEX IF NOT EXISTS idx_ula_org_id        ON user_location_access(org_id);
CREATE INDEX IF NOT EXISTS idx_ula_location_id   ON user_location_access(location_id);
ALTER TABLE user_location_access ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_location_access FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON user_location_access;
CREATE POLICY org_isolation ON user_location_access
    USING (org_id = current_setting('app.current_org_id', true)::UUID) WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

CREATE TABLE IF NOT EXISTS employees (
    employee_id  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id       UUID NOT NULL REFERENCES organizations(org_id),
    location_id  UUID NOT NULL REFERENCES locations(location_id),
    user_id      UUID REFERENCES users(user_id),
    display_name TEXT NOT NULL,
    pin_hash     TEXT,
    role         TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'active'
                 CHECK (status IN ('active', 'inactive', 'terminated')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_employees_org_id      ON employees(org_id);
CREATE INDEX IF NOT EXISTS idx_employees_org_status  ON employees(org_id, status);
CREATE INDEX IF NOT EXISTS idx_employees_org_location ON employees(org_id, location_id);
CREATE INDEX IF NOT EXISTS idx_employees_user_id     ON employees(user_id);
ALTER TABLE employees ENABLE ROW LEVEL SECURITY;
ALTER TABLE employees FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON employees;
CREATE POLICY org_isolation ON employees
    USING (org_id = current_setting('app.current_org_id', true)::UUID) WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

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
CREATE INDEX IF NOT EXISTS idx_audit_org_id      ON audit_log(org_id);
CREATE INDEX IF NOT EXISTS idx_audit_org_time     ON audit_log(org_id, created_at);
CREATE INDEX IF NOT EXISTS idx_audit_action       ON audit_log(org_id, action, created_at);
ALTER TABLE audit_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_log FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON audit_log;
CREATE POLICY org_isolation ON audit_log
    USING (org_id = current_setting('app.current_org_id', true)::UUID) WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- =========================================================================
-- 4. LOGIN FUNCTION (from 000)
-- =========================================================================
CREATE OR REPLACE FUNCTION fn_login_lookup(p_email TEXT)
RETURNS TABLE (
    user_id       UUID,
    org_id        UUID,
    role          TEXT,
    display_name  TEXT,
    password_hash TEXT
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
BEGIN
    RETURN QUERY
    SELECT u.user_id, u.org_id, u.role, u.display_name, u.password_hash
    FROM users u
    WHERE u.email = p_email
      AND u.status = 'active'
    LIMIT 1;
END;
$$;
REVOKE EXECUTE ON FUNCTION fn_login_lookup(TEXT) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION fn_login_lookup(TEXT) TO watchdog_app;

-- =========================================================================
-- 5. VISION SCHEMA (002)
-- =========================================================================
CREATE TABLE IF NOT EXISTS vision_cameras (
    camera_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id            UUID NOT NULL REFERENCES organizations(org_id),
    location_id       UUID REFERENCES locations(location_id),
    name              TEXT NOT NULL,
    rtsp_url          TEXT NOT NULL,
    status            TEXT NOT NULL DEFAULT 'pending'
                      CHECK (status IN ('online', 'offline', 'degraded', 'pending')),
    last_heartbeat_at TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_cameras_org_id       ON vision_cameras(org_id);
CREATE INDEX IF NOT EXISTS idx_cameras_org_location ON vision_cameras(org_id, location_id);
CREATE INDEX IF NOT EXISTS idx_cameras_org_status   ON vision_cameras(org_id, status);
ALTER TABLE vision_cameras ENABLE ROW LEVEL SECURITY;
ALTER TABLE vision_cameras FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON vision_cameras;
CREATE POLICY org_isolation ON vision_cameras
    USING (org_id = current_setting('app.current_org_id', true)::UUID) WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

CREATE TABLE IF NOT EXISTS vision_zones (
    zone_id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    camera_id  UUID NOT NULL REFERENCES vision_cameras(camera_id) ON DELETE CASCADE,
    org_id     UUID NOT NULL REFERENCES organizations(org_id),
    type       TEXT NOT NULL CHECK (type IN ('checkout','aisles','stockroom','back_office','entrance','restroom','restricted','privacy_mask')),
    name       TEXT,
    polygon    JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_zones_org_id      ON vision_zones(org_id);
CREATE INDEX IF NOT EXISTS idx_zones_camera_id   ON vision_zones(camera_id);
ALTER TABLE vision_zones ENABLE ROW LEVEL SECURITY;
ALTER TABLE vision_zones FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON vision_zones;
CREATE POLICY org_isolation ON vision_zones
    USING (org_id = current_setting('app.current_org_id', true)::UUID) WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

CREATE TABLE IF NOT EXISTS detections (
    detection_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id       UUID NOT NULL REFERENCES organizations(org_id),
    camera_id    UUID NOT NULL REFERENCES vision_cameras(camera_id),
    event_type   TEXT NOT NULL,
    severity     TEXT NOT NULL DEFAULT 'info' CHECK (severity IN ('critical','warning','info')),
    confidence   DOUBLE PRECISION,
    timestamp    TIMESTAMPTZ NOT NULL DEFAULT now(),
    payload      JSONB,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_detections_org_id       ON detections(org_id);
CREATE INDEX IF NOT EXISTS idx_detections_org_camera   ON detections(org_id, camera_id);
CREATE INDEX IF NOT EXISTS idx_detections_org_time     ON detections(org_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_detections_org_event    ON detections(org_id, event_type);
CREATE INDEX IF NOT EXISTS idx_detections_org_severity ON detections(org_id, severity);
ALTER TABLE detections ENABLE ROW LEVEL SECURITY;
ALTER TABLE detections FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON detections;
CREATE POLICY org_isolation ON detections
    USING (org_id = current_setting('app.current_org_id', true)::UUID) WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

CREATE TABLE IF NOT EXISTS vision_clips (
    clip_id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        UUID NOT NULL REFERENCES organizations(org_id),
    camera_id     UUID NOT NULL REFERENCES vision_cameras(camera_id),
    detection_id  UUID REFERENCES detections(detection_id),
    b2_key        TEXT,
    thumbnail_key TEXT,
    duration_secs INTEGER,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_clips_org_id       ON vision_clips(org_id);
CREATE INDEX IF NOT EXISTS idx_clips_org_camera   ON vision_clips(org_id, camera_id);
CREATE INDEX IF NOT EXISTS idx_clips_detection_id ON vision_clips(detection_id);
ALTER TABLE vision_clips ENABLE ROW LEVEL SECURITY;
ALTER TABLE vision_clips FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON vision_clips;
CREATE POLICY org_isolation ON vision_clips
    USING (org_id = current_setting('app.current_org_id', true)::UUID) WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- =========================================================================
-- 6. IDENTITY SCHEMA (003)
-- =========================================================================
CREATE TABLE IF NOT EXISTS identity_persons (
    person_id     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        UUID NOT NULL REFERENCES organizations(org_id),
    name          TEXT NOT NULL,
    kind          TEXT NOT NULL DEFAULT 'employee' CHECK (kind IN ('employee','contractor','visitor')),
    consent_status TEXT NOT NULL DEFAULT 'pending' CHECK (consent_status IN ('given','revoked','pending')),
    enrolled_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_persons_org_id   ON identity_persons(org_id);
CREATE INDEX IF NOT EXISTS idx_persons_org_kind ON identity_persons(org_id, kind);
ALTER TABLE identity_persons ENABLE ROW LEVEL SECURITY;
ALTER TABLE identity_persons FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON identity_persons;
CREATE POLICY org_isolation ON identity_persons
    USING (org_id = current_setting('app.current_org_id', true)::UUID) WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

CREATE TABLE IF NOT EXISTS identity_templates (
    template_id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id     UUID NOT NULL REFERENCES identity_persons(person_id) ON DELETE CASCADE,
    org_id        UUID NOT NULL REFERENCES organizations(org_id),
    embedding     BYTEA NOT NULL,
    algorithm     TEXT NOT NULL DEFAULT 'arcface',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_templates_org_id    ON identity_templates(org_id);
CREATE INDEX IF NOT EXISTS idx_templates_person_id ON identity_templates(person_id);
ALTER TABLE identity_templates ENABLE ROW LEVEL SECURITY;
ALTER TABLE identity_templates FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON identity_templates;
CREATE POLICY org_isolation ON identity_templates
    USING (org_id = current_setting('app.current_org_id', true)::UUID) WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

CREATE TABLE IF NOT EXISTS identity_consents (
    consent_id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id     UUID NOT NULL REFERENCES identity_persons(person_id) ON DELETE CASCADE,
    org_id        UUID NOT NULL REFERENCES organizations(org_id),
    status        TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('given','revoked','pending')),
    lawful_basis  TEXT,
    granted_at    TIMESTAMPTZ,
    revoked_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_consents_org_id    ON identity_consents(org_id);
CREATE INDEX IF NOT EXISTS idx_consents_person_id ON identity_consents(person_id);
ALTER TABLE identity_consents ENABLE ROW LEVEL SECURITY;
ALTER TABLE identity_consents FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON identity_consents;
CREATE POLICY org_isolation ON identity_consents
    USING (org_id = current_setting('app.current_org_id', true)::UUID) WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

CREATE TABLE IF NOT EXISTS identity_audit_log (
    log_id      BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    org_id      UUID NOT NULL REFERENCES organizations(org_id),
    person_id   UUID REFERENCES identity_persons(person_id),
    action      TEXT NOT NULL,
    performed_by UUID REFERENCES users(user_id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_idaudit_org_id  ON identity_audit_log(org_id);
CREATE INDEX IF NOT EXISTS idx_idaudit_person  ON identity_audit_log(person_id);
ALTER TABLE identity_audit_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE identity_audit_log FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON identity_audit_log;
CREATE POLICY org_isolation ON identity_audit_log
    USING (org_id = current_setting('app.current_org_id', true)::UUID) WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- =========================================================================
-- 7. WEBRTC SCHEMA (004)
-- =========================================================================
CREATE TABLE IF NOT EXISTS webrtc_sessions (
    session_id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id       UUID NOT NULL REFERENCES organizations(org_id),
    camera_id    UUID NOT NULL REFERENCES vision_cameras(camera_id),
    viewer_id    UUID REFERENCES users(user_id),
    sdp_offer    TEXT NOT NULL,
    sdp_answer   TEXT,
    status       TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','connected','closed','failed')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_webrtc_org_id     ON webrtc_sessions(org_id);
CREATE INDEX IF NOT EXISTS idx_webrtc_camera_id  ON webrtc_sessions(camera_id);
ALTER TABLE webrtc_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE webrtc_sessions FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON webrtc_sessions;
CREATE POLICY org_isolation ON webrtc_sessions
    USING (org_id = current_setting('app.current_org_id', true)::UUID) WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- =========================================================================
-- 8. NOTIFICATIONS SCHEMA (005)
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
CREATE INDEX IF NOT EXISTS idx_notif_rules_org_id        ON notification_rules(org_id);
CREATE INDEX IF NOT EXISTS idx_notif_rules_org_event     ON notification_rules(org_id, event_type);
CREATE INDEX IF NOT EXISTS idx_notif_rules_org_severity  ON notification_rules(org_id, severity);
CREATE INDEX IF NOT EXISTS idx_notif_rules_org_enabled   ON notification_rules(org_id, enabled);
ALTER TABLE notification_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification_rules FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON notification_rules;
CREATE POLICY org_isolation ON notification_rules
    USING (org_id = current_setting('app.current_org_id', true)::UUID) WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

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
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_notif_log_org_id      ON notification_log(org_id);
CREATE INDEX IF NOT EXISTS idx_notif_log_org_time    ON notification_log(org_id, created_at);
CREATE INDEX IF NOT EXISTS idx_notif_log_org_status  ON notification_log(org_id, status);
CREATE INDEX IF NOT EXISTS idx_notif_log_rule_id     ON notification_log(rule_id);
ALTER TABLE notification_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification_log FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS org_isolation ON notification_log;
CREATE POLICY org_isolation ON notification_log
    USING (org_id = current_setting('app.current_org_id', true)::UUID) WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- =========================================================================
-- 9. GRANTS
-- =========================================================================
GRANT SELECT, INSERT, UPDATE, DELETE ON
    organizations, locations, users, user_location_access, employees,
    vision_cameras, vision_zones, detections, vision_clips,
    identity_persons, identity_templates, identity_consents, identity_audit_log,
    webrtc_sessions, notification_rules, notification_log
TO watchdog_app;

GRANT SELECT, INSERT ON audit_log TO watchdog_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO watchdog_app;

-- =========================================================================
-- 10. SEED: Demo organization + owner user
-- =========================================================================
-- Password: DemoPassword1234! (bcrypt hash generated with cost 10)
INSERT INTO organizations (name, slug)
SELECT 'Easy Mart', 'easy-mart'
WHERE NOT EXISTS (SELECT 1 FROM organizations WHERE slug = 'easy-mart');

INSERT INTO users (org_id, email, password_hash, display_name, role)
SELECT
    (SELECT org_id FROM organizations WHERE slug = 'easy-mart'),
    'owner@easymart.com',
    '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', -- "DemoPassword1234!"
    'Store Owner',
    'owner'
WHERE NOT EXISTS (SELECT 1 FROM users WHERE email = 'owner@easymart.com');