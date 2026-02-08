# CI/CD Observability Guide

This guide explains how to configure and use the CI/CD Observability features in Helios Platform's Backstage portal, including Tekton Pipeline visibility and ArgoCD Sync status.

## Overview

The Backstage portal integrates with:
- **Tekton**: View PipelineRuns and TaskRuns directly in the entity's CI/CD tab
- **ArgoCD**: View deployment sync status and history in the entity overview

```
┌─────────────────────────────────────────────────────────────────┐
│                     Backstage Portal                            │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐      │
│  │   Catalog    │    │   CI/CD Tab  │    │   Overview   │      │
│  │   Entity     │───▶│   (Tekton)   │    │   (ArgoCD)   │      │
│  └──────────────┘    └──────┬───────┘    └──────┬───────┘      │
└─────────────────────────────┼──────────────────┼───────────────┘
                              │                  │
                              ▼                  ▼
                    ┌─────────────────┐  ┌─────────────────┐
                    │   Kubernetes    │  │     ArgoCD      │
                    │   (Tekton CRs)  │  │     Server      │
                    └─────────────────┘  └─────────────────┘
```

## Architecture

### Backend Proxy Flow

```
Browser → Backstage Backend → Kubernetes API (Tekton CRDs)
                           → ArgoCD API (via proxy)
```

### Plugins Used

| Plugin | Package | Purpose |
|--------|---------|---------|
| Tekton | `@backstage-community/plugin-tekton` | Display PipelineRuns/TaskRuns |
| ArgoCD | `@roadiehq/backstage-plugin-argo-cd` | Display sync status & history |
| Kubernetes | `@backstage/plugin-kubernetes` | Core K8s connectivity |

## Configuration

### Configuration Files

| File | Purpose |
|------|---------|
| `app-config.yaml` | Production defaults (in-cluster) |
| `app-config.local.yaml` | Local development overrides |

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `ARGOCD_AUTH_TOKEN` | Yes | ArgoCD API token for authentication |

## Setup Instructions

### Local Development (Minikube)

**Prerequisites:**
- Minikube running with Tekton and ArgoCD installed
- `kubectl` configured to connect to Minikube

**Step 1: Start kubectl proxy** (Terminal 1)
```bash
kubectl proxy --port=8001
```

**Step 2: Port-forward ArgoCD** (Terminal 2)
```bash
kubectl port-forward svc/argocd-server -n argocd 8081:443
```

**Step 3: Get ArgoCD token and start Backstage** (Terminal 3)
```bash
# Get the ArgoCD admin password
export ARGOCD_AUTH_TOKEN=$(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d)

# Start Backstage (from apps/portal directory)
cd apps/portal
yarn start
```

### Production (In-Cluster Deployment)

When Backstage runs as a pod in the same Kubernetes cluster:

1. **ArgoCD URL** is automatically resolved via internal DNS:
   ```
   http://argocd-server.argocd.svc.cluster.local/api/v1/
   ```

2. **Kubernetes API** is accessed via:
   ```
   https://kubernetes.default.svc
   ```

3. **Authentication** uses the pod's ServiceAccount (auto-mounted token)

