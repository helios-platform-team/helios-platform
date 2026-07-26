-- Rollback initial schema setup

-- =====================================================
-- Drop tables
-- =====================================================

DROP TABLE IF EXISTS ${{ values.apiSchema }}.items CASCADE;

-- =====================================================
-- Drop roles (only if they exist and no other tables depend on them)
-- =====================================================

-- Note: In production, be careful dropping roles - they may have permissions
-- on other objects. This is safe for the example template.
DO $$ BEGIN
  DROP ROLE IF EXISTS ${{ values.anonRole }};
  DROP ROLE IF EXISTS ${{ values.jwtRole }};
EXCEPTION WHEN OTHERS THEN
  RAISE NOTICE 'Roles may still be in use: %', SQLERRM;
END $$;

-- =====================================================
-- Drop schema
-- =====================================================

DROP SCHEMA IF EXISTS ${{ values.apiSchema }} CASCADE;
