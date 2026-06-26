-- 002_vision_schema.sql
-- Watch Dog retail compliance system — CCTV / vision pipeline schema.
-- Depends on 001_core_schema.sql (organizations, locations).

-- ---------------------------------------------------------------------------
-- cameras
-- ---------------------------------------------------------------------------
CREATE TABLE cameras (
    camera_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id           UUID NOT NULL REFERENCES organizations(org_id),
    location_id      UUID NOT NULL REFERENCES locations(location_id),
    name             TEXT NOT NULL,
    rtsp_url         TEXT,
    local_agent_id   TEXT,
    status           TEXT NOT NULL DEFAULT 'pending'
                     CHECK (status IN ('pending', 'online', 'offline', 'degraded')),
    last_heartbeat_at TIMESTAMPTZ,
    feature_flags    JSONB NOT NULL DEFAULT '{}',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_cameras_org_location ON cameras(org_id, location_id);
CREATE INDEX idx_cameras_org_status   ON cameras(org_id, status);

ALTER TABLE cameras ENABLE ROW LEVEL SECURITY;
ALTER TABLE cameras FORCE ROW LEVEL SECURITY;
CREATE POLICY org_isolation ON cameras
    USING (org_id = current_setting('app.current_org_id', true)::UUID) WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- ---------------------------------------------------------------------------
-- zones
-- ---------------------------------------------------------------------------
CREATE TABLE zones (
    zone_id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id    UUID NOT NULL REFERENCES organizations(org_id),
    camera_id UUID NOT NULL REFERENCES cameras(camera_id) ON DELETE CASCADE,
    name      TEXT NOT NULL,
    kind      TEXT NOT NULL
              CHECK (kind IN ('checkout', 'aisles', 'stockroom', 'back_office',
                              'entrance', 'restroom', 'restricted', 'privacy_mask')),
    polygon   JSONB,
    capacity  INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_zones_org_id     ON zones(org_id);
CREATE INDEX idx_zones_camera_id  ON zones(camera_id);

ALTER TABLE zones ENABLE ROW LEVEL SECURITY;
ALTER TABLE zones FORCE ROW LEVEL SECURITY;
CREATE POLICY org_isolation ON zones
    USING (org_id = current_setting('app.current_org_id', true)::UUID) WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- ---------------------------------------------------------------------------
-- clips
-- ---------------------------------------------------------------------------
CREATE TABLE clips (
    clip_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id           UUID NOT NULL REFERENCES organizations(org_id),
    location_id      UUID NOT NULL REFERENCES locations(location_id),
    camera_id        UUID NOT NULL REFERENCES cameras(camera_id),
    s3_bucket        TEXT NOT NULL,
    s3_key           TEXT NOT NULL,
    duration_seconds INT,
    starts_at        TIMESTAMPTZ NOT NULL,
    retention_tier   TEXT NOT NULL DEFAULT 'hot'
                     CHECK (retention_tier IN ('hot', 'warm', 'glacier', 'deleted')),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_clips_org_location ON clips(org_id, location_id);
CREATE INDEX idx_clips_org_time     ON clips(org_id, created_at);
CREATE INDEX idx_clips_camera_time  ON clips(camera_id, starts_at);

ALTER TABLE clips ENABLE ROW LEVEL SECURITY;
ALTER TABLE clips FORCE ROW LEVEL SECURITY;
CREATE POLICY org_isolation ON clips
    USING (org_id = current_setting('app.current_org_id', true)::UUID) WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- ---------------------------------------------------------------------------
-- detections
-- ---------------------------------------------------------------------------
CREATE TABLE detections (
    detection_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id       UUID NOT NULL REFERENCES organizations(org_id),
    location_id  UUID NOT NULL REFERENCES locations(location_id),
    camera_id    UUID NOT NULL REFERENCES cameras(camera_id),
    zone_id      UUID REFERENCES zones(zone_id) ON DELETE SET NULL,
    event_type   TEXT NOT NULL,
    severity     TEXT NOT NULL DEFAULT 'info'
                 CHECK (severity IN ('critical', 'warning', 'info')),
    confidence   NUMERIC(5,4),
    payload      JSONB,
    clip_id      UUID REFERENCES clips(clip_id) ON DELETE SET NULL,
    detected_at  TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_detections_org_location     ON detections(org_id, location_id);
CREATE INDEX idx_detections_org_status       ON detections(org_id, severity);
CREATE INDEX idx_detections_camera_time      ON detections(camera_id, detected_at);
CREATE INDEX idx_detections_org_event_time   ON detections(org_id, event_type, detected_at);
CREATE INDEX idx_detections_org_severity_time ON detections(org_id, severity, detected_at);

ALTER TABLE detections ENABLE ROW LEVEL SECURITY;
ALTER TABLE detections FORCE ROW LEVEL SECURITY;
CREATE POLICY org_isolation ON detections
    USING (org_id = current_setting('app.current_org_id', true)::UUID) WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- ---------------------------------------------------------------------------
-- vision_severity_overrides
-- ---------------------------------------------------------------------------
CREATE TABLE vision_severity_overrides (
    org_id    UUID NOT NULL REFERENCES organizations(org_id),
    event_type TEXT NOT NULL,
    severity  TEXT NOT NULL
              CHECK (severity IN ('critical', 'warning', 'info')),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (org_id, event_type)
);

ALTER TABLE vision_severity_overrides ENABLE ROW LEVEL SECURITY;
ALTER TABLE vision_severity_overrides FORCE ROW LEVEL SECURITY;
CREATE POLICY org_isolation ON vision_severity_overrides
    USING (org_id = current_setting('app.current_org_id', true)::UUID) WITH CHECK (org_id = current_setting('app.current_org_id', true)::UUID);

-- ---------------------------------------------------------------------------
-- Grants
-- ---------------------------------------------------------------------------
GRANT SELECT, INSERT, UPDATE, DELETE ON
    cameras,
    zones,
    clips,
    detections,
    vision_severity_overrides
TO watchdog_app;