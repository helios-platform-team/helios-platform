# PostGraphile Template: Database Migration Pipeline

This specification guide details the inner workings of the underlying zero-downtime schema evolution engine engineered for PostGraphile components on the Helios Platform.

## Architecture & Communication Flow

```text
Git Commit (to db/migrations/)
            ↓
 Gitea Webhook Interceptor
            ↓ (CEL Filters paths matching 'db/**')
  Tekton Trigger Engine
            ↓
  PipelineRun: db-migrate
            ├── Step 1: Clone Workspace State
            ├── Step 2: Run golang-migrate up
            └── Step 3: Schema Introspection Update
            ↓
   PostGraphile Engine (Schema reloads in milliseconds with ZERO pod restarts)
```

---

## Engine Breakdown

### 1. Version State Manager: `golang-migrate`

The system utilizes `migrate/migrate:v4.17.0` to handle database schema drift orchestration.

- **Idempotency**: It verifies target database state via a dedicated tracker table (`public.schema_migrations`) ensuring assertions apply sequentially without structural collision.
- **Formatting Structure**: All migrations require exact matching prefix padding pairs: `NNNNNN_description.{up|down}.sql`.

### 2. Auto-Reload System: Dynamic `--watch` Engine

Unlike legacy setups requiring full cluster application restarts to track schema adjustments, PostGraphile utilizes a robust internal reflection network combined with connection level surveillance.

- **Live Cache Refresh**: Our configuration initializes PostGraphile with the `watch: true` option inside `.postgraphilerc.js`.
- **The Workflow**: The moment the `db-migrate` pipeline executes your SQL statements against PostgreSQL, PostGraphile instantly catches the server schema generation changes, invalidates its internal cache, and morphs the GraphQL schema tree in milliseconds without dropping active user connections.

### 3. Path Interception: Common Expression Language (CEL) Filters

To protect resources, the infrastructure webhook relies on Tekton trigger interceptors evaluating with Common Expression Language (CEL) blocks.

- **Filtering Rule**: It extracts change vectors from incoming Gitea payloads, filtering events to block irrelevant codebase alterations (like updating text documents or application metadata) from incorrectly firing your database migrations.

---

## Resolving Operator Variable Conflicts

A standard limitation within the default Helios Operator Go reconciliation logic forces an injection of a hardcoded string layout directly into any container utilizing environment variables matching the literal tag `DATABASE_URL`.

To seamlessly integrate standard Kubernetes `valueFrom.secretKeyRef` patterns without crashing under legacy engine overwrites, this template isolates connection routing under a unique custom variable:

```javascript
// From .postgraphilerc.js
connection: process.env.POSTGRAPHILE_DB_URI
```

This structural bypass shields your PostGraphile deployment, allowing it to extract credentials via the operator's managed secret data keys safely.

---

## Verification & Tracking Checklist

### Monitoring Execution Progress

You can stream and track execution passes on the platform cluster namespace using the following commands:

```bash
# 1. Watch for active Pipeline runs matching your component
kubectl get pipelinerun -n ${{ values.namespace }} -w

# 2. Extract migration task processing output logs
kubectl logs -f -l tekton.dev/pipeline=db-migrate -c step-migrate -n ${{ values.namespace }}
```

### Inspecting Internal Migration State

To audit versioning history directly inside PostgreSQL, interface with the stateful tracker tracking table:

```bash
# Port forward to database service layer
kubectl port-forward svc/${{ values.name }}-db 5432:5432 -n ${{ values.namespace }}

# Run check query inside another workspace instance
psql postgres://postgres:password@localhost:5432/${{ values.name }}-db -c "SELECT * FROM schema_migrations;"
```

---

## Common Error Resolution Matrix

### 1. Problem: `schemas.map is not a function` Error

**Root Cause**: PostGraphile configuration expects an array list format to scan schema trees, but a raw, un-bracketed string object (`schema: "public"`) was processed.

**Remediation**: Open `.postgraphilerc.js` and wrap the configuration target explicitly in Array notation: `schema: ["public"]`.

### 2. Problem: Exit Code 127 / Command Not Found

**Root Cause**: Custom container build commands attempted calling a global binary string executable command using a raw shell entrypoint hook, but PostGraphile is locally embedded via node modules inside its runtime space.

**Remediation**: Use the default entrypoint sequence provided by the template and defer CLI execution to the cosmiconfig `.postgraphilerc.js` scanner wrapper layer.

### 3. Problem: Pipeline Run Status is Marked As Dirty

**Root Cause**: A database migration query script failed midway due to a syntax error or colliding schema constraints. `golang-migrate` locks down execution runs to guard against corruption.

**Remediation**: Connect to your database instance, repair your schema state to its preceding safe baseline point, force state clear via `UPDATE schema_migrations SET dirty = false;`, correct your SQL syntax errors inside Git, and re-push.