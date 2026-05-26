# Testing db-migrate Trigger Implementation

## Quick Test Summary

| Level | Test Type | Command | Time |
|-------|-----------|---------|------|
| **Unit** | Mapper + CUE rendering | `go test ./...` | ~30s |
| **Integration** | Local K8s cluster | `kubectl apply -f` | ~5m |
| **E2E** | Actual webhook + git push | Manual push | ~2m |

---

## 1️⃣ Unit Tests (Go)

### Test 1: Verify TriggerType is read from HeliosApp

Create or update test file: `apps/operator/internal/controller/tekton/mapper_test.go`

```go
package tekton

import (
	"testing"

	appv1alpha1 "github.com/helios-platform-team/helios-platform/apps/operator/api/v1alpha1"
)

func TestMapCRDToTektonInput_TriggerType(t *testing.T) {
	tests := []struct {
		name          string
		triggerType   string
		expectedTrigger string
	}{
		{
			name:        "Default trigger type is gitea-push",
			triggerType: "",
			expectedTrigger: "gitea-push",
		},
		{
			name:        "db-migrate trigger type is preserved",
			triggerType: "db-migrate",
			expectedTrigger: "db-migrate",
		},
		{
			name:        "Custom trigger type is passed through",
			triggerType: "github-push",
			expectedTrigger: "github-push",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &appv1alpha1.HeliosApp{
				Spec: appv1alpha1.HeliosAppSpec{
					GitRepo:       "https://gitea.local/user/repo",
					ImageRepo:     "myregistry/myapp",
					GitOpsRepo:    "https://gitea.local/user/gitops",
					GitOpsPath:    "./",
					TriggerType:   tt.triggerType,
				},
			}

			input := MapCRDToTektonInput(app)

			if input.TriggerType != tt.expectedTrigger {
				t.Errorf("Expected TriggerType=%s, got %s", tt.expectedTrigger, input.TriggerType)
			}
		})
	}
}

func TestMapCRDToTektonInput_PostgRESTDefaults(t *testing.T) {
	// Verify that PostgREST template sets triggerType to db-migrate
	app := &appv1alpha1.HeliosApp{
		Spec: appv1alpha1.HeliosAppSpec{
			GitRepo:     "https://gitea.local/user/my-api",
			ImageRepo:   "myregistry/my-api",
			GitOpsRepo:  "https://gitea.local/user/my-api-gitops",
			GitOpsPath:  "./",
			TriggerType: "db-migrate",
		},
	}

	input := MapCRDToTektonInput(app)

	if input.TriggerType != "db-migrate" {
		t.Errorf("PostgREST should have db-migrate trigger, got %s", input.TriggerType)
	}
}
```

Run test:
```bash
cd apps/operator
go test ./internal/controller/tekton/... -v -run TestMapCRDToTektonInput_TriggerType
```

### Test 2: Update existing CUE test to include db-migrate

File: `apps/operator/internal/cue/tekton_test.go`

Update the `validTektonInput()` function OR create tests for db-migrate:

```go
// Add this test function
func validTektonInputDbMigrate() TektonInput {
	input := validTektonInput()
	input.TriggerType = "db-migrate"
	return input
}

func TestRenderTektonResources_DbMigrateTrigger(t *testing.T) {
	cuePath := getCuePath(t)
	renderer, err := NewTektonRenderer(cuePath)
	if err != nil {
		t.Fatalf("Failed to create TektonRenderer: %v", err)
	}

	input := validTektonInputDbMigrate()
	objects, err := renderer.RenderTektonResources(input)
	if err != nil {
		t.Fatalf("RenderTektonResources failed: %v", err)
	}

	// Verify db-migrate specific resources exist
	var hasDbMigrateBinding bool
	var hasDbMigrateTemplate bool
	var hasDbMigrateListener bool

	for _, obj := range objects {
		name := obj.GetName()
		kind := obj.GetKind()

		if kind == "TriggerBinding" && contains(name, "db-migrate") {
			hasDbMigrateBinding = true
		}
		if kind == "TriggerTemplate" && contains(name, "db-migrate") {
			hasDbMigrateTemplate = true
		}
		if kind == "EventListener" && contains(name, "db-migrate") {
			hasDbMigrateListener = true
		}
	}

	if !hasDbMigrateBinding {
		t.Error("Expected db-migrate TriggerBinding not found")
	}
	if !hasDbMigrateTemplate {
		t.Error("Expected db-migrate TriggerTemplate not found")
	}
	if !hasDbMigrateListener {
		t.Error("Expected db-migrate EventListener not found")
	}
}

func contains(s string, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && len(s) >= len(substr) && 
		(s == substr || (len(s) > len(substr) && findStringIndex(s, substr) >= 0))
}

func findStringIndex(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
```

