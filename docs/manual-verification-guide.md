# Manual Verification Guide — Automated Secret Injection (#39)

This guide walks through manually verifying the acceptance criteria for the
Automated Secret Injection feature on a local k3d cluster.

## Acceptance Criteria

1. **Secret Injection**: `kubectl describe pod <backend-pod>` shows database
   credentials (DB_HOST, DB_USER, DB_PASS) successfully mounted as environment
   variables via `secretKeyRef`.
2. **Database Connectivity**: A NestJS server connects to the provisioned
   PostgreSQL database using the injected env vars and responds to health checks.

## Prerequisites

- **Docker** running locally
- **k3d** (`v5+`)
- **kubectl** configured
- **kustomize** (or `kubectl kustomize`)

Install via mise (if used):

```bash
mise install k3d kubectl
```

---

## Step 1: Create a k3d Cluster

```bash
k3d cluster create helios-test --wait
```

Fix kubeconfig if needed (k3d may set the server to `host.docker.internal`):

```bash
# Get the port k3d assigned
PORT=$(kubectl config view -o jsonpath='{.clusters[?(@.name=="k3d-helios-test")].cluster.server}' | grep -oP ':\K\d+$')

# Point to 127.0.0.1 for local access
kubectl config set-cluster k3d-helios-test --server="https://127.0.0.1:${PORT}"
```

Verify:

```bash
kubectl get nodes
# Should show: k3d-helios-test-server-0   Ready
```

---

## Step 2: Build and Load the Operator Image

The operator needs CUE files bundled in the image. Use the test Dockerfile
(`apps/operator/Dockerfile.test`) which copies `cue/` into the image:

```bash
# From project root (helios-platform/)
docker build -f apps/operator/Dockerfile.test -t helios-operator:local .

# Load into k3d
k3d image import helios-operator:local -c helios-test
```

---

## Step 3: Deploy the Operator

### 3a. Install CRDs

```bash
kubectl apply -f apps/operator/config/crd/bases/app.helios.io_heliosapps.yaml
```

### 3b. Deploy operator via kustomize

The operator kustomization (`config/default/`) sets namespace `operator-system`
and adds prefix `operator-`. The manager image is already mapped to
`helios-operator:local` in `config/manager/kustomization.yaml`.

```bash
cd apps/operator

# Build kustomize output and patch imagePullPolicy for local images
kustomize build config/default \
  | sed 's/imagePullPolicy: Always/imagePullPolicy: Never/' \
  | kubectl apply -f -

cd ../..
```

Wait for the operator to be ready:

```bash
kubectl -n operator-system get pods -w
# Wait for: operator-controller-manager-xxx   2/2   Running
```

Check the operator logs to confirm CUE engine initialized:

```bash
kubectl -n operator-system logs deployment/operator-controller-manager -c manager | head -20
# Look for: "CUE engine initialized"
```

---

## Step 4: Build and Load the Test NestJS App

Create a minimal NestJS + Prisma test app. You can use the template source at
`apps/portal/examples/nestjs-prisma-template/content/source/` as a base, with
template variables replaced (`${{ values.* }}` -> concrete values).

### 4a. Create a test app directory

```bash
mkdir -p /tmp/nestjs-test-app/src/prisma
mkdir -p /tmp/nestjs-test-app/prisma/migrations/0001_init
```

### 4b. Key files

**package.json** — Note: use `npm install` in Dockerfile if no lockfile exists.
Set `start:migrate:prod` to run prisma migrations then start the app.

```json
{
  "name": "test-backend",
  "version": "0.0.1",
  "private": true,
  "scripts": {
    "build": "nest build",
    "start:prod": "node dist/main",
    "start:migrate:prod": "npx prisma migrate deploy && node dist/main"
  },
  "dependencies": {
    "@nestjs/common": "11.1.16",
    "@nestjs/core": "11.1.16",
    "@nestjs/platform-express": "11.1.16",
    "@prisma/adapter-pg": "7.5.0",
    "@prisma/client": "7.5.0",
    "pg": "8.20.0",
    "reflect-metadata": "0.2.2",
    "rxjs": "7.8.2"
  },
  "devDependencies": {
    "@nestjs/cli": "11.0.16",
    "@types/pg": "8.18.0",
    "prisma": "7.5.0",
    "ts-node": "10.9.2",
    "typescript": "5.9.3"
  }
}
```

**prisma.config.ts** — Prisma 7 requires this instead of `url` in schema:

```typescript
import { defineConfig } from 'prisma/config';

function buildDatabaseUrl(): string {
  const host = process.env.DB_HOST || 'localhost';
  const user = process.env.DB_USER || 'postgres';
  const pass = encodeURIComponent(process.env.DB_PASS || 'postgres');
  const name = process.env.DB_NAME || 'postgres';
  const port = process.env.DB_PORT || '5432';
  return `postgresql://${user}:${pass}@${host}:${port}/${name}?schema=public`;
}

