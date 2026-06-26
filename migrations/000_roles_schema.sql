-- 000_roles_schema.sql
-- Watch Dog retail compliance system — roles, grants, and RLS-safe login function.
-- Must run BEFORE all other migrations.

-- -----------------------------------------------------------------------
-- Application roles
-- -----------------------------------------------------------------------
-- watchdog_owner: runs migrations, owns tables
-- watchdog_app:   application runtime role, NOBYPASSRLS, subject to all RLS policies
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'watchdog_owner') THEN
        CREATE ROLE watchdog_owner NOLOGIN NOBYPASSRLS;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'watchdog_app') THEN
        CREATE ROLE watchdog_app NOLOGIN NOBYPASSRLS;
    END IF;
END $$;

-- -----------------------------------------------------------------------
-- SECURITY DEFINER function for login credential lookup
-- Bypasses RLS on users table to check credentials across all orgs.
-- Returns at most one row. The function runs as the table owner (SECURITY DEFINER)
-- so RLS is not applied, but it only returns user_id/org_id/role/display_name —
-- never password_hash or other sensitive columns.
-- -----------------------------------------------------------------------
CREATE OR REPLACE FUNCTION fn_login_lookup(p_email TEXT, p_password_hash TEXT)
RETURNS TABLE (
    user_id      UUID,
    org_id       UUID,
    role         TEXT,
    display_name TEXT
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
BEGIN
    RETURN QUERY
    SELECT u.user_id, u.org_id, u.role, u.display_name
    FROM users u
    WHERE u.email = p_email
      AND u.password_hash = p_password_hash
      AND u.status = 'active'
    LIMIT 1;
END;
$$;

-- Only watchdog_app can call the login function
REVOKE EXECUTE ON FUNCTION fn_login_lookup(TEXT, TEXT) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION fn_login_lookup(TEXT, TEXT) TO watchdog_app;

-- In Docker/dev, the postgres superuser (watchdog) needs to be able to
-- SET ROLE watchdog_app so RLS policies apply to application queries.
-- In production, the app connects directly as watchdog_app with its own credentials.
GRANT watchdog_app TO watchdog;
