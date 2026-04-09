# ${{ values.name }} - PostgREST API

This is a PostgREST instant REST API service for PostgreSQL.

## What is PostgREST?

PostgREST automatically generates a production-ready REST API from any PostgreSQL database schema. You define your data structures in SQL, and PostgREST instantly exposes them via standard HTTP verbs (GET, POST, PUT, DELETE).

## Quick Start

### 1. Define Your Schema

Add your database tables in `schema/` directory:

```bash
# Example: schema/01-tables.sql
CREATE TABLE posts (
  id SERIAL PRIMARY KEY,
  title TEXT NOT NULL,
  body TEXT,
  created_at TIMESTAMP DEFAULT NOW()
);
```

### 2. Build Your Custom Image

The `Dockerfile` builds your custom image with your schema:

```bash
docker build -t ${{ values.image }}:latest .
docker push ${{ values.image }}:latest
```

The Tekton CI/CD pipeline automates this on every commit.

### 3. Deploy

```bash
kubectl apply -f gitops/helios-app.yaml
```

Helios Operator will:
- Provision PostgreSQL database
- Apply your schema from the image
- Start PostgREST container
- Expose REST API automatically

## Key Features

- **Auto-Generated CRUD Operations**: Full REST endpoints from your database schema
- **JWT Authentication**: Secure endpoints with JWT tokens
- **Role-Based Access Control**: Database-enforced permissions
- **OpenAPI Documentation**: Auto-generated API documentation
- **Custom Schema**: Define your tables in SQL

## Architecture

```
Your Source Repo (Docker Image)
      ↓
  Tekton CI/CD (build & push image)
      ↓
PostgREST Container (your custom image)
      ↓
PostgreSQL Database (automatically provisioned by Helios Operator)
```

## Configuration

The Helios Operator automatically:

- Creates a PostgreSQL database
- Applies your schema from `schema/` directory
- Injects `PGRST_DB_URI` environment variable
- Exposes PostgREST on port `${{ values.port }}`

## Customizing Your Schema

Edit the SQL files in `schema/` directory:

- **`01-tables.sql`**: Define your database tables (replace example)
- **`02-permissions.sql`**: Set up roles and access control (customize for your needs)
- Add more `.sql` files as needed (e.g., `03-views.sql`, `04-functions.sql`)

See [schema/README.md](schema/README.md) for detailed examples.

## Building and Deploying

### Step 1: Update Your Schema

```bash
# Edit example tables to match your data model
vim schema/01-tables.sql

# Add your permissions and roles
vim schema/02-permissions.sql
```

### Step 2: Configure PostgREST (Optional)

Edit `postgrestrc.conf` to customize:
- API schema to expose
- Authentication method
- CORS settings
- Request limits

### Step 3: Push to Repository

```bash
git add -A
git commit -m "Add my database schema"
git push origin main
```

Webhook automatically triggers Tekton CI/CD to:
1. Build Docker image with your schema
2. Push image to `${{ values.dockerOrg }}/${{ values.repoName }}`
3. Deploy to Kubernetes via ArgoCD

### Step 4: Test Your API

```bash
# Get the API URL (run these in your cluster)
kubectl get ingress -n ${{ values.namespace }}

# List users
curl https://your-api.example.com/users

# Create a user
curl -X POST https://your-api.example.com/users \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@example.com","name":"John Doe"}'
```

## Generated Repositories

This template generates **two repositories**:

### 1. Source Repository (this one)

Contains your application code:
- `Dockerfile` - Builds custom image with your schema
- `schema/` - SQL files defining your database tables
- `postgrestrc.conf` - PostgREST configuration
- CI/CD pipeline metadata

When you push changes, Tekton automatically:
1. Builds a new Docker image
2. Runs tests
3. Pushes to `docker.io/${{ values.dockerOrg }}/${{ values.repoName }}`

### 2. GitOps Repository

Contains Kubernetes manifests:
- `helios-app.yaml` - HeliosApp CRD (main deployment manifest)
- `argocd-app.yaml` - ArgoCD Application for GitOps sync
- `kustomization.yaml` - Kubernetes bundle configuration

When ArgoCD syncs, it:
1. Pulls your custom Docker image from the registry
2. Creates PostgreSQL database (via Helios Operator)
3. Deploys PostgREST container
4. Exposes REST API via Ingress

## Environment Variables

The Helios Operator automatically sets these for PostgREST:

- `PGRST_DB_URI` - PostgreSQL connection string (injected as secret)
- `PGRST_DB_SCHEMA` - Schema to expose as REST API (default: `${{ values.apiSchema }}`)
- `PGRST_DB_ANON_ROLE` - Role for unauthenticated requests (default: `${{ values.anonRole }}`)
- `PGRST_JWT_SECRET` - Secret key for JWT verification (from `${{ values.jwtSecret }}`)

## Complete Workflow

```
1. You edit schema/01-tables.sql
                ↓
2. Git push to source repository
                ↓
3. Webhook triggers Tekton CI/CD
                ↓
4. Docker image built with your schema
                ↓
5. Image pushed to docker.io/${{ values.dockerOrg }}/${{ values.repoName }}:latest
                ↓
6. GitOps repository's helios-app.yaml is updated
                ↓
7. ArgoCD detects change and syncs
                ↓
8. Helios Operator creates PostgreSQL + applies schema
                ↓
9. PostgREST container starts with your custom image
                ↓
10. Your REST API is live at https://your-api.example.com/
```

## Next Steps

1. **Define Your Data Model**: Edit `schema/01-tables.sql` with your tables
2. **Set Permissions**: Configure `schema/02-permissions.sql` for your roles
3. **Configure PostgREST**: Customize `postgrestrc.conf` if needed
4. **Commit & Push**: Changes automatically trigger deployment
5. **Test Your API**: Use the REST endpoints exposed by PostgREST

## Troubleshooting

### API not responding?
```bash
# Check PostgREST logs
kubectl logs -f deployment/${{ values.name }} -c api

# Check database connection
kubectl exec -it deployment/${{ values.name }} -- psql "$PGRST_DB_URI" -c "SELECT 1"
```

### Database not initialized?
```bash
# Check Operator logs
kubectl logs -f deployment/helios-operator

# Check database status
kubectl get database -n ${{ values.namespace }}
```

### Changes not deployed?
```bash
# Check ArgoCD sync status
argocd app get ${{ values.name }}-gitops

# Manual sync
argocd app sync ${{ values.name }}-gitops
```

## Documentation

- [PostgREST Official Docs](https://postgrest.org)
- [Helios Operator Guide](../../../../../../docs/OPERATOR.md)
- [Sample Schema](schema/README.md)
