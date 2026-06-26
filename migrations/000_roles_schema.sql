-- 000_roles_schema.sql
-- Watch Dog retail compliance system — roles, grants, and RLS-safe login function.
-- Must run BEFORE all other migrations.

-- -----------------------------------------------------------------------
-- Application roles
-- -----------------------------------------------------------------------
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'watchdog_owner') THEN
        CREATE ROLE watchdog_owner NOLOGIN NOBYPASSRLS;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'watchdog_app') THEN
        CREATE ROLE watchdog_app NOLOGIN NOBYPASSRLS;
    END IF;
END $$;

-- In Docker/dev, the postgres superuser needs to be able to SET ROLE watchdog_app
GRANT watchdog_app TO watchdog;

-- -----------------------------------------------------------------------
-- SECURITY DEFINER function for login credential lookup
-- Bypasses RLS to look up a user by email. Returns the password_hash so
-- the Go backend can verify it with bcrypt (not direct string comparison).
-- -----------------------------------------------------------------------
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
