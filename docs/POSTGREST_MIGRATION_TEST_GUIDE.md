# PostgREST Database Migration Testing Guide

## Overview

This guide provides step-by-step instructions to verify that database migrations run automatically before PostgREST deployment, and that failed migrations block ArgoCD syncs.

**What we're testing:**
1. ✅ Tekton pipeline builds migration Docker image
2. ✅ Migration image is pushed to registry
3. ✅ ArgoCD PreSync Job runs migrations before deploying PostgREST
4. ✅ Failed migrations block ArgoCD sync
5. ✅ PostgREST reflects schema changes immediately after sync

---

## Prerequisites

### Cluster Setup
```bash
# Verify cluster is running
kubectl cluster-info

# Verify required namespaces
kubectl get ns | grep -E "tekton|argocd|default"
```

### Dependencies
- Kubernetes 1.24+
- Tekton Pipelines v0.50+
- ArgoCD v2.8+
- Helios Operator v0.1+
- Backstage Portal with PostgREST template

### Install/Verify Components
```bash
# Check Tekton Pipelines
kubectl get pods -n tekton-pipelines | grep tekton-pipelines-controller

# Check ArgoCD
kubectl get pods -n argocd | grep argocd-

# Check Helios Operator
kubectl get pods -n helios-system | grep operator
```

---

## Test 1: Scaffold a New PostgREST Application

### Step 1.1: Create Application via Backstage

1. Open Backstage Portal: `http://localhost:3000`
2. Navigate to **Create** → **PostgREST API Template**
3. Fill in the form:
   - **Service Name:** `test-api` (will be used for all resources)
   - **API Port:** `3000`
   - **Docker Registry Org:** `<your-docker-org>` (e.g., `mycompany`)
   - **API Schema:** `public`
   - **JWT Secret:** `your-secret-key-min-32-chars-required-here` (min 32 chars)
   - **Repository URL:** Point to your Gitea instance

### Step 1.2: Verify Scaffolding Completed

The template should create:
- **Source repository** with `db/migrations/` directory
- **GitOps repository** with minimal manifests (namespace.yaml, helios-app.yaml)
- **HeliosApp CRD** that triggers operator to:
  -  Auto-generate PreSync Job from CUE definitions
  - Auto-generate Tekton EventListener for migration triggers
  - Auto-generate RBAC permissions for Job execution

```bash
# Verify source repo was created
git clone <source-repo-url>
cd test-api
ls -la db/migrations/

# Verify GitOps repo (now minimal - just base files)
git clone <gitops-repo-url>
cd test-api-gitops
ls -la  # Should show: namespace.yaml, helios-app.yaml
```

---

## Test 2: Verify Operator Auto-Generated Resources

### Step 2.1: Check HeliosApp Status

```bash
# Verify HeliosApp CRD was created by scaffolder
kubectl get heliosapp -n <namespace>
kubectl describe heliosapp test-api -n <namespace>

# Check operator generated annotations
kubectl get heliosapp test-api -o jsonpath='{.metadata.annotations}' | jq .
# Look for: helios.io/has-database-trait=true, helios.io/presync-job=...
```

### Step 2.2: Verify Operator Generated PreSync Resources

```bash
# Check ServiceAccount was auto-created
kubectl get sa -n <namespace> | grep migrator

# Verify ClusterRole for Job management
kubectl get clusterrole | grep presync-job-role

# Verify ClusterRoleBinding
kubectl get clusterrolebinding | grep presync-job-binding
```

### Step 2.3: Check Tekton EventListener (Auto-Generated)

```bash
# Once HeliosApp is created, operator should create EventListener
kubectl get eventlistener -n <namespace> | grep test-api

# Find the external URL
kubectl get svc -n el-services | grep test-api-db-migrate-listener
```

### Step 2.3: Verify Pipeline Trigger Configuration

```bash
# Check Trigger definition
kubectl get triggers -n default
kubectl describe trigger test-api-db-migrate-trigger

# Verify PipelineRun definition
kubectl logs -n tekton-pipelines -l app=tekton-pipelines-controller -f
```

