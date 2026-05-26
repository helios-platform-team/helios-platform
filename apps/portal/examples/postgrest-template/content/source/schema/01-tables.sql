-- Example: Create basic tables
-- Replace this with your own schema

CREATE TABLE users (
  id SERIAL PRIMARY KEY,
  email TEXT NOT NULL UNIQUE,
  name TEXT,
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE posts (
  id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL REFERENCES users(id),
  title TEXT NOT NULL,
  body TEXT,
  published BOOLEAN DEFAULT FALSE,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

-- PostgREST automatically creates REST endpoints for these tables:
-- GET    /users           - List all users
-- POST   /users           - Create a new user
-- GET    /users?id=eq.1   - Get user with id=1
-- PATCH  /users?id=eq.1   - Update user with id=1
-- DELETE /users?id=eq.1   - Delete user with id=1
--
-- Same for /posts, /posts/{id}, etc.
