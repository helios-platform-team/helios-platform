# API Reference - HeliosApp CRD

Complete API documentation for `HeliosApp` Custom Resource.

## Resource Definition

```yaml
apiVersion: platform.helios.io/v1
kind: HeliosApp
```

## Spec Fields

### Application Source

| Field         | Type   | Required | Description                                      |
| ------------- | ------ | -------- | ------------------------------------------------ |
| `gitRepo`     | string | Yes      | URL of Git repository containing source code     |
| `gitBranch`   | string | Yes      | Git branch to build                              |
| `gitRevision` | string | No       | Specific commit SHA (optional, defaults to HEAD) |
| `imageRepo`   | string | Yes      | Container registry and image name                |
| `imageTag`    | string | No       | Image tag (defaults to git commit SHA)           |

**Example**:

```yaml
spec:
  gitRepo: "https://github.com/my-org/my-app"
  gitBranch: "main"
  imageRepo: "ghcr.io/my-org/my-app"
```

### Tekton Configuration

| Field            | Type              | Required | Description                         |
| ---------------- | ----------------- | -------- | ----------------------------------- |
| `pipelineName`   | string            | Yes      | Name of Tekton Pipeline to use      |
| `serviceAccount` | string            | Yes      | ServiceAccount with Git credentials |
| `pipelineParams` | map[string]string | No       | Additional parameters for Pipeline  |

**Example**:

```yaml
spec:
  pipelineName: "helios-manifest-pipeline"
  serviceAccount: "helios-tekton-sa"
  pipelineParams:
    timeout: "30m"
    verbosity: "debug"
```

### GitOps Template

| Field          | Type   | Required | Description                                    |
| -------------- | ------ | -------- | ---------------------------------------------- |
| `templateRepo` | string | Yes      | Git repo containing Helm charts or Kustomize   |
| `templatePath` | string | Yes      | Path to template in repo                       |
| `templateType` | string | No       | `helm` or `kustomize` (auto-detect if not set) |

**Example**:

```yaml
spec:
  templateRepo: "https://github.com/my-org/k8s-templates"
  templatePath: "apps/microservice-base"
  templateType: "helm"
```

### GitOps Destination

| Field          | Type   | Required | Description                           |
| -------------- | ------ | -------- | ------------------------------------- |
| `gitopsRepo`   | string | Yes      | Git repo to commit rendered manifests |
| `gitopsPath`   | string | Yes      | Path in gitops repo                   |
| `gitopsBranch` | string | No       | Branch in gitops repo (default: main) |

**Example**:

```yaml
spec:
  gitopsRepo: "git@github.com:my-org/gitops.git"
  gitopsPath: "environments/prod/my-app"
  gitopsBranch: "main"
```

### ArgoCD Configuration

| Field             | Type   | Required | Description                                                      |
| ----------------- | ------ | -------- | ---------------------------------------------------------------- |
| `argocdNamespace` | string | No       | ArgoCD namespace (default: argocd)                               |
| `targetNamespace` | string | No       | Kubernetes namespace cho deployment (default: same as HeliosApp) |
| `autoSync`        | bool   | No       | Enable ArgoCD auto-sync (default: true)                          |
| `prune`           | bool   | No       | Enable pruning resources (default: true)                         |
| `selfHeal`        | bool   | No       | Enable self-healing (default: true)                              |

**Example**:

```yaml
spec:
  argocdNamespace: "argocd"
  targetNamespace: "production"
  autoSync: true
  prune: true
  selfHeal: true
```

### Custom Values

| Field    | Type              | Required | Description                       |
| -------- | ----------------- | -------- | --------------------------------- |
| `values` | map[string]string | No       | Custom values to pass to template |

**Example**:

```yaml
spec:
  values:
    replicas: "3"
    ingress.enabled: "true"
    ingress.host: "app.example.com"
    resources.requests.cpu: "500m"
    resources.requests.memory: "512Mi"
```

