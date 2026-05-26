# PostgREST Backstage Template

A complete Backstage software template for scaffolding PostgREST instant REST APIs with automatic database provisioning, zero-downtime migrations, and full CI/CD via the Helios Platform.

## Overview

This template creates:
- **Source Repository**: Dockerfile, SQL schema, PostgREST configuration, migration files
- **GitOps Repository**: Kubernetes manifests with HeliosApp CRD, Tekton CI/CD, ArgoCD integration
- **Automatic Webhook Integration**: Triggers CI/CD pipelines on code changes

## Key Features

✅ **Auto-Generated REST API**: PostgREST creates CRUD endpoints directly from your database schema  
✅ **Automatic Database Provisioning**: Helios Operator manages PostgreSQL databases on Kubernetes  
✅ **Zero-Downtime Migrations**: Push migration files to `db/migration/` → automatic execution without redeployment  
✅ **CI/CD Pipeline**: Tekton builds custom Docker image with your application code  
✅ **GitOps Deployment**: ArgoCD syncs manifests from separate GitOps repository  
✅ **JWT Authentication**: Built-in JWT token support for API security  
✅ **Role-Based Access Control**: Database-level permissions with Postgres roles  
✅ **Automated Webhooks**: Git webhooks trigger pipelines on every push  

## Architecture

```
Backstage Template
    ↓
Creates Two Repos + Webhook Integration:
    ├── Source Repo (Dockerfile + schema + migrations)
    │       ↓ (on every push)
    │   Tekton CI/CD Pipeline
    │       ├─→ Build: Docker image with your code
    │       ├─→ Test: Run any tests
    │       └─→ Push: docker.io/org/repo:latest
    │
    ├── GitOps Repo (Kubernetes manifests)
    │       ↓ (on every push)
    │   ArgoCD Sync
    │       └─→ Update deployments
    │
    └── Database Migrations (automatic trigger)
            ↓ (when db/migration/*.sql files change)
        Tekton db-migrate Pipeline
            ├─→ Clone repo at commit
            ├─→ Run: golang-migrate up
            └─→ Reload: PostgREST schema
                (NO downtime, NO redeployment)
```

## Database Migrations (The Game Changer)

Instead of rebuilding and redeploying your container for database changes:

**Old way** (❌ Slow, downtime):
```
Edit schema → Commit → Build Docker image → Push → Redeploy → Downtime
```

**Helios way** (✅ Fast, zero downtime):
```
Create migration file → Push → Automatic execution → Done
```

### How It Works

1. Create migration files in `db/migration/` folder:
   ```bash
   db/migration/
     001_initial.up.sql        # Creates tables
     001_initial.down.sql      # Rollback script
     002_add_indexes.up.sql    # Add indexes (does not recreate data)
     002_add_indexes.down.sql  # Rollback
   ```

2. Push to source repository:
   ```bash
   git add db/migration/
   git commit -m "Add user profiles table"
   git push origin main
   ```

3. **Automatic Pipeline Trigger** 🚀
   - Helios Operator creates a webhook on your repo
   - CEL filter triggers ONLY on migration path changes
   - db-migrate pipeline:
     - Clones repository
     - Runs `golang-migrate up`
     - Reloads PostgREST schema cache via `NOTIFY` command
   - **API continues serving requests** ✨

4. Done! Your database is updated, PostgREST knows about it.

### Benefits

- **Zero downtime**: Migrations run in background, API never stops
- **Database-first development**: Evolve schema independently from code
- **Fast iterations**: Deploy schema changes in seconds, not minutes
- **Safe rollbacks**: Each migration has `down.sql` for instant rollback
- **Audit trail**: All migrations tracked in Git with timestamps

## Design Philosophy

This template follows three core principles:

### 1. **GitOps-First**
- All infrastructure is version-controlled
- Changes are Pull Request → Review → Merge → Auto-deploy
- Rollback is just `git revert`

### 2. **Zero-Downtime Operations**
- Database migrations don't require redeployment
- Code updates use rolling deployments
- API continues serving while schema evolves

