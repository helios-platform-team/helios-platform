# Automated Secret Injection (#39)

Inject database credentials (DB_HOST, DB_USER, DB_PASS) from the Operator-generated K8s Secret into the Backend Pod's env block, and create a NestJS + Prisma Node.js template that connects using these injected env vars.

## Proposed Changes

### Operator — Secret Injection into Backend Deployment

The CUE engine renders a Deployment for each component (via GitOps sync). After the GitOps sync pushes the manifest, ArgoCD deploys it. However, the Deployment currently has **no database env vars** — the CUE `#Deployment` base only supports simple `{name, value}` env vars, not `secretKeyRef`.

**Approach**: Add a new reconciliation phase (**Phase 0.9**) in [heliosapp_controller.go](file:///home/phuochoan/Workspace/HCMUS/4th_Year/Capstone_Projects/helios-platform/apps/operator/internal/controller/heliosapp_controller.go) that **patches the live Deployment** (deployed by ArgoCD) to inject `DB_HOST`, `DB_USER`, and `DB_PASS` as `envFrom` / env vars referencing the K8s Secret. This runs AFTER database secrets and instances are created.

> [!IMPORTANT]
> The CUE engine generates the base Deployment manifest (pushed via GitOps). The Go operator patches the **live Deployment** in-cluster to add secret env vars. This keeps secrets out of the GitOps repo entirely.

#### [MODIFY] [database_resources.go](file:///home/phuochoan/Workspace/HCMUS/4th_Year/Capstone_Projects/helios-platform/apps/operator/internal/controller/database_resources.go)

Add a new function `InjectDatabaseEnvVars` that:
- Takes a Deployment and a database secret name
- Adds `DB_HOST`, `DB_USER`, `DB_PASS` env vars to the first container using `valueFrom.secretKeyRef`
- Also adds `DATABASE_URL` as a convenience env var for Prisma ORM (constructed from the other vars via an init container or as a direct string referencing the secret values)
- Is idempotent: skips if env vars already exist

Add a new reconciler method `reconcileDatabaseSecretInjection` that:
- Finds components with database traits
- Gets or waits for the corresponding Deployment (by component name)
- Calls `InjectDatabaseEnvVars` to patch the Deployment's env block
- Uses `r.Update()` to apply changes to the live Deployment

#### [MODIFY] [heliosapp_controller.go](file:///home/phuochoan/Workspace/HCMUS/4th_Year/Capstone_Projects/helios-platform/apps/operator/internal/controller/heliosapp_controller.go)

- Add a call to `r.reconcileDatabaseSecretInjection(ctx, &heliosApp)` as **Phase 0.9** (after database instance provisioning at Phase 0.7, before image validation)
- Add structured logging for the injection phase

#### [MODIFY] [database_resources_test.go](file:///home/phuochoan/Workspace/HCMUS/4th_Year/Capstone_Projects/helios-platform/apps/operator/internal/controller/database_resources_test.go)

Add new test cases:
- `TestInjectDatabaseEnvVars` — verifies env vars are correctly injected into a Deployment
- `TestInjectDatabaseEnvVars_Idempotent` — verifies running injection twice doesn't duplicate env vars
- `TestReconcileDatabaseSecretInjection` — integration test with fake client verifying the full reconciliation flow
- `TestReconcileDatabaseSecretInjection_NoTraits` — skips when no database traits
- `TestReconcileDatabaseSecretInjection_DeploymentNotFound` — handles missing Deployment gracefully (returns nil, logs warning — Deployment may not be deployed by ArgoCD yet)

---

### Node.js Template — NestJS + Prisma ORM

Create a new Backstage template for NestJS + Prisma that connects to the database using the injected env vars.

#### [NEW] `apps/portal/examples/nestjs-prisma-template/` directory

Structure:
```
nestjs-prisma-template/
  template.yaml                  # Backstage scaffolder template
  content/
    source/
      package.json               # NestJS + Prisma deps (latest)
      tsconfig.json              # TypeScript config
      tsconfig.build.json        # Build-specific TS config
      nest-cli.json              # NestJS CLI config
      .env.example               # Example env vars
      Dockerfile                 # Multi-stage NestJS build
      catalog-info.yaml          # Backstage catalog entry
      prisma/
        schema.prisma            # Prisma schema with env-based DATABASE_URL
      src/
        main.ts                  # NestJS bootstrap
        app.module.ts            # Root module
        app.controller.ts        # Health check controller
        app.service.ts           # App service
        prisma/
          prisma.module.ts       # Prisma module
          prisma.service.ts      # Prisma service (extends PrismaClient)
    gitops/
      helios-app.yaml            # HeliosApp CRD manifest with database trait
```

Key design decisions:
- `DATABASE_URL` is built from `DB_HOST`, `DB_USER`, `DB_PASS` env vars (Prisma convention)
- The Prisma schema reads `DATABASE_URL` from `env("DATABASE_URL")`
- The [Dockerfile](file:///home/phuochoan/Workspace/HCMUS/4th_Year/Capstone_Projects/helios-platform/apps/operator/Dockerfile) runs `prisma generate` at build time and `prisma migrate deploy` at startup
- Uses latest stable NestJS v11+ and Prisma v6+

---

## Verification Plan

### Automated Tests

All existing tests plus new tests, run from `apps/operator/`:

```sh
# 1. Compilation check
go build ./...

# 2. Static analysis
go vet ./...

# 3. Full test suite (includes all database_resources_test.go tests)
make test

# 4. Build operator binary
make build

# 5. Run database-specific tests in verbose mode
go test ./internal/controller/... -v -count=1 -run TestInjectDatabaseEnvVars
go test ./internal/controller/... -v -count=1 -run TestReconcileDatabaseSecretInjection
```

### Manual Verification

After deploying a backend with a database trait:
1. Run `kubectl describe pod <backend-pod>` to verify DB_HOST, DB_USER, DB_PASS are present as environment variables referencing the secret
2. Verify the NestJS application boots and connects to the database by checking pod logs
