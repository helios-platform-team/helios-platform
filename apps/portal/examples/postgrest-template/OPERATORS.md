# PostgREST Template Integration Guide

## For Platform Operators & DevOps Engineers

This guide covers deploying, configuring, and managing the PostgREST Backstage template in production environments.

## Prerequisites

- Helios Platform v0.2.0 or later
- Backstage Portal v1.31 or later
- Helios Operator running (manages database provisioning)
- Tekton Pipelines v0.50+
- ArgoCD v2.8+
- PostgreSQL 14+ (auto-provisioned by operator)
- Kubernetes 1.28+

## Installation

### 1. Verify Template Files

```bash
# Template directory structure
ls -la apps/portal/examples/postgrest-template/

# Should contain:
# ├── template.yaml              # Backstage scaffolder configuration
# ├── REGISTRATION.md            # Registration instructions
# ├── QUICKSTART.md              # User guide
# ├── validate.sh                # Validation script
# └── content/
#     ├── source/                # Source code templates (5 files)
#     └── gitops/                # Kubernetes manifests (5 files)
```

### 2. Register in Backstage

The template is already registered in `apps/portal/app-config.yaml`:

```yaml
catalog:
  locations:
    - type: file
      target: ../../examples/postgrest-template/template.yaml
      rules:
        - allow: [Template]
```

### 3. Validate Template

```bash
cd apps/portal/examples/postgrest-template

# Run validation
bash validate.sh

# Output should show:
# ✓ Template structure is valid
# ✓ YAML syntax is correct
# ✓ HeliosApp CRD correctly configured
# ✓ Tekton CI/CD pipeline defined
# ✓ PostgREST configuration present
# ✓ PGRST_DB_URI integration configured
```

### 4. Deploy Backstage

```bash
cd apps/portal

# Install dependencies
npm install

# Run in development
npm run start

# Or build for production
npm run build
docker build -t backstage:latest .
```

### 5. Verify Template Registration

Once Backstage starts:

1. Navigate to http://localhost:3000
2. Click **Create** in top menu
3. Search for "PostgREST"
4. Verify template appears in list

## Configuration Reference

### Template Configuration (app-config.yaml)

```yaml
scaffolder:
  # Set working directory for template operations
  workingDirectory: /tmp/scaffolder

  # Configure default environment variables
  defaultEnvironment:
    secrets:
      GITEA_TOKEN: ${GITEA_TOKEN}      # Git credentials
      ARGOCD_AUTH_TOKEN: ${ARGOCD_TOKEN}  # ArgoCD auth (if needed)

catalog:
  locations:
    # PostgREST Template Location
    - type: file
      target: ../../examples/postgrest-template/template.yaml
      rules:
        - allow: [Template]
```

### Template Parameters

**Component Information**
- `name`: Human-readable service name
- `port`: API listen port (1024-65535)
- `dockerOrg`: Docker registry organization
- `repoName`: Git repository name (URL-safe)

**PostgREST Configuration**
- `apiSchema`: PostgreSQL schema to expose (default: `public`)
- `jwtSecret`: JWT signing secret (min 32 characters)
- `jwtRole`: JWT audience claim (default: `authenticated`)
- `anonRole`: Role for anonymous requests (default: `anon`)

**Database Configuration**  
- `databaseConfig.dbType`: Always `postgres` for PostgREST
- `databaseConfig.dbName`: PostgreSQL database name
- `databaseConfig.port`: PostgreSQL port (default: 5432)

### Environment Variable Injection

The Helios Operator automatically injects these into PostgREST pods:

```
PGRST_DB_URI=postgres://user:pass@host:5432/dbname
PGRST_DB_SCHEMA=public
PGRST_DB_ANON_ROLE=anon
PGRST_JWT_AUDIENCE=authenticated
PGRST_MAX_ROWS=1000
PGRST_LOG_LEVEL=notice
```

**Important:** The `PGRST_DB_URI` format is:
```
postgres://[user]:[password]@[host]:[port]/[database]
```

Credentials containing special characters (`:`, `@`, `/`, `%`) are URL-encoded by the operator.

