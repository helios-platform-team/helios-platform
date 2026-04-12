-- Initial schema setup for PostgREST API
-- This migration creates the foundational tables and roles for the API

-- =====================================================
-- Create the API schema
-- =====================================================
CREATE SCHEMA IF NOT EXISTS api;
COMMENT ON SCHEMA api IS 'Public API schema exposed by PostgREST';

-- =====================================================
-- Create database roles
-- =====================================================

-- Role for unauthenticated API access
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'anon') THEN
    CREATE ROLE anon NOLOGIN;
    COMMENT ON ROLE anon IS 'Role for anonymous (unauthenticated) API requests';
  END IF;
END $$;

-- Role for authenticated API access
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'authenticated') THEN
    CREATE ROLE authenticated NOLOGIN;
    COMMENT ON ROLE authenticated IS 'Role for authenticated API requests';
  END IF;
END $$;

-- =====================================================
-- Schema permissions
-- =====================================================

-- Grant access to the API schema
GRANT USAGE ON SCHEMA api TO anon;
GRANT USAGE ON SCHEMA api TO authenticated;

-- =====================================================
-- Example: Create a simple table
-- =====================================================

CREATE TABLE IF NOT EXISTS api.items (
  id SERIAL PRIMARY KEY,
  title TEXT NOT NULL,
  description TEXT,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

COMMENT ON TABLE api.items IS 'Example table exposed by the PostgREST API';
COMMENT ON COLUMN api.items.id IS 'Unique identifier';
COMMENT ON COLUMN api.items.title IS 'Item title';
COMMENT ON COLUMN api.items.description IS 'Item description';

-- Grant permissions on tables
GRANT SELECT, INSERT, UPDATE, DELETE ON api.items TO authenticated;
GRANT SELECT ON api.items TO anon;

-- Grant sequence access for INSERT operations
GRANT USAGE, SELECT ON SEQUENCE api.items_id_seq TO authenticated;
GRANT USAGE, SELECT ON SEQUENCE api.items_id_seq TO anon;

-- =====================================================
-- Notify PostgREST to reload the schema
-- =====================================================
NOTIFY pgrst, 'reload schema';