### 3. **Developer Experience**
- One command: `git push` → Everything happens automatically
- No manual kubectl commands needed after initial setup
- Clear error messages on failures
- Audit trail of all changes

## Template Parameters

When scaffolding via Backstage, you'll provide:

| Parameter | Required? | Description | Example |
|-----------|-----------|-------------|---------|
| **Component Name** | ✅ | Service display name | `My Awesome API` |
| **Repository Name** | ✅ | Git repo name | `my-awesome-api` |
| **Docker Org** | ✅ | Docker Hub username/org | `mycompany` |
| **API Port** | ✅ | PostgREST listen port | `3000` |
| **Namespace** | ✅ | Kubernetes namespace | `production` |
| **Database Name** | ✅ | PostgreSQL database | `my-awesome-api-db` |
| **JWT Secret** | ⚠️ | For token signing (32+ chars) | (random) |
| **JWT Role** | Optional | Default role for tokens | `authenticated` |
| **Anon Role** | Optional | Unauthenticated role | `anon` |
| **API Schema** | Optional | PostgreSQL schema to expose | `public` |

**Security tip:** Generate a random JWT secret: `openssl rand -base64 32`

## What Gets Generated

### Source Repository

Your application code + schema:

```
source-repo/
  ├── Dockerfile                    # Builds custom image FROM postgrest/postgrest
  ├── postgrestrc.conf              # PostgREST configuration (port, logging, etc)
  ├── schema/
  │   ├── 01-tables.sql             # Table definitions (edit this!)
  │   ├── 02-permissions.sql        # Role permissions (edit this!)
  │   └── README.md                 # Schema documentation
  ├── db/
  │   └── migration/                # Database migrations (see below)
  │       ├── 001_initial.up.sql
  │       └── 001_initial.down.sql
  ├── catalog-info.yaml             # Backstage component metadata
  ├── README.md                      # User documentation
  └── .git/                          # Git repository
```

**What you edit:**
- `schema/01-tables.sql` - Your database design
- `schema/02-permissions.sql` - Access control rules
- `db/migration/*.sql` - Zero-downtime schema updates
- `postgrestrc.conf` - PostgREST behavior
- `Dockerfile` - Build configuration (rarely needed)

**How it works:**
```
Push to main branch
    ↓ (webhook)
Tekton builds Docker image
    ↓
Image includes: postgrest + your schema + your migrations
    ↓
Push to docker.io/<org>/<repo>:latest
    ↓
ArgoCD detects new image
    ↓
Kubernetes restarts PostgREST pod (rolling deployment)
    ↓
Old requests drain, new requests use new schema
```

### GitOps Repository

Infrastructure declarations:

```
gitops-repo/
  ├── helios-app.yaml               # Main application definition
  │                                 # (database + postgrest + webhook settings)
  ├── namespace.yaml                # Kubernetes namespace
  ├── kustomization.yaml            # Bundle all manifests
  ├── tekton/
  │   ├── eventlistener.yaml        # Webhook configuration
  │   ├── triggerbinding.yaml       # Extract params from webhook
  │   └── triggertemplate.yaml      # Define what to run
  └── README.md                      # Deployment documentation
```

**How webhooks work:**

1. Source repo has webhook pointing to: `http://el-<name>-listener.<namespace>.svc.cluster.local:8080`
2. Every git push sends a JSON payload to that endpoint
3. EventListener validates signature + filters by branch/path
4. Matching commits trigger a PipelineRun
5. Pipeline executes in Kubernetes (build, test, push)

**db-migrate webhook is separate:**

1. db-migrate EventListener listens for changes to `db/migration/` folder only
2. Different trigger than the main CI/CD webhook
3. Only executes golang-migrate, doesn't rebuild image
4. Runs in <10 seconds (no container rebuild)

## Complete Workflow

### Initial Setup (One-Time)

