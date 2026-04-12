-- Rollback initial schema setup

-- =====================================================
-- Drop tables
-- =====================================================

DROP TABLE IF EXISTS api.items CASCADE;

-- =====================================================
-- Drop roles
-- =====================================================

DROP ROLE IF EXISTS {{ values.anonRole }};
DROP ROLE IF EXISTS {{ values.jwtRole }};

-- =====================================================
-- Drop schema
-- =====================================================

DROP SCHEMA IF EXISTS api CASCADE;
