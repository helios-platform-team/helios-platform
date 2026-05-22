# Helios Operator

The **Helios Operator** is the core control plane component of the Helios Platform. It manages the lifecycle of `HeliosApp` resources, translating high-level application definitions into low-level Kubernetes resources using a CUE-based rendering engine.

## 🚀 Overview

- **Custom Resource Definition (CRD)**: Defines the `HeliosApp` API.
- **Controller**: Watches for `HeliosApp` changes and reconciles them.
- **CUE Integration**: Utilizes the logic in the `/cue` directory to generate Kubernetes manifests (Deployments, Services, Ingresses, etc.).

## 📋 Prerequisites

To develop or run the operator, you need:

- **Go**: v1.26.0+
- **Docker**: 17.03+
- **kubectl**: v1.11.3+
- **Kubernetes Cluster**: Access to a `v1.11.3+` cluster (e.g., Kind, Minikube, or EKS).
- **CUE CLI**: Recommended for local template testing.

See the [Setup Guide](./SETUP.md) for detailed installation instructions.

## 🛠 Development & Build

The project uses a `Makefile` to automate common tasks. Commands should be run from the **project root** using `make -C apps/operator`.

### 1. Build the Operator

Generate code, manifests, and build the binary:

```bash
make -C apps/operator build
```

### 2. Run Locally

Run the controller against your current `~/.kube/config` context:

```bash
make -C apps/operator run
```

### 3. Generate Manifests

Update CRDs and RBAC rules after changing API types:

```bash
make -C apps/operator manifests
```

## 🧪 Testing

### 1. Run All Tests

Runs unit tests, controller tests (using `envtest`), and CUE engine integration tests:

```bash
# Note: Ensure you are in the project root
go -C apps/operator test ./...
```

### 2. Test Specific Components

- **CUE Engine Integration**:

  ```bash
  go -C apps/operator test -v ./internal/cue/...
  ```

- **Controllers**:

  ```bash
  go -C apps/operator test -v ./internal/controller/...
  ```

### 3. End-to-End (e2e) Tests

Requires `kind` installed. This creates a local cluster and runs integration scenarios:

```bash
make -C apps/operator test-e2e
```

## 🚢 Deployment

### 1. Build and Push Container Image

```bash
export IMG=your-registry/helios-operator:latest
make -C apps/operator docker-build docker-push
```

## Verification (Local Dev)

To verify the API and schema validation without deploying the full operator:

### 1. Install CRD only

```bash
make -C apps/operator install
```

### 2. Check CRD Status

```bash
kubectl get crd heliosapps.app.helios.io
```

### 3. Apply Sample Application

Apply the sample to verify the cluster accepts `HeliosApp` objects:

```bash
kubectl apply -f apps/operator/config/samples/app_v1alpha1_heliosapp.yaml

# Verify the object exists
kubectl get heliosapp sample-app -o yaml
```

## 🚢 Deployment (Cluster)

### 1. Install CRDs

```bash
make -C apps/operator install
```

### 2. Deploy to Cluster

```bash
make -C apps/operator deploy IMG=$IMG
```

### 3. Verify Installation

Check if the CRDs were applied correctly:

```bash
kubectl get crd heliosapps.app.helios.io
```

Test the API by applying a sample application:

```bash
kubectl apply -f apps/operator/config/samples/app_v1alpha1_heliosapp.yaml

# Check the created resource
kubectl get heliosapp sample-app
```

### 4. Cleanup Sample

```bash
kubectl delete -f apps/operator/config/samples/app_v1alpha1_heliosapp.yaml
```

### 5. Uninstall/Undeploy

```bash
# undeploy removes the controller AND the CRDs
make -C apps/operator undeploy

# install/uninstall handles ONLY the CRDs
make -C apps/operator uninstall
```

## 📂 Project Structure

- `api/v1alpha1/`: API definitions for `HeliosApp`.
- `cmd/main.go`: Entry point for the operator manager.
- `config/`: Kustomize manifests for CRDs, RBAC, and deployment.
- `internal/controller/`: Reconciliation logic for `HeliosApp`.
- `internal/cue/`: Go wrapper for the CUE rendering engine.

## 🗄️ PostgREST Database Migration Flow

The operator supports automatic database migrations for PostgREST services via an integrated CI/CD → GitOps → ArgoCD workflow.

### Overview

When a HeliosApp component has a `database` trait, the platform automatically:

1. **Builds Migration Image** (Tekton): Creates a Docker image with `golang-migrate` tool and SQL migration scripts
2. **Pushes to Registry** (Tekton): Tags the image as `<registry>/<app>-migrate:latest`
3. **Runs PreSync Job** (ArgoCD): Before syncing PostgREST Deployment, ArgoCD runs the PreSync Job to execute migrations
4. **Blocks on Failure** (ArgoCD): If migrations fail, ArgoCD sync is blocked and PostgREST is not updated
5. **Restarts Pods** (ArgoCD): After migrations succeed, PostSync Job restarts PostgREST pods to invalidate schema cache

