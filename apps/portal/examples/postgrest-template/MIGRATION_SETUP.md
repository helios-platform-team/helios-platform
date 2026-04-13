# PostgREST Template: Database Migration Pipeline

This guide explains how the database migration pipeline works end-to-end and how to use it with your PostgREST API.

## Overview

The **db-migrate pipeline** automatically runs database migrations and reloads the PostgREST schema cache when changes are committed to the `db/migrations/` directory. This ensures your API immediately reflects the latest database schema without rebuilding container images.

### Pipeline Steps

```
Git Commit (to db/migrations/) 
    ↓
EventListener (filters db/** changes)
    ↓
PipelineRun (db-migrate)
    ├─ Clone Repository
    ├─ Run golang-migrate
    └─ Trigger Schema Reload (NOTIFY)
    ↓
PostgREST API (schema updated, no restart needed)
```

---

## Architecture

### 1. Migration Tool: golang-migrate

**Why golang-migrate?**
- ✅ Clean separation of up/down migrations
- ✅ Works in containers (no Go installation needed)
- ✅ Tracks migration status in database schema_migrations table
- ✅ Supports multiple databases (PostgreSQL, MySQL, etc.)
- ✅ Idempotent (safe to re-run)

**Used Image:** `migrate/migrate:v4.17.0`

**Key Features:**
- Version-based migrations (000001_, 000002_, etc.)
- Up/Down SQL files for each version
- Automatic transaction handling per migration
- Database state tracking to prevent duplicate runs