---

## Test 3: Trigger Migration Pipeline

### Step 3.1: Create Initial Migration File

```bash
# In the source repository
cd test-api
mkdir -p db/migrations

# Create first migration
cat > db/migrations/001_init.sql << 'EOF'
-- Create initial schema
CREATE SCHEMA IF NOT EXISTS api;

CREATE TABLE api.users (
  id SERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  email TEXT UNIQUE NOT NULL,
  created_at TIMESTAMP DEFAULT NOW()
);

-- Grant permissions for PostgREST
GRANT USAGE ON SCHEMA api TO anon;
GRANT SELECT ON api.users TO anon;
EOF

git add db/migrations/
git commit -m "Add initial migration"
git push
```

### Step 3.2: Monitor Pipeline Execution

```bash
# Watch for PipelineRun creation
kubectl get pipelinerun -n default -w

# Get latest PipelineRun
PIPELINERUN=$(kubectl get pipelinerun -n default --sort-by=.metadata.creationTimestamp -o jsonpath='{.items[-1].metadata.name}')

# Monitor task execution
kubectl describe pipelinerun $PIPELINERUN -n default

# Watch task logs
kubectl get taskrun -n default -w
kubectl logs -n default -f $(kubectl get taskrun -n default --sort-by=.metadata.creationTimestamp -o jsonpath='{.items[-1].metadata.name}')
```

### Step 3.3: Verify Image Build

```bash
# Check if image was built and pushed
# Option 1: Check Docker registry
docker pull <your-docker-org>/test-api-migrate:latest
docker inspect <your-docker-org>/test-api-migrate:latest

# Option 2: Check Tekton task results
kubectl get taskrun -n default -o yaml | grep -A 5 "image-digest"
```

---

## Test 4: Verify Migration Image Contents

### Step 4.1: Inspect Migration Image

```bash
# Verify golang-migrate binary is present
docker run --rm <your-docker-org>/test-api-migrate:latest which migrate

# Verify migration scripts are copied
docker run --rm <your-docker-org>/test-api-migrate:latest ls -la /migrations/

# Test migration execution (dry run)
docker run --rm \
  -e PGRST_DB_URI="postgres://user:password@localhost:5432/testdb" \
  <your-docker-org>/test-api-migrate:latest \
  migrate -path /migrations -database "$PGRST_DB_URI" version
```

---

## Test 5: Deploy HeliosApp and Check ArgoCD

### Step 5.1: Verify HeliosApp Creation

```bash
# Check if HeliosApp was created in cluster
kubectl get heliosapp -n default
kubectl describe heliosapp test-api -n default

# Verify HeliosApp status
kubectl get heliosapp test-api -o jsonpath='{.status}' | jq .
```

### Step 5.2: Verify ArgoCD Application Created

```bash
# Check ArgoCD Application
kubectl get application -n argocd test-api
kubectl describe application test-api -n argocd

# Get ArgoCD UI URL
kubectl port-forward svc/argocd-server -n argocd 8080:443

# Login and navigate to Applications
# Look for: test-api (should show PreSync hook status)
```

### Step 5.3: Monitor ArgoCD Application Sync

```bash
# Watch application status
kubectl get application -n argocd test-api -w

# Check sync result
kubectl get application test-api -n argocd -o jsonpath='{.status.operationState.phase}'

# View detailed status
kubectl describe application test-api -n argocd
```

---

## Test 6: Verify PreSync Job Execution

### Step 6.1: Check PreSync Job Created

```bash
# List Jobs in default namespace
kubectl get jobs -n default

# Find PreSync Job
PRESYNC_JOB=$(kubectl get jobs -n default -o jsonpath='{.items[*].metadata.name}' | grep presync)
kubectl describe job $PRESYNC_JOB -n default

# Get Job Pod
PRESYNC_POD=$(kubectl get pods -n default -l job-name=$PRESYNC_JOB -o jsonpath='{.items[0].metadata.name}')
```

