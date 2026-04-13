# PostgREST Backstage Template

A complete Backstage software template for scaffolding PostgREST instant REST APIs with automatic database provisioning via the Helios Operator.

## Overview

This template creates:
- **Source Repository**: Dockerfile, SQL schema, PostgREST configuration
- **GitOps Repository**: Kubernetes manifests with HeliosApp CRD, Tekton CI/CD, ArgoCD integration

## Key Features

✅ **Auto-Generated REST API**: PostgREST creates CRUD endpoints from your database schema  
✅ **Automatic Database Provisioning**: Helios Operator provision PostgreSQL on Kubernetes  
✅ **CI/CD Pipeline**: Tekton builds custom Docker image with your schema  
✅ **GitOps Deployment**: ArgoCD syncs from separate GitOps repository  
✅ **JWT Authentication**: Built-in JWT support for securing your API  
✅ **Role-Based Access Control**: Database-level permissions with Postgres roles  

## Architecture

```
Backstage Template
    ↓
Creates Two Repos:
    ├── Source Repo (Dockerfile + schema + config)
    │       ↓
    │   Tekton CI/CD
    │       ↓
    │   Docker Image → docker.io/org/repo:latest
    │
    └── GitOps Repo (Kubernetes manifests)
            ↓
        ArgoCD
            ↓
        Helios Operator
            ├── PostgreSQL Database
            ├── PostgREST Container
            └── Ingress for REST API
```

## Requirements Met ✅

### Requirement 1: Scaffolding Only (No K8s Deployment)
- Template generates manifests but does NOT apply them
- Users manually deploy when ready via `kubectl apply -f gitops/`
- Reduces risk of accidental deployments in the wrong cluster

### Requirement 2: Use Official PostgREST Image
- Dockerfile: `FROM postgrest/postgrest:v12.2.0` (official image)
- Users customize by copying their SQL schema into the image
- Results in: `Dockerfile` → `docker build` → `docker.io/org/repo:latest` custom image
- This custom image still uses official PostgREST as its base, but includes user's schema

### Requirement 3: Parameterize Namespace
- Template includes `namespace` parameter (required in template.yaml)
- Dynamically injected into ALL Kubernetes manifests:
  - `helios-app.yaml`: `metadata.namespace: ${{ parameters.namespace }}`
  - `argocd-app.yaml`: `metadata.namespace: ${{ parameters.namespace }}`
  - `kustomization.yaml`: All resources in same namespace
  - `pipeline.yaml`: `metadata.namespace: ${{ parameters.namespace }}`
  - `triggers.yaml`: `metadata.namespace: ${{ parameters.namespace }}`

### Requirement 4: Support Custom Images with Registry
- Template includes Docker registry parameters: `dockerOrg`, `repoName`
- Dockerfile builds custom image FROM official postgrest/postgrest
- Tekton pipeline (in gitops repo) builds and pushes to registry
- GitOps manifests reference: `docker.io/${{ dockerOrg }}/${{ repoName }}:latest`
- Enables same flexibility as Node.js templates: users build their own images

## Template Parameters

### Component Information (Required)
| Parameter | Description | Example |
|-----------|-------------|---------|
| `name` | Service name | `my-api` |
| `port` | PostgREST listen port | `3000` |
| `dockerOrg` | Docker registry org/user | `mycompany` |
| `repoName` | Docker repository name | `my-api` |

### PostgREST Configuration (Optional)
| Parameter | Description | Default |
|-----------|-------------|---------|
| `apiSchema` | PostgreSQL schema to expose | `public` |
| `jwtSecret` | JWT signing secret (32+ chars) | - |
| `jwtRole` | Default JWT role claim | `authenticated` |
| `anonRole` | Role for unauthenticated requests | `anon` |

### Infrastructure (Optional)
| Parameter | Description |
|-----------|-------------|
| `namespace` | Kubernetes namespace | (required, input per cluster) |
| `databaseConfig` | Database type, name, etc. | (picker UI) |
| `repoUrl` | Git repository URL | (picker UI) |

## What Gets Generated

