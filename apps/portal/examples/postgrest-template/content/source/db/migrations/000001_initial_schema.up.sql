-- Initial schema setup for PostgREST API
-- This migration creates the foundational tables and roles for the API

-- =====================================================
-- Create the API schema
-- =====================================================
CREATE SCHEMA IF NOT EXISTS ${{ values.apiSchema }};
COMMENT ON SCHEMA ${{ values.apiSchema }} IS 'Public API schema exposed by PostgREST';

-- =====================================================
-- Create database roles
-- =====================================================

-- Role for unauthenticated API access
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '${{ values.anonRole }}') THEN
    CREATE ROLE ${{ values.anonRole }} NOLOGIN;
    COMMENT ON ROLE ${{ values.anonRole }} IS 'Role for anonymous (unauthenticated) API requests';
  END IF;
END $$;

-- Role for authenticated API access
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '${{ values.jwtRole }}') THEN
    CREATE ROLE ${{ values.jwtRole }} NOLOGIN;
    COMMENT ON ROLE ${{ values.jwtRole }} IS 'Role for authenticated API requests';
  END IF;
END $$;

-- =====================================================
-- Grant roles to the authenticator user
-- =====================================================

-- PostgREST needs the connection role (authenticator) to have permission
-- to switch to the anon and authenticated roles
GRANT ${{ values.anonRole }} TO CURRENT_USER;
GRANT ${{ values.jwtRole }} TO CURRENT_USER;

-- =====================================================
-- Schema permissions
-- =====================================================

-- Grant access to the API schema
GRANT USAGE ON SCHEMA ${{ values.apiSchema }} TO ${{ values.anonRole }};
GRANT USAGE ON SCHEMA ${{ values.apiSchema }} TO ${{ values.jwtRole }};

-- =====================================================
-- Example: Create a simple table
-- =====================================================

CREATE TABLE IF NOT EXISTS ${{ values.apiSchema }}.items (
  id SERIAL PRIMARY KEY,
  title TEXT NOT NULL,
  description TEXT,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

COMMENT ON TABLE ${{ values.apiSchema }}.items IS 'Example table exposed by the PostgREST API';
COMMENT ON COLUMN ${{ values.apiSchema }}.items.id IS 'Unique identifier';
COMMENT ON COLUMN ${{ values.apiSchema }}.items.title IS 'Item title';
COMMENT ON COLUMN ${{ values.apiSchema }}.items.description IS 'Item description';

-- Grant permissions on tables
GRANT SELECT, INSERT, UPDATE, DELETE ON ${{ values.apiSchema }}.items TO ${{ values.jwtRole }};
GRANT SELECT ON ${{ values.apiSchema }}.items TO ${{ values.anonRole }};

-- Grant sequence access for INSERT operations
GRANT USAGE, SELECT ON SEQUENCE ${{ values.apiSchema }}.items_id_seq TO ${{ values.jwtRole }};
GRANT USAGE, SELECT ON SEQUENCE ${{ values.apiSchema }}.items_id_seq TO ${{ values.anonRole }};

-- =====================================================
-- Notify PostgREST to reload the schema
-- =====================================================
NOTIFY pgrst, 'reload schema';
