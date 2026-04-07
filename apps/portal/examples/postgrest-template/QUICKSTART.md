# PostgREST Template Quick Start Guide

## Overview

The **PostgREST API Template** enables developers to instantly create production-ready REST APIs directly from PostgreSQL database schemas—without writing any backend code.

## What is PostgREST?

[PostgREST](https://postgrest.org) automatically generates a RESTful API from any PostgreSQL database. Define your tables in SQL, and PostgREST exposes them via standard HTTP endpoints.

### Example: Define Once, Get API Instantly

```sql
-- 1. Create table in PostgreSQL
CREATE TABLE posts (
  id SERIAL PRIMARY KEY,
  title TEXT NOT NULL,
  body TEXT,
  created_at TIMESTAMP DEFAULT NOW()
);

-- 2. PostgREST automatically generates these endpoints:
GET    /posts           # List all posts
POST   /posts           # Create a post
GET    /posts/{id}      # Read a post
PUT    /posts/{id}      # Update a post
DELETE /posts/{id}      # Delete a post

-- 3. Filtering, sorting, pagination work automatically:
GET /posts?title=ilike.*Hello*&order=created_at.desc&limit=10
```

## How to Use the Template

### Step 1: Access the Template

1. Open **Backstage Portal**: http://localhost:3000
2. Click **Create** (top menu)
3. Find **"PostgREST API Template"** in the list

### Step 2: Configure Your Service

**Component Information**
- **Name**: Human-readable service name (e.g., "Blog API")
- **Port**: API server port (default: 3000)
- **Docker Organization**: Docker registry namespace
- **Repository Name**: URL-safe name (e.g., blog-api)

**PostgREST Configuration**
- **API Schema**: PostgreSQL schema to expose (default: `public`)
  - Use `public` for all tables
  - Use `api` for curated subset
- **JWT Secret**: Secret for signing JWT tokens (32+ chars recommended)
- **JWT Role**: Role claim in authentication tokens (default: `authenticated`)
- **Anonymous Role**: Role for unauthenticated requests (default: `anon`)

**Database Configuration**
- Database type: PostgreSQL (auto-selected)
- Database name: Auto-generated or custom

**Repository & Webhook**
- Choose your Git provider (Gitea, GitHub, etc.)
- Template automatically creates source and GitOps repositories
- Webhooks trigger CI/CD on code changes

### Step 3: Review and Execute

- Click through remaining steps
- Accept defaults or customize as needed
- Click **Create** to scaffold your service

### Step 4: Scaffolding Completes Automatically

The Helios Operator will:
1. ✅ Provision PostgreSQL database
2. ✅ Generate secure credentials
3. ✅ Inject connection string to PostgREST
4. ✅ Deploy PostgREST container
5. ✅ Expose service via Ingress
6. ✅ Setup Tekton CI/CD pipeline
7. ✅ Configure ArgoCD GitOps sync

## Access Your API

### From Within Cluster
```bash
# Internal DNS
http://my-api-service.default.svc.cluster.local:3000
```

### From Outside Cluster (via Ingress)
```bash
# External hostname (configured during scaffolding)
http://my-api.local/
```

## Working with Your PostgREST API

### Write SQL, Get REST Endpoints

Connect to your PostgreSQL database and create tables:

```sql
CREATE SCHEMA api;

CREATE TABLE api.users (
  id SERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  email TEXT UNIQUE NOT NULL
);

CREATE TABLE api.posts (
  id SERIAL PRIMARY KEY,
  user_id INT REFERENCES api.users(id),
  title TEXT NOT NULL,
  body TEXT,
  published_at TIMESTAMP
);
```

PostgREST automatically generates endpoints:
- `/users` — User CRUD operations
- `/posts` — Post CRUD operations

### Filter, Sort, Paginate

```bash
# Filter
GET /posts?published_at=gt.2026-01-01

# Sort
GET /posts?order=published_at.desc

# Paginate
GET /posts?limit=10&offset=20

# Complex queries
GET /posts?user_id=eq.5&title=like.*PostgreSQL*&order=published_at.desc&limit=5
```

### Authentication with JWT

```bash
# 1. Provide JWT secret in token (if JWT_SECRET set)
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

# 2. Include in requests
curl -H "Authorization: Bearer $TOKEN" http://api/posts

# 3. PostgREST extracts role from token
# And enforces PostgreSQL row-level security (RLS)
```

### Row-Level Security (RLS)

Enforce database-level permissions:

```sql
-- Only users can see their own posts
CREATE POLICY user_posts ON posts 
  USING (user_id = current_user_id());

-- Admins can see everything
CREATE POLICY admin_all ON posts 
  USING (current_role = 'admin');
```

## Environment Variables

Automatically set by Helios Operator:

| Variable | Value | Notes |
|----------|-------|-------|
| `PGRST_DB_URI` | `postgres://user:pass@host:5432/db` | PostgreSQL connection string |
| `PGRST_DB_SCHEMA` | Configured schema (default: `public`) | Exposed schema |
| `PGRST_DB_ANON_ROLE` | Configured role (default: `anon`) | Role for public access |
| `PGRST_JWT_AUDIENCE` | Configured role (default: `authenticated`) | JWT audience |
| `PGRST_MAX_ROWS` | `1000` | Max rows returned per request |

See [PostgREST Configuration](https://postgrest.org/en/latest/references/config.html) for all options.

## Example Use Cases

### 1. Real-Time Dashboard Backend
```yaml
# Store metrics in PostgreSQL
table: metrics (id, timestamp, value, metric_name)
schema: public

# PostgREST serves:
GET /metrics?metric_name=eq.cpu&order=timestamp.desc&limit=100
```

### 2. GraphQL-Compatible API
```yaml
# PostgREST serves REST-QL
POST /rpc/graphql { query: "..." }
```

### 3. Mobile App Backend
```yaml
# Simple REST endpoints
GET /users/{id}
POST /users
PUT /users/{id}
DELETE /users/{id}

# With RLS for multi-tenant isolation
```

### 4. Internal Tools
```yaml
# Zero setup, immediate REST API
# Perfect for admin dashboards, reporting tools
GET /data?filter=value&download=csv
```

## Architecture

```
┌─────────────────────────────────────────────────┐
│          Backstage Portal                       │
│  (Template Registration & Scaffolding UI)      │
└─────────────────┬───────────────────────────────┘
                  │
      ┌───────────┴──────────────┐
      │                          │
      ▼                          ▼
┌──────────────┐         ┌──────────────┐
│  Git Repos   │         │  K8s Cluster │
│  (Gitea)     │         │  (Helios)    │
└──────────────┘         └──────┬───────┘
                                │
                  ┌─────────────┼─────────────┐
                  ▼             ▼             ▼
            ┌──────────┐  ┌──────────┐  ┌──────────┐
            │ Operator │  │ Tekton   │  │ ArgoCD   │
            │(DB Prov) │  │(CI/CD)   │  │(GitOps)  │
            └─────┬────┘  └──────────┘  └──────────┘
                  │
      ┌───────────┴────────────┬──────────────┐
      ▼                        ▼              ▼
┌──────────────┐        ┌────────────┐  ┌──────────┐
│ PostgreSQL   │        │ PostgREST  │  │ Service  │
│ Database     │   ────▶│ Container  │  │& Ingress │
└──────────────┘        └────────────┘  └──────────┘
      │                        ▲
      │                        │
      └────── REST API ────────┘
```

## Troubleshooting

### API Not Responding
```bash
# Check PostgREST pod
kubectl get pods -l app=my-api

# Check logs
kubectl logs deployment/my-api

# Verify database connection
kubectl exec deployment/my-api -- \
  curl -I http://localhost:3000/
```

### Authentication Issues
```bash
# Verify JWT secret is set correctly
 kubectl get secret my-api-db-secret -o yaml | grep JWT

# Test with curl
curl -H "Authorization: Bearer INVALID_TOKEN" http://api/posts
# Should return 401 if token validation enabled
```

### Database Schema Issues
```bash
# Connect to database
kubectl exec postgres-pod -- \
  psql -U user -d database \
  -c "SELECT * FROM information_schema.tables;"

# Check if schema exists
psql -c "\\dn"  # List schemas
```

## Best Practices

### 1. Use Schemas for Organization
```sql
CREATE SCHEMA public;    -- User-facing API
CREATE SCHEMA internal;  -- Internal tools only
CREATE SCHEMA audit;     -- Audit logs
```

### 2. Enable Row-Level Security (RLS)
```sql
ALTER TABLE posts ENABLE ROW LEVEL SECURITY;

-- Only show posts user owns
CREATE POLICY user_posts ON posts 
  USING (author_id = current_user_id());
```

### 3. Use Computed Columns for Complex Logic
```sql
ALTER TABLE posts
  ADD COLUMN author_name TEXT 
  GENERATED ALWAYS AS (
    SELECT name FROM users WHERE id = author_id
  ) STORED;
```

### 4. Version Your API
```sql
CREATE SCHEMA api_v1;
-- Maintain backward compatibility by not renaming tables
```

### 5. Monitor Performance
```sql
-- Add indexes for filtered/sorted columns
CREATE INDEX idx_posts_user_id ON posts(user_id);
CREATE INDEX idx_posts_published_at ON posts(published_at);
```

## Documentation & Resources

- **PostgREST Official**: https://postgrest.org
- **API Guide**: https://postgrest.org/en/latest/references/api.html
- **RLS Guide**: https://postgrest.org/en/latest/how-tos/basics.html#row-level-security
- **JWT & Auth**: https://postgrest.org/en/latest/how-tos/jwt.html
- **Helios Platform**: See `ARCHITECTURE.md` in project root

## Getting Help

- **Template Issues**: Check `REGISTRATION.md` in postgrest-template directory
- **PostgREST Questions**: Refer to https://postgrest.org/en/latest/
- **Helios Platform**: See project documentation
- **Backstage Scaffolder**: https://backstage.io/docs/features/software-templates/

---

**Ready to build?** Go to Backstage Create page and select PostgREST API Template! 🚀
