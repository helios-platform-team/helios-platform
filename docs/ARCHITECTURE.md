# Helios Platform — High-Level Architecture

## Overview

Helios is an **internal developer platform (IDP)** built on Kubernetes that provides a self-service experience for deploying and managing applications. It combines a **Backstage developer portal** with a **custom Kubernetes operator** to deliver a fully automated workflow: developers describe their applications via a UI, and the platform handles source code scaffolding, CI/CD pipeline creation, GitOps-based deployment, database provisioning, and observability — all without requiring deep Kubernetes expertise.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          Developer Workflow                             │
│                                                                         │
│   Developer ──► Backstage Portal ──► Scaffold Repos ──► HeliosApp CR   │
│                                                                         │
│   HeliosApp CR ──► Operator ──► CUE Render ──► GitOps Push             │
│                                    │                                    │
│                                    ├──► Tekton Pipelines (CI)           │
│                                    └──► Argo CD Application (CD)        │
│                                                                         │
│   Git Push ──► Webhook ──► Tekton PipelineRun ──► Build & Deploy        │
└─────────────────────────────────────────────────────────────────────────┘
```

## Repository Structure

```
helios-platform/
├── apps/
│   ├── operator/          # Kubernetes operator (Go, Kubebuilder)
│   └── portal/            # Backstage developer portal (TypeScript)
├── cue/                   # CUE definitions for manifest & Tekton generation
│   ├── definitions/       # Schemas, components, traits, Tekton resources
│   ├── engine/            # Builder logic that produces K8s & Tekton objects
│   └── examples/          # Sample inputs for testing
├── docs/                  # Project documentation
├── scripts/               # Prerequisite checks, cluster cleanup, dev helpers
├── infrastructure/        # (Reserved, currently empty)
├── .github/               # CI workflows and PR template
├── Taskfile.yml           # Task runner for setup, dev, test, clean
└── .env.example           # Environment variable reference
```

## Core Components

### 1. Backstage Developer Portal (`apps/portal/`)

| Attribute       | Value                                               |
|-----------------|-----------------------------------------------------|
| **Framework**   | Backstage 1.49.1                                    |
| **Runtime**     | Node 22/24, Yarn 4.4.1                              |
| **Database**    | SQLite (dev), PostgreSQL (production)                |
| **Auth**        | Guest (default), GitHub OAuth (optional)             |

The portal is the primary UI for developers. It serves as a **software catalog**, **scaffolder**, and **observability dashboard**.

**Key capabilities:**

- **Software Catalog** — Registers and displays all platform-managed components with metadata, ownership, and lifecycle status. Includes automatic discovery from Gitea organizations.
- **Software Templates** — Three built-in templates let developers scaffold new services:
  - *Example Node.js* — Basic Node.js service with Gitea repo + GitOps repo creation.
  - *Advanced Node.js (GitOps)* — Full pipeline: source repo, GitOps repo, Tekton triggers, HeliosApp CR applied to the cluster.
  - *NestJS + Prisma (Database-backed)* — NestJS service with Prisma ORM and operator-managed PostgreSQL.
- **CI/CD Observability** — Tekton PipelineRun/TaskRun status and Argo CD sync state displayed directly on entity pages.
- **Kubernetes Views** — Live pod/deployment status from the cluster.
- **TechDocs** — Embedded documentation site generated from Markdown.
- **Notifications** — In-app notification delivery (e.g., post-scaffold success messages).

**Custom backend extensions:**

| Action                                  | Purpose                                              |
|-----------------------------------------|------------------------------------------------------|
| `kubernetes:apply`                      | Applies a YAML manifest from the scaffolder workspace|
| `kubernetes:create-git-credentials-secret`| Creates a Kubernetes Secret with Gitea credentials |
| `gitea:create-webhook`                  | Registers a push webhook on a Gitea repository       |

**Custom frontend extensions:**

- `DatabasePickerExtension` — A scaffolder field that lets users select database configurations when using the NestJS + Prisma template.
- Rich `EntityPage` wiring with Tekton, Argo CD, and Kubernetes tabs.

### 2. Helios Operator (`apps/operator/`)

| Attribute       | Value                                               |
|-----------------|-----------------------------------------------------|
| **Language**    | Go 1.26                                             |
| **Framework**   | Kubebuilder / controller-runtime v0.23.3            |
| **CRD**        | `HeliosApp` (`app.helios.io/v1alpha1`)              |

The operator is the control plane of the platform. It watches `HeliosApp` custom resources and reconciles the desired state into a complete application stack.

#### HeliosApp CRD Schema

```yaml
apiVersion: app.helios.io/v1alpha1
kind: HeliosApp
metadata:
  name: my-app
  namespace: default
