-- 007_updated_at_triggers.sql
-- Watch Dog retail compliance system — auto-update updated_at on row change.

-- -----------------------------------------------------------------------
-- Reusable trigger function: sets updated_at = now() on row update
-- -----------------------------------------------------------------------
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$;

-- -----------------------------------------------------------------------
-- Apply to all tables that have an updated_at column
-- -----------------------------------------------------------------------

-- Core schema (001)
CREATE TRIGGER trg_organizations_updated_at
    BEFORE UPDATE ON organizations
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_locations_updated_at
    BEFORE UPDATE ON locations
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_employees_updated_at
    BEFORE UPDATE ON employees
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Vision schema (002)
CREATE TRIGGER trg_cameras_updated_at
    BEFORE UPDATE ON cameras
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Identity schema (003)
CREATE TRIGGER trg_identity_persons_updated_at
    BEFORE UPDATE ON identity_persons
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- WebRTC schema (004)
CREATE TRIGGER trg_webrtc_sessions_updated_at
    BEFORE UPDATE ON webrtc_sessions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Notifications schema (005)
CREATE TRIGGER trg_notification_rules_updated_at
    BEFORE UPDATE ON notification_rules
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
