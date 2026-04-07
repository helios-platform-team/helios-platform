# PostgREST Template Testing Guide

## Testing Levels

- **Level 1 (5 min)**: Quick Validation — Verify template is registered
- **Level 2 (15 min)**: Local Testing — Start Backstage and test UI
- **Level 3 (30 min)**: End-to-End — Actually scaffold a service
- **Level 4 (60+ min)**: Full Deployment — Test with real Kubernetes cluster

---

## Level 1: Quick Validation (5 minutes)

Verify template files, configuration, and syntax without starting services.

### 1.1 Check Template Files Exist

```bash
cd /home/nghia/project/helios-platform

# Verify template structure
echo "=== Checking Template Files ==="
ls -lh apps/portal/examples/postgrest-template/template.yaml
ls -lh apps/portal/examples/postgrest-template/content/source/
ls -lh apps/portal/examples/postgrest-template/content/gitops/

# Expected output: 
# - template.yaml (182 lines)
# - 5 source files (catalog, dockerfile, config, readme, gitignore)
# - 5 gitops files (heliosapp, argocd, pipeline, triggers, kustomization)
```

### 1.2 Verify Configuration Registration

```bash
echo "=== Checking Backstage Configuration ==="

# Check if template is registered in app-config.yaml
grep -A 3 "postgrest-template" apps/portal/app-config.yaml

# Expected output:
# # PostgREST Template - Instant REST API over PostgreSQL
# - type: file
#   target: ../../examples/postgrest-template/template.yaml
#   rules:
#     - allow: [Template]
```

### 1.3 Run Validation Script

```bash
cd apps/portal/examples/postgrest-template

echo "=== Running Template Validation ==="
bash validate.sh

# Expected output:
# ✓ Template structure is valid
# ✓ YAML syntax is correct
# ✓ HeliosApp CRD correctly configured
# ✓ Tekton CI/CD pipeline defined
# ✓ PostgREST configuration present
# ✓ PGRST_DB_URI integration configured
```

### 1.4 Verify Operator Changes

```bash
echo "=== Checking Operator Integration ==="

# Check if PGRST_DB_URI is in injection code
grep "PGRST_DB_URI" apps/operator/internal/controller/database/injection.go && \
  echo "✓ PGRST_DB_URI found in injection.go" || \
  echo "✗ PGRST_DB_URI not found"

# Check if formatPostgresURI function exists
grep "func formatPostgresURI" apps/operator/internal/controller/database/resources.go && \
  echo "✓ formatPostgresURI function found" || \
  echo "✗ formatPostgresURI function not found"

# Check if URI tests exist
grep "TestFormatPostgresURI" apps/operator/internal/controller/database/resources_test.go && \
  echo "✓ URI tests found" || \
  echo "✗ URI tests not found"
```

### 1.5 Run Operator Tests

```bash
cd apps/operator

echo "=== Running Operator Tests ==="
go test ./internal/controller/database/... -v -count=1 2>&1 | tail -20

# Expected: All tests should PASS, specifically:
# - TestFormatPostgresURI (5 test cases)
# - TestGenerateDatabaseSecret
# - TestInjectDatabaseEnvVars
# - Total: 43/43 tests passing
```

✅ **Level 1 Complete** — If all checks pass, template is properly configured!

---

## Level 2: Local Testing (15 minutes)

Start Backstage Portal locally and verify template appears in UI.

### 2.1 Prerequisites

```bash
# Check Node.js and npm
node --version  # Should be v18+
npm --version   # Should be v9+

# Check Git
git --version

# Check Docker (optional, for building images)
docker --version
```

### 2.2 Install Dependencies

```bash
cd apps/portal

echo "=== Installing Backstage Dependencies ==="
npm install

# This may take 2-3 minutes on first run
# Expected: node_modules/ directory created with all dependencies
```

### 2.3 Start Backstage Development Server

```bash
echo "=== Starting Backstage Portal ==="
npm run start

# Expected output:
# [0] [22:45:00] Performing initial build
# [0] Backend started on port 7007
# [0] Frontend started on port 3000
# [0] Waiting for initial build
# [1] ✔ Compilation successful after 45 seconds
# [1] Listening on http://localhost:3000

# Once running, leave this terminal open
```

### 2.4 Access Backstage in Browser (New Terminal)

```bash
# Open http://localhost:3000 in your browser
# Or use curl:

curl -s http://localhost:3000 | head -20

# Expected: HTML content with "Backstage" title
```

### 2.5 Verify Template Appears in UI