### Step 6.2: View PreSync Job Logs

```bash
# Get logs from migration Job
kubectl logs $PRESYNC_POD -n default

# Expected output should show:
# - Connecting to database
# - Running migrations
# - "1 migration(s) applied"

# Example successful output:
# flyway Validate
# flyway Repair
# Repairing schema history table in schema [public]...
# Repairing successful.
# flyway Migrate
# Migrating schema [public] to version 1 - init
# Successfully applied 1 migration to schema [public]
```

### Step 6.3: Verify Job Status

```bash
# Check Job completion
kubectl get job $PRESYNC_JOB -n default -o jsonpath='{.status}'

# Expected: { "succeeded": 1 }  (or active: 0, failed: 0, succeeded: 1)

# Verify Job cleanup (TTL after finished)
kubectl get job $PRESYNC_JOB -n default -o jsonpath='{.spec.ttlSecondsAfterFinished}'
# Expected: 3600 (1 hour)
```

---

## Test 7: Verify PostgREST Deployment

### Step 7.1: Check PostgREST Pod

```bash
# Get PostgREST pods
kubectl get pod -n default -l app=test-api

# Describe Pod
kubectl describe pod -n default -l app=test-api

# Verify environment variables
kubectl get pod -n default -l app=test-api -o jsonpath='{.items[0].spec.containers[0].env}' | jq .
```

### Step 7.2: Verify PostgREST is Ready

```bash
# Port-forward to PostgREST
kubectl port-forward svc/test-api 3000:3000 -n default

# Test API endpoint
curl http://localhost:3000/

# Should return either:
# - OpenAPI documentation (if available)
# - API version info
# - Or 404 (if no default route configured)

# Query created table via REST
curl http://localhost:3000/api/users

# Expected response: []  (empty array for new table)
```

### Step 7.3: Verify Database Connection

```bash
# Check PGRST_DB_URI environment variable
kubectl get pod -n default -l app=test-api -o jsonpath='{.items[0].spec.containers[0].env[?(@.name=="PGRST_DB_URI")].value}'

# Exec into pod and verify database connectivity
kubectl exec -it $(kubectl get pod -n default -l app=test-api -o jsonpath='{.items[0].metadata.name}') -n default -- bash

# Inside pod:
psql $PGRST_DB_URI -c "\dt api.*"  # List tables in api schema
psql $PGRST_DB_URI -c "SELECT * FROM api.users;"  # Query the table
```

---

## Test 8: Verify Schema Changes Reflected

### Step 8.1: Add New Migration

```bash
# In source repository
cd test-api

cat > db/migrations/002_add_posts.sql << 'EOF'
CREATE TABLE api.posts (
  id SERIAL PRIMARY KEY,
  user_id INTEGER REFERENCES api.users(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  content TEXT,
  created_at TIMESTAMP DEFAULT NOW()
);

GRANT SELECT ON api.posts TO anon;
EOF

git add db/migrations/
git commit -m "Add posts table"
git push
```

### Step 8.2: Trigger New Pipeline Run

```bash
# Webhook should trigger automatically
# Monitor PipelineRun creation
kubectl get pipelinerun -n default --sort-by=.metadata.creationTimestamp

# Wait for pipeline to complete
kubectl wait --for=condition=Succeeded pipelinerun/<pipelinerun-name> -n default --timeout=5m
```

### Step 8.3: Sync ArgoCD Application

```bash
# Force sync ArgoCD Application
argocd app sync test-api --server $ARGOCD_SERVER

# OR via kubectl
kubectl patch application test-api -n argocd --type merge -p '{"metadata":{"annotations":{"argocd.argoproj.io/refresh":"hard"}}}'

# Monitor sync
kubectl get application test-api -n argocd -w
```

### Step 8.4: Test New API Endpoint

