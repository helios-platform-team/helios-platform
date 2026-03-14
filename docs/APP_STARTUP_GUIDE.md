# Helios Platform Setup Guide

> **Goal:** Everyone should be able to run the same app. A new user can set up the app and have it running within 30 minutes.

---

## Quick Start (Recommended)

The fastest way to get up and running uses [Task](https://taskfile.dev/) to automate all setup steps.

### 1. Install Task

```bash
# Linux
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b /usr/local/bin

# macOS (Homebrew)
brew install go-task

# Windows (Scoop)
scoop install task

# Windows (Chocolatey)
choco install go-task
```

> **Windows users:** Task requires a POSIX shell for its commands. Install [Git for Windows](https://git-scm.com/download/win) (includes Git Bash) or use [WSL2](https://learn.microsoft.com/en-us/windows/wsl/install). You can also run `scripts\check-prereqs.bat` directly from `cmd.exe` to verify tools before using Task.

### 2. Configure credentials

```bash
cp .env.example .env
# Edit .env and fill in your GitHub PAT, OAuth app, and Docker Hub credentials
```

### 3. Verify prerequisites

```bash
task check
```

### 4. Bootstrap the environment

This creates a k3d cluster (lightweight k3s-in-Docker), installs Tekton, ArgoCD, CRDs, and all dependencies (~5-10 min):

```bash
task setup
```

### 5. Start developing

Runs the Go operator and Backstage portal concurrently:

```bash
task dev
```

The portal will be available at http://localhost:3000 and the backend API at http://localhost:7007.

### Other useful commands

| Command | Description |
|---------|-------------|
| `task check` | Verify all prerequisites are installed |
| `task setup` | Bootstrap the full local environment |
| `task dev` | Run operator + portal concurrently |
| `task dev:operator` | Run only the operator |
| `task dev:portal` | Run only the portal |
| `task test` | Run all tests |
| `task clean` | Delete the k3d cluster |

---

## Manual Setup (Step-by-Step Reference)

The sections below describe each step that `task setup` performs automatically. Use these for troubleshooting or if you prefer manual control.

### Prerequisites

#### Required Tools

Ensure you have the following installed (run `task check` to verify):

- **Go** >= 1.24 ([install](https://go.dev/dl/))
- **Docker** ([install](https://docs.docker.com/get-docker/))
- **kubectl** ([install](https://kubernetes.io/docs/tasks/tools/))
- **k3d** ([install](https://k3d.io/) or `curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash`)
- **CUE** (`go install cuelang.org/go/cmd/cue@latest`)
- **Node.js** >= 22 ([install](https://nodejs.org/) or `nvm install 22`)
- **Yarn** 4 (`corepack enable && corepack prepare yarn@4 --activate`)

#### Environment Variables

Create a `.env` file at the **repository root** with all credentials (see `.env.example`):

```bash
cp .env.example .env
```

The Taskfile distributes these to the operator and portal automatically. If running manually, you also need a `.env` in `apps/portal/` with at minimum:

```env
AUTH_GITHUB_CLIENT_ID=
AUTH_GITHUB_CLIENT_SECRET=
GITHUB_ORG=helios-platform-team
GITHUB_TOKEN=
```

---

## Step 1: Create a Fresh k3d Cluster

```bash
k3d cluster create helios-dev --agents 1 --wait
```

---

## Step 2: Install Tekton (Pipeline, Triggers & Interceptors)

### 2.1 Install Components

```bash
# Install Pipeline, Triggers & Interceptors
kubectl apply --filename https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
kubectl apply --filename https://storage.googleapis.com/tekton-releases/triggers/latest/release.yaml
kubectl apply --filename https://storage.googleapis.com/tekton-releases/triggers/latest/interceptors.yaml
```

### 2.2 Configure Feature Flags

> [!IMPORTANT]
> Disabling affinity assistant and coschedule prevents pod scheduling issues.

```bash
kubectl patch configmap feature-flags -n tekton-pipelines -p '{"data":{"disable-affinity-assistant":"true","coschedule":"disabled"}}'
```

### 2.3 Restart Controller

```bash
kubectl rollout restart deployment tekton-pipelines-controller -n tekton-pipelines
```

### 2.4 Apply Compatibility Patch (Kubernetes < 1.28 Only)

> [!WARNING]
> **Only apply this patch if your k3s node version is below 1.28.**
> 
> k3d typically ships with a recent k3s version (1.28+), so this patch is rarely needed. Check with `kubectl version`.

```bash
# PATCH: Fix incompatibility with older Kubernetes (Prevent CrashLoopBackOff)
kubectl set env deployment/tekton-pipelines-controller -n tekton-pipelines KUBERNETES_MIN_VERSION=1.20.0
kubectl set env deployment/tekton-pipelines-webhook -n tekton-pipelines KUBERNETES_MIN_VERSION=1.20.0
```

---

## Step 3: Install ArgoCD

```bash
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
```

---

## Step 4: Run the Operator

### 4.1 Install CRDs

```bash
make -C apps/operator install
```

### 4.2 Run the Controller Locally

```bash
make -C apps/operator run
```

---

## Step 5: Create Tekton Resources

> [!NOTE]
> This installs: `git-clone`, `kaniko-build`, `git-update-manifest`, and the pipeline itself.

```bash
# Generate YAML from CUE
cue export ./cue/definitions/tekton/tasks/*.cue --out yaml > apps/operator/tekton/tasks.yaml

# Apply the generated file
kubectl apply -f apps/operator/tekton/tasks.yaml
```

---

## Step 6: Grant Tekton Triggers ServiceAccount Permissions

This command allows the EventListener to start the pipeline by creating a `PipelineRun` resource.

```bash
kubectl create clusterrolebinding tekton-triggers-sa-admin \
  --clusterrole=cluster-admin \
  --serviceaccount=default:tekton-triggers-sa
```

---

## Step 7: Create Docker Credentials Secret

### 7.1 Create the Secret

> [!CAUTION]
> Replace with your own Docker Hub credentials.

```bash
kubectl create secret docker-registry docker-credentials \
  --docker-server=https://index.docker.io/v1/ \
  --docker-username=<your-docker-username> \
  --docker-password=<your-docker-password> \
  --docker-email=<your-email>
```

### 7.2 Link to Pipeline ServiceAccount

```bash
kubectl patch sa pipeline -p '{"secrets": [{"name": "docker-credentials"}]}'
```

---

## Step 8: Start the Portal

### 8.1 Set Up Node.js (Node 22 Required)

```bash
export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"  
[ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion"  
```

```bash
source ~/.bashrc  # or source ~/.zshrc
```

```bash
nvm use 22
```

### 8.2 Install Dependencies and Start

```bash
yarn install
```

```bash
# In the portal folder, run the starting script
./start-dev.sh
```

---

## Step 9: Create a New App Using Backstage Template

### 9.1 Start ngrok

```bash
ngrok http 8080
```

### 9.2 Configure the App

1. Copy the ngrok URL
2. Enter the URL into the webhooks field in the app creation form
3. Add secret

**Example Values:**

| Field | Value |
|-------|-------|
| Username | `hoangphuc841` |
| Repo Name | `advanced-nodejs-app-v1` |
| Webhook URL | `https://unstaggering-unoptionally-hugh.ngrok-free.dev` |

> [!IMPORTANT]
> - Make sure to enter a repo name that **doesn't already exist**.
> - Ensure the PAT in `.env` matches the username. If you use your GitHub username, you must use your own `GITHUB_TOKEN`.

---

## Step 10: Operator Processing

After clicking create and successfully adding the app to Backstage, the operator will:

1. Render templates
2. SmartSync
3. Push manifest to GitOps
4. Apply the helios-app.yaml from the app's GitOps repo
5. Operator will begin reconcile the new app.

### Troubleshooting: GitOps Token Empty Error

> [!WARNING]
> If you see a "GitOps token empty" error in the operator terminal,you might have not set the GITHUB_TOKEN in the .env file. If you have set the GITHUB_TOKEN but still see this error, you need to create the secret manually.

```bash
# Replace [app-name-here] with your actual app name
kubectl delete secret github-credentials-[app-name-here] --ignore-not-found

kubectl create secret generic github-credentials-[app-name-here] \
  --from-literal=token=<your-github-token> \
  --from-literal=password=<your-github-token> \
  --from-literal=username=<your-github-username> \
  --from-literal=secretToken=github-credentials-[app-name-here]
```

**After this:**
- The template applies the `helios-app.yaml` from the app's GitOps repo
- Check the operator terminal to see the running process:
  1. Cloning repository
  2. Creating and committing `manifest.yaml`
  3. Pushing to GitOps repo
  4. Operator will begin reconcile the new app.

---

## Step 11: Port-Forward to the App

```bash
# Format: el-[app-name]-listener
kubectl port-forward svc/[app-name]-backend 9090:8080
```

---

## Step 12: Test the Tekton Pipeline

### Trigger a Pipeline Run

1. Commit a meaningful change to the app source repo
2. GitHub detects the change and sends a webhook payload to the EventListener
3. The EventListener creates a PipelineRun

### Monitor Pipeline Runs

**See pipeline run names and status:**

```bash
kubectl get pipelineruns -n default --sort-by=.metadata.creationTimestamp
```

**See pipeline pods:**

```bash
# Example: grep for your app name
kubectl get pods -n default | grep "advanced-nodejs-app-v14-run"
```

---

## Quick Reference

### Automated (Taskfile)

| Command | Description |
|---------|-------------|
| `task setup` | Steps 1-8 in one command |
| `task dev` | Run operator + portal |
| `task clean` | Tear down cluster |

### Manual

| Step | Description | Command |
|------|-------------|---------|
| 1 | Create cluster | `k3d cluster create helios-dev --agents 1 --wait` |
| 2 | Install Tekton | `kubectl apply -f https://storage.googleapis.com/tekton-releases/...` |
| 3 | Install ArgoCD | `kubectl apply -n argocd -f https://...argo-cd/.../install.yaml` |
| 4 | Run operator | `make -C apps/operator run` |
| 5 | Apply Tekton resources | `kubectl apply -f apps/operator/tekton/` |
| 6 | Grant permissions | `kubectl create clusterrolebinding tekton-triggers-sa-admin ...` |
| 7 | Docker credentials | `kubectl create secret docker-registry docker-credentials ...` |
| 8 | Start portal | `yarn install && ./start-dev.sh` |