```bash
# You should see:
# 1. Backstage Portal homepage at http://localhost:3000
# 2. Top menu with: Home | Docs | Create | Catalog
# 3. Click "Create" → List of available templates

# Check if PostgREST template appears:
curl -s http://localhost:7007/api/scaffolder/v2/templates | \
  grep -i postgrest

# Expected output should include:
# "name":"postgrest-template"
# "title":"PostgREST API Template"
```

### 2.6 Manual UI Check

**In Browser:**

1. Open http://localhost:3000
2. Click **Create** in the top menu
3. Should see templates listed:
   - ✅ Basic Template
   - ✅ Advanced Node.js Template
   - ✅ NestJS + Prisma Template
   - ✅ **PostgREST API Template** ← Find this one

4. Click on "PostgREST API Template"
5. Verify scaffolder form appears with sections:
   - Component Information (name, port, docker org)
   - PostgREST Configuration (schema, jwt, anon role)
   - Database Configuration
   - Repository & Webhook
   - Optional Extras

✅ **Level 2 Complete** — Template is discoverable and editable in UI!

---

## Level 3: End-to-End Testing (30 minutes)

Actually scaffold a complete PostgREST service using the template.

### Prerequisites for Level 3

You need these running:
- ✅ Backstage Portal (running from Level 2)
- ⚠️ Gitea (for Git repository operations)
- ⚠️ Kubernetes cluster (for deployment)
- ⚠️ Helios Operator (for database provisioning)

**Note:** If you don't have Kubernetes/Gitea running, you can still complete Levels 1-2 or do a dry-run (Level 3.1 below).

### 3.1 Dry-Run: Template Parameter Validation (No K8s needed)

```bash
cd apps/portal/examples/postgrest-template

cat > /tmp/test_params.yaml << 'EOF'
name: test-api
port: 3000
dockerOrg: myregistry
repoName: test-postgrest-api
apiSchema: public
jwtSecret: your-secret-key-minimum-32-chars-long
jwtRole: authenticated
anonRole: anon
databaseConfig:
  dbType: postgres
  dbName: test_api_db
owner: platform-team
EOF

echo "=== Validating Template Parameters ==="

python3 << 'PYTHON'
import yaml

with open('/tmp/test_params.yaml', 'r') as f:
    params = yaml.safe_load(f)

# Validate required fields
required = ['name', 'port', 'dockerOrg', 'repoName', 'apiSchema', 'jwtSecret', 'jwtRole', 'anonRole']
for field in required:
    if field in params:
        print(f"✓ {field}: {params[field]}")
    else:
        print(f"✗ Missing: {field}")

print("\n✓ All required parameters present!")
PYTHON
```

### 3.2 Full End-to-End (Requires Kubernetes + Gitea)

**Prerequisites:**
```bash
# Check Kubernetes connection
kubectl cluster-info
kubectl get nodes

# Check Gitea availability
curl -s http://localhost:3030/api/v1/version | jq .

# Check Helios Operator
kubectl get deployment -n helios
```

**via Backstage UI:**

1. **Access Template**
   - Open http://localhost:3000
   - Click **Create**
   - Select **PostgREST API Template**

2. **Fill in Parameters**

   **Component Information:**
   - Name: `test-api`
   - Port: `3000`
   - Docker Organization: `myregistry`
   - Repository Name: `test-postgrest-api`

   **PostgREST Configuration:**
   - API Schema: `public`
   - JWT Secret: `my-super-secret-key-that-is-at-least-32-chars-long`
   - JWT Role: `authenticated`
   - Anonymous Role: `anon`

   **Database Configuration:**
   - Database Type: `postgres`
   - Database Name: `test_api_db`

   **Repository & Webhook:**
   - Source Repository: Point to your Gitea instance
   - (e.g., `localhost:3030/myorg/test-postgrest-api`)

   **Optional Extras:**
   - ☑ Register component in catalog
   - ☑ Send Backstage notification

3. **Execute Scaffolding**
   - Review parameters
   - Click **Create**
   - Watch the progress (should complete in ~30 seconds)

4. **Verify Resources Created**

   ```bash
   # Check Git repositories created
   curl -H "Authorization: token $GITEA_TOKEN" \
     http://localhost:3030/api/v1/user/repos

   # Check Kubernetes resources
   kubectl get heliosapp
   kubectl describe heliosapp test-api

   # Check database provisioning
   kubectl get statefulset -l app=test-api-db
   kubectl get secret test-api-db-secret -o yaml

   # Verify PGRST_DB_URI
   kubectl get secret test-api-db-secret -n default \
     -o jsonpath='{.data.PGRST_DB_URI}' | base64 -d
   ```