1. **Scaffold via Backstage UI**
   ```
   Backstage → Create Component → PostgREST API Template
   Fill in: name, registry, namespace, database settings
   Template creates: source + gitops repos with webhooks
   ```

2. **Deploy to Kubernetes** (Manual, one-time)
   ```bash
   cd gitops/
   kubectl apply -f helios-app.yaml
   # Helios Operator creates PostgreSQL database + PostgREST container
   ```

3. **Get your API endpoint**
   ```bash
   kubectl get ingress -n <namespace>
   # Your REST API is now live!
   ```

### Day-to-Day Operations

#### Add Features (Code Changes)
```bash
# 1. Edit your code in source repo
nano postgrestrc.conf
vim schema/01-tables.sql

# 2. Push changes
git add .
git commit -m "Add profile schema"
git push origin main

# 3. Automatic CI/CD
# Webhook triggers Tekton:
#   ✓ Builds Docker image
#   ✓ Runs tests
#   ✓ Pushes to registry
#
# GitOps repo auto-updates with new image version
# ArgoCD syncs → PostgREST pod restarts with new code
```

#### Migrate Database (Zero Downtime)
```bash
# 1. Create migration files
mkdir -p db/migration
cat > db/migration/003_add_users.up.sql << 'EOF'
CREATE TABLE users (
  id SERIAL PRIMARY KEY,
  email TEXT UNIQUE NOT NULL,
  created_at TIMESTAMP DEFAULT NOW()
);
EOF

cat > db/migration/003_add_users.down.sql << 'EOF'
DROP TABLE IF EXISTS users;
EOF

# 2. Push migration files
git add db/migration/
git commit -m "Add users table"
git push origin main

# 3. Automatic db-migrate pipeline
# (You don't do anything else!)
# 
# Webhook triggers db-migrate pipeline:
#   ✓ Clones repository
#   ✓ Runs golang-migrate
#   ✓ Updates schema cache
#   ✓ API continues working
#
# kubectl logs -f deployment/el-{name}-db-migrate-listener
# to watch the webhook trigger

# Done! Your new endpoint is live:
# GET /users
# POST /users (insert)
# PATCH /users (update)
# DELETE /users (delete)
```

#### Update GitOps Configuration
```bash
# 1. Edit manifests in gitops repo
nano helios-app.yaml  # Change replicas, ports, etc

# 2. Push changes
git add .
git commit -m "Scale to 3 replicas"
git push origin main

# 3. Automatic sync
# GitOps webhook triggers
# ArgoCD syncs → Kubernetes updates
```

## Getting Started

### Prerequisites
- Kubernetes 1.28+ with Helios Operator 0.2.0+
- ArgoCD 2.8+ installed
- Backstage instance with scaffolder plugin
- Docker Hub account (for image registry)

### Step 1: Create Your API via Backstage

```
Backstage → "Create Component" button → "PostgREST API" template

Fill in:
  Component Name:     my-awesome-api
  Docker Org:         mycompany  (or Docker Hub username)
  Repository Name:    my-awesome-api
  API Port:           3000
  Namespace:          production  (or your target namespace)
  JWT Secret:         (random string, 32+ chars)
  Database Name:      my-awesome-api-db
```

**What gets created:**
- ✅ Source repository with Dockerfile, schema templates
- ✅ GitOps repository with deployed manifests
- ✅ Git webhooks (automatically registered)

### Step 2: Deploy to Your Cluster

```bash
# Clone the generated gitops repository
git clone https://your-gitea/mycompany/my-awesome-api-gitops.git
cd my-awesome-api-gitops

# Deploy HeliosApp (this is what everything depends on)
kubectl apply -f helios-app.yaml

# Watch rollout
kubectl rollout status deployment/api -n production --timeout=5m

# Get your API endpoint
kubectl get ingress -n production
# Result: my-awesome-api.company.internal or https://api.company.internal
```

### Step 3: Design Your Database Schema

Edit the source repository (`schema/` folder):