## Status Fields

### Phase

| Value                          | Description                |
| ------------------------------ | -------------------------- |
| `Pending`                      | HeliosApp just created     |
| `ManifestGenerationInProgress` | Tekton Pipeline running    |
| `ManifestGenerationFailed`     | Pipeline failed            |
| `DeployingWithArgoCD`          | ArgoCD Application created |
| `Syncing`                      | ArgoCD syncing             |
| `SyncedAndHealthy`             | Deployment successful ✅   |
| `Degraded`                     | ArgoCD reports unhealthy   |

### Conditions

```yaml
status:
  conditions:
    - type: Ready
      status: "True" | "False" | "Unknown"
      reason: <machine-readable>
      message: <human-readable>
      lastTransitionTime: <timestamp>
```

**Condition Types**:

| Type                     | Description                                                         |
| ------------------------ | ------------------------------------------------------------------- |
| `Ready`                  | Overall readiness                                                   |
| `TektonTriggersReady`    | EventListener, TriggerBinding, TriggerTemplate created successfully |
| `PipelineRunSucceeded`   | PipelineRun completed successfully                                  |
| `ArgoApplicationCreated` | ArgoCD Application resource created                                 |
| `ArgoSynced`             | ArgoCD sync successful                                              |
| `ArgoHealthy`            | ArgoCD reports healthy                                              |

### Other Status Fields

| Field                 | Type      | Description                                                                |
| --------------------- | --------- | -------------------------------------------------------------------------- |
| `observedGeneration`  | int64     | Generation of spec that has been processed                                 |
| `lastPipelineRun`     | string    | Name of most recent PipelineRun                                            |
| `lastPipelineRunTime` | timestamp | Time of most recent PipelineRun                                            |
| `argoApplicationName` | string    | Name of ArgoCD Application                                                 |
| `syncStatus`          | string    | ArgoCD sync status: Synced, OutOfSync, Unknown                             |
| `healthStatus`        | string    | ArgoCD health: Healthy, Progressing, Degraded, Suspended, Missing, Unknown |
| `deployedRevision`    | string    | Deployed Git commit SHA                                                    |

**Example Status**:

```yaml
status:
  phase: "SyncedAndHealthy"
  observedGeneration: 2
  lastPipelineRun: "my-app-pipelinerun-abc123"
  lastPipelineRunTime: "2024-01-15T10:30:00Z"
  argoApplicationName: "my-app"
  syncStatus: "Synced"
  healthStatus: "Healthy"
  deployedRevision: "abc123def456"
  conditions:
    - type: Ready
      status: "True"
      reason: "SyncedAndHealthy"
      message: "Application is synced and healthy"
      lastTransitionTime: "2024-01-15T10:35:00Z"
```

## Complete Example

```yaml
apiVersion: platform.helios.io/v1
kind: HeliosApp
metadata:
  name: my-production-app
  namespace: default
  labels:
    app: my-app
    environment: production
spec:
  # Application source
  gitRepo: "https://github.com/my-org/my-app"
  gitBranch: "main"
  imageRepo: "ghcr.io/my-org/my-app"

  # Tekton configuration
  pipelineName: "helios-manifest-pipeline"
  serviceAccount: "helios-tekton-sa"
  pipelineParams:
    timeout: "30m"

  # GitOps template
  templateRepo: "https://github.com/my-org/k8s-templates"
  templatePath: "apps/microservice-base"
  templateType: "helm"

  # GitOps destination
  gitopsRepo: "git@github.com:my-org/gitops.git"
  gitopsPath: "environments/prod/my-app"
  gitopsBranch: "main"

  # ArgoCD configuration
  argocdNamespace: "argocd"
  targetNamespace: "production"
  autoSync: true
  prune: true
  selfHeal: true

  # Custom values
  values:
    replicas: "5"
    ingress.enabled: "true"
    ingress.host: "app.example.com"
    ingress.tls: "true"
    resources.requests.cpu: "1000m"
    resources.requests.memory: "2Gi"
    resources.limits.cpu: "2000m"
    resources.limits.memory: "4Gi"
    autoscaling.enabled: "true"
    autoscaling.minReplicas: "3"
    autoscaling.maxReplicas: "10"
    monitoring.enabled: "true"
```