spec:
  owner: team-backend
  description: "My application"
  imageRepo: registry.example.com/my-app
  gitRepo: http://gitea:3000/org/my-app.git
  gitBranch: main
  gitopsRepo: http://gitea:3000/org/my-app-gitops.git
  gitopsPath: environments/dev
  gitopsBranch: main
  replicas: 2
  port: 3000
  env:
    - name: NODE_ENV
      value: production
  components:
    - name: web
      type: web-service
      properties:
        image: registry.example.com/my-app:latest
        replicas: 2
        port: 3000
      traits:
        - type: service
          properties:
            port: 3000
        - type: ingress
          properties:
            host: my-app.example.com
        - type: database
          properties:
            type: postgres
```

#### Reconciliation Flow

```
HeliosApp CR created/updated
        │
        ▼
┌─────────────────────┐
│  1. Map CRD → Model │  Convert spec to internal Application/TektonInput structs
└────────┬────────────┘
         │
         ▼
┌─────────────────────┐
│  2. CUE Render       │  engine.Render() → multi-doc YAML for app manifests
│     (App Manifests)  │  (Deployment, Service, Ingress, ConfigMap)
└────────┬────────────┘
         │
         ▼
┌─────────────────────┐
│  3. Tekton Resources │  TektonRenderer → Tasks, Pipeline, Triggers, EventListener
│     (CUE + Go RBAC)  │  + Go-managed ServiceAccount, RoleBinding, ClusterRoleBinding
└────────┬────────────┘
         │
         ▼
┌─────────────────────┐
│  4. Database         │  For "database" traits: Secret (credentials),
│     Provisioning     │  StatefulSet (Postgres), headless Service,
│                      │  Deployment env injection (DB_HOST, DB_USER, etc.)
└────────┬────────────┘
         │
         ▼
┌─────────────────────┐
│  5. Image Gate       │  Wait until all components have non-empty image;
│                      │  trigger initial PipelineRun if not yet done
└────────┬────────────┘
         │
         ▼
┌─────────────────────┐
│  6. GitOps Sync      │  Clone gitops repo, write manifest.yaml,
│                      │  commit & push (skip if content unchanged)
└────────┬────────────┘
         │
         ▼
┌─────────────────────┐
│  7. Argo CD App      │  Create/ensure Application CR pointing at
│                      │  gitops repo/path/branch → app namespace
└─────────────────────┘
```

**Status tracking:** The operator maintains `phase` (Pending → Ready → Failed), `conditions`, `lastAppliedHash` (to avoid redundant GitOps pushes), and `initialBuildTriggered`.

### 3. CUE Template Engine (`cue/`)

CUE serves as the **declarative rendering engine** for both application manifests and Tekton CI/CD resources. It decouples resource generation from the operator's Go code, making templates composable and testable independently.

#### Application Rendering Path

```
#HeliosApp (input)
    │
    ├── Component Registry ──► web-service → Deployment
    │
    └── Trait Registry
            ├── service  → Service
            ├── ingress  → Ingress (nginx)
            └── database → ConfigMap (connection metadata)
    │
    ▼
kubernetesObjects[]  →  Multi-doc YAML pushed to GitOps repo
```

**OAM-inspired model:** Applications are composed of typed **components** (e.g., `web-service`) and optional **traits** (e.g., `service`, `ingress`, `database`). Each type maps to a CUE definition that renders standard Kubernetes objects.

#### Tekton Rendering Path

```
#TektonInput (input)
    │
    ├── Task Registry
    │     ├── git-clone        (checkout source)
    │     ├── buildah-build     (container image build + push)
    │     └── git-update-manifest (update GitOps repo with new image)
    │
    ├── Pipeline Registry
    │     ├── from-code-to-cluster  (clone → test → build → gitops update)
    │     └── build-only            (clone → test → build)
    │
    └── Trigger Registry
          └── gitea-push  →  TriggerBinding + TriggerTemplate
                              + EventListener + optional Ingress
    │
    ▼
tektonObjects[]  →  Applied directly to the cluster by the operator
```

### 4. GitOps Layer

The platform follows a **GitOps** pattern where the desired state of deployed applications is stored in Git:

1. The **operator** renders Kubernetes manifests via CUE and pushes them to a dedicated GitOps repository (one per application).
2. An **Argo CD Application** CR watches that repository and syncs manifests into the application's namespace.
3. Argo CD is configured with **automated sync**, **pruning**, and **self-heal** enabled.
4. `ignoreDifferences` is set on Deployment env vars matching `DB_*` to prevent Argo CD from reverting operator-injected database credentials.

### 5. CI/CD Pipeline (Tekton)

The CI pipeline is fully automated via Tekton, triggered by Git push events:

```
Git Push → Gitea Webhook → EventListener → TriggerTemplate → PipelineRun
                                                                   │
                              ┌────────────────────────────────────┘
                              │
                    ┌─────────┴─────────┐
                    │  1. git-clone      │  Clone source repository
                    ├────────────────────┤
                    │  2. run-tests      │  Execute test command (inline taskSpec)
                    ├────────────────────┤
                    │  3. buildah-build   │  Build container image, push to registry
                    ├────────────────────┤
                    │  4. git-update-    │  Update GitOps repo with new image tag
                    │     manifest       │  → Argo CD syncs automatically
                    └────────────────────┘