```bash
# Clone source repository
git clone https://your-gitea/mycompany/my-awesome-api.git
cd my-awesome-api

# Edit schema files
cat > schema/01-tables.sql << 'EOF'
CREATE TABLE posts (
  id SERIAL PRIMARY KEY,
  title TEXT NOT NULL,
  body TEXT,
  author_id INTEGER REFERENCES users(id),
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE users (
  id SERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  email TEXT UNIQUE NOT NULL
);
EOF

cat > schema/02-permissions.sql << 'EOF'
-- Allow public read-only access
GRANT SELECT ON posts, users TO anon;
GRANT SELECT ON posts, users TO authenticated;

-- Allow authenticated users to create posts
GRANT INSERT ON posts TO authenticated;
EOF

# Commit and push
git add schema/
git commit -m "Add posts and users tables"
git push origin main
```

Your schema is now compiled into the Docker image. **Automatic pipeline triggers:**
- Builds Docker image with schema
- Pushes to docker.io/mycompany/my-awesome-api:latest
- ArgoCD detects the change and redeploys
- **Your API endpoints are live** (e.g., GET /posts, POST /posts)

### Step 4: Evolve Your Schema (Zero-Downtime Migrations)

Instead of rebuilding images, use migrations:

```bash
# Create migration files
mkdir -p db/migration

cat > db/migration/001_initial.up.sql << 'EOF'
CREATE TABLE posts (id SERIAL PRIMARY KEY, title TEXT, author_id INTEGER);
CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT, email TEXT UNIQUE);
EOF

cat > db/migration/001_initial.down.sql << 'EOF'
DROP TABLE IF EXISTS posts, users;
EOF

# Later: add a new feature without redeploying
cat > db/migration/002_add_published_at.up.sql << 'EOF'
ALTER TABLE posts ADD COLUMN published_at TIMESTAMP;
EOF

cat > db/migration/002_add_published_at.down.sql << 'EOF'
ALTER TABLE posts DROP COLUMN published_at;
EOF

# Push your changes
git add db/migration/
git commit -m "Add published_at to posts"
git push origin main

# ✨ MAGIC: db-migrate pipeline automatically:
#   • Clones repo
#   • Runs golang-migrate
#   • Reloads PostgREST
#   • No downtime, no redeployment, no pod restarts
```

Monitor the migration:

```bash
# Check webhook triggered
kubectl get eventlistener -w -n production
kubectl logs deployment/el-my-awesome-api-db-migrate-listener -n production

# Check migration results
kubectl get pipelinerun -n production | grep db-migrate
kubectl describe pipelinerun <name> -n production

# Your new endpoints work immediately
curl https://api.company.internal/posts  # includes published_at
```

## Troubleshooting

### API Not Responding
```bash
# Check PostgREST pod is running
kubectl get pod -n production -l app=api

# Check logs for connection errors
kubectl logs deployment/api -n production

# Verify database is running
kubectl get pod -n production -l app=api-db

# Test DB connection from API pod
kubectl exec -it deployment/api -n production -- \
  sh -c 'curl -s http://localhost:3000/ | head'
```

### Migrations Not Triggering
```bash
# Check db-migrate EventListener is running
kubectl get eventlistener -n production

# Check if webhook is registered in Gitea
kubectl logs deployment/el-my-awesome-api-db-migrate-listener -n production -f --tail=50

# Manually trigger by pushing to db/migration/ folder:
echo "-- test" > db/migration/999_test.up.sql
git add db/migration/999_test.up.sql
git commit -m "Trigger migration"
git push origin main

# Watch the pipeline get created
kubectl get pipelinerun -n production -w
```

### Failed Migrations
```bash
# Check migration run status
kubectl get pipelinerun -n production -l tekton.dev/pipeline=db-migrate

# Get detailed error
kubectl describe pipelinerun <name> -n production

# Check migration logs
kubectl logs -f -l tekton.dev/pipeline=db-migrate -c step-migrate -n production

# Common issues:
# - SQL syntax error: Check *.up.sql file format
# - File naming: Must be NNN_description.up.sql (with leading zeros)
# - Database not found: Check dbName matches in helios-app.yaml
```

