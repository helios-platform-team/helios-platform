# Development Guide

Detailed guide for developing, building, and contributing to Helios Operator.

## Development Environment Setup

### Prerequisites

- **Go**: 1.25+
- **Docker**: 28.5+
- **kubectl**: 1.34+
- **Kind**: 0.30+ (or Minikube)
- **Kubebuilder**: 4.9+
- **Make**: GNU Make
- **Git**: Latest version

### Clone Repository

```bash
git clone https://github.com/your-org/helios-operator.git
cd helios-operator
```

### Install Dependencies

```bash
# Install Go dependencies
go mod download

# Install development tools
make install-tools
```

## Project Structure

```text
helios-operator/
├── api/v1/                  # CRD API definitions
│   ├── heliosapp_types.go   # HeliosApp spec & status
│   └── zz_generated.*       # Generated code
├── cmd/main.go              # Operator entrypoint
├── internal/controller/     # Reconciliation logic
│   ├── heliosapp_controller.go      # Main controller
│   ├── tekton_resources.go          # Tekton integration
│   └── heliosapp_controller_test.go # Unit tests
├── config/                  # Kustomize configs
│   ├── crd/                 # CRD manifests
│   ├── manager/             # Operator deployment
│   ├── rbac/                # RBAC rules
│   └── samples/             # Example CRs
├── test/e2e/                # End-to-end tests
├── tekton/                  # Tekton Pipeline definitions
├── docs/                    # Documentation
└── Makefile                 # Build automation
```

## Build & Run Locally

### Run Against Local Cluster

**Step 1: Create Kind cluster**:

```bash
make setup-test-e2e  # Creates cluster + installs Tekton + ArgoCD
```

**Step 2: Install CRD**:

```bash
make install
```

**Step 3: Run operator locally**:

```bash
make run
```

The operator runs outside the cluster, connecting via kubeconfig.

### Run Inside Cluster

**Step 1: Build & push image**:

```bash
export IMG=your-registry/helios-operator:dev
make docker-build docker-push
```

**Step 2: Deploy**:

```bash
make deploy IMG=$IMG
```

**Step 3: Check logs**:

```bash
kubectl logs -n helios-system deployment/helios-operator-controller-manager -f
```

## Development Workflow

### 1. Make Code Changes

Edit files in `api/v1/` or `internal/controller/`.

### 2. Generate Code

After modifying CRD types:

```bash
make generate  # Generate DeepCopy methods
make manifests # Generate CRD YAML
```

### 3. Run Tests

```bash
# Unit tests
make test

# E2E tests
make test-e2e

# Coverage
make test-coverage
```

### 4. Lint & Format

```bash
# Format code
make fmt

# Lint
make vet

# Golangci-lint
make lint
```

### 5. Build

```bash
# Build binary
make build

# Build Docker image
make docker-build
```

## Testing

### Unit Tests

Located in `internal/controller/*_test.go`.

**Run**:

```bash
make test
```

**Write new test**:

```go
var _ = Describe("HeliosApp Controller", func() {
    Context("When reconciling a resource", func() {
        It("Should create Tekton Triggers", func() {
            ctx := context.Background()

            heliosApp := &platformv1.HeliosApp{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "test-app",
                    Namespace: "default",
                },
                Spec: platformv1.HeliosAppSpec{
                    GitRepo: "https://github.com/test/app",
                    // ... full spec
                },
            }

            Expect(k8sClient.Create(ctx, heliosApp)).To(Succeed())

            // Verify EventListener created
            el := &triggersv1.EventListener{}
            Eventually(func() bool {
                err := k8sClient.Get(ctx, types.NamespacedName{
                    Name:      "test-app-listener",
                    Namespace: "default",
                }, el)
                return err == nil
            }, timeout, interval).Should(BeTrue())
        })
    })
})
```

### E2E Tests

Located in `test/e2e/e2e_test.go`.

**Run**:

```bash
make test-e2e
```

Tests cover:

- ✅ Phase 1: Tekton Triggers creation
- ✅ Phase 2: PipelineRun execution → ArgoCD App creation
- ✅ Phase 3: Status sync from ArgoCD
- ✅ Spec changes trigger reconciliation
- ✅ Failure handling

### Manual Testing

**Step 1: Create test HeliosApp**:

```bash
kubectl apply -f config/samples/heliosapp_v1_heliosapp.yaml
```

**Step 2: Watch status**:

```bash
kubectl get heliosapp -w
```

**Step 3: Check resources**:

```bash
# Tekton Triggers
kubectl get eventlistener,triggerbinding,triggertemplate

# PipelineRuns
kubectl get pipelinerun

# ArgoCD Applications
kubectl get application -n argocd
```