5. **Test API Connectivity**

   ```bash
   # Wait for deployment to be ready
   kubectl wait --for=condition=ready pod \
     -l app=test-api --timeout=120s

   # Port forward to the service
   kubectl port-forward svc/test-api 3000:3000 &

   # Test health endpoint (should return PostgREST version info)
   curl -s http://localhost:3000/ | jq .

   # Expected output:
   # {
   #   "version": "12.2.0",
   #   "name": "PostgREST"
   # }
   ```

✅ **Level 3 Complete** — Template successfully scaffolds production-ready services!

---

## Level 4: Full Deployment Testing (60+ minutes)

Complete end-to-end testing with database schema, API usage, and monitoring.

### 4.1 Create Database Schema

```bash
# Connect to the provisioned PostgreSQL database
PGRST_DB_URI=$(kubectl get secret test-api-db-secret -n default \
  -o jsonpath='{.data.PGRST_DB_URI}' | base64 -d)

# Parse connection string
# Format: postgres://user:password@host:port/database

# Create test tables
psql "$PGRST_DB_URI" << 'SQL'
CREATE TABLE IF NOT EXISTS posts (
  id SERIAL PRIMARY KEY,
  title TEXT NOT NULL,
  body TEXT,
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
  id SERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  email TEXT UNIQUE NOT NULL
);

ALTER TABLE posts ENABLE ROW LEVEL SECURITY;

INSERT INTO posts (title, body) VALUES
  ('Getting Started', 'This is my first post'),
  ('PostgREST Benefits', 'Instant REST API from SQL');

INSERT INTO users (name, email) VALUES
  ('Alice', 'alice@example.com'),
  ('Bob', 'bob@example.com');

SELECT * FROM posts;
SQL
```

### 4.2 Test REST Endpoints

```bash
# Ensure port forward is active
kubectl port-forward svc/test-api 3000:3000 &

# Wait a moment
sleep 2

# Test CRUD operations
echo "=== Testing POST (Create) ==="
curl -X POST http://localhost:3000/posts \
  -H "Content-Type: application/json" \
  -d '{"title":"Testing Template","body":"This is a test"}'

echo -e "\n=== Testing GET (Read) ==="
curl -s http://localhost:3000/posts | jq .

echo -e "\n=== Testing GET with Filter ==="
curl -s 'http://localhost:3000/posts?title=like.*Getting*' | jq .

echo -e "\n=== Testing GET with Sort/Limit ==="
curl -s 'http://localhost:3000/posts?order=id.desc&limit=5' | jq .

echo -e "\n=== Testing PUT (Update) ==="
curl -X PUT http://localhost:3000/posts?id=eq.1 \
  -H "Content-Type: application/json" \
  -d '{"title":"Updated Title"}'

echo -e "\n=== Testing DELETE ==="
curl -X DELETE http://localhost:3000/posts?id=eq.99

echo -e "\n✓ All REST operations working!"
```

### 4.3 Test Authentication (JWT)

```bash
PGRST_DB_URI=$(kubectl get secret test-api-db-secret -n default \
  -o jsonpath='{.data.PGRST_DB_URI}' | base64 -d)

# Get JWT secret from PostgreSQL (if configured)
JWT_SECRET=$(psql "$PGRST_DB_URI" -tc "SELECT current_setting('app.jwt_secret', true);")

if [ -z "$JWT_SECRET" ]; then
  echo "JWT not configured in this deployment (optional)"
  echo "To enable JWT, set in postgrestrc.conf:"
  echo "  jwt-secret = 'your-secret-here'"
else
  echo "JWT Secret is set: $JWT_SECRET"
fi

# If JWT is configured, test with token:
# TOKEN="<your-jwt-token-here>"
# curl -H "Authorization: Bearer $TOKEN" http://localhost:3000/posts
```

### 4.4 Verify Operator Injection

```bash
echo "=== Verifying Operator Injected Environment Variables ==="

# Check pod environment variables
kubectl exec deployment/test-api -- env | grep -E "PGRST|DB_"

# Expected output:
# PGRST_DB_URI=postgres://user:pass@test-api-db:5432/test_api_db
# PGRST_DB_SCHEMA=public
# PGRST_DB_ANON_ROLE=anon
# PGRST_JWT_AUDIENCE=authenticated
# PGRST_MAX_ROWS=1000
# DB_HOST=test-api-db
# DB_USER=<generated-user>
# DB_PASS=<generated-password>
# DB_PORT=5432
```

### 4.5 Monitor Logs

```bash
echo "=== Monitoring PostgREST Logs ==="
kubectl logs deployment/test-api -f

echo -e "\n=== Monitoring Operator Logs ==="
kubectl logs deployment/helios-operator -n helios -f | grep -i postgrest

echo -e "\n=== Monitoring Database Logs ==="
kubectl logs statefulset/test-api-db -f | head -20
```

