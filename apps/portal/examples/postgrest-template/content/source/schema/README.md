# Database Schema

This directory contains SQL files that define your database schema. Add your database tables, views, and permissions here.

## How It Works

1. **Add SQL Files**: Create `.sql` files in this directory (e.g., `01-tables.sql`, `02-permissions.sql`)
2. **Automatic Loading**: When the Docker image is built, these files are copied to the container
3. **Schema Applied**: When PostgreSQL starts (via Helios Operator), the Operator applies your schema
4. **REST API Generated**: PostgREST automatically exposes your tables as REST endpoints

## File Structure

Use numbered prefixes to control execution order:

```
schema/
  01-tables.sql          # CREATE TABLE statements
  02-permissions.sql     # GRANT statements for roles
  03-views.sql           # CREATE VIEW statements
```

## Example: 01-tables.sql

```sql
-- Create a users table
CREATE TABLE users (
  id SERIAL PRIMARY KEY,
  email TEXT NOT NULL UNIQUE,
  name TEXT,
  created_at TIMESTAMP DEFAULT NOW()
);

-- Create a posts table
CREATE TABLE posts (
  id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL REFERENCES users(id),
  title TEXT NOT NULL,
  body TEXT,
  published BOOLEAN DEFAULT FALSE,
  created_at TIMESTAMP DEFAULT NOW()
);
```

## Example: 02-permissions.sql

```sql
-- Create roles for authentication
CREATE ROLE anon NOLOGIN;
CREATE ROLE authenticated NOLOGIN IN ROLE anon;

-- Grant read access to anon role
GRANT SELECT ON users, posts TO anon;

-- Grant full access to authenticated role
GRANT ALL ON users, posts TO authenticated;
```

## How PostgREST Exposes Your Schema

Once your schema is loaded, PostgREST automatically creates REST endpoints:

```bash
# List all users
GET /users

# Create a new user
POST /users
Content-Type: application/json
{ "email": "user@example.com", "name": "John Doe" }

# Get a specific user
GET /users?id=eq.1

# Update a user
PATCH /users?id=eq.1
{ "name": "Jane Doe" }

# Delete a user
DELETE /users?id=eq.1
```

## Authentication

Set up JWT authentication in your PostgREST configuration (`postgrestrc.conf`):

```ini
db-uri = "postgres://..."
db-schema = "public"
db-anon-role = "anon"
jwt-secret = "your-secret-key"
jwt-claim-check-aud = false
jwt-claim-check-sub = false
```

Then reference this:

```bash
# Include JWT token in Authorization header
curl -H "Authorization: Bearer $JWT_TOKEN" \
  http://api.example.com/posts
```

## Best Practices

1. **Use descriptive names**: `users`, `posts`, `comments` (plural)
2. **Add timestamps**: `created_at`, `updated_at` for audit trails
3. **Use constraints**: NOT NULL, UNIQUE, FOREIGN KEY for data integrity
4. **Define roles early**: Separate `anon` and `authenticated` for different access levels
5. **Number your files**: `01-`, `02-`, `03-` to ensure correct load order
6. **Keep it simple**: Start with basic CRUD, add views and functions later

## Testing Locally

```bash
docker build -t my-api:latest .
docker run -e PGRST_DB_URI="postgres://user:pass@localhost/mydb" my-api:latest
```

## More Information

- [PostgREST Documentation](https://postgrest.org/en/stable/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Helios Operator Documentation](../../docs/OPERATOR.md)
