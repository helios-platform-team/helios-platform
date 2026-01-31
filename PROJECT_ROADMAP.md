# Helios V2: Project Roadmap & Technical Guide

This document provides a comprehensive gap analysis between the **Helios V2 Re-designed** requirements and the current codebase, along with a technical implementation guide for the missing features and new proposed capabilities. It is designed to assist the project management and development planning for a team of 6 students over a 6-month period.

---

## Part 1: Gap Analysis (Current vs. Required)

The following table compares the requirements outlined in the "Helios V2" document against the current state of the repository.

| Feature Area | Requirement | Current Status | Implementation Gap |
| :--- | :--- | :--- | :--- |
| **1. Architecture** | **OAM & CUE Engine** | ✅ **Implemented** | Core CUE engine exists (`cue/engine`). Separation of logic is handled. |
| | **GitOps Single Writer** | ✅ **Implemented** | Operator syncs to Git (`apps/operator/internal/gitops`). |
| **2. Golden Path** | **Multi-language Support** | ❌ **Missing** | Only generic `web-service` exists. Need specific templates for .NET, Java, React/Vue in Backstage & CUE. |
| | **Fullstack Scaffolding** | ❌ **Missing** | Current templates only scaffold single services. Need a "Monorepo" or "Multi-repo" scaffolder action for FE+BE+DB. |
| | **Database Provisioning** | ❌ **Missing** | No logic to provision RDS/SQL instances. Currently assumes stateless workloads. |
| **3. Automated Wiring** | **DB Credentials & Injection** | ❌ **Missing** | No Operator logic to generate Secrets, create random passwords, or inject env vars automatically. |
| | **Service Discovery (BE -> FE)** | ❌ **Missing** | No logic to inject Backend URLs into Frontend configurations. |
| **4. Developer Exp.** | **Dynamic Wizard (Schema-UI)** | ⚠️ **Partial** | Standard Backstage JSONSchema forms exist, but they are not dynamically generated from CUE/OpenAPI as requested. |
| | **Built-in Observability** | ⚠️ **Partial** | Standard Backstage plugins used. No pre-configured Grafana/Prometheus dashboards or libraries injected into app templates. |
| **5. CI/CD** | **Tekton Pipelines** | ✅ **Implemented** | Strong Tekton integration exists, but currently hardcoded in Go (`tekton_resources.go`). |

---

## Part 2: Implementation Guide (Core Missing Features)

### A. One-Click Fullstack Project Bootstrapping

**Goal:** Create a single template in Backstage that provisions a Frontend, Backend, and Database.