### Components

#### Tekton Pipeline: `db-migrate-image`
- **Trigger**: Activates when `db/migrations/` path changes in source repository
- **Tasks**:
  1. Clone source repository
  2. Build Docker image with `Dockerfile.migrate` and migration scripts
  3. Push image to registry with `:latest` tag
- **Location**: `cue/definitions/tekton/pipelines/db-migrate-image.cue`

#### GitOps Manifests (created via Template Scaffolder)
- **Dockerfile.migrate**: Multi-stage build with golang-migrate
- **presync-job.yaml**: ArgoCD PreSync hook Job that runs migrations
- **kustomization.yaml**: Bundles namespace, HeliosApp, and presync-job resources
- **Location**: `apps/portal/examples/postgrest-template/content/gitops/`

#### ArgoCD Hooks
- **PreSync**: Runs migration Job before Deployment sync
- **PostSync**: Restarts PostgREST pods after successful sync
- **Hook Deletion Policy**: `BeforeHookCreation` ensures old Jobs are cleaned up before new ones start

### Operation

#### Scaffolding a PostgREST App with Migrations

```bash
# Use the Backstage scaffolder: choose "PostgREST API Template"
# The template will:
# 1. Create source repository with Dockerfile, migrations, etc.
# 2. Create GitOps repository with kustomization.yaml, helios-app.yaml, presync-job.yaml
# 3. Create db-migrate-image Tekton trigger to watch db/migrations/ changes
# 4. Apply HeliosApp and namespace to cluster
```

#### Triggering Database Migrations

```bash
# Add or modify a migration file
echo "CREATE TABLE new_table (id SERIAL PRIMARY KEY);" > db/migrations/000002_add_table.up.sql

# Push to source repository
git add db/migrations/000002_add_table.up.sql
git commit -m "Add new_table migration"
git push origin main

# Automated flow:
# 1. Webhook triggers db-migrate-image Tekton pipeline
# 2. Pipeline builds migration image with tag :latest
# 3. Pipeline pushes image to registry
# 4. ArgoCD detects presync-job.yaml referencing <app>-migrate:latest
# 5. ArgoCD's PreSync hook runs the Job with fresh image
# 6. Job executes: migrate -path /migrations -database $PGRST_DB_URI up
# 7. If successful: ArgoCD syncs Deployment, PostSync restarts pods
# 8. If failed: ArgoCD blocks sync, operator logs failure
```

#### Migration Script in Docker Image

The `Dockerfile.migrate` embeds an entrypoint script:

```bash
#!/bin/bash
set -e
echo "Running database migrations..."
migrate -path /migrations -database "$PGRST_DB_URI" up
echo "Migrations completed successfully"
```

The `PGRST_DB_URI` environment variable is injected from the `<app>-db-credentials` Secret (managed by the operator's database trait).

### Configuration

#### HeliosApp Example

```yaml
apiVersion: app.helios.io/v1alpha1
kind: HeliosApp
metadata:
  name: my-api
spec:
  components:
  - name: postgrest-api
    type: web-service
    properties:
      image: index.docker.io/postgrest/postgrest:latest
    traits:
    - type: database
      properties:
        dbType: postgres
        version: "16"
        storage: "1Gi"
```

When the `database` trait is present, the scaffolder automatically includes:
- PreSync Job configured to use `<org>/my-api-migrate:latest` image
- ServiceAccount with permissions to run the Job
- Kustomization to apply all resources

### Troubleshooting

#### Migration Job Fails

Check the ArgoCD Application status:

```bash
kubectl get application <app-name>-argocd -n argocd -o yaml
# Look for: syncResult.syncPhase = Failed or Synced=false
```

View Job logs:

```bash
kubectl logs job/<app>-db-migrate-presync -n <namespace>
```

Retry migration:

```bash
# Edit source repository migration file or create new migration
# Commit and push changes
# db-migrate-image pipeline will re-run and retry migrations
```

#### Image Not Pulling

Verify image exists in registry:

```bash
docker pull <registry>/<app>-migrate:latest
# or check via registry UI
```

Verify PreSync Job has correct image reference:

```bash
kubectl get job <app>-db-migrate-presync -n <namespace> -o yaml | grep image
```

#### Pod Not Restarting After Migration

Verify PostSync Job ran:

```bash
kubectl get job <app>-postgrest-restart-postsync -n <namespace> -o yaml
```

Manually restart pods if needed:

```bash
kubectl rollout restart deployment/<app> -n <namespace>
```

### RBAC Permissions

The operator requires these permissions for migration management:

```yaml
apiGroups:
  - batch
resources:
  - jobs
verbs:
  - create
  - delete
  - get
  - list
  - patch
  - watch
```

These are configured in `apps/operator/config/rbac/role.yaml`.