**Reference:** [golang-migrate Documentation](https://github.com/golang-migrate/migrate/blob/master/database/postgres/TUTORIAL.md)

### 2. Schema Reload: PostgreSQL NOTIFY

**Standard Mechanism:**
```sql
NOTIFY pgrst, 'reload schema';
```

**Why NOTIFY instead of Kubernetes rollout restart?**
- ✅ **Zero Downtime:** API stays running, schema reloaded in milliseconds
- ✅ **Cleaner:** No pod restarts, no session disruption
- ✅ **Scalable:** Works with multiple replicas without rollout waits
- ✅ **PostgREST Designed For:** PostgREST specifically watches for this NOTIFY event
- ✅ **Reliable:** PostgreSQL guarantees event delivery to all connected clients

**How PostgREST Listens:**
PostgREST automatically listens for the `pgrst` channel. When it receives `'reload schema'`, it:
1. Introspects the database schema again
2. Rebuilds its internal API definition
3. Applies changes immediately to subsequent requests

### 3. Git Trigger: CEL Filter

**Trigger Configuration:**
- **File:** `content/gitops/triggers.yaml` (generated from CUE)
- **Filter:** Only fires when commits modify `db/**` path
- **Uses:** CEL (Common Expression Language) interceptor
- **Ignores:** Changes to other directories (code, docs, etc.)

**Example Filter Logic:**
```
has(body.commits) && 
body.commits.filter(c, has(c.modified) && c.modified.exists(m, m.startsWith('db/'))).size() > 0
```

This ensures the migration pipeline:
- Runs automatically on `db/migrations/` changes
- Ignores code changes, reducing noise
- Stays focused on its mission (migrations)

---

## How to Add a New Migration

### Step 1: Create Migration Files

Migrations go in `db/migrations/` with the naming convention: `NNNNNN_description.{up,down}.sql`

```bash
# Create two files in db/migrations/

# File: db/migrations/000002_add_users_table.up.sql
-- Create users table
CREATE TABLE IF NOT EXISTS api.users (
  id SERIAL PRIMARY KEY,
  email TEXT UNIQUE NOT NULL,
  name TEXT NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
GRANT SELECT, INSERT, UPDATE, DELETE ON api.users TO authenticated;


# File: db/migrations/000002_add_users_table.down.sql
-- Rollback users table
DROP TABLE IF EXISTS api.users CASCADE;
```

### Step 2: Commit Your Changes

```bash
git add db/migrations/000002_add_users_table.*
git commit -m "feat: add users table"
git push origin main
```

### Step 3: Pipeline Automatically Runs

1. **Webhook Fires:** Gitea sends webhook to EventListener
2. **Filter Matches:** EventListener sees `db/migrations/` in changed files
3. **PipelineRun Created:** Tekton schedules the `db-migrate` pipeline
4. **Migration Executes:**
   - Task 1: Clone the repository
   - Task 2: Run `golang-migrate up` on `db/migrations/`
   - Task 3: Execute `NOTIFY pgrst, 'reload schema';`
5. **Done:** API automatically serves the new schema

### Step 4: Verify

```bash
# Check PostgREST API now includes new endpoints
curl http://your-postgrest-api/users

# Check the schema_migrations table
psql $DATABASE_URL -c "SELECT * FROM schema_migrations;"
```

---

## Security: Secrets & Networking

### Database Access in Pipeline

The migration pipeline needs access to the database. This is handled via:

**1. Kubernetes Secret Injection:**
- Secret Name: `{app-name}-db` (created by Helios Operator)
- Contains: `DB_USER`, `DB_PASS`, `DB_HOST`, `DB_PORT`, `PGRST_DB_URI`
- Mounted to migration task pod

**2. Volume Mount:**
```yaml
volumes:
  - name: db-credentials
    secret:
      secretName: myapp-db

volumeMounts:
  - name: db-credentials
    mountPath: /etc/db-credentials
    readOnly: true
```

**3. Environment Variable:**
```bash
DATABASE_URL=postgres://user:pass@host:5432/dbname
```

**Network Access:**
- Database must be reachable from Tekton cluster
- Typically run in same Kubernetes cluster (internal DNS)
- Operator provision prevents network issues

### Least Privilege

The task uses the database user created by the Operator:
- Username: Randomly generated (8 chars)
- Password: Randomly generated (32 chars)
- Permissions: Only owns the application database
- No superuser or dangerous privileges

---

## CUE Pipeline Structure

### 1. Pipeline Definition

**File:** `cue/definitions/tekton/pipelines/db-migrate.cue`

```cue
#PipelineRegistry: "db-migrate": {
  name: "db-migrate"
  description: "Database migration pipeline for PostgREST"
  config: {
    params: [
      {name: "app-repo-url", type: "string"},
      {name: "app-repo-revision", type: "string"},
      {name: "database-url", type: "string"},
      {name: "migration-source", type: "string", default: "db/migrations"},
    ]
    tasks: [
      {name: "clone-repo", taskRef: {name: "git-clone"}, ...},
      {name: "run-migrations", taskRef: {name: "db-migrate"}, ...},
      {name: "reload-postgrest", taskRef: {name: "postgrest-reload"}, ...},
    ]
  }
}
```

### 2. Task Registry

**File:** `cue/definitions/tekton/tasks/registry.cue`

```cue
#TaskRegistry: {
  "git-clone": #GitClone
  "db-migrate": #DBMigrate
  "postgrest-reload": #PostgRESTReload
}
```

### 3. Trigger Registry

**File:** `cue/definitions/tekton/triggers/registry.cue`

```cue
#TriggerRegistry: {
  "db-migrate": #DatabaseMigrationTriggerBundle
}
```

All pieces are registered in CUE and automatically rendered as Kubernetes resources.

---

## HeliosApp Configuration

Your template includes automatic db-migrate trigger:

```yaml
apiVersion: helios.io/v1alpha1
kind: HeliosApp
metadata:
  name: my-postgrest-api
spec:
  # This line enables the db-migrate trigger
  triggerType: db-migrate
  
  # Operator will provision database with these traits
  components:
    - name: api
      traits:
        - type: database
          properties:
            dbType: postgres
            dbName: my-api-db
            port: 5432
```

---

## Running Migrations Manually

If you need to run migrations outside the automated pipeline:

### Option 1: Via kubectl (pod exec)

```bash
# Get the PostgreSQL pod name
kubectl get pods -n default -l app=my-api-db

# Exec into the pod and run migrations manually
kubectl run -it --rm migration-job \
  --image=migrate/migrate:v4.17.0 \
  --restart=Never \
  -- migrate -path /migrations -database "$DATABASE_URL" up
```

### Option 2: Trigger PipelineRun Manually

```bash
kubectl create -f - <<EOF
apiVersion: tekton.dev/v1beta1
kind: PipelineRun
metadata:
  name: manual-migration-run
  namespace: default
spec:
  pipelineRef:
    name: db-migrate
  params:
    - name: app-repo-url
      value: https://github.com/myorg/myrepo.git
    - name: app-repo-revision
      value: main
    - name: database-url
      value: postgres://user:pass@db-service:5432/mydb
    - name: migration-source
      value: db/migrations
  workspaces:
    - name: source
      volumeClaimTemplate:
        spec:
          accessModes: ["ReadWriteOnce"]
          resources:
            requests:
              storage: 1Gi
EOF
```

---

## Troubleshooting

### Problem: Migrations Not Triggering

**Check 1: Verify EventListener is running**
```bash
kubectl get all -l app={appname}-db-migrate-listener
kubectl logs -l app={appname}-db-migrate-listener
```

**Check 2: Verify webhook is set up**
```bash
# In Gitea: Settings → Webhooks
# Should see a webhook to the EventListener service
# Should show successful deliveries on recent commits
```

**Check 3: Verify CEL filter matches**
```bash
# Only commits with db/** changes should match
# Check git log to ensure db/migrations/ files were modified
git log --name-status | grep "db/"
```

### Problem: Migrations Executing but Schema Not Updating

**Check PostgREST Logs:**
```bash
kubectl logs -l app=my-api
# Look for: "reloading schema" or schema update messages
```

**Verify NOTIFY Was Successful:**
```bash
kubectl exec -it pod/db-service-0 -- psql postgres://user:pass@localhost/dbname
postgres=# SELECT * FROM pg_stat_get_activity();
postgres=# LISTEN pgrst;
postgres=# NOTIFY pgrst, 'reload schema';
```

### Problem: golang-migrate Fails

**Check Migration Files:**
```bash
# Ensure files follow naming convention
# NNNNNN_description.up.sql
# NNNNNN_description.down.sql

# No spaces or special chars (except underscore)
```

**Check SQL Syntax:**
```bash
# Validate migration SQL locally
psql $DATABASE_URL < db/migrations/000002_add_users_table.up.sql
```

---

## Advanced: Custom Migration Workflows

### A. Pre-migration Backup

If you need to backup before migrations:

```cue
// In db-migrate pipeline, add task after clone-repo
{
  name: "backup-database"
  taskRef: {name: "postgres-backup"}
  runAfter: ["clone-repo"]
  params: [
    {name: "database-url", value: "$(params.database-url)"},
    {name: "backup-destination", value: "s3://my-backups/"},
  ]
}
```

### B. Post-migration Testing

Add a SQL validation step:

```sql
-- db/migrations/000002_add_users_table.up.sql

CREATE TABLE api.users (
  id SERIAL PRIMARY KEY,
  email TEXT UNIQUE NOT NULL
);

-- Validate schema
DO $$ BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.tables 
    WHERE table_schema = 'api' AND table_name = 'users'
  ) THEN
    RAISE EXCEPTION 'Users table creation failed';
  END IF;
END $$;
```

### C. Conditional Migrations

```sql
CREATE TABLE IF NOT EXISTS api.users (
  id SERIAL PRIMARY KEY,
  email TEXT UNIQUE NOT NULL
);

-- Safe: no error if table already exists
```

---

## Best Practices

1. **Test Locally First**
   ```bash
   docker run -it --rm \
     -e DATABASE_URL=postgres://user:pass@db:5432/testdb \
     -v $(pwd)/db/migrations:/migrations \
     migrate/migrate:v4.17.0 \
     -path /migrations -database $DATABASE_URL up
   ```

2. **Keep Migrations Small**
   - One logical change per migration
   - Easier to rollback if needed

3. **Always Provide Down Migrations**
   - Test rollbacks: `migrate ... down`
   - Safer deployments with rollback capability

4. **Lock Critical Rows**
   ```sql
   -- In migrations, use FOR UPDATE when needed
   BEGIN;
   LOCK TABLE api.users IN EXCLUSIVE MODE;
   ALTER TABLE api.users ADD COLUMN status TEXT;
   COMMIT;
   ```

5. **Monitor Schema Drift**
   ```bash
   # Check schema_migrations table regularly
   psql $DATABASE_URL -c "SELECT version, dirty FROM schema_migrations ORDER BY version DESC LIMIT 5;"
   ```

---

## FAQ

**Q: Can I run the code pipeline and db-migrate at the same time?**
A: Yes! The db-migrate trigger only fires on `db/**` changes. Normal build pipeline fires on code changes.

**Q: What happens if a migration fails?**
A: golang-migrate marks it as _dirty_ and won't proceed. You must fix and re-run (or manually rollback).

**Q: Can I skip a migration version?**
A: Not recommended. golang-migrate expects sequential versions.

**Q: How often does PostgREST check for schema changes?**
A: PostgREST listens constantly. Schema reload is immediate upon NOTIFY.

**Q: What if the database is down during migration?**
A: Pipeline fails. Fix database connectivity, then re-push (or manually trigger PipelineRun).

---

## Related Documentation

- [golang-migrate Tutorial](https://github.com/golang-migrate/migrate/blob/master/database/postgres/TUTORIAL.md)
- [PostgREST Schema Introspection](https://postgrest.org/en/stable/schema_structure.html)
- [Tekton Pipelines Guide](https://tekton.dev/docs/pipelines/)
- [PostgreSQL NOTIFY/LISTEN](https://www.postgresql.org/docs/current/sql-notify.html)
