# Helios Operator - Setup Guide

Complete guide for setting up the Helios Operator development and testing environment.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Installation](#installation)
  - [Automated Setup (Recommended)](#automated-setup-recommended)
  - [Alternative: Using Makefile](#alternative-using-makefile)
  - [Manual Setup](#manual-setup)
- [Verification](#verification)
- [Troubleshooting](#troubleshooting)
- [Quick Reference](#quick-reference)

---

## Prerequisites

Before starting, ensure you have the following installed:

| Tool    | Version | Purpose                | Installation                                                              |
| ------- | ------- | ---------------------- | ------------------------------------------------------------------------- |
| Go      | 1.25+   | Compile operator       | [golang.org/dl](https://golang.org/dl/)                                   |
| Docker  | 28.5+   | Build container images | [docs.docker.com/get-docker](https://docs.docker.com/get-docker/)         |
| kubectl | 1.34+   | Kubernetes CLI         | [kubernetes.io/docs/tasks/tools](https://kubernetes.io/docs/tasks/tools/) |
| Kind    | 0.30+   | Local K8s cluster      | [kind.sigs.k8s.io](https://kind.sigs.k8s.io/docs/user/quick-start/)       |
| Make    | latest  | Build automation       | Pre-installed on most systems                                             |

**Verify installation:**

```bash
go version        # Should show 1.25+
docker --version  # Should show 28.5+
kubectl version --client
kind version
make --version
```

---

## Installation

### Automated Setup (Recommended)

The fastest way to get started with a complete local e2e testing environment:

```bash
# 1. Clone repository
git clone https://github.com/hoangphuc841/helios-operator.git
cd helios-operator

# 2. Run the automated setup script
./scripts/setup-e2e-local.sh
```

This single command will:

- ✅ Check prerequisites (Docker, kubectl, kind, Go)
- ✅ Create Kind cluster
- ✅ Install Tekton Pipelines and Triggers
- ✅ Install ArgoCD
- ✅ Install Tekton resources (Pipeline, Tasks, ServiceAccount, PVC)
- ✅ Build operator image
- ✅ Install CRDs
- ✅ Deploy operator

**Timeline:** ~10-15 minutes total

**After setup completes**, you can immediately test:

```bash
# Apply sample HeliosApp
kubectl apply -f config/samples/heliosapp_v1_heliosapp.yaml

# Watch the status
kubectl get heliosapp --watch
```

### Alternative: Using Makefile

If you prefer step-by-step control:

```bash
# 1. Clone repository
git clone https://github.com/hoangphuc841/helios-operator.git
cd helios-operator

# 2. Setup cluster and dependencies
make setup-test-e2e

# 3. Build and deploy operator
make docker-build IMG=helios-operator:dev
kind load docker-image helios-operator:dev --name helios-operator-test-e2e
make deploy IMG=helios-operator:dev
```

### Manual Setup

For more control over the process:

#### Step 1: Create Kubernetes Cluster

```bash
# Option A: Kind (recommended for development)
kind create cluster --name helios-test

# Option B: Minikube
minikube start --cpus=4 --memory=8192

# Option C: Use existing cluster
kubectl config use-context <your-context>
```

#### Step 2: Install Tekton Pipelines

```bash
# Install Tekton Pipelines
kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml

# Wait for readiness
kubectl wait --for=condition=ready pod \
  -l app=tekton-pipelines-controller \
  -n tekton-pipelines \
  --timeout=300s

# Install Tekton Triggers
kubectl apply -f https://storage.googleapis.com/tekton-releases/triggers/latest/release.yaml
kubectl apply -f https://storage.googleapis.com/tekton-releases/triggers/latest/interceptors.yaml

# Wait for readiness
kubectl wait --for=condition=ready pod \
  -l app=tekton-triggers-controller \
  -n tekton-pipelines \
  --timeout=300s
```

#### Step 3: Install ArgoCD

```bash
# Create namespace
kubectl create namespace argocd

# Install ArgoCD
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# Wait for ArgoCD to be ready
kubectl wait --for=condition=ready pod \
  -l app.kubernetes.io/name=argocd-server \
  -n argocd \
  --timeout=300s

# (Optional) Expose ArgoCD UI
kubectl port-forward svc/argocd-server -n argocd 8080:443
# Access at: https://localhost:8080
# Username: admin
# Password: kubectl get secret argocd-initial-admin-secret -n argocd -o jsonpath="{.data.password}" | base64 -d
```

#### Step 4: Build and Deploy Operator

```bash
# Install CRDs
make install

# Install Tekton Pipeline resources (Tasks and Pipeline)
kubectl apply -f tekton/

# Build operator image
make docker-build IMG=helios-operator:dev

# Load image to Kind (if using Kind)
kind load docker-image helios-operator:dev --name helios-test

# Deploy operator
make deploy IMG=helios-operator:dev

# Wait for operator to be ready
kubectl wait --for=condition=available deployment/helios-operator-controller-manager \
  -n helios-operator-system \
  --timeout=300s
```

---

## Verification

### 1. Check Operator Status

```bash
# Check operator deployment
kubectl get deployment -n helios-operator-system
# Expected: helios-operator-controller-manager (1/1 Ready)

# Check operator logs
kubectl logs -n helios-operator-system deployment/helios-operator-controller-manager -c manager

# Check CRDs installed
kubectl get crd heliosapps.platform.helios.io
```

### 2. Check Tekton Components

```bash
# Check Tekton Pipelines
kubectl get pods -n tekton-pipelines
# Expected: All pods Running

# Check Tekton CRDs
kubectl api-resources | grep tekton
# Expected: pipelines, pipelineruns, tasks, taskruns, etc.
```

### 3. Check ArgoCD Components

```bash
# Check ArgoCD pods
kubectl get pods -n argocd
# Expected: argocd-server, argocd-repo-server, argocd-application-controller Running

# Check ArgoCD CRDs
kubectl get crd applications.argoproj.io
```

### 4. Test with Sample Application

```bash
# Create sample HeliosApp
kubectl apply -f config/samples/heliosapp_v1_heliosapp.yaml

# Watch status
kubectl get heliosapp sample-heliosapp --watch

# Check created resources
kubectl get eventlistener,triggerbinding,triggertemplate
kubectl get pipelinerun
kubectl get application -n argocd
```

**Note**: The `make setup-test-e2e` command automatically installs:

- PVC for pipeline workspace (`shared-workspace-pvc`)
- ServiceAccount for Tekton Triggers (`tekton-triggers-sa`)
- All Tekton Tasks and Pipeline resources

If you skipped the automated setup, install them manually:

```bash
kubectl apply -f tekton/
```

---

## Troubleshooting

### Issue: "CRD not found"

**Symptoms:**

```text
error: the server doesn't have a resource type "heliosapps"
```

**Solution:**

```bash
make install
kubectl get crd heliosapps.platform.helios.io
```

### Issue: "Operator pod crashloop"

**Symptoms:**

```text
kubectl get pods -n helios-operator-system
# Shows CrashLoopBackOff
```

**Solution:**

```bash
# Check logs
kubectl logs -n helios-operator-system deployment/helios-operator-controller-manager -c manager --previous

# Common causes:
# 1. RBAC permissions missing -> Re-deploy with: make deploy IMG=...
# 2. Invalid webhook config -> Check cert-manager is installed
# 3. Resource limits -> Adjust in config/manager/manager.yaml
```

### Issue: "Tekton PipelineRun not starting"

**Symptoms:**

```text
kubectl get pipelinerun
# Shows "Pending" or "CouldntGetPipeline" for long time
```

**Solution:**

```bash
# Check if Tekton Pipeline resources are installed
kubectl get pipeline

# If no pipelines found, install them:
kubectl apply -f tekton/

# Check Tekton controller logs
kubectl logs -n tekton-pipelines deployment/tekton-pipelines-controller

# Check ServiceAccount exists
kubectl get sa <serviceaccount-name> -n <namespace>

# Check PVC exists (if pipeline uses workspaces)
kubectl get pvc -n <namespace>
```

### Issue: "ArgoCD Application not syncing"

**Symptoms:**

```text
kubectl get application -n argocd
# Shows "OutOfSync" or "Unknown"
```

**Solution:**

```bash
# Check ArgoCD application details
kubectl describe application <app-name> -n argocd

# Check ArgoCD has access to git repo
# For private repos, configure SSH keys or tokens in ArgoCD

# Check manifest repo exists and has valid manifests
```

### Issue: "PVC not found" or "Pod pending due to PVC"

**Symptoms:**

```text
kubectl get pods
# Shows Pending
kubectl describe pod <pod-name>
# Shows: persistentvolumeclaim "shared-workspace-pvc" not found
```

**Solution:**

```bash
# Install the PVC from the tekton directory
kubectl apply -f tekton/workspace-pvc.yaml

# Or install all tekton resources
kubectl apply -f tekton/

# Verify PVC is bound
kubectl get pvc
```

### Issue: "EventListener pod not starting - ServiceAccount not found"

**Symptoms:**

```text
kubectl get eventlistener
# Shows AVAILABLE: False

kubectl get events
# Shows: serviceaccount "tekton-triggers-sa" not found
```

**Solution:**

```bash
# Install the ServiceAccount from the tekton directory
kubectl apply -f tekton/tekton-triggers-sa.yaml

# Or install all tekton resources
kubectl apply -f tekton/

# Verify ServiceAccount exists
kubectl get sa tekton-triggers-sa
```

### Issue: "Kind cluster creation fails"

**Symptoms:**

```text
ERROR: failed to create cluster: ...
```

**Solution:**

```bash
# Delete existing cluster
kind delete cluster --name helios-operator-test-e2e

# Try again with verbose logging
kind create cluster --name helios-operator-test-e2e --verbosity=3

# Check Docker is running
docker ps

# Check available resources
docker system df
```

---

## Quick Reference

### Common Commands

```bash
# Development
make fmt                    # Format code
make vet                    # Run go vet
make test                   # Run unit tests
make build                  # Build operator binary

# Docker
make docker-build IMG=...   # Build container image
make docker-push IMG=...    # Push to registry

# Deployment
make install                # Install CRDs
make deploy IMG=...         # Deploy operator
make undeploy               # Remove operator
make uninstall              # Remove CRDs

# E2E Testing
make setup-test-e2e         # Create Kind cluster and install Tekton + ArgoCD
make test-e2e               # Run full E2E suite (includes setup and cleanup)

# Debugging
kubectl logs -n helios-operator-system deployment/helios-operator-controller-manager -c manager
kubectl describe heliosapp <name>
kubectl get events --sort-by='.lastTimestamp'
```

### Environment Variables

```bash
# Operator configuration
export IMG=helios-operator:dev           # Image name
export KIND_CLUSTER=helios-test          # Kind cluster name
export CERT_MANAGER_INSTALL_SKIP=true    # Skip cert-manager installation

# Kubebuilder markers
export KUBEBUILDER_ASSETS=/path/to/testbin
```

### Useful Aliases

Add to your `~/.bashrc` or `~/.zshrc`:

```bash
# Helios Operator aliases
alias ho-logs='kubectl logs -n helios-operator-system deployment/helios-operator-controller-manager -c manager'
alias ho-deploy='make docker-build IMG=helios-operator:dev && kind load docker-image helios-operator:dev && make deploy IMG=helios-operator:dev'
alias ho-test='make test-e2e'
alias ho-watch='kubectl get heliosapp --watch'
```

---

## Next Steps

After successful setup:

1. **Read Testing Guide**: See [TESTING_GUIDE.md](./TESTING_GUIDE.md) for testing strategies
2. **Run E2E Tests**: Execute `make test-e2e` to validate setup
3. **Deploy Sample App**: Test with sample HeliosApp from `config/samples/`
4. **Production Setup**: Follow [PRODUCTION_DEPLOYMENT.md](./PRODUCTION_DEPLOYMENT.md) for production deployment

---

## Additional Resources

- **Main README**: [README.md](../README.md)
- **Contributing**: [CONTRIBUTING.md](../CONTRIBUTING.md)
- **Development**: [DEVELOPMENT.md](../DEVELOPMENT.md)
- **Tekton Documentation**: <https://tekton.dev/docs/>
- **ArgoCD Documentation**: <https://argo-cd.readthedocs.io/>
- **Kubebuilder Book**: <https://book.kubebuilder.io/>

---

**Need Help?**

- GitHub Issues: <https://github.com/hoangphuc841/helios-operator/issues>
- Slack Channel: [TBD]
- Email: [maintainer-email]