**Technical Approach:**
1.  **Backstage Template (`template.yaml`):**
    *   Create a new template `fullstack-template`.
    *   Use `parameters` to ask for: App Name, Frontend Framework (React/Vue), Backend Language (Java/C#), Database Type (Postgres/SQL Server).
2.  **Scaffolder Actions:**
    *   Use `fetch:template` multiple times to fetch skeleton code for FE and BE into a single directory structure (e.g., `/packages/client`, `/packages/server`).
    *   **Monorepo Strategy:** Recommend scaffolding a standard monorepo (e.g., Nx or Turbo) or simple folder separation.
3.  **CUE Definition Update:**
    *   Update `cue/examples/simple-app.cue` (and the `HeliosApp` CRD) to support multiple components in one file.
    *   Example CUE structure:
        ```cue
        components: [
          { name: "web", type: "frontend", ... },
          { name: "api", type: "backend", ... },
          { name: "db",  type: "database", ... }
        ]
        ```

### B. Automated Wiring (The "Killer Feature")

**Goal:** Automatically connect Database -> Backend -> Frontend without manual config.

**Technical Approach (Operator Logic in `apps/operator`):**
1.  **Database Credential Generation:**
    *   **Where:** In `HeliosAppReconciler.Reconcile` (Phase -1 or 0).
    *   **Logic:**
        *   Check if a "Database" component exists.
        *   Check if a `Secret` named `${AppName}-db-creds` exists.
        *   If not, generate random username/password and create the Kubernetes Secret.
2.  **Backend Injection:**
    *   **Where:** In `cue/engine/builder.cue` or Operator Pre-processing.
    *   **Logic:**
        *   Modify the Backend component's `env` list to include `DB_HOST`, `DB_USER`, `DB_PASS` (referencing the Secret).
        *   *Tip:* Pass the Secret name into the CUE `input` so CUE can render the `valueFrom: secretKeyRef`.
3.  **Frontend Injection:**
    *   **Where:** In Operator.
    *   **Logic:**
        *   Identify the Service URL of the Backend (e.g., `http://${AppName}-backend.${Namespace}.svc`).
        *   Inject this URL into the Frontend's build-time environment variables (e.g., `REACT_APP_API_URL`).
        *   *Note:* Since Frontend is static (usually), this might require a runtime config injection strategy (e.g., a `config.js` file generated at startup).

---

## Part 3: New Feature Technical Plan

### 1. Multiple Environments (Dev, Staging, Prod)

**Goal:** Allow users to promote applications across environments.

**Technical Strategy: GitOps Branching**
*   **Concept:** Use Git Branches in the **GitOps Repo** to represent environments.
    *   `main` -> Production
    *   `staging` -> Staging Cluster/Namespace
    *   `dev` -> Dev Cluster/Namespace
*   **Implementation:**
    1.  **Backstage:** Add a "Promote" action/button that merges a Pull Request from `dev` branch to `staging` branch in the GitOps repo.
    2.  **ArgoCD:** Configure ArgoCD to have 3 Applications pointing to the same Repo but different **targetRevision** (branches).
    3.  **Operator:** Ensure the Operator can sync to specific branches based on `spec.gitOpsBranch` in the `HeliosApp` CRD.

### 2. UI/UX Re-design & Branding

**Goal:** Override default Backstage styles.

**Technical Strategy:**
*   **Theme:** Backstage uses Material UI v4 (mostly).
*   **File:** `apps/portal/packages/app/src/theme/myTheme.ts`.
*   **Action:**
    *   Create a custom theme implementing `BackstageTheme`.
    *   Override `palette` (colors), `typography` (fonts), and specific component styles (`overrides`).
    *   Inject this theme in `App.tsx`.
*   **Figma:** Use Figma to design the color palette and logo, then export assets to `apps/portal/packages/app/src/assets`.

### 3. Documentation & APIs (OpenAPI/Swagger)

**Goal:** Visualize API docs automatically.

**Technical Strategy:**
1.  **Standard:** Use the `api-docs` plugin (already in `package.json`).
2.  **Catalog Info:** In the `catalog-info.yaml` scaffolded for the Backend, ensure it includes:
    ```yaml
    apiVersion: backstage.io/v1alpha1
    kind: API
    metadata:
      name: my-app-api
    spec:
      type: openapi
      lifecycle: production
      owner: user:guest
      definition:
        $text: https://github.com/.../openapi.yaml
    ```
3.  **Automation:**
    *   Modify the **Tekton Pipeline** to extract `swagger.json` / `openapi.yaml` from the code (e.g., generated by Spring Boot or Swashbuckle).
    *   Commit this file to the Source Repo or a specific docs folder so Backstage can read it.

### 4. CUE for Tekton Pipelines

**Goal:** Replace hardcoded Go logic (`tekton_resources.go`) with CUE rendering.

**Technical Strategy:**
1.  **Define CUE Schema:** Create `cue/definitions/tekton/pipeline.cue`.
    *   Model `Task`, `Pipeline`, `TriggerBinding` in CUE.
2.  **Migration:**
    *   Move the logic from `GeneratePipeline()` (Go) into a CUE template.
    *   Update `HeliosAppReconciler` to call `CueEngine.Render(pipelineModel)` instead of calling the Go functions.
3.  **Benefit:** This allows you to have different Pipeline templates (e.g., `JavaPipeline`, `NodePipeline`) defined in CUE files, which are easier to edit than Go code.

---

## Part 4: Recommended Additional Features

These are realistic features that add high value within the 6-month timeframe.

### 1. Cost Insights (OpenCost/KubeCost)
*   **Why:** "Cloud-Native" usually brings cost concerns.
*   **How:** Install OpenCost in the cluster. Use the Backstage **Cost Radius** or **OpenCost** plugin to show estimated monthly costs for the created `Deployment` (based on CPU/RAM requests).

### 2. Security Scanning (Trivy)
*   **Why:** DevSecOps is a standard requirement.
*   **How:**
    *   Add a **Trivy** step in the Tekton Pipeline (`kaniko-build` task).
    *   Scan the image *before* pushing.
    *   If vulnerabilities > Critical, fail the build.
    *   Visualize results in Backstage using the **Security Insights** plugin.

### 3. RBAC with Keycloak
*   **Why:** "Self-Service" requires identity management.
*   **How:**
    *   Deploy **Keycloak** in the cluster.
    *   Configure Backstage `auth` provider (OIDC) to talk to Keycloak.
    *   Map Keycloak Groups to Backstage Groups to control who can see/edit which Templates.

### 4. Ephemeral Environments (PR Previews)
*   **Why:** High-velocity development.
*   **How:**
    *   When a PR is opened in GitHub, trigger a separate Tekton pipeline.
    *   Deploy the app to a temporary namespace (`pr-123`).
    *   Post the URL back to the GitHub PR comment.
    *   Delete namespace when PR is closed.
