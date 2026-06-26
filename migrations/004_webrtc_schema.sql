-- 004_webrtc_schema.sql
-- Watch Dog retail compliance system — WebRTC signaling & TURN credentials.
-- Depends on 001_core_schema.sql (organizations), 002_vision_schema.sql (cameras), users.

-- ---------------------------------------------------------------------------
-- webrtc_sessions
-- ---------------------------------------------------------------------------
CREATE TABLE webrtc_sessions (
    session_id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES organizations(org_id),
    camera_id       UUID NOT NULL REFERENCES cameras(camera_id),
    viewer_user_id  UUID REFERENCES users(user_id),
    sdp_offer       TEXT NOT NULL,
    sdp_answer      TEXT,
    ice_candidates  JSONB,
    status          TEXT NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'connected', 'closed', 'failed')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_webrtc_sessions_org_id     ON webrtc_sessions(org_id);
CREATE INDEX idx_webrtc_sessions_org_camera  ON webrtc_sessions(org_id, camera_id);
CREATE INDEX idx_webrtc_sessions_org_status  ON webrtc_sessions(org_id, status);
CREATE INDEX idx_webrtc_sessions_org_time    ON webrtc_sessions(org_id, created_at);

ALTER TABLE webrtc_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE webrtc_sessions FORCE ROW LEVEL SECURITY;
CREATE POLICY org_isolation ON webrtc_sessions
    USING (org_id = current_setting('app.current_org_id')::UUID);

-- ---------------------------------------------------------------------------
-- webrtc_turn_credentials
-- ---------------------------------------------------------------------------
CREATE TABLE webrtc_turn_credentials (
    credential_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        UUID NOT NULL REFERENCES organizations(org_id),
    username      TEXT NOT NULL,
    credential    TEXT NOT NULL,
    expires_at    TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_turn_creds_org_id    ON webrtc_turn_credentials(org_id);
CREATE INDEX idx_turn_creds_expires    ON webrtc_turn_credentials(expires_at);

ALTER TABLE webrtc_turn_credentials ENABLE ROW LEVEL SECURITY;
ALTER TABLE webrtc_turn_credentials FORCE ROW LEVEL SECURITY;
CREATE POLICY org_isolation ON webrtc_turn_credentials
    USING (org_id = current_setting('app.current_org_id')::UUID);

-- ---------------------------------------------------------------------------
-- Grants
-- ---------------------------------------------------------------------------
GRANT SELECT, INSERT, UPDATE, DELETE ON
    webrtc_sessions,
    webrtc_turn_credentials
TO watchdog_app;