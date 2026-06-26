-- 003_identity_schema.sql
-- Watch Dog retail compliance system — Face ID / identity stack (Egypt PDP Law compliant).
-- Depends on 001_core_schema.sql (organizations, employees).

-- pgvector for biometric template embeddings
CREATE EXTENSION IF NOT EXISTS "vector";

-- ---------------------------------------------------------------------------
-- identity_persons
-- ---------------------------------------------------------------------------
CREATE TABLE identity_persons (
    person_id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id       UUID NOT NULL REFERENCES organizations(org_id),
    kind         TEXT NOT NULL CHECK (kind IN ('employee', 'customer')),
    employee_id  UUID REFERENCES employees(employee_id),
    customer_id  UUID,
    display_name TEXT NOT NULL,
    enrolled_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_identity_persons_org_id      ON identity_persons(org_id);
CREATE INDEX idx_identity_persons_org_kind   ON identity_persons(org_id, kind);
CREATE INDEX idx_identity_persons_employee   ON identity_persons(employee_id);

ALTER TABLE identity_persons ENABLE ROW LEVEL SECURITY;
ALTER TABLE identity_persons FORCE ROW LEVEL SECURITY;
CREATE POLICY org_isolation ON identity_persons
    USING (org_id = current_setting('app.current_org_id')::UUID);

-- ---------------------------------------------------------------------------
-- identity_templates  (encrypted-at-rest biometric embeddings)
-- ---------------------------------------------------------------------------
CREATE TABLE identity_templates (
    template_id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        UUID NOT NULL REFERENCES organizations(org_id),
    person_id     UUID NOT NULL REFERENCES identity_persons(person_id) ON DELETE CASCADE,
    embedding     VECTOR(512) NOT NULL,
    embedding_ct  BYTEA,          -- ciphertext of embedding (envelope encryption)
    embedding_nonce BYTEA,         -- AEAD nonce
    kms_dek_id    TEXT NOT NULL,
    quality_score NUMERIC(4,3),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_identity_templates_org_id   ON identity_templates(org_id);
CREATE INDEX idx_identity_templates_person_id ON identity_templates(person_id);

-- HNSW vector index for cosine-similarity face search
CREATE INDEX idx_identity_templates_embedding_hnsw
    ON identity_templates
    USING hnsw (embedding vector_cosine_ops);

ALTER TABLE identity_templates ENABLE ROW LEVEL SECURITY;
ALTER TABLE identity_templates FORCE ROW LEVEL SECURITY;
CREATE POLICY org_isolation ON identity_templates
    USING (org_id = current_setting('app.current_org_id')::UUID);

-- ---------------------------------------------------------------------------
-- identity_consents  (PDP Law — explicit, revocable consent record)
-- ---------------------------------------------------------------------------
CREATE TABLE identity_consents (
    consent_id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES organizations(org_id),
    person_id       UUID NOT NULL REFERENCES identity_persons(person_id) ON DELETE CASCADE,
    captured_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    consent_text    TEXT NOT NULL,
    consent_locale  TEXT NOT NULL CHECK (consent_locale IN ('en', 'ar')),
    signature_sha256 BYTEA NOT NULL,
    captured_by     TEXT NOT NULL,
    captured_ip     INET,
    revoked         BOOLEAN NOT NULL DEFAULT false,
    revoked_at     TIMESTAMPTZ,
    revoked_reason TEXT
);

CREATE INDEX idx_identity_consents_org_id    ON identity_consents(org_id);
CREATE INDEX idx_identity_consents_person_id ON identity_consents(person_id);
CREATE INDEX idx_identity_consents_revoked   ON identity_consents(org_id, revoked);

ALTER TABLE identity_consents ENABLE ROW LEVEL SECURITY;
ALTER TABLE identity_consents FORCE ROW LEVEL SECURITY;
CREATE POLICY org_isolation ON identity_consents
    USING (org_id = current_setting('app.current_org_id')::UUID);

-- ---------------------------------------------------------------------------
-- identity_access_audit  (every access to biometric data is logged)
-- ---------------------------------------------------------------------------
CREATE TABLE identity_access_audit (
    audit_id     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id       UUID NOT NULL REFERENCES organizations(org_id),
    person_id    UUID,
    accessed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    purpose      TEXT NOT NULL
                 CHECK (purpose IN ('recognize', 'enroll', 'review', 'export', 'erase')),
    triggered_by TEXT NOT NULL,
    camera_id    UUID,
    detection_id UUID,
    notes        TEXT
);

CREATE INDEX idx_identity_access_audit_org_id   ON identity_access_audit(org_id);
CREATE INDEX idx_identity_access_audit_person   ON identity_access_audit(person_id);
CREATE INDEX idx_identity_access_audit_time      ON identity_access_audit(org_id, accessed_at);

ALTER TABLE identity_access_audit ENABLE ROW LEVEL SECURITY;
ALTER TABLE identity_access_audit FORCE ROW LEVEL SECURITY;
CREATE POLICY org_isolation ON identity_access_audit
    USING (org_id = current_setting('app.current_org_id')::UUID);

-- ---------------------------------------------------------------------------
-- identity_dek  (per-org data-encryption-key envelope; supports rotation)
-- ---------------------------------------------------------------------------
CREATE TABLE identity_dek (
    dek_id        TEXT PRIMARY KEY,
    org_id        UUID NOT NULL REFERENCES organizations(org_id),
    wrapped_key   BYTEA NOT NULL,
    kms_key_arn   TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    rotated_out_at TIMESTAMPTZ,
    UNIQUE (org_id, dek_id)
);

CREATE INDEX idx_identity_dek_org_id ON identity_dek(org_id);

ALTER TABLE identity_dek ENABLE ROW LEVEL SECURITY;
ALTER TABLE identity_dek FORCE ROW LEVEL SECURITY;
CREATE POLICY org_isolation ON identity_dek
    USING (org_id = current_setting('app.current_org_id')::UUID);

-- ---------------------------------------------------------------------------
-- face_erasure_log  (right-to-erasure audit trail per PDP Law)
-- ---------------------------------------------------------------------------
CREATE TABLE face_erasure_log (
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

CREATE INDEX idx_face_erasure_log_org_id    ON face_erasure_log(org_id);
CREATE INDEX idx_face_erasure_log_person_id ON face_erasure_log(person_id);

ALTER TABLE face_erasure_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE face_erasure_log FORCE ROW LEVEL SECURITY;
CREATE POLICY org_isolation ON face_erasure_log
    USING (org_id = current_setting('app.current_org_id')::UUID);

-- ---------------------------------------------------------------------------
-- Grants
-- ---------------------------------------------------------------------------
GRANT SELECT, INSERT, UPDATE, DELETE ON
    identity_persons,
    identity_templates,
    identity_consents,
    identity_access_audit,
    identity_dek,
    face_erasure_log
TO watchdog_app;