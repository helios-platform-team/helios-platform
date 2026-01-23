# Helios Operator - Testing Guide

Comprehensive guide for testing the Helios Operator.

## Testing Overview

| Test Type         | Purpose                            | Duration | Command         |
| ----------------- | ---------------------------------- | -------- | --------------- |
| Unit Tests        | Test individual functions/methods  | ~30s     | `make test`     |
| Integration Tests | Test controller logic with envtest | ~2min    | `make test`     |
| E2E Tests         | Test complete GitOps workflow      | ~10min   | `make test-e2e` |
| Manual Tests      | Validate real-world scenarios      | varies   | see below       |

---

## Quick Start: Automated E2E Setup

### One-Command Setup

For a complete local e2e testing environment:

```bash
# Run the automated setup script
./scripts/setup-e2e-local.sh
```

This script will:

- ✅ Check prerequisites (Docker, kubectl, kind, Go)
- ✅ Create Kind cluster (`helios-operator-test-e2e`)
- ✅ Install Tekton Pipelines and Triggers
- ✅ Install ArgoCD
- ✅ Install Tekton resources (Pipeline, Tasks, ServiceAccount, PVC)
- ✅ Build and load operator image
- ✅ Install CRDs
- ✅ Deploy operator

**After setup completes**, test immediately:

```bash
# Apply sample HeliosApp
kubectl apply -f config/samples/heliosapp_v1_heliosapp.yaml

# Watch status
kubectl get heliosapp --watch
```

### Using Makefile (Alternative)

```bash
# Setup e2e environment (cluster + dependencies)
make setup-test-e2e

# Then build and deploy operator manually:
make install
make docker-build IMG=helios-operator:dev
kind load docker-image helios-operator:dev --name helios-operator-test-e2e
make deploy IMG=helios-operator:dev

# Test with sample
kubectl apply -f config/samples/heliosapp_v1_heliosapp.yaml

# Run full e2e test suite (setup + test + cleanup)
make test-e2e
```

---

## Unit Tests

### Running Unit Tests

```bash
# Run all unit tests
make test

# Run tests with coverage
go test ./... -coverprofile=coverage.out

# View coverage report
go tool cover -html=coverage.out

# Run specific package
go test ./internal/controller/ -v

# Run specific test
go test ./internal/controller/ -v -run TestHeliosAppReconciler
```

### Test Structure

```text
internal/controller/
├── heliosapp_controller.go       # Main controller logic
├── heliosapp_controller_test.go  # Controller unit tests
├── tekton_resources.go           # Tekton resource generators
└── suite_test.go                 # Test suite setup
```

### Example Unit Test

```go
var _ = Describe("HeliosApp Controller", func() {
    Context("When reconciling a HeliosApp", func() {
        It("Should create EventListener", func() {
            ctx := context.Background()

            heliosApp := &platformv1.HeliosApp{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "test-app",
                    Namespace: "default",
                },
                Spec: platformv1.HeliosAppSpec{
                    AppRepoUrl:      "https://github.com/example/app",
                    AppRepoRevision: "main",
                    ImageRepo:       "ghcr.io/example/app",
                    // ... other required fields
                },
            }

            Expect(k8sClient.Create(ctx, heliosApp)).To(Succeed())

            // Verify EventListener is created
            Eventually(func() error {
                el := &triggersv1beta1.EventListener{}
                return k8sClient.Get(ctx, types.NamespacedName{
                    Name:      heliosApp.Name + "-listener",
                    Namespace: heliosApp.Namespace,
                }, el)
            }, timeout, interval).Should(Succeed())
        })
    })
})
```

---

## E2E Tests

### Architecture

The e2e tests validate the complete GitOps workflow:

1. **Phase 1**: HeliosApp creation → Tekton Triggers (EventListener, TriggerBinding, TriggerTemplate)
2. **Phase 2**: Initial PipelineRun execution → Manifest generation
3. **Phase 3**: ArgoCD Application creation → Deployment
4. **Phase 4**: Status synchronization and monitoring

### Prerequisites

Before running e2e tests, ensure:

```bash
# Check if cluster exists
kubectl cluster-info

# Check if operator is deployed
kubectl get deployment -n helios-operator-system

# Check if Tekton is installed
kubectl get pods -n tekton-pipelines

# Check if ArgoCD is installed
kubectl get pods -n argocd
```

### Running E2E Tests

**Full e2e test suite** (setup + test + cleanup):

```bash
make test-e2e
```

This will:

1. Create Kind cluster and install dependencies
2. Install CRDs
3. Run e2e test suite
4. Cleanup (delete cluster)

**Run e2e tests manually** (more control):

```bash
# 1. Setup environment
make setup-test-e2e

# 2. Install CRDs
make install

# 3. Build and deploy operator
make docker-build IMG=helios-operator:dev
kind load docker-image helios-operator:dev --name helios-operator-test-e2e
make deploy IMG=helios-operator:dev

# 4. Run tests
cd test/e2e
go test -v -ginkgo.v

# 5. Cleanup when done
kind delete cluster --name helios-operator-test-e2e
```

**Run specific test**:

```bash
cd test/e2e
go test -v -ginkgo.focus="Should create EventListener"
```

### E2E Test Scenarios

The e2e tests cover these scenarios:

#### Test 1: Tekton Triggers Creation

**What it tests:**

- HeliosApp CR creation
- EventListener creation with correct name and spec
- TriggerBinding creation
- TriggerTemplate creation
- Initial PipelineRun triggering
- Status update to `ManifestGenerationInProgress`

**Verification:**

```bash
kubectl get eventlistener
# Expected: <heliosapp-name>-listener

kubectl get triggerbinding,triggertemplate

kubectl get heliosapp <name> -o jsonpath='{.status.phase}'
# Expected: ManifestGenerationInProgress
```

#### Test 2: PipelineRun Execution

**What it tests:**

- PipelineRun created with correct parameters
- Parameters match HeliosApp spec (app-repo-url, app-repo-revision, etc.)
- Workspaces correctly configured
- ServiceAccount reference

**Verification:**

```bash
kubectl get pipelinerun

kubectl get pipelinerun <name> -o yaml | grep -A 20 "spec:"
# Verify parameters and workspaces
```

#### Test 3: ArgoCD Application Creation

**What it tests:**

- ArgoCD Application creation after PipelineRun success
- Application spec matches GitOps configuration
- Status update to `DeployingWithArgoCD`

**Verification:**

```bash
kubectl get application -n argocd

kubectl get heliosapp <name> -o jsonpath='{.status.phase}'
# Expected: DeployingWithArgoCD
```

#### Test 4: Status Synchronization

**What it tests:**

- observedGeneration tracking
- Phase transitions
- Message updates
- Condition management

**Verification:**

```bash
kubectl get heliosapp <name> -o jsonpath='{.status}'
```

#### Test 5: Spec Updates

**What it tests:**

- Generation increment on spec changes
- New PipelineRun creation
- No duplicate EventListeners

**Verification:**

```bash
# Update spec
kubectl patch heliosapp <name> --type=merge -p '{"spec":{"appRepoRevision":"develop"}}'

# Check generation
kubectl get heliosapp <name> -o jsonpath='{.metadata.generation}'

# Check new PipelineRun
kubectl get pipelinerun --sort-by=.metadata.creationTimestamp
```

---

## Manual Testing

### Prerequisites

Ensure your local e2e environment is set up:

```bash
./scripts/setup-e2e-local.sh
```

### Scenario 1: Deploy Sample Application

```bash
# 1. Apply sample HeliosApp
kubectl apply -f config/samples/heliosapp_v1_heliosapp.yaml

# 2. Watch reconciliation
kubectl get heliosapp sample-heliosapp --watch

# 3. Check created resources
echo "Tekton Triggers:"
kubectl get eventlistener,triggerbinding,triggertemplate

echo "PipelineRuns:"
kubectl get pipelinerun

echo "ArgoCD Applications:"
kubectl get application -n argocd

# 4. Check detailed status
kubectl describe heliosapp sample-heliosapp

# 5. View operator logs
kubectl logs -n helios-operator-system \
  deployment/helios-operator-controller-manager \
  -c manager \
  --tail=50 \
  -f
```

### Scenario 2: Update Application Configuration

```bash
# 1. Update repository revision
kubectl patch heliosapp sample-heliosapp --type=merge \
  -p '{"spec":{"appRepoRevision":"develop"}}'

# 2. Verify new PipelineRun created
kubectl get pipelinerun --sort-by=.metadata.creationTimestamp

# 3. Check generation incremented
kubectl get heliosapp sample-heliosapp -o jsonpath='{.metadata.generation}'
echo ""

# 4. Watch status update
kubectl get heliosapp sample-heliosapp -o jsonpath='{.status.phase}'
echo ""
```

### Scenario 3: Test with Custom Values

```bash
# Create HeliosApp with custom configuration
cat <<EOF | kubectl apply -f -
apiVersion: platform.helios.io/v1
kind: HeliosApp
metadata:
  name: custom-app
  namespace: default
spec:
  appRepoUrl: https://github.com/your-org/your-app
  appRepoRevision: main
  imageRepo: ghcr.io/your-org/your-app
  gitopsRepoUrl: https://github.com/your-org/gitops-repo
  gitopsRepoBranch: main
  manifestPathInGitopsRepo: apps/custom-app
  pipelineName: from-code-to-cluster
  serviceAccount: pipeline-sa
  dockerHubUsername: your-username
  dockerHubToken: your-token
EOF

# Watch the deployment
kubectl get heliosapp custom-app --watch
```

### Scenario 4: Inspect Tekton Pipeline Logs

```bash
# Get latest PipelineRun
PR_NAME=$(kubectl get pipelinerun \
  --sort-by=.metadata.creationTimestamp \
  -o jsonpath='{.items[-1].metadata.name}')

echo "Watching PipelineRun: $PR_NAME"

# Watch logs (using tkn CLI if installed)
tkn pipelinerun logs $PR_NAME -f

# Or using kubectl
kubectl logs -l tekton.dev/pipelineRun=$PR_NAME --all-containers=true -f
```

### Scenario 5: Debug Failure Scenarios

```bash
# Check PipelineRun status
kubectl get pipelinerun -o wide

# Get failed PipelineRun details
PR_NAME=<failed-pipelinerun-name>
kubectl describe pipelinerun $PR_NAME

# Check TaskRun logs
kubectl get taskrun -l tekton.dev/pipelineRun=$PR_NAME
TR_NAME=<taskrun-name>
kubectl logs $TR_NAME

# Check HeliosApp status for error message
kubectl get heliosapp <name> -o jsonpath='{.status.message}'
echo ""
```

---

## Test Coverage

### Generate Coverage Reports

```bash
# Run tests with coverage
go test ./... -coverprofile=coverage.out -covermode=atomic

# Generate HTML report
go tool cover -html=coverage.out -o coverage.html

# View in browser
open coverage.html  # macOS
xdg-open coverage.html  # Linux

# View coverage summary
go tool cover -func=coverage.out
```

### Coverage Targets

| Package               | Target | Focus Areas                               |
| --------------------- | ------ | ----------------------------------------- |
| `internal/controller` | 80%    | Reconciliation logic, resource generation |
| `api/v1`              | 70%    | Type definitions, validation              |

---

## Troubleshooting Tests

### Common Issues

#### "No kind clusters found"

```bash
# Create cluster manually
./scripts/setup-e2e-local.sh

# Or use Makefile
make setup-test-e2e
```

#### "Tekton Pipelines not ready"

```bash
# Check Tekton status
kubectl get pods -n tekton-pipelines

# Wait for ready
kubectl wait --for=condition=ready pod \
  -l app=tekton-pipelines-controller \
  -n tekton-pipelines \
  --timeout=300s

# Reinstall if needed
kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
```

#### "CRD not found"

```bash
# Install CRDs
make install

# Verify
kubectl get crd heliosapps.platform.helios.io
```

#### "Operator not running"

```bash
# Check operator status
kubectl get pods -n helios-operator-system

# Check logs
kubectl logs -n helios-operator-system \
  deployment/helios-operator-controller-manager \
  -c manager \
  --tail=50

# Redeploy if needed
make deploy IMG=helios-operator:local
```

#### "Test timeout"

```bash
# E2E tests may timeout if cluster is slow
# Increase timeout in test code or ensure sufficient resources

# Check resource usage
kubectl top nodes
kubectl top pods -A
```

#### "Package not found" in unit tests

```bash
# Tidy dependencies
go mod tidy

# Download dependencies
go mod download

# Verify modules
go mod verify
```

#### "TestEnv not starting"

```bash
# Install testenv binaries
make setup-envtest

# Verify KUBEBUILDER_ASSETS
echo $KUBEBUILDER_ASSETS

# If not set, run:
export KUBEBUILDER_ASSETS="$(go run sigs.k8s.io/controller-runtime/tools/setup-envtest@latest use -p path)"
```

---

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Tests
on: [push, pull_request]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25"
      - name: Run unit tests
        run: make test
      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          file: ./coverage.out

  e2e-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25"
      - name: Install Kind
        run: |
          curl -Lo ./kind https://kind.sigs.k8s.io/dl/latest/kind-linux-amd64
          chmod +x ./kind
          sudo mv ./kind /usr/local/bin/kind
      - name: Run e2e tests
        run: make test-e2e
        timeout-minutes: 30
```

---

## Best Practices

### 1. Test Organization

```text
test/
├── e2e/                  # End-to-end tests
│   ├── e2e_test.go       # Test scenarios
│   └── e2e_suite_test.go # Ginkgo suite setup
├── fixtures/             # Test data
│   └── sample-heliosapp.yaml
└── utils/                # Test helpers
    └── utils.go          # Common utilities
```

### 2. Test Naming Conventions

```go
// Good - descriptive and specific
var _ = Describe("HeliosApp Controller", func() {
    Context("When creating a new HeliosApp", func() {
        It("Should create EventListener with correct name", func() {
            // test code
        })
    })

    Context("When updating HeliosApp spec", func() {
        It("Should trigger new PipelineRun", func() {
            // test code
        })
    })
})

// Bad - vague and unclear
var _ = Describe("Test", func() {
    It("works", func() {
        // test code
    })
})
```

### 3. Test Data Management

```go
// Use fixtures for complex resources
func loadHeliosAppFixture() *platformv1.HeliosApp {
    // Load from test/fixtures/sample-heliosapp.yaml
    // or construct programmatically
    return &platformv1.HeliosApp{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "test-app",
            Namespace: "default",
        },
        Spec: platformv1.HeliosAppSpec{
            AppRepoUrl:      "https://github.com/example/app",
            AppRepoRevision: "main",
            // ... other fields
        },
    }
}
```

### 4. Cleanup and Isolation

```go
var _ = Describe("HeliosApp Controller", func() {
    var (
        ctx       context.Context
        heliosApp *platformv1.HeliosApp
    )

    BeforeEach(func() {
        ctx = context.Background()
        heliosApp = loadHeliosAppFixture()
    })

    AfterEach(func() {
        // Cleanup resources
        Expect(k8sClient.Delete(ctx, heliosApp)).To(Succeed())
    })

    It("Should create resources", func() {
        // test code
    })
})
```

### 5. Assertion Best Practices

```go
// Good - use Eventually for async operations
Eventually(func() error {
    return k8sClient.Get(ctx, key, resource)
}, timeout, interval).Should(Succeed())

// Good - clear expectations
Expect(resource.Spec.Field).To(Equal("expected-value"))

// Bad - direct assertions on async operations
resource := &Resource{}
Expect(k8sClient.Get(ctx, key, resource)).To(Succeed())
// This may fail due to timing
```

---

## Quick Reference Commands

### Setup and Cleanup

```bash
# Full automated setup
./scripts/setup-e2e-local.sh

# Setup via Makefile
make setup-test-e2e

# Cleanup Kind cluster
kind delete cluster --name helios-operator-test-e2e
```

### Running Tests

```bash
# Unit tests
make test

# E2E tests (full - includes setup and cleanup)
make test-e2e
```

### Debugging

```bash
# Watch resources
kubectl get heliosapp --watch
kubectl get pipelinerun --watch
kubectl get application -n argocd --watch

# Check logs
kubectl logs -n helios-operator-system deployment/helios-operator-controller-manager -c manager -f

# Describe resources
kubectl describe heliosapp <name>
kubectl describe pipelinerun <name>
kubectl describe application -n argocd <name>

# Check status
kubectl get heliosapp <name> -o jsonpath='{.status}' | jq '.'
```

---

## Next Steps

- **[Setup Guide](./SETUP_GUIDE.md)** - Complete setup instructions
- **[Development Guide](./DEVELOPMENT.md)** - Development workflow
- **[Troubleshooting](./TROUBLESHOOTING.md)** - Common issues and solutions
- **[Architecture](./ARCHITECTURE.md)** - System architecture

---

**Need Help?**

- 📖 Full documentation: [docs/](.)
- 🐛 Report issues: [GitHub Issues](https://github.com/hoangphuc841/helios-operator/issues)
- 💬 Discussions: [GitHub Discussions](https://github.com/hoangphuc841/helios-operator/discussions)