## Operational Tasks

### Monitor PostgREST Deployments

```bash
# List all PostgREST services
kubectl get deployment -l app.kubernetes.io/name=postgrest -A

# Check specific service
kubectl describe deployment my-api -n default

# View pod logs
kubectl logs deployment/my-api -n default -f

# Check operator provisioning
kubectl logs deployment/helios-operator -n helios -f | grep PGRST
```

### Verify Database Provisioning

```bash
# Check StatefulSet for PostgreSQL
kubectl get statefulset -l helios.io/trait=database -n default

# Inspect database credentials secret
kubectl get secret my-api-db-secret -n default -o yaml

# Verify PGRST_DB_URI was created
kubectl get secret my-api-db-secret -n default \
  -o jsonpath='{.data.PGRST_DB_URI}' | base64 -d | echo

# Connect to database
kubectl exec -it statefulset/my-api-db -n default -- \
  psql -U $USER -d my-api-db -c "SELECT version();"
```

### Troubleshoot Template Scaffolding

**Issue: Template not appearing in Backstage UI**

```bash
# 1. Check configuration syntax
cat apps/portal/app-config.yaml | python3 -c "import sys, yaml; yaml.safe_load(sys.stdin)" && echo "✓ Valid YAML"

# 2. Verify template file exists
ls -la apps/portal/examples/postgrest-template/template.yaml

# 3. Check Backstage backend logs
kubectl logs deployment/backstage-backend -f | grep -i "postgrest\|catalog"

# 4. Restart Backstage
kubectl delete pod -l app=backstage-backend
```

**Issue: Scaffolding fails during execution**

```bash
# 1. Check Backstage backend logs for action failures
kubectl logs deployment/backstage-backend -f

# 2. Verify Gitea credentials
echo "GITEA_TOKEN='${GITEA_TOKEN}'" | grep -v "null"

# 3. Check Git repository creation
curl -H "Authorization: token ${GITEA_TOKEN}" \
  http://localhost:3030/api/v1/user/repos

# 4. Verify HeliosApp CRD registration
kubectl get crd heliosapps.helios.io

# 5. Check operator reconciliation
kubectl logs deployment/helios-operator -n helios -f | grep "HeliosApp"
```

**Issue: Database not provisioning**

```bash
# 1. Verify HeliosApp is created
kubectl get heliosapp -n default
kubectl describe heliosapp my-api -n default

# 2. Check operator logs for database trait
kubectl logs deployment/helios-operator -n helios -f | grep "database\|trait"

# 3. Verify database provisioning resources
kubectl get statefulset,secret -l app=my-api-db -n default

# 4. Check operator RBAC permissions
kubectl get role,rolebinding -n default | grep operator
```

**Issue: PGRST_DB_URI not injected**

```bash
# 1. Check if secret contains PGRST_DB_URI
kubectl get secret my-api-db-secret -n default -o yaml | grep PGRST_DB_URI

# 2. Verify secret key is properly set
kubectl get secret my-api-db-secret -n default \
  -o jsonpath='{.data}' | jq 'keys'

# 3. Check PostgREST pod environment
kubectl exec deployment/my-api -n default -- env | grep PGRST

# 4. Test connection string format
PGRST_DB_URI=$(kubectl get secret my-api-db-secret -n default \
  -o jsonpath='{.data.PGRST_DB_URI}' | base64 -d)
echo "Connection test: $PGRST_DB_URI"
```

### Upgrade Considerations

**Template Updates**
1. Update template files in `apps/portal/examples/postgrest-template/`
2. No config changes needed (already registered)
3. Changes apply to new scaffolding requests
4. Existing deployed services unaffected

**PostgREST Image Updates**
1. Edit `content/source/Dockerfile`
   ```dockerfile
   # Old
   FROM postgrest/postgrest:v12.1.0
   
   # New
   FROM postgrest/postgrest:v12.2.0
   ```
2. Update Tekton pipeline to push new image on commits
3. Existing deployments update via ArgoCD GitOps sync

**Operator Updates**
1. Redeploy Helios Operator to cluster
2. Operator continues working with existing HeliosApps
3. New database trait features available for new services

## Performance Tuning

### PostgREST Configuration

Edit `content/source/postgrestrc.conf`:

```conf
# Connection pooling
db-pool = 10           # Increase for high concurrency
db-pool-timeout = 10   # Seconds to wait for available connection

# Request limits
max-rows = 1000        # Reduce for large tables
server-port = 3000     # Ensure port not in use

# Logging
log-level = notice     # Use 'info' for debugging, 'warn' for production
```

### Database Optimization

Add indexes for common queries:

```sql
-- Assumed public schema
CREATE INDEX idx_posts_user_id ON public.posts(user_id);
CREATE INDEX idx_posts_created_at ON public.posts(created_at DESC);
CREATE INDEX idx_posts_search ON public.posts USING gin(
  to_tsvector('english', title || ' ' || body)
);
```

### Kubernetes Resource Limits

Edit `content/gitops/helios-app.yaml`:

```yaml
components:
  - name: api
    container:
      port: 3000
      resources:
        requests:
          cpu: 100m
          memory: 256Mi
        limits:
          cpu: 500m
          memory: 512Mi
```

## Security Best Practices

### 1. JWT Token Validation

```sql
-- Enable JWT secret in template
PGRST_JWT_SECRET="your-secret-key"
```

Clients must provide valid token:
```bash
curl -H "Authorization: Bearer $JWT_TOKEN" http://api/posts
```

### 2. Row-Level Security (RLS)

Always use RLS for multi-tenant services:

```sql
CREATE TABLE users (
  id SERIAL PRIMARY KEY,
  name TEXT,
  tenant_id INT
);

ALTER TABLE users ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON users
  USING (tenant_id = current_setting('app.current_tenant_id')::INT);
```

### 3. Schema Isolation

Use PostgreSQL schemas to control API exposure:

```sql
-- Exposed API
CREATE SCHEMA api;

-- Internal only
CREATE SCHEMA internal;

-- Audit logs
CREATE SCHEMA audit;
```

### 4. Network Policies

Restrict PostgREST network access:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: postgrest-lockdown
spec:
  podSelector:
    matchLabels:
      template: postgrest-api
  policyTypes:
    - Ingress
  ingress:
    - from:
        - podSelector:
            matchLabels:
              role: client
```

### 5. Secrets Management

Never commit secrets:

```bash
# ✓ Good: Use environment variables
export GITEA_TOKEN=$(kubectl get secret gitea-creds -o jsonpath='{.data.token}')

# ✗ Bad: Hardcode credentials
GITEA_TOKEN="ghp_xxxxxxxxxxxxxxxxxxxx"
```

## Maintenance

### Regular Tasks

**Daily**
- Monitor PostgREST pod health
- Check database backups completed

**Weekly**
- Review operator and PostgREST logs
- Verify argoCD sync status

**Monthly**
- Update dependencies (PostgREST, operator)
- Review API performance metrics
- Test disaster recovery procedures

### Backup Strategy

```bash
# PostgreSQL backup (from pod)
kubectl exec statefulset/my-api-db -n default -- \
  pg_dump -U user my-api-db > backup.sql

# GitOps repository backup (handled by Git provider)
# Manifests stored in Gitea/GitHub

# Kubernetes cluster backup (using Velero recommended)
velero backup create postgrest-backup \
  --include-namespaces default
```

## Support & Documentation

- **PostgREST Docs**: https://postgrest.org
- **Backstage Docs**: https://backstage.io/docs/features/software-templates/
- **Helios Platform**: See project README
- **Tekton Pipelines**: https://tekton.dev/docs/
- **ArgoCD**: https://argo-cd.readthedocs.io/

## Change Log

### v1.0.0 (April 2026)
- Initial PostgREST template release
- PGRST_DB_URI environment variable injection
- Full Tekton CI/CD integration
- ArgoCD GitOps sync
- Backstage catalog integration

---

**Last Updated:** April 6, 2026  
**Status:** Production Ready ✅