export default defineConfig({
  schema: 'prisma/schema.prisma',
  migrations: { path: 'prisma/migrations' },
  datasource: { url: process.env.DATABASE_URL || buildDatabaseUrl() },
});
```

**prisma/schema.prisma** — No `url` in datasource (Prisma 7 breaking change):

```prisma
generator client {
  provider = "prisma-client"
}

datasource db {
  provider = "postgresql"
}

model User {
  id        Int      @id @default(autoincrement())
  email     String   @unique
  name      String?
  createdAt DateTime @default(now()) @map("created_at")
  updatedAt DateTime @updatedAt @map("updated_at")
  @@map("users")
}
```

**prisma/migrations/migration_lock.toml**:

```toml
# Prisma Migrate lockfile v1

provider = "postgresql"
```

**prisma/migrations/0001_init/migration.sql**:

```sql
CREATE TABLE "users" (
    "id" SERIAL NOT NULL,
    "email" TEXT NOT NULL,
    "name" TEXT,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,
    CONSTRAINT "users_pkey" PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX "users_email_key" ON "users"("email");
```

**src/prisma/prisma.service.ts** — Use `import * as pg` (not default import):

```typescript
import { Injectable, OnModuleInit, OnModuleDestroy } from '@nestjs/common';
import { PrismaClient } from '@prisma/client';
import { PrismaPg } from '@prisma/adapter-pg';
import * as pg from 'pg';

@Injectable()
export class PrismaService extends PrismaClient implements OnModuleInit, OnModuleDestroy {
  private pool: pg.Pool;

  constructor() {
    const pool = new pg.Pool({
      host: process.env.DB_HOST || 'localhost',
      user: process.env.DB_USER || 'postgres',
      password: process.env.DB_PASS || 'postgres',
      database: process.env.DB_NAME || 'postgres',
      port: parseInt(process.env.DB_PORT || '5432', 10),
    });
    const adapter = new PrismaPg(pool as any);
    super({ adapter });
    this.pool = pool;
  }

  async onModuleInit(): Promise<void> { await this.$connect(); }
  async onModuleDestroy(): Promise<void> { await this.$disconnect(); await this.pool.end(); }
}
```

**src/app.controller.ts** — Health endpoint:

```typescript
import { Controller, Get } from '@nestjs/common';

@Controller()
export class AppController {
  @Get('health')
  health() { return { status: 'ok' }; }
}
```

(Also create standard `src/main.ts`, `src/app.module.ts`, `src/prisma/prisma.module.ts`,
`tsconfig.json`, `tsconfig.build.json`, `nest-cli.json` — use the template source
as reference.)

### 4c. Dockerfile for test app

```dockerfile
FROM node:24-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY prisma ./prisma/
COPY prisma.config.ts ./
RUN npx prisma generate
COPY . .
RUN npm run build

FROM node:24-alpine AS production
WORKDIR /app
COPY package*.json ./
RUN npm install --omit=dev
COPY --from=builder /app/prisma ./prisma/
COPY --from=builder /app/prisma.config.ts ./
COPY --from=builder /app/node_modules/.prisma ./node_modules/.prisma/
COPY --from=builder /app/node_modules/@prisma ./node_modules/@prisma/
COPY --from=builder /app/dist ./dist/
EXPOSE 3000
CMD ["npm", "run", "start:migrate:prod"]
```

### 4d. Build and load

```bash
cd /tmp/nestjs-test-app
docker build -t test-backend:latest .
k3d image import test-backend:latest -c helios-test
cd -
```

---

## Step 5: Create Test Resources

### 5a. Backend Deployment (simulates ArgoCD output)

The operator injects env vars into an **existing** Deployment. In production,
ArgoCD creates this. For testing, create it manually:

```bash
cat <<'EOF' | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-backend
  namespace: default
  labels:
    app: test-backend
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test-backend
  template:
    metadata:
      labels:
        app: test-backend
    spec:
      containers:
        - name: test-backend
          image: test-backend:latest
          imagePullPolicy: Never
          ports:
            - containerPort: 3000
          env:
            - name: PORT
              value: "3000"
            - name: DB_NAME
              value: "test_backend_db"
            - name: DB_PORT
              value: "5432"
EOF
```

### 5b. HeliosApp Custom Resource

This triggers the operator's reconcile loop:

```bash
cat <<'EOF' | kubectl apply -f -
apiVersion: app.helios.io/v1alpha1
kind: HeliosApp
metadata:
  name: test-app
  namespace: default
spec:
  owner: test-team
  description: "Test app for database secret injection verification"
  gitRepo: "https://github.com/test/test-repo"
  gitopsRepo: "https://github.com/test/test-gitops"
  gitopsPath: "test-app"
  imageRepo: "test-backend"
  gitBranch: main
  gitopsBranch: main
  gitopsSecretRef: ""
  pipelineName: "from-code-to-cluster"
  webhookDomain: "test.example.com"
  webhookSecret: "github-webhook-secret"
  port: 3000
  replicas: 1
  components:
    - name: test-backend
      type: web-service
      properties:
        image: "test-backend:latest"
        port: 3000
        replicas: 1
      traits:
        - type: service
          properties:
            port: 3000
        - type: database
          properties:
            dbType: postgres
            dbName: test_backend_db
            version: "16"
            storage: "1Gi"
EOF
```

---

## Step 6: Verify Acceptance Criteria

### 6a. Verify Secret Created (Phase 0.5)

```bash
kubectl get secret test-backend-db-secret -o jsonpath='{.data}' | python3 -m json.tool
```

Expected: Secret contains `DB_HOST`, `DB_USER`, `DB_PASS` keys (base64-encoded).

Decode values:

```bash
kubectl get secret test-backend-db-secret -o jsonpath='{.data.DB_HOST}' | base64 -d
# Expected: test-backend-db

kubectl get secret test-backend-db-secret -o jsonpath='{.data.DB_USER}' | base64 -d
# Expected: some generated username

kubectl get secret test-backend-db-secret -o jsonpath='{.data.DB_PASS}' | base64 -d
# Expected: some generated password
```

### 6b. Verify Postgres StatefulSet Running (Phase 0.7)

```bash
kubectl get statefulset test-backend-db
# Expected: READY 1/1

kubectl get svc test-backend-db
# Expected: headless service (ClusterIP: None)
```

### 6c. Verify Env Var Injection (Phase 0.9) — Acceptance Criterion #1

```bash
kubectl get deployment test-backend -o jsonpath='{.spec.template.spec.containers[0].env}' | python3 -m json.tool
```

Expected output includes `secretKeyRef` entries:

```json
[
  { "name": "PORT", "value": "3000" },
  { "name": "DB_NAME", "value": "test_backend_db" },
  { "name": "DB_PORT", "value": "5432" },
  {
    "name": "DB_HOST",
    "valueFrom": { "secretKeyRef": { "name": "test-backend-db-secret", "key": "DB_HOST" } }
  },
  {
    "name": "DB_USER",
    "valueFrom": { "secretKeyRef": { "name": "test-backend-db-secret", "key": "DB_USER" } }
  },
  {
    "name": "DB_PASS",
    "valueFrom": { "secretKeyRef": { "name": "test-backend-db-secret", "key": "DB_PASS" } }
  }
]
```

Also verify via `kubectl describe pod`:

```bash
kubectl describe pod -l app=test-backend | grep -A2 "DB_HOST\|DB_USER\|DB_PASS"
```

### 6d. Verify NestJS Connects to Database — Acceptance Criterion #2

Wait for the backend pod to be ready:

```bash
kubectl get pods -l app=test-backend -w
# Wait for: test-backend-xxx   1/1   Running
```

Check application logs:

```bash
kubectl logs -l app=test-backend
```

Expected: Prisma migration runs, NestJS starts without connection errors.

Port-forward and test health endpoint:

```bash
kubectl port-forward deployment/test-backend 3000:3000 &
curl http://localhost:3000/health
# Expected: {"status":"ok"}
kill %1
```

---

## Step 7: Check Operator Logs

```bash
kubectl -n operator-system logs deployment/operator-controller-manager -c manager \
  | grep -E "secret|inject|database|StatefulSet"
```

Look for:

- `"Created database secret"` (Phase 0.5)
- `"Created database StatefulSet"` / `"Created database Service"` (Phase 0.7)
- `"Injected database env vars"` (Phase 0.9)

**Note:** Errors about Tekton CRDs or GitOps tokens are expected in this test
environment — the operator tolerates these and continues to database phases.

---

## Step 8: Cleanup

```bash
kubectl delete heliosapp test-app
kubectl delete deployment test-backend
kubectl delete statefulset test-backend-db
kubectl delete svc test-backend-db
kubectl delete secret test-backend-db-secret
k3d cluster delete helios-test
```

---

## Known Gotchas

| Issue | Cause | Fix |
|---|---|---|
| `import pg from 'pg'` fails at runtime | CommonJS module system — `pg` has no default export | Use `import * as pg from 'pg'` |
| `url = env("DATABASE_URL")` in schema.prisma | Prisma 7 removed `url` from schema datasource | Use `prisma.config.ts` with `defineConfig()` |
| `prisma migrate deploy` fails with "lock file missing" | No `migration_lock.toml` in migrations dir | Create `migration_lock.toml` with `provider = "postgresql"` |
| `PrismaPg(pool)` TypeScript error | `@types/pg` Pool type conflicts with adapter's bundled types | Cast with `pool as any` |
| k3d kubeconfig uses `host.docker.internal` | k3d default for Docker Desktop | Override with `kubectl config set-cluster ... --server=https://127.0.0.1:<port>` |
| CUE engine fails to initialize | CUE files not in operator image | Use `Dockerfile.test` which copies `cue/` into image |

---

## Summary of Operator Phases Tested

| Phase | Action | Resource Created |
|---|---|---|
| 0.5 | Generate DB secret | `Secret/<component>-db-secret` |
| 0.7 | Provision Postgres | `StatefulSet/<component>-db` + `Service/<component>-db` |
| 0.9 | Inject env vars | Patches `Deployment/<component>` with `secretKeyRef` |
