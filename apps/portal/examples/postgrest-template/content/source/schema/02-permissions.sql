-- Example: Set up roles and permissions
-- Customize this based on your authentication requirements

-- Create roles (if not already created by Helios Operator)
DO $$
BEGIN
  CREATE ROLE anon NOLOGIN;
EXCEPTION WHEN DUPLICATE_OBJECT THEN
  NULL;
END $$;

DO $$
BEGIN
  CREATE ROLE authenticated NOLOGIN;
EXCEPTION WHEN DUPLICATE_OBJECT THEN
  NULL;
END $$;

-- Remove old grants (if any)
REVOKE ALL ON users, posts FROM anon, authenticated;

-- Grant SELECT access to anonymous users
GRANT SELECT ON users, posts TO anon;

-- Grant CRUD access to authenticated users
GRANT SELECT, INSERT, UPDATE, DELETE ON users, posts TO authenticated;

-- Allow users to update their own records (example)
-- Note: PostgREST also supports row-level security for fine-grained control
CREATE POLICY user_self_update ON users FOR UPDATE
  USING (id = current_user_id());