### Source Repository Structure
```
source/
  Dockerfile                  # Builds custom image from postgrest/postgrest
  postgrestrc.conf           # PostgREST configuration
  schema/                    # User's database schema
    README.md                # Schema documentation
    01-tables.sql            # Example tables
    02-permissions.sql       # Example permissions
  catalog-info.yaml          # Backstage component metadata
  README.md                  # User documentation
```

### GitOps Repository Structure
```
gitops/
  helios-app.yaml           # Main CRD: defines API + database requirement
  argocd-app.yaml           # Points ArgoCD to this repo
  kustomization.yaml        # Kubernetes bundle
  pipeline.yaml             # Tekton PipelineRun for CI/CD
  triggers.yaml             # Webhook EventListener + TriggerBinding
  README.md                 # Deployment documentation
```

## Complete Workflow

1. **Scaffold via Backstage UI**
   - User fills in parameters (name, Docker org, etc.)
   - Template generates source + gitops repositories

2. **Customize Schema** (source repo)
   - User edits `schema/01-tables.sql` with their tables
   - Edits `schema/02-permissions.sql` for roles/access control
   - Commits and pushes to source repo

3. **Automatic CI/CD** (Webhook → Tekton)
   - Webhook triggers Tekton pipeline
   - Builds: `docker build -t docker.io/org/repo:latest .`
   - Pushes: Docker image to registry

4. **Manual GitOps Sync** (User → ArgoCD)
   - User applies: `kubectl apply -f gitops/helios-app.yaml`
   - ArgoCD watches gitops repo for changes
   - ArgoCD deploys HeliosApp to cluster

5. **Automatic Operator Deployment** (ArgoCD → Helios Operator)
   - Helios Operator sees HeliosApp CRD
   - Creates PostgreSQL database (with schema from Docker image)
   - Starts PostgREST container
   - Exposes REST API via Ingress

6. **Use REST API**
   - Users interact with auto-generated endpoints
   - PostgREST serves REST CRUD operations

## Deployment

### Prerequisites
- Kubernetes 1.28+
- Helios Operator 0.2.0+ installed
- ArgoCD 2.8+ installed
- Docker registry access (Docker Hub username)
- Backstage instance with scaffolder plugin

### First-Time Setup

```bash
# After template scaffolding
cd gitops/

# Review manifests
cat helios-app.yaml

# Deploy to cluster
kubectl apply -f helios-app.yaml

# Watch deployment progress
kubectl get heliosapp -w

# List generated endpoints
kubectl get ingress -n your-namespace
```

### Update Deployment

```bash
# From source repo: push schema changes
git add schema/
git commit -m "Add new tables"
git push

# Tekton auto-builds and pushes image

# From gitops repo: review and commit any changes
cd ../gitops/
git add .
git commit -m "Update configuration"
git push

# ArgoCD auto-syncs
```

## Security Considerations

1. **JWT Secrets**: Provide secure `jwtSecret` (32+ characters, random)
2. **Database Credentials**: Not stored in manifests; injected by Operator
3. **Role-Based Access**: Configure `anonRole` and `authenticated` roles in schema
4. **Docker Registry**: Use private registries for proprietary APIs
5. **Ingress TLS**: Ensure HTTPS is configured for production

## Troubleshooting

### API Not Responding
```bash
# Check PostgREST logs
kubectl logs -f deployment/my-api -c api

# Verify database connection
kubectl exec -it deployment/my-api -- \
  sh -c 'curl -v http://localhost:3000/'
```

### Database Not Initialized
```bash
# Check Operator logs
kubectl logs -f deployment/helios-operator

# Check database status
kubectl get database -n your-namespace

# Check secrets injected
kubectl get secret -n your-namespace | grep db
```

### CI/CD Not Triggering
```bash
# Check webhook
kubectl get eventlisteners -n default

# Check Tekton logs
kubectl logs -f tekton-triggers-controller -n tekton-pipelines

# Verify webhook URL in Git repo settings
# Should point to: http://el-{repoName}-listener.default.svc.cluster.local:8080
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

## References

- [PostgREST Documentation](https://postgrest.org)
- [Helios Operator Documentation](../../../../docs/OPERATOR.md)
- [Backstage Scaffolder Docs](https://backstage.io/docs/features/software-templates)
- [ArgoCD Documentation](https://argo-cd.readthedocs.io/)
- [Tekton Pipelines](https://tekton.dev)
