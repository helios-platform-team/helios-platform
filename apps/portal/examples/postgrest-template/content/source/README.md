# ${{ values.name }} - PostgREST API

This is a PostgREST instant REST API service for PostgreSQL.

## What is PostgREST?

PostgREST automatically generates a production-ready REST API from any PostgreSQL database schema. You define your data structures in SQL, and PostgREST instantly exposes them via standard HTTP verbs (GET, POST, PUT, DELETE).

## Key Features

- **Auto-Generated CRUD Operations**: Full REST endpoints from your database schema
- **JWT Authentication**: Secure endpoints with JWT tokens
- **Role-Based Access Control**: Database-enforced permissions
- **OpenAPI Documentation**: Auto-generated API documentation
- **Zero Configuration**: Just point it at your database

## Architecture

```
Client Requests
      ↓
PostgREST Container (port ${{ values.port }})
      ↓
PostgreSQL Database (automatically provisioned by Helios Operator)
```

## Configuration

The Helios Operator automatically:

1. **Provisions a PostgreSQL database** with persistent storage
2. **Creates secure credentials** (random username and password)
3. **Injects the connection string** via `PGRST_DB_URI` environment variable
4. **Configures role-based access** for the API schema `${{ values.apiSchema }}`

## Usage

### Define Your Schema

Connect to the database and define your tables:

```sql
CREATE TABLE IF NOT EXISTS posts (
  id SERIAL PRIMARY KEY,
  title TEXT NOT NULL,
  body TEXT,
  author_id INT REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS users (
  id SERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  email TEXT UNIQUE NOT NULL
);
```

### Access the API

Once deployed, you'll have generated endpoints:

- `GET /posts` - List all posts
- `POST /posts` - Create a post
- `GET /posts/{id}` - Read a post
- `PUT /posts/{id}` - Update a post
- `DELETE /posts/{id}` - Delete a post

## Environment Variables

set automatically by the Helios Operator:

- `PGRST_DB_URI` - PostgreSQL connection string
- `PGRST_DB_SCHEMA` - Schema to expose (default: `${{ values.apiSchema }}`)
- `PGRST_DB_ANON_ROLE` - Role for unauthenticated requests (default: `${{ values.anonRole }}`)
- `PGRST_JWT_AUDIENCE` - JWT audience claim (default: `${{ values.jwtRole }}`)

## Documentation

- [PostgREST Official Docs](https://postgrest.org)
- [API Endpoints Guide](https://postgrest.org/en/latest/references/api.html)
- [JWT Authentication](https://postgrest.org/en/latest/how-tos/jwt.html)

## Deployment

This template creates:

1. **HeliosApp CRD** - Defines the PostgREST component and its database requirement
2. **PostgreSQL Database** - Automatically provisioned by the Helios Operator
3. **Container Deployment** - Runs the PostgREST service
4. **Service** - Exposes PostgREST within the cluster
5. **Ingress** - Provides external access to the API
6. **Tekton Pipeline** - CI/CD for building the container
7. **ArgoCD Sync** - GitOps-based deployment

All of this is managed declaratively through Kubernetes manifests.