### 4.6 Test Persistence & Recovery

```bash
# Delete the PostgREST pod to verify recovery
echo "=== Testing Pod Recovery ==="
kubectl delete pod -l app=test-api

# Wait for recreation
kubectl get pods -l app=test-api -w

# Verify API still works after recovery
sleep 5
curl -s http://localhost:3000/posts | jq '.[] | .title'
```

### 4.7 Cleanup

```bash
echo "=== Cleanup Test Resources ==="

# Stop port forward
pkill -f "kubectl port-forward"

# Delete the scaffolded resources (optional)
kubectl delete heliosapp test-api
kubectl delete secret test-api-db-secret
kubectl delete statefulset test-api-db

# Verify deletion
kubectl get heliosapp
```

✅ **Level 4 Complete** — Full end-to-end testing validated!

---

## Testing Checklist

Use this checklist to verify each level:

### ✓ Level 1: Quick Validation
- [ ] Template files exist (13 files total)
- [ ] Configuration registered in app-config.yaml
- [ ] Validation script passes (7/7 checks)
- [ ] Operator tests pass (43/43)
- [ ] Operator integration confirmed (PGRST_DB_URI in code)

### ✓ Level 2: Local Testing
- [ ] Node.js v18+ installed
- [ ] npm dependencies installed
- [ ] Backstage starts without errors
- [ ] Template appears in **Create** menu
- [ ] Template form renders correctly (5 parameter sections)
- [ ] No console errors in browser

### ✓ Level 3: End-to-End Scaffolding
- [ ] Template parameters accepted
- [ ] Scaffolding completes successfully
- [ ] Git repositories created (source + gitops)
- [ ] HeliosApp CRD created
- [ ] Database StatefulSet provisioned
- [ ] PGRST_DB_URI secret created
- [ ] Pod logs show successful startup

### ✓ Level 4: Full Deployment
- [ ] Database tables created
- [ ] GET /posts returns data
- [ ] POST /posts creates new record
- [ ] PUT /posts updates record
- [ ] DELETE /posts removes record
- [ ] Filtering works (?title=like.*)
- [ ] Sorting works (?order=id.desc)
- [ ] Pagination works (?limit=5&offset=0)
- [ ] Pod recovery works (delete + recreate)
- [ ] All env vars properly injected
- [ ] Logs show no errors

---

## Quick Commands Reference

```bash
# Level 1: Validation
cd /home/nghia/project/helios-platform/apps/portal/examples/postgrest-template
bash validate.sh
cd ../../operator && go test ./internal/controller/database/... -v

# Level 2: Start Backstage
cd /home/nghia/project/helios-platform/apps/portal
npm install
npm run start
# Then open http://localhost:3000

# Level 3: Verify Scaffolding
kubectl get heliosapp
kubectl get deployment -l app=test-api
kubectl get secret test-api-db-secret -o yaml

# Level 4: Test API
kubectl port-forward svc/test-api 3000:3000
curl http://localhost:3000/posts | jq .
```

---

## Troubleshooting Tests

**Template not appearing in Backstage UI**
```bash
# Check configuration syntax
yaml-lint apps/portal/app-config.yaml

# Restart Backstage
npm run start  # Control-C to stop, then restart
```

**Scaffolding fails**
```bash
# Check Backstage backend logs
kubectl logs deployment/backstage-backend --tail=50

# Verify Gitea credentials
echo $GITEA_TOKEN
```

**Database not provisioning**
```bash
# Check operator logs
kubectl logs deployment/helios-operator -n helios -f | grep -i error

# Check HeliosApp status
kubectl describe heliosapp test-api
```

**PGRST_DB_URI not working**
```bash
# Decode and verify the URI
kubectl get secret test-api-db-secret -n default \
  -o jsonpath='{.data.PGRST_DB_URI}' | base64 -d

# Test connection directly
psql $(kubectl get secret test-api-db-secret -n default \
  -o jsonpath='{.data.PGRST_DB_URI}' | base64 -d) \
  -c "SELECT version();"
```

---

## CI/CD Integration

For automated testing in CI/CD pipelines:

```yaml
# .github/workflows/test.yml (example)
name: PostgREST Template Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Level 1 - Validation
        run: |
          cd apps/portal/examples/postgrest-template
          bash validate.sh
      
      - name: Level 1 - Operator Tests
        run: |
          cd apps/operator
          go test ./internal/controller/database/... -v
      
      - name: Level 2 - Backstage Setup
        run: |
          cd apps/portal
          npm ci
          npm run build
```

---

**Status:** Ready for testing! 🚀

Start with **Level 1** (5 min) to quickly verify everything is set up, then progress to higher levels as needed.