```bash
# Port-forward to PostgREST (if not already)
kubectl port-forward svc/test-api 3000:3000 -n default &

# Test new endpoint
curl http://localhost:3000/api/posts

# Expected: [] (empty array)

# Test with data
curl -X POST http://localhost:3000/api/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice","email":"alice@example.com"}'

curl -X POST http://localhost:3000/api/posts \
  -H "Content-Type: application/json" \
  -d '{"user_id":1,"title":"Hello World","content":"This is my first post"}'

curl http://localhost:3000/api/posts
# Expected: [{"id":1,"user_id":1,"title":"Hello World",...}]
```

---

## Test 9: Test Migration Failure Scenario

### Step 9.1: Create Intentional Migration Error

```bash
# In source repository
cat > db/migrations/003_bad_migration.sql << 'EOF'
-- This will fail: invalid SQL syntax
CREAT TABLE invalid_table (
  id INTEGER
);
EOF

git add db/migrations/
git commit -m "Add intentional bad migration"
git push
```

### Step 9.2: Monitor Pipeline Failure

```bash
# Watch PipelineRun
kubectl get pipelinerun -n default -w

# Get failed PipelineRun
kubectl get pipelinerun -n default --sort-by=.metadata.creationTimestamp -o name | tail -1

# Check failure details
kubectl describe pipelinerun <failed-pipelinerun> -n default

# Check task logs for error
kubectl get taskrun -n default -o jsonpath='{.items[-1].metadata.name}' | xargs -I {} kubectl logs {} -n default
```

### Step 9.3: Verify PreSync Job Fails

```bash
# The Job should fail after backoff limit
kubectl get job -n default -w

# Find PreSync Job
PRESYNC_JOB=$(kubectl get jobs -n default -o jsonpath='{.items[*].metadata.name}' | grep presync | tail -1)

# Check job status
kubectl get job $PRESYNC_JOB -n default -o jsonpath='{.status}'
# Expected: { "failed": 1 } (failed after 3 retries)

# View failure logs
kubectl logs -n default $(kubectl get pods -n default -l job-name=$PRESYNC_JOB -o jsonpath='{.items[0].metadata.name}')
# Expected: SQL syntax error message
```

### Step 9.4: Verify ArgoCD Blocks Sync

```bash
# Check Application status
kubectl describe application test-api -n argocd | grep -A 10 "OperationState"

# Expected status:
# - Phase: Failed
# - Message: PreSync job failed

# Verify PostgREST pods are NOT updated
kubectl describe pod -n default -l app=test-api | grep -A 3 "Image:"
# Should still show old image

# Verify existing tables are still there (no schema corruption)
kubectl exec -it $(kubectl get pod -n default -l app=test-api -o jsonpath='{.items[0].metadata.name}') -n default -- \
  psql $PGRST_DB_URI -c "SELECT * FROM api.posts;"
```

### Step 9.5: Fix and Recover

```bash
# Fix the migration file
cat > db/migrations/003_good_migration.sql << 'EOF'
CREATE TABLE api.comments (
  id SERIAL PRIMARY KEY,
  post_id INTEGER REFERENCES api.posts(id) ON DELETE CASCADE,
  content TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT NOW()
);

GRANT SELECT ON api.comments TO anon;
EOF

# Remove bad migration
git rm db/migrations/003_bad_migration.sql

git commit -m "Fix migration"
git push
```

```bash
# Monitor new PipelineRun (should succeed)
kubectl get pipelinerun -n default -w

# Wait for migration to complete
PRESYNC_JOB=$(kubectl get jobs -n default -o jsonpath='{.items[*].metadata.name}' | grep presync | tail -1)
kubectl wait --for=condition=complete job/$PRESYNC_JOB -n default --timeout=2m

# Check ArgoCD sync now succeeds
kubectl get application test-api -n argocd -o jsonpath='{.status.sync.status}'
# Expected: In Sync or Synced
```

---

## Test 10: Kubernetes Resource Verification Checklist

### Resources Created