### CI/CD Pipeline Not Triggering
```bash
# Check Tekton EventListener for source code changes
kubectl get eventlistener -n production

# Verify webhook URL in Gitea
# Should be: http://el-my-awesome-api-listener.production.svc.cluster.local:8080

# Check Tekton controller logs
kubectl logs -f deployment/tekton-triggers-controller -n tekton-pipelines

# Manual test: push any file to source repo
git add .
git commit -m "Test CI/CD" --allow-empty
git push origin main

# Check if PipelineRun gets created
kubectl get pipelinerun -n production -w
```

## Common Patterns

### JWT Authentication
```bash
# Generate a strong JWT secret
openssl rand -base64 32

# Pass to template when creating component
JWT Secret: <generated-value>

# Clients use JWT token to access protected endpoints:
AUTH_TOKEN=$(your-auth-system-generates-jwt)
curl -H "Authorization: Bearer $AUTH_TOKEN" https://api.company.internal/protected-data
```

### Role-Based Access Control
```sql
-- In schema/02-permissions.sql
CREATE ROLE anon NOLOGIN;
CREATE ROLE authenticated NOLOGIN;

-- Public endpoints (unauthenticated)
GRANT SELECT ON public_posts TO anon;

-- Private endpoints (authenticated users only)
GRANT SELECT ON user_profile TO authenticated;
GRANT INSERT, UPDATE ON user_profile TO authenticated;
```

### Custom Docker Image
The template uses `postgrest/postgrest:latest` as base, allowing you to:
- Add system packages
- Install extensions
- Configure PostgREST before runtime

Edit `Dockerfile` in source repo:
```dockerfile
FROM postgrest/postgrest:v12.2.0

# Add custom packages or configs
RUN apt-get update && apt-get install -y custom-tool

# Copy your schema
COPY schema/ /schema/

# Build pushes to docker.io/<org>/<repo>:latest
```

## Files in This Template

- **template.yaml** - Backstage template definition
- **validate.sh** - Template validation script
- **content/source/** - Source repository template
- **content/gitops/** - GitOps repository template
- **README.md** (this file) - Template documentation

## Next Steps

1. Register this template in Backstage (`catalog-info.yaml`)
2. Users access via Backstage UI → Create Component → PostgREST API Template
3. Follow the scaffolding flow to generate their repositories
4. See generated `README.md` in source repo for customization guide

## Quick Reference

### Common Commands
```bash
# Monitor API deployment
kubectl get pods -n production -l app=api
kubectl logs -f deployment/api -n production

# Watch database migrations
kubectl get pipelinerun -n production -l tekton.dev/pipeline=db-migrate
kubectl logs -f pipelinerun/<name> -n production

# Check webhook integration
kubectl get eventlistener -n production
kubectl logs -f deployment/el-my-awesome-api-listener -n production

# Direct database access
kubectl port-forward -n production svc/api-db 5432:5432
psql -h localhost -U postgres -d my-awesome-api-db
```

### Debugging Checklist
| Issue | Check This |
|-------|-----------|
| API not responding | `kubectl get pod -n production`, then `kubectl logs` |
| Migrations not running | `kubectl get eventlistener -n production`, check git push was to `db/migration/` |
| Database connection error | Verify Secret exists: `kubectl get secret -n production <app>-db-credentials` |
| Migration fails with SQL error | Review `*.up.sql` file format and `NNN_description` naming |
| Webhook not triggered | Check Gitea webhook URL is: `http://el-<name>-listener.<namespace>.svc.cluster.local:8080` |

## References

- [PostgREST Documentation](https://postgrest.org)
- [Helios Operator Documentation](../../../../docs/OPERATOR.md)
- [Backstage Scaffolder Docs](https://backstage.io/docs/features/software-templates)
- [ArgoCD Documentation](https://argo-cd.readthedocs.io/)
- [Tekton Pipelines](https://tekton.dev)