Run test:
```bash
cd apps/operator
go test ./internal/cue/... -v -run TestRenderTektonResources_DbMigrateTrigger
```

### Test 3: Verify CEL filter in EventListener

```go
func TestEventListener_DbMigrate_CELFilter(t *testing.T) {
	cuePath := getCuePath(t)
	renderer, err := NewTektonRenderer(cuePath)
	if err != nil {
		t.Fatalf("Failed to create TektonRenderer: %v", err)
	}

	input := validTektonInputDbMigrate()
	objects, err := renderer.RenderTektonResources(input)
	if err != nil {
		t.Fatalf("RenderTektonResources failed: %v", err)
	}

	// Find EventListener
	var eventListener map[string]interface{}
	for _, obj := range objects {
		if obj.GetKind() == "EventListener" && contains(obj.GetName(), "db-migrate") {
			eventListener = obj.Object
			break
		}
	}

	if eventListener == nil {
		t.Fatal("EventListener not found")
	}

	// Verify CEL filter exists
	spec := eventListener["spec"].(map[string]interface{})
	triggers := spec["triggers"].([]interface{})
	if len(triggers) == 0 {
		t.Fatal("No triggers found in EventListener")
	}

	trigger := triggers[0].(map[string]interface{})
	interceptors := trigger["interceptors"].([]interface{})

	// Find CEL interceptor
	var hasCelFilter bool
	for _, interceptor := range interceptors {
		i := interceptor.(map[string]interface{})
		ref := i["ref"].(map[string]interface{})
		if ref["name"] == "cel" {
			hasCelFilter = true
			// Verify filter contains db/ check
			params := i["params"].([]interface{})
			if len(params) > 0 {
				param := params[0].(map[string]interface{})
				value := param["value"].(string)
				if !contains(value, "db/") {
					t.Errorf("CEL filter doesn't check for db/ path: %s", value)
				}
			}
		}
	}

	if !hasCelFilter {
		t.Error("CEL interceptor not found in EventListener")
	}
}
```

### Run all tests:

```bash
cd apps/operator

# Run mapper tests
go test ./internal/controller/tekton/... -v

# Run CUE rendering tests
go test ./internal/cue/... -v

# Run both with coverage
go test ./... -v -cover
```

---

## 2️⃣ Integration Tests (Local K8s)

### Setup local cluster:

```bash
# Create test namespace
kubectl create namespace test-migrations
kubectl label namespace test-migrations dev=true

# Install required CRDs and dependencies
kubectl apply -f docs/deployment/
```

### Test 1: Deploy HeliosApp with db-migrate trigger

Create test file: `/tmp/test-postgrest-app.yaml`

```yaml
apiVersion: helios.io/v1alpha1
kind: HeliosApp
metadata:
  name: test-postgrest-api
  namespace: test-migrations
spec:
  owner: test-team
  description: "Test PostgREST API with db-migrate trigger"
  
  # ← Critical: Set triggerType to db-migrate
  triggerType: db-migrate
  
  gitRepo: https://gitea.local/test/my-api
  imageRepo: test-registry/my-api
  gitopsRepo: https://gitea.local/test/my-api-gitops
  gitopsPath: ./
  
  webhookDomain: webhook.test.local
  webhookSecret: test-webhook-secret
  
  replicas: 1
  port: 3000
  
  components:
    - name: api
      type: web-service
      properties:
        image: myregistry/my-api:latest
        port: 3000
      traits:
        - type: database
          properties:
            dbType: postgres
            dbName: test_db
            version: 15
        - type: service
          properties:
            port: 3000
```

Deploy:
```bash
kubectl apply -f /tmp/test-postgrest-app.yaml

# Verify HeliosApp creation
kubectl get heliosapp -n test-migrations
kubectl describe heliosapp test-postgrest-api -n test-migrations
```