```bash
# Namespace
kubectl get ns | grep default

# HeliosApp CRD
kubectl get heliosapp -n default

# ArgoCD Application
kubectl get application -n argocd

# Tekton PipelineRun
kubectl get pipelinerun -n default

# Tekton PipelineRun resources
kubectl get triggerbinding -n default
kubectl get trigger -n default
kubectl get eventlistener -n default

# Kubernetes Jobs (PreSync)
kubectl get jobs -n default

# Kubernetes Deployment (PostgREST)
kubectl get deployment -n default

# Kubernetes Service (PostgREST)
kubectl get svc -n default

# Kubernetes Secret (Database credentials, Webhook)
kubectl get secret -n default | grep -E "test-api|webhook"

# ServiceAccount (for Jobs)
kubectl get sa -n default | grep migrator
```

### Verify RBAC Permissions

```bash
# Check Operator ClusterRole has required permissions
kubectl get clusterrole manager-role -o yaml | grep -A 20 "batch"

# Expected:
# - apiGroups:
#   - batch
#   resources:
#   - jobs
#   verbs:
#   - create
#   - delete
#   - get
#   - list
#   - patch
#   - watch

# Verify ServiceAccount bindings
kubectl get clusterrolebinding | grep manager-role
```

---

## Troubleshooting Guide

### Issue: Webhook Not Triggering Pipeline

**Symptoms:** Push to repository but no PipelineRun created

**Diagnosis:**
```bash
# Check EventListener is accessible
kubectl get svc -n el-services
curl http://el-test-api-db-migrate-listener.el-services.svc.cluster.local:8080

# Check EventListener logs
kubectl logs -n tekton-pipelines -l app=tekton-events-controller -f

# Verify webhook secret matches
kubectl get secret test-api-webhook-secret -n default -o jsonpath='{.data.secret}' | base64 -d

# Verify CEL filter in trigger
kubectl get trigger test-api-db-migrate-trigger -n default -o yaml | grep -A 5 "filter:"
```

**Solution:**
- Verify webhook URL is correct and accessible
- Ensure webhook secret matches in repository settings
- Check CEL filter matches your migration file paths (should be `db/migrations/`)

---

### Issue: PreSync Job Stuck in Pending

**Symptoms:** Job created but Pod never starts

**Diagnosis:**
```bash
# Check Job status
kubectl describe job <presync-job-name> -n default

# Check Pod events
kubectl describe pod <presync-pod-name> -n default

# Check ServiceAccount permissions
kubectl auth can-i create jobs --as=system:serviceaccount:default:<service-account-name>

# Check resource availability
kubectl describe nodes | grep -A 5 "Allocated resources"
```

**Solution:**
- Verify ServiceAccount has required permissions (RBAC)
- Check cluster has sufficient resources (CPU, memory)
- Verify migration image is accessible from cluster

---

### Issue: Migration Runs But Schema Not Visible in PostgREST

**Symptoms:** Migration succeeds, but new tables not accessible via REST API

**Diagnosis:**
```bash
# Verify migration actually ran in database
kubectl exec -it $(kubectl get pod -n default -l app=test-api -o jsonpath='{.items[0].metadata.name}') -n default -- \
  psql $PGRST_DB_URI -c "\dt"

# Check PostgREST logs
kubectl logs -n default -l app=test-api | grep -i schema

# Verify PGRST_DB_SCHEMA matches migration schema
kubectl get pod -n default -l app=test-api -o jsonpath='{.items[0].spec.containers[0].env[?(@.name=="PGRST_DB_SCHEMA")].value}'

# Check permissions in database
psql $PGRST_DB_URI -c "\z"  # List all permissions
```

**Solution:**
- Grant proper permissions in migration: `GRANT SELECT ON schema.table TO anon;`
- Restart PostgREST pod to reload schema: `kubectl rollout restart deployment/test-api -n default`
- Verify `PGRST_DB_SCHEMA` environment variable matches schema name

---

### Issue: ArgoCD Application Stuck in Syncing

