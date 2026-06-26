-- 001_core_schema.sql
-- Watch Dog retail compliance system — base schema (organizations, locations, users, employees, audit_log)
-- Forward-only migration. All tables use org_id for Row-Level Security.

-- Extensions
CREATE EXTENSION IF NOT EXISTS "pgcrypto";   -- gen_random_uuid()

-- ---------------------------------------------------------------------------
-- organizations
-- ---------------------------------------------------------------------------
CREATE TABLE organizations (
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
CREATE POLICY org_isolation ON organizations
    USING (org_id = current_setting('app.current_org_id', true)::UUID) WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- ---------------------------------------------------------------------------
-- locations
-- ---------------------------------------------------------------------------
CREATE TABLE locations (
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

CREATE INDEX idx_locations_org_id       ON locations(org_id);
CREATE INDEX idx_locations_org_status    ON locations(org_id, status);

ALTER TABLE locations ENABLE ROW LEVEL SECURITY;
ALTER TABLE locations FORCE ROW LEVEL SECURITY;
CREATE POLICY org_isolation ON locations
    USING (org_id = current_setting('app.current_org_id', true)::UUID) WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- ---------------------------------------------------------------------------
-- users
-- ---------------------------------------------------------------------------
CREATE TABLE users (
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

CREATE INDEX idx_users_org_id    ON users(org_id);
CREATE INDEX idx_users_org_status ON users(org_id, status);

ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE users FORCE ROW LEVEL SECURITY;
CREATE POLICY org_isolation ON users
    USING (org_id = current_setting('app.current_org_id', true)::UUID) WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- ---------------------------------------------------------------------------
-- user_location_access
-- ---------------------------------------------------------------------------
CREATE TABLE user_location_access (
    user_id     UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    location_id UUID NOT NULL REFERENCES locations(location_id) ON DELETE CASCADE,
    org_id      UUID NOT NULL REFERENCES organizations(org_id),
    PRIMARY KEY (user_id, location_id)
);

CREATE INDEX idx_ula_org_id        ON user_location_access(org_id);
CREATE INDEX idx_ula_location_id   ON user_location_access(location_id);

ALTER TABLE user_location_access ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_location_access FORCE ROW LEVEL SECURITY;
CREATE POLICY org_isolation ON user_location_access
    USING (org_id = current_setting('app.current_org_id', true)::UUID) WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- ---------------------------------------------------------------------------
-- employees
-- ---------------------------------------------------------------------------
CREATE TABLE employees (
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

CREATE INDEX idx_employees_org_id      ON employees(org_id);
CREATE INDEX idx_employees_org_status  ON employees(org_id, status);
CREATE INDEX idx_employees_org_location ON employees(org_id, location_id);
CREATE INDEX idx_employees_user_id     ON employees(user_id);

ALTER TABLE employees ENABLE ROW LEVEL SECURITY;
ALTER TABLE employees FORCE ROW LEVEL SECURITY;
CREATE POLICY org_isolation ON employees
    USING (org_id = current_setting('app.current_org_id', true)::UUID) WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- ---------------------------------------------------------------------------
-- audit_log
-- ---------------------------------------------------------------------------
CREATE TABLE audit_log (
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

CREATE INDEX idx_audit_org_id      ON audit_log(org_id);
CREATE INDEX idx_audit_org_time     ON audit_log(org_id, created_at);
CREATE INDEX idx_audit_action       ON audit_log(org_id, action, created_at);

ALTER TABLE audit_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_log FORCE ROW LEVEL SECURITY;
CREATE POLICY org_isolation ON audit_log
    USING (org_id = current_setting('app.current_org_id', true)::UUID) WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- ---------------------------------------------------------------------------
-- Grants
-- ---------------------------------------------------------------------------
GRANT SELECT, INSERT, UPDATE, DELETE ON
    organizations,
    locations,
    users,
    user_location_access,
    employees
TO watchdog_app;

-- Audit log is append-only — no UPDATE or DELETE
GRANT SELECT, INSERT ON
    audit_log
TO watchdog_app;

GRANT USAGE, SELECT ON
    audit_log_log_id_seq
TO watchdog_app;