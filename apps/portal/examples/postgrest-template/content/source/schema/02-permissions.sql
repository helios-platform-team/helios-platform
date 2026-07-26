-- Example: Set up roles and permissions
-- Customize this based on your authentication requirements

-- Create roles (if not already created by Helios Operator)
DO $$
BEGIN
  CREATE ROLE ${{ values.anonRole }} NOLOGIN;
EXCEPTION WHEN DUPLICATE_OBJECT THEN
  NULL;
END $$;

DO $$
BEGIN
  CREATE ROLE ${{ values.jwtRole }} NOLOGIN;
EXCEPTION WHEN DUPLICATE_OBJECT THEN
  NULL;
END $$;

-- PostgREST needs the connection role (authenticator) to have permission
-- to switch to the anon and authenticated roles
GRANT ${{ values.anonRole }} TO CURRENT_USER;
GRANT ${{ values.jwtRole }} TO CURRENT_USER;

-- Remove old grants (if any)
REVOKE ALL ON users, posts FROM ${{ values.anonRole }}, ${{ values.jwtRole }};

-- Grant SELECT access to anonymous users
GRANT SELECT ON users, posts TO ${{ values.anonRole }};

-- Grant CRUD access to authenticated users
GRANT SELECT, INSERT, UPDATE, DELETE ON users, posts TO ${{ values.jwtRole }};

-- Allow users to update their own records (example)
-- Note: PostgREST also supports row-level security for fine-grained control
CREATE POLICY user_self_update ON users FOR UPDATE
  USING (id = current_user_id());