**Symptoms:** Application shows "Syncing" status indefinitely

**Diagnosis:**
```bash
# Check Application status
kubectl get application test-api -n argocd -o yaml | grep -A 10 "operationState"

# Check ArgoCD server logs
kubectl logs -n argocd -l app.kubernetes.io/name=argocd-server -f

# Check PreSync Job status
PRESYNC_JOB=$(kubectl get jobs -n default -o jsonpath='{.items[*].metadata.name}' | grep presync | tail -1)
kubectl get job $PRESYNC_JOB -n default -o yaml

# Check if Job Pod is stuck
kubectl get pod -n default -l job-name=$PRESYNC_JOB
```

**Solution:**
- Delete stuck job: `kubectl delete job <presync-job-name> -n default`
- Sync application again: `argocd app sync test-api`
- Check application manifests have valid syntax: `argocd app get test-api`

---

### Issue: Image 404 When Running Migration Job

**Symptoms:** Migration Job fails with "ImagePullBackOff"

**Diagnosis:**
```bash
# Check image exists in registry
docker pull <your-docker-org>/test-api-migrate:latest

# Verify image is accessible from cluster
kubectl run -it --rm debug --image=<your-docker-org>/test-api-migrate:latest --restart=Never -- bash

# Check ImagePullSecret if using private registry
kubectl get secret -n default | grep docker
```

**Solution:**
- Verify pipeline successfully built and pushed image
- Check Docker registry credentials in cluster
- Verify image name in presync-job.yaml matches registry

---

## Success Criteria

✅ **All tests pass if:**

1. ✅ Tekton pipeline triggered on migration file changes
2. ✅ Pipeline builds and pushes migration image successfully
3. ✅ ArgoCD PreSync Job runs before PostgREST deployment
4. ✅ PostgREST pods deploy after successful migration
5. ✅ Schema changes immediately visible via PostgREST API
6. ✅ Failed migration Job blocks ArgoCD sync
7. ✅ ArgoCD Application status shows correct sync phase
8. ✅ PostgREST pods NOT updated if migration fails
9. ✅ Migration can be fixed and deployment retried
10. ✅ All Kubernetes resources properly created and configured

---

## Quick Health Check Script

```bash
#!/bin/bash
set -e

echo "🔍 PostgREST Migration Setup Health Check"
echo ""

# Check Kubernetes cluster
echo "✅ Kubernetes cluster:"
kubectl cluster-info | head -2

# Check namespaces
echo "✅ Tekton namespace:"
kubectl get ns tekton-pipelines -o name

echo "✅ ArgoCD namespace:"
kubectl get ns argocd -o name

# Check Tekton controller
echo "✅ Tekton controller running:"
kubectl get pods -n tekton-pipelines -l app=tekton-pipelines-controller -o name

# Check ArgoCD components
echo "✅ ArgoCD server running:"
kubectl get pods -n argocd -l app.kubernetes.io/name=argocd-server -o name

# Check example app
if kubectl get heliosapp test-api -n default 2>/dev/null; then
    echo "✅ HeliosApp 'test-api' exists"
    echo "✅ ArgoCD Application:"
    kubectl get application test-api -n argocd -o name 2>/dev/null || echo "⚠️  Not found yet"
else
    echo "⚠️  HeliosApp 'test-api' not found (scaffold first)"
fi

echo ""
echo "🎉 Health check complete!"
```

Save as `scripts/health-check-postgrest.sh` and run:
```bash
bash scripts/health-check-postgrest.sh
```

---

## References

- [Tekton Pipelines Documentation](https://tekton.dev/docs/)
- [ArgoCD Hooks Documentation](https://argo-cd.readthedocs.io/en/stable/user-guide/resource_hooks/)
- [PostgREST API Documentation](https://postgrest.org/en/stable/)
- [golang-migrate Documentation](https://github.com/golang-migrate/migrate)
- [Kubernetes Jobs Documentation](https://kubernetes.io/docs/concepts/workloads/controllers/job/)