## Field Validation

### Required Fields

Operator will reject if missing:

- `spec.gitRepo`
- `spec.gitBranch`
- `spec.imageRepo`
- `spec.pipelineName`
- `spec.serviceAccount`
- `spec.templateRepo`
- `spec.templatePath`
- `spec.gitopsRepo`
- `spec.gitopsPath`

### Format Validation

- **gitRepo**: Must be valid Git URL (https:// or git@)
- **imageRepo**: Must be valid image reference
- **pipelineName**: Must match existing Tekton Pipeline
- **serviceAccount**: Must exist in same namespace

## Default Values

| Field             | Default Value       |
| ----------------- | ------------------- |
| `imageTag`        | Git commit SHA      |
| `templateType`    | Auto-detected       |
| `gitopsBranch`    | "main"              |
| `argocdNamespace` | "argocd"            |
| `targetNamespace` | HeliosApp namespace |
| `autoSync`        | true                |
| `prune`           | true                |
| `selfHeal`        | true                |

## Immutable Fields

After creation, the following fields **cannot be changed**:

- `metadata.name`
- `metadata.namespace`

All spec fields can be updated → triggers new reconciliation.

## Annotations

### Special Annotations

| Annotation                       | Description                            |
| -------------------------------- | -------------------------------------- |
| `helios.io/pause-reconciliation` | Set "true" to pause reconciliation     |
| `helios.io/manual-sync`          | Set "true" to disable auto-sync ArgoCD |
| `helios.io/webhook-url`          | Auto-generated webhook URL             |

**Example**:

```yaml
metadata:
  annotations:
    helios.io/pause-reconciliation: "true"
```

## Labels

### Auto-Generated Labels

Operator automatically adds:

- `helios.io/app-name: <name>`
- `helios.io/managed-by: helios-operator`

### Recommended Labels

```yaml
metadata:
  labels:
    app: my-app
    environment: production
    team: platform
    cost-center: engineering
```

## Events

Operator emits Kubernetes Events:

| Reason                   | Type    | Description               |
| ------------------------ | ------- | ------------------------- |
| `TektonTriggersCreated`  | Normal  | Triggers setup successful |
| `PipelineRunTriggered`   | Normal  | PipelineRun triggered     |
| `PipelineRunSucceeded`   | Normal  | PipelineRun successful    |
| `PipelineRunFailed`      | Warning | PipelineRun failed        |
| `ArgoApplicationCreated` | Normal  | ArgoCD App created        |
| `SyncedAndHealthy`       | Normal  | Deployment healthy        |
| `ReconciliationError`    | Warning | Error in reconciliation   |

View events:

```bash
kubectl describe heliosapp <name>
```

## Finalizers

Operator uses finalizer:

```yaml
metadata:
  finalizers:
    - helios.io/cleanup
```

When deleting HeliosApp:

1. Delete ArgoCD Application (cascading delete resources)
2. Delete Tekton Triggers
3. Delete PipelineRuns
4. Remove finalizer

## RBAC

ServiceAccount needs permissions:

```yaml
# For HeliosApp controller
- platform.helios.io/heliosapps: get, list, watch, update, patch

# For Tekton
- tekton.dev/pipelineruns: create, get, list, watch
- triggers.tekton.dev/*: create, get, list, watch, update, patch

# For ArgoCD
- argoproj.io/applications: create, get, list, watch, update, patch, delete
```

## See Also

- [Getting Started](./GETTING_STARTED.md) - Tutorial
- [Architecture](./ARCHITECTURE.md) - How it works
- [Examples](../config/samples/) - Sample YAMLs