**Step 4: Cleanup**:

```bash
kubectl delete heliosapp --all
```

## Debugging

### Debug Operator Locally

Use VS Code or Delve:

```bash
dlv debug ./cmd/main.go -- --kubeconfig=$HOME/.config/kubeconfig
```

### Debug in Cluster

**Step 1: Enable verbose logging**:

Edit `config/manager/manager.yaml`:

```yaml
args:
  - --leader-elect
  - --zap-log-level=debug
```

**Step 2: Port-forward for debugging**:

```bash
kubectl port-forward -n helios-system deployment/helios-operator-controller-manager 8080:8080
```

### Common Issues

**Issue**: CRD not found

```bash
make install  # Reinstall CRD
```

**Issue**: Operator crashes

```bash
kubectl logs -n helios-system deployment/helios-operator-controller-manager --previous
```

**Issue**: Reconciliation not triggered

```bash
# Force reconciliation by annotation
kubectl annotate heliosapp <name> force-reconcile=$(date +%s)
```

## Code Guidelines

### Go Style

- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use `gofmt` and `goimports`
- Max line length: 120 characters

### Naming Conventions

- **Functions**: `CreateEventListener`, `UpdateStatus`
- **Variables**: `heliosApp`, `pipelineRun`
- **Constants**: `DefaultTimeout`, `MaxRetries`

### Error Handling

```go
// Good
if err := r.Create(ctx, obj); err != nil {
    return ctrl.Result{}, fmt.Errorf("failed to create object: %w", err)
}

// Bad
if err := r.Create(ctx, obj); err != nil {
    log.Error(err, "error")
    return ctrl.Result{}, err
}
```

### Logging

```go
// Use structured logging
log := log.FromContext(ctx)
log.Info("Creating Tekton Triggers", "heliosapp", heliosApp.Name)
log.Error(err, "Failed to create PipelineRun", "name", name)
```

### Comments

```go
// CreateEventListener creates a Tekton EventListener for the given HeliosApp.
// It returns an error if the EventListener already exists or creation fails.
func CreateEventListener(heliosApp *platformv1.HeliosApp) error {
    // Implementation
}
```

## CI/CD Pipeline

### GitHub Actions

`.github/workflows/ci.yaml`:

```yaml
name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v4
        with:
          go-version: "1.25"
      - run: make test
      - run: make test-e2e

  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: make docker-build
      - run: make docker-push
```

### Release Process

**Step 1: Tag version**:

```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

**Step 2: Build release image**:

```bash
make docker-build docker-push IMG=ghcr.io/your-org/helios-operator:v1.0.0
```

**Step 3: Generate release manifests**:

```bash
make build-installer IMG=ghcr.io/your-org/helios-operator:v1.0.0
```

## Makefile Targets

## Best Practices

1. **Set up alerts** for operator down & high error rates
2. **Monitor PipelineRun success rate** to detect infrastructure issues
3. **Track ArgoCD sync status** to ensure deployments work
4. **Aggregate logs** for easier debugging
5. **Create Grafana dashboards** for team visibility
6. **Set retention policies** for metrics & logs
7. **Regular reviews** of metrics to optimize

## See Also

- [Troubleshooting Guide](./TROUBLESHOOTING.md)
- [Architecture](./ARCHITECTURE.md)
- [API Reference](./API_REFERENCE.md)

| Target              | Description      |
| ------------------- | ---------------- |
| `make help`         | Show all targets |
| `make install`      | Install CRD      |
| `make uninstall`    | Remove CRD       |
| `make deploy`       | Deploy operator  |
| `make undeploy`     | Remove operator  |
| `make run`          | Run locally      |
| `make test`         | Run unit tests   |
| `make test-e2e`     | Run E2E tests    |
| `make build`        | Build binary     |
| `make docker-build` | Build image      |
| `make docker-push`  | Push image       |
| `make generate`     | Generate code    |
| `make manifests`    | Generate CRD     |
| `make fmt`          | Format code      |
| `make vet`          | Run go vet       |
| `make lint`         | Run linters      |

## Resources

- [Kubebuilder Book](https://book.kubebuilder.io/)
- [Operator SDK](https://sdk.operatorframework.io/)
- [Kubernetes Controller Best Practices](https://kubernetes.io/docs/concepts/architecture/controller/)
- [Go Testing](https://golang.org/doc/tutorial/add-a-test)
- [Ginkgo Framework](https://onsi.github.io/ginkgo/)

## Getting Help

- 💬 Slack: #helios-dev channel
- 📧 Mailing list: <dev@helios.io>
- 📝 Wiki: [Development Wiki](https://github.com/your-org/helios-operator/wiki)