```

## Integration Map

```
┌──────────────┐         ┌──────────────────┐        ┌──────────────┐
│              │ scaffold │                  │ apply   │              │
│  Backstage   ├────────►│  Gitea           │◄───────►│  Kubernetes  │
│  Portal      │ webhook │  (Git hosting)   │  sync   │  Cluster     │
│              │         │                  │         │              │
└──────┬───────┘         └────────┬─────────┘        └──────┬───────┘
       │                          │                          │
       │ view CI/CD               │ push event               │ runs on
       │ status                   │                          │
       │                          ▼                          │
       │                 ┌──────────────────┐                │
       │                 │  Tekton          │────────────────┘
       │                 │  (CI Pipelines)  │
       │                 └────────┬─────────┘
       │                          │ updates manifest
       │                          ▼
       │                 ┌──────────────────┐
       └────────────────►│  Argo CD         │────── syncs to ───► App Namespace
         view sync       │  (CD / GitOps)   │
         status          └──────────────────┘
```

| System       | Role                                                     |
|-------------|----------------------------------------------------------|
| **Backstage**| Developer portal: catalog, scaffolder, observability UI  |
| **Gitea**    | Self-hosted Git server for source and GitOps repositories|
| **Tekton**   | CI pipeline engine: clone, test, build, push manifests   |
| **Argo CD**  | CD engine: syncs GitOps repo to cluster, auto-heal       |
| **CUE**      | Declarative template engine for K8s and Tekton manifests |
| **Operator** | Control plane: reconciles HeliosApp → infrastructure     |
| **PostgreSQL**| Optional per-app database, provisioned by the operator  |

## Local Development Stack

The entire platform runs locally on a **k3d** cluster, orchestrated by `Taskfile.yml`:

```bash
task setup    # Creates k3d cluster, installs Tekton, Argo CD, Gitea,
              # configures tokens, secrets, CRDs, and portal dependencies

task dev      # Runs operator (Go) + portal (Backstage) in dev mode

task test     # Runs operator unit/integration tests + portal checks

task clean    # Tears down the k3d cluster
```

**Environment variables** (`.env.example`):

| Category        | Variables                                                  |
|----------------|-------------------------------------------------------------|
| **Gitea**       | `GITEA_URL`, `GITEA_INTERNAL_URL`, `GITEA_TOKEN`, `GITEA_USER`, `GITEA_ADMIN_USER/PASS` |
| **Docker**      | `DOCKER_SERVER`, `DOCKER_USERNAME`, `DOCKER_PASSWORD`, `DOCKER_EMAIL` |
| **GitHub OAuth** | `AUTH_GITHUB_CLIENT_ID`, `AUTH_GITHUB_CLIENT_SECRET` (optional) |
| **Git**         | `GIT_AUTHOR_NAME`, `GIT_AUTHOR_EMAIL` (optional)           |

## CI/CD (GitHub Actions)

| Workflow             | Trigger                           | Steps                              |
|---------------------|------------------------------------|------------------------------------|
| `operator-ci.yml`   | Push/PR to `main` or `features/*` | Go 1.26, `make test`               |
| `portal-ci.yml`     | Push/PR to `main` or `features/*` | Node 24, `prettier:check`, `tsc`, `build:all` |

## Testing Strategy

| Layer                | Tool                    | Scope                                        |
|----------------------|------------------------|----------------------------------------------|
| **Operator unit**    | Go test + envtest       | Controller logic, CUE engine, GitOps client  |
| **Operator E2E**     | Ginkgo + Kind cluster   | Full reconciliation against a real cluster    |
| **CUE validation**   | `cue vet` + Go tests   | Schema conformance and rendering correctness  |
| **Portal**           | Jest + Playwright       | Frontend components and integration tests    |
| **CI checks**        | GitHub Actions          | Lint, type-check, build verification         |

## Key Design Decisions

1. **OAM-inspired component model** — Applications are described as components + traits rather than raw Kubernetes manifests, providing a higher-level abstraction.
2. **CUE over Go templates** — Manifest generation is delegated to CUE for type safety, composability, and independent testability outside of Go.
3. **Operator as orchestrator** — A single `HeliosApp` CR drives the entire lifecycle: CI pipelines, GitOps repos, Argo CD applications, and database provisioning.
4. **GitOps as the deployment mechanism** — The operator never deploys workloads directly; it writes manifests to Git and lets Argo CD handle synchronization.
5. **Gitea for self-contained hosting** — The platform uses Gitea instead of external Git providers, enabling a fully local and air-gapped development experience.
6. **Backstage as the developer interface** — Provides a unified portal for service creation, catalog browsing, and CI/CD observability without requiring kubectl access.
