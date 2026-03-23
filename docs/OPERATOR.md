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
