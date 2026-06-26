-- 005_notifications_schema.sql
-- Watch Dog retail compliance system — alert routing & notification log.
-- Depends on 001_core_schema.sql (organizations).

-- ---------------------------------------------------------------------------
-- notification_rules
-- ---------------------------------------------------------------------------
CREATE TABLE notification_rules (
    rule_id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     UUID NOT NULL REFERENCES organizations(org_id),
    event_type TEXT NOT NULL,
    severity   TEXT NOT NULL
               CHECK (severity IN ('critical', 'warning', 'info')),
    channel    TEXT NOT NULL
               CHECK (channel IN ('telegram', 'email', 'sms', 'dashboard')),
    target     TEXT NOT NULL,
    enabled    BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_notif_rules_org_id        ON notification_rules(org_id);
CREATE INDEX idx_notif_rules_org_event     ON notification_rules(org_id, event_type);
CREATE INDEX idx_notif_rules_org_severity  ON notification_rules(org_id, severity);
CREATE INDEX idx_notif_rules_org_enabled   ON notification_rules(org_id, enabled);

ALTER TABLE notification_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification_rules FORCE ROW LEVEL SECURITY;
CREATE POLICY org_isolation ON notification_rules
    USING (org_id = current_setting('app.current_org_id', true)::UUID) WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- ---------------------------------------------------------------------------
-- notification_log
-- ---------------------------------------------------------------------------
CREATE TABLE notification_log (
    log_id     BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    org_id     UUID NOT NULL REFERENCES organizations(org_id),
    rule_id    UUID REFERENCES notification_rules(rule_id),
    event_type TEXT NOT NULL,
    severity   TEXT NOT NULL
               CHECK (severity IN ('critical', 'warning', 'info')),
    channel    TEXT NOT NULL
               CHECK (channel IN ('telegram', 'email', 'sms', 'dashboard')),
    target     TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'pending'
               CHECK (status IN ('sent', 'failed', 'pending')),
    error      TEXT,
    sent_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_notif_log_org_id      ON notification_log(org_id);
CREATE INDEX idx_notif_log_org_time    ON notification_log(org_id, created_at);
CREATE INDEX idx_notif_log_org_status  ON notification_log(org_id, status);
CREATE INDEX idx_notif_log_rule_id     ON notification_log(rule_id);

ALTER TABLE notification_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification_log FORCE ROW LEVEL SECURITY;
CREATE POLICY org_isolation ON notification_log
    USING (org_id = current_setting('app.current_org_id', true)::UUID) WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- ---------------------------------------------------------------------------
-- Grants
-- ---------------------------------------------------------------------------
GRANT SELECT, INSERT, UPDATE, DELETE ON
    notification_rules,
    notification_log
TO watchdog_app;

GRANT USAGE, SELECT ON
    notification_log_log_id_seq
TO watchdog_app;