### Test 2: Verify Tekton resources were created

```bash
# List all Tekton resources created by operator
kubectl get eventlisteners -n test-migrations
kubectl get triggerbindings -n test-migrations
kubectl get triggertemplates -n test-migrations
kubectl get tasks -n test-migrations
kubectl get pipelines -n test-migrations

# Get details of EventListener
kubectl describe eventlistener test-postgrest-api-db-migrate-listener -n test-migrations

# Check EventListener has correct trigger
kubectl get eventlistener test-postgrest-api-db-migrate-listener -n test-migrations -o json | \
  jq '.spec.triggers[0]'
```

### Test 3: Verify CEL filter configuration

```bash
# Get full EventListener spec to check interceptors
kubectl get eventlistener test-postgrest-api-db-migrate-listener -n test-migrations -o json | \
  jq '.spec.triggers[0].interceptors'

# Should see output like:
# [
#   {
#     "params": [
#       {"name": "secretRef", "value": {...}},
#       {"name": "eventTypes", "value": ["push"]}
#     ],
#     "ref": {"kind": "ClusterInterceptor", "name": "github"}
#   },
#   {
#     "params": [
#       {"name": "filter", "value": "has(body.commits) && ..."}
#     ],
#     "ref": {"kind": "ClusterInterceptor", "name": "cel"}
#   }
# ]
```

### Test 4: Simulate webhook (manual PipelineRun)

Since actual webhook requires Git setup, manually trigger migration:

```bash
# Create manual PipelineRun to simulate webhook
cat > /tmp/test-migration-run.yaml << 'EOF'
apiVersion: tekton.dev/v1beta1
kind: PipelineRun
metadata:
  name: test-db-migrate-run-001
  namespace: test-migrations
spec:
  pipelineRef:
    name: db-migrate
  params:
    - name: app-repo-url
      value: https://gitea.local/test/my-api
    - name: app-repo-revision
      value: main
    - name: database-url
      valueFrom:
        secretKeyRef:
          name: test-postgrest-api-db
          key: DATABASE_URL
    - name: migration-source
      value: db/migrations
    - name: namespace
      value: test-migrations
  workspaces:
    - name: source
      volumeClaimTemplate:
        spec:
          accessModes:
            - ReadWriteOnce
          resources:
            requests:
              storage: 1Gi
EOF

kubectl apply -f /tmp/test-migration-run.yaml

# Monitor execution
kubectl get pipelinerun -n test-migrations -w
kubectl describe pipelinerun test-db-migrate-run-001 -n test-migrations
kubectl logs test-db-migrate-run-001-* -n test-migrations --tail=50
```

### Test 5: Compare with gitea-push trigger

```bash
# Create second HeliosApp with gitea-push trigger for comparison
cat > /tmp/test-gitea-push-app.yaml << 'EOF'
apiVersion: helios.io/v1alpha1
kind: HeliosApp
metadata:
  name: test-standard-app
  namespace: test-migrations
spec:
  triggerType: gitea-push  # ← Different trigger
  gitRepo: https://gitea.local/test/my-service
  imageRepo: test-registry/my-service
  gitopsRepo: https://gitea.local/test/my-service-gitops
  gitopsPath: ./
  
  components:
    - name: service
      type: web-service
      properties:
        image: myregistry/my-service:latest
        port: 8080
EOF

kubectl apply -f /tmp/test-gitea-push-app.yaml

# Compare EventListeners
kubectl get eventlisteners -n test-migrations

# db-migrate should have CEL filter for db/
# gitea-push should NOT have CEL filter
echo "=== db-migrate listener (should have CEL) ==="
kubectl get eventlistener test-postgrest-api-db-migrate-listener -n test-migrations -o json | \
  jq '.spec.triggers[0].interceptors | map(.ref.name)'

echo "=== gitea-push listener (should NOT have CEL) ==="
kubectl get eventlistener test-standard-app-listener -n test-migrations -o json | \
  jq '.spec.triggers[0].interceptors | map(.ref.name)'
```

---

## 3️⃣ E2E Test (Actual Webhook)

### Setup real repositories and test webhook

#### Step 1: Prepare test repos

```bash
# Create source repo with migrations
git clone https://gitea.local/test/my-api
cd my-api

# Create migration files
mkdir -p db/migrations
cat > db/migrations/000001_initial.up.sql << 'EOF'
CREATE TABLE api.test (id SERIAL PRIMARY KEY);
NOTIFY pgrst, 'reload schema';
EOF

cat > db/migrations/000001_initial.down.sql << 'EOF'
DROP TABLE IF EXISTS api.test;
NOTIFY pgrst, 'reload schema';
EOF

git add .
git commit -m "Add test migrations"
git push origin main
```

#### Step 2: Deploy HeliosApp

```bash
cat > /tmp/postgrest-app.yaml << 'EOF'
apiVersion: helios.io/v1alpha1
kind: HeliosApp
metadata:
  name: my-api
  namespace: default
spec:
  triggerType: db-migrate
  gitRepo: https://gitea.local/test/my-api
  imageRepo: myregistry/my-api
  gitopsRepo: https://gitea.local/test/my-api-gitops
  gitopsPath: ./
  
  webhookDomain: webhook.yourdomain.com
  webhookSecret: my-webhook-secret
  
  components:
    - name: api
      type: web-service
      properties:
        image: myregistry/my-api:latest
        port: 3000
      traits:
        - type: database
          properties:
            dbType: postgres
            dbName: my_custom_db
EOF

kubectl apply -f /tmp/postgrest-app.yaml
```

#### Step 3: Configure webhook in Git

In Gitea UI:
- Go to my-api repo → Settings → Webhooks
- Should see webhook auto-created by operator
- URL: `http://el-my-api-db-migrate-listener.default.svc.cluster.local:8080`
- Events: **push**

#### Step 4: Test by pushing migrations

```bash
cd my-api

# Create new migration
cat > db/migrations/000002_add_users.up.sql << 'EOF'
CREATE TABLE api.users (
  id SERIAL PRIMARY KEY,
  email TEXT NOT NULL
);
NOTIFY pgrst, 'reload schema';
EOF

cat > db/migrations/000002_add_users.down.sql << 'EOF'
DROP TABLE IF EXISTS api.users;
NOTIFY pgrst, 'reload schema';
EOF

# Push - should trigger db-migrate pipeline
git add db/migrations/
git commit -m "Add users table migration"
git push origin main

# Check pipeline triggered
kubectl get pipelineruns -n default -w

# View logs
kubectl logs <pipelinerun-name>-run-migrations-* -n default
```

#### Step 5: Test CEL filter (negative case)

```bash
# Push non-db changes - should NOT trigger
echo "some code" >> src/main.rs
git add src/
git commit -m "Update app code"
git push origin main

# Check: NO new PipelineRun should be created
kubectl get pipelineruns -n default | tail -1
# Should not show new entry for this push
```

---

## 4️⃣ Cleanup

```bash
# Delete test namespace
kubectl delete namespace test-migrations

# Delete test files
rm -f /tmp/test-*.yaml

# Delete test repos
rm -rf ~/test-repos/
```

---

## Monitoring & Debugging

### Check operator logs

```bash
kubectl logs -f deployment/helios-operator -n helios-system
```

### Check EventListener logs

```bash
kubectl logs -f deployment/el-my-api-db-migrate-listener -n default
```

### Check webhook deliveries (Gitea UI)

- Settings → Webhooks → Click webhook
- Recent Deliveries tab
- Look for:
  - ✅ Green = successful
  - ❌ Red = failed
  - Check request/response body

### Verify database migrations

```bash
kubectl exec -it postgres-0 -- \
  psql -U postgres -d my_custom_db -c \
  "SELECT * FROM schema_migrations;"
```

---

## Summary Checklist

- [ ] Unit tests pass (`go test ./...`)
- [ ] Mapper correctly reads TriggerType from HeliosApp
- [ ] CUE renderer correctly switches between triggers
- [ ] db-migrate EventListener created with CEL filter
- [ ] gitea-push EventListener created without CEL filter
- [ ] Manual PipelineRun executes successfully
- [ ] Webhook deliveries show in Git UI
- [ ] Push to db/** triggers migration
- [ ] Push to other paths does NOT trigger migration
- [ ] Migration successfully applies schema changes
- [ ] `NOTIFY pgrst` executes and reloads schema