**Required RBAC for ServiceAccount:**
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: backstage-tekton-reader
rules:
  - apiGroups: ["tekton.dev"]
    resources: ["pipelineruns", "taskruns"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["pods", "pods/log"]
    verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: backstage-tekton-reader
subjects:
  - kind: ServiceAccount
    name: backstage
    namespace: backstage
roleRef:
  kind: ClusterRole
  name: backstage-tekton-reader
  apiGroup: rbac.authorization.k8s.io
```

**ArgoCD Token Secret:**
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: backstage-argocd-token
  namespace: backstage
type: Opaque
stringData:
  ARGOCD_AUTH_TOKEN: <your-argocd-api-token>
```

## Entity Annotations

To link a Backstage entity to its CI/CD resources, add these annotations to your `catalog-info.yaml`:

```yaml
apiVersion: backstage.io/v1alpha1
kind: Component
metadata:
  name: my-service
  annotations:
    # ArgoCD: Links to the ArgoCD Application name
    argocd/app-name: my-service
    
    # Tekton: Label selector to find relevant PipelineRuns
    janus-idp.io/tekton: my-service
    
    # Kubernetes: General workload visibility
    backstage.io/kubernetes-label-selector: 'app.kubernetes.io/part-of=my-service'
spec:
  type: service
  lifecycle: production
  owner: team-a
```

### Annotation Reference

| Annotation | Plugin | Description |
|------------|--------|-------------|
| `argocd/app-name` | ArgoCD | Name of the ArgoCD Application CR |
| `janus-idp.io/tekton` | Tekton | Label value to match PipelineRuns |
| `backstage.io/kubernetes-label-selector` | Kubernetes | K8s label selector for workloads |
| `backstage.io/kubernetes-id` | Kubernetes | Alternative: exact name match |

## Using the UI

### CI/CD Tab (Tekton)

1. Navigate to a Component in the Catalog
2. Click the **CI/CD** tab
3. View recent PipelineRuns with:
   - Status (Running, Succeeded, Failed)
   - Duration
   - Start time
   - TaskRun details

### Overview Card (ArgoCD)

1. Navigate to a Component in the Catalog
2. On the **Overview** tab, find the ArgoCD card showing:
   - Sync Status (Synced, OutOfSync, Unknown)
   - Health Status (Healthy, Degraded, Progressing)
   - Recent sync history

## Troubleshooting

### Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| "No CI/CD available" | Missing annotations | Add `janus-idp.io/tekton` annotation to entity |
| ArgoCD card not showing | Missing annotation | Add `argocd/app-name` annotation to entity |
| 401 Unauthorized (ArgoCD) | Invalid/expired token | Regenerate `ARGOCD_AUTH_TOKEN` |
| No PipelineRuns found | Wrong label selector | Verify PipelineRuns have matching labels |
| Connection refused (local) | Port-forward not running | Start `kubectl proxy` and ArgoCD port-forward |

### Verify Tekton Labels

Ensure your PipelineRuns have the correct labels:

```bash
# Check PipelineRun labels
kubectl get pipelineruns -l app.kubernetes.io/part-of=my-service

# Or with the Tekton annotation label
kubectl get pipelineruns -l janus-idp.io/tekton=my-service
```

### Verify ArgoCD Application

```bash
# List ArgoCD applications
kubectl get applications -n argocd

# Check specific app
kubectl get application my-service -n argocd -o yaml
```

### Debug Kubernetes Connectivity

```bash
# Test kubectl proxy
curl http://localhost:8001/api/v1/namespaces

# Test ArgoCD proxy
curl -k -H "Authorization: Bearer $ARGOCD_AUTH_TOKEN" \
  https://localhost:8081/api/v1/applications
```

## File Reference

### apps/portal/app-config.yaml (Production)

```yaml
proxy:
  endpoints:
    '/argocd/api':
      target: http://argocd-server.argocd.svc.cluster.local/api/v1/
      changeOrigin: true
      secure: false
      headers:
        Authorization: Bearer ${ARGOCD_AUTH_TOKEN}

kubernetes:
  serviceLocatorMethod:
    type: 'multiTenant'
  clusterLocatorMethods:
    - type: 'config'
      clusters:
        - url: https://kubernetes.default.svc
          name: in-cluster
          authProvider: 'serviceAccount'
          skipTLSVerify: true
  customResources:
    - group: 'tekton.dev'
      apiVersion: 'v1'
      plural: 'pipelineruns'
    - group: 'tekton.dev'
      apiVersion: 'v1'
      plural: 'taskruns'
```

### apps/portal/app-config.local.yaml (Local Dev)

```yaml
proxy:
  endpoints:
    '/argocd/api':
      target: https://localhost:8081/api/v1/
      changeOrigin: true
      secure: false
      headers:
        Authorization: Bearer ${ARGOCD_AUTH_TOKEN}

kubernetes:
  serviceLocatorMethod:
    type: 'multiTenant'
  clusterLocatorMethods:
    - type: 'localKubectlProxy'
  customResources:
    - group: 'tekton.dev'
      apiVersion: 'v1'
      plural: 'pipelineruns'
    - group: 'tekton.dev'
      apiVersion: 'v1'
      plural: 'taskruns'
```

## Related Documentation

- [Backstage Kubernetes Plugin](https://backstage.io/docs/features/kubernetes/)
- [Tekton Plugin (Janus IDP)](https://github.com/janus-idp/backstage-plugins/tree/main/plugins/tekton)
- [Roadie ArgoCD Plugin](https://github.com/RoadieHQ/roadie-backstage-plugins/tree/main/plugins/frontend/backstage-plugin-argo-cd)
- [Helios Automation Guide](../apps/operator/AUTOMATION_GUIDE.md)
