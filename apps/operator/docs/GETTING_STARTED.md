# Getting Started with Helios Operator

Hướng dẫn triển khai ứng dụng với Helios Operator - GitOps workflow tự động (Development).

## 📋 Yêu Cầu

- Minikube
- kubectl, Git, Docker Hub account
- 2 GitHub repos: source code (có Dockerfile), GitOps manifests (có initial manifest)

## 🚀 Setup

### Checklist Trước Khi Bắt Đầu

Đảm bảo bạn đã có:

- [ ] Minikube đang chạy
- [ ] Docker Hub account và đã login: `docker login`
- [ ] GitHub Personal Access Token với quyền repo
- [ ] 2 GitHub repos: source code (có Dockerfile), GitOps manifests (có initial manifest)

### 1. Cluster

```bash
minikube start
```

### 2. Tekton & ArgoCD

```bash
# Tekton
kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
kubectl apply -f https://storage.googleapis.com/tekton-releases/triggers/latest/release.yaml
kubectl apply -f https://storage.googleapis.com/tekton-releases/triggers/latest/interceptors.yaml

# Fix single-node cluster
kubectl patch configmap feature-flags -n tekton-pipelines --type merge -p '{"data":{"disable-affinity-assistant":"true","coschedule":"disabled"}}'
kubectl delete pod -n tekton-pipelines -l app=tekton-pipelines-controller

# ArgoCD
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# Tùy chọn: Sync nhanh hơn (30s thay vì 3min)
kubectl patch configmap argocd-cm -n argocd --type merge -p '{"data":{"timeout.reconciliation":"30s"}}'
kubectl rollout restart deployment argocd-repo-server -n argocd
kubectl rollout restart statefulset argocd-application-controller -n argocd
```

### 3. Operator

```bash
git clone https://github.com/hoangphuc841/helios-operator
cd helios-operator
make install
# Build image local với tag latest
make docker-build IMG=helios-operator:latest
# Load image mới vào minikube
minikube image load helios-operator:latest
# Deploy operator
make deploy IMG=helios-operator:latest
```

### 4. Tekton Resources & ServiceAccounts

⚠️ **QUAN TRỌNG**: Phải tạo theo đúng thứ tự này:

```bash
# 1. PVC (bắt buộc tạo trước)
kubectl apply -f tekton/workspace-pvc.yaml

# 2. ServiceAccounts (bắt buộc tạo trước HeliosApp)
kubectl apply -f tekton/tekton-triggers-sa.yaml
kubectl apply -f tekton/service-account.yaml

# 3. Tasks và Pipeline
kubectl apply -f tekton/task-git-clone.yaml
kubectl apply -f tekton/task-kaniko-build.yaml
kubectl apply -f tekton/task-git-update-manifest.yaml
kubectl apply -f tekton/pipeline.yaml

# Verify
kubectl get pvc,sa,task,pipeline
```

**Lưu ý**: Operator sẽ tự động tạo TriggerBinding, TriggerTemplate và EventListener khi bạn apply HeliosApp. Không cần apply manually.

### 5. Secrets

```bash
# Docker Hub
docker login -u <YOUR_DOCKERHUB_USERNAME>
kubectl create secret generic docker-credentials --from-file=.dockerconfigjson=$HOME/.docker/config.json --type=kubernetes.io/dockerconfigjson

# GitHub
kubectl create secret generic git-credentials --from-literal=username=<YOUR_GITHUB_USERNAME> --from-literal=password=<YOUR_GITHUB_TOKEN> --type=kubernetes.io/basic-auth
kubectl annotate secret git-credentials tekton.dev/git-0=https://github.com

# Webhook
kubectl create secret generic github-webhook-secret --from-literal=secretToken=$(openssl rand -base64 32)
```

### 6. GitOps Repo - Initial Manifest

⚠️ **QUAN TRỌNG**: Tạo manifest ban đầu (pipeline chỉ update, không tạo mới):

```bash
cd <your-gitops-repo>
mkdir -p apps/dev/<your-app-name>

cat > apps/dev/<your-app-name>/deployment.yaml << 'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: <your-deployment-name>     # VD: api, backend, frontend
spec:
  replicas: <initial-replicas>     # VD: 2, 3
  selector:
    matchLabels:
      app: <your-app-label>        # VD: api, backend
  template:
    metadata:
      labels:
        app: <your-app-label>
    spec:
      containers:
      - name: app
        image: docker.io/<your-dockerhub-username>/<your-image-name>:latest
        ports:
        - containerPort: <your-app-port>  # VD: 8080, 3000, 5000
        resources:
          requests:
            memory: '128Mi'
            cpu: '100m'
          limits:
            memory: '256Mi'
            cpu: '500m'
---
apiVersion: v1
kind: Service
metadata:
  name: <your-service-name>        # VD: api-service, backend-svc
spec:
  selector:
    app: <your-app-label>
  ports:
  - port: 80
    targetPort: <your-app-port>
EOF

git add . && git commit -m "Init manifest for <your-app-name>" && git push
```

**Ví dụ cụ thể**:

```bash
cd gitops-manifests
mkdir -p apps/dev/nodejs-api

cat > apps/dev/nodejs-api/deployment.yaml << 'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: api
  template:
    metadata:
      labels:
        app: api
    spec:
      containers:
      - name: app
        image: docker.io/john/nodejs-api:latest
        ports:
        - containerPort: 3000
        resources:
          requests:
            memory: '128Mi'
            cpu: '100m'
          limits:
            memory: '256Mi'
            cpu: '500m'
---
apiVersion: v1
kind: Service
metadata:
  name: api-service
spec:
  selector:
    app: api
  ports:
  - port: 80
    targetPort: 3000
EOF

git add . && git commit -m "Init manifest for nodejs-api" && git push
```

### 7. Deploy

```yaml
# <your-app-name>.yaml (ví dụ: nodejs-api.yaml, python-service.yaml)
apiVersion: platform.helios.io/v1
kind: HeliosApp
metadata:
  name: <your-app-name> # VD: nodejs-api, python-service
  namespace: default
spec:
  gitRepo: "https://github.com/<your-github-username>/<your-source-repo>"
  gitBranch: "main"
  imageRepo: "docker.io/<your-dockerhub-username>/<your-image-name>"
  port: <your-app-port> # VD: 8080, 3000, 5000
  replicas: <desired-replicas> # VD: 2, 3, 5
  pipelineName: "from-code-to-cluster"
  serviceAccount: "pipeline-sa"
  webhookSecret: "github-webhook-secret"
  pvcName: "shared-workspace-pvc"
  gitopsRepo: "https://github.com/<your-github-username>/<your-gitops-repo>"
  gitopsPath: "apps/dev/<your-app-name>" # phải khớp với folder trong GitOps repo
  contextSubpath: "" # "" nếu Dockerfile ở root, "backend/" nếu ở subfolder
  values:
    environment: "dev"
```

**Ví dụ cụ thể**:

```yaml
# nodejs-api.yaml
apiVersion: platform.helios.io/v1
kind: HeliosApp
metadata:
  name: nodejs-api
  namespace: default
spec:
  gitRepo: "https://github.com/john/my-nodejs-app"
  gitBranch: "main"
  imageRepo: "docker.io/john/nodejs-api"
  port: 3000
  replicas: 3
  pipelineName: "from-code-to-cluster"
  serviceAccount: "pipeline-sa"
  webhookSecret: "github-webhook-secret"
  pvcName: "shared-workspace-pvc"
  gitopsRepo: "https://github.com/john/gitops-manifests"
  gitopsPath: "apps/dev/nodejs-api"
  contextSubpath: ""
  values:
    environment: "dev"
```

```bash
kubectl apply -f <your-app-name>.yaml
kubectl get heliosapp -A -w
kubectl get pr -A -w
kubectl get application -A -w
```

### 8. Webhook (Tùy chọn)

```bash
# Port forward EventListener (tên service do operator tạo)
# Format: el-<your-app-name>-el
# VD: nếu HeliosApp name là "nodejs-api", service sẽ là "el-nodejs-api-el"
kubectl port-forward svc/el-<your-app-name>-el 8080:8080

# Trong terminal khác chạy ngrok
ngrok http 8080

# GitHub Settings → Webhooks (vào repo source code của bạn):
# URL: https://<YOUR_NGROK_URL>/
# Content type: application/json
# Secret: kubectl get secret github-webhook-secret -o jsonpath='{.data.secretToken}' | base64 -d
# Events: Just the push event
```

**Ví dụ cụ thể**:

```bash
# Nếu HeliosApp name là "nodejs-api"
kubectl port-forward svc/el-nodejs-api-el 8080:8080

# Ngrok URL sẽ giống: https://abc123.ngrok.io
# Paste URL này vào GitHub webhook
```

**Lưu ý phân bổ port**:

- Port **8080**: EventListener webhook (dùng với ngrok)
- Port **8443**: ArgoCD UI

## 📊 Kiểm Tra

### ArgoCD UI (Khuyến nghị xem trước)

```bash
# Lấy password ArgoCD admin
kubectl get secret argocd-initial-admin-secret -n argocd -o jsonpath='{.data.password}' | base64 -d
# Lưu password này lại

# Port forward ArgoCD UI (dùng port 8443 để tránh conflict)
kubectl port-forward svc/argocd-server -n argocd 8443:443

# Mở browser: https://localhost:8443
# Username: admin
# Password: (password từ lệnh trên)
```

**Lưu ý**:

- Browser sẽ cảnh báo certificate không an toàn, click "Advanced" → "Proceed" (an toàn cho local)
- Dùng port **8443** cho ArgoCD UI
- Port **8080** để dành cho EventListener webhook

Trong ArgoCD UI, bạn sẽ thấy:

- Application name: `<your-app-name>-argocd`
- Sync status: Synced (màu xanh)
- Health status: Healthy (màu xanh)
- Resources: Deployment, Service, Pods

### Quick Status Check

```bash
# QUAN TRỌNG:
# - <your-app-name> = tên HeliosApp (metadata.name trong HeliosApp YAML)
# - <your-deployment-name> = tên deployment trong GitOps manifest (metadata.name)
# - <your-app-label> = label trong pod template (spec.template.metadata.labels.app)
APP_NAME=<your-app-name>
DEPLOYMENT_NAME=<your-deployment-name>
APP_LABEL=<your-app-label>

echo "=== HeliosApp Status ==="
kubectl get heliosapp $APP_NAME -o jsonpath='{.status.conditions[0]}' | jq '.'

echo -e "\n=== ArgoCD Application ==="
kubectl get application -n argocd ${APP_NAME}-argocd -o jsonpath='Sync: {.status.sync.status}, Health: {.status.health.status}{"\n"}'

echo -e "\n=== Deployment ==="
kubectl get deploy $DEPLOYMENT_NAME -o jsonpath='replicas={.spec.replicas} image={.spec.template.spec.containers[0].image}{"\n"}'

echo -e "\n=== Pods ==="
kubectl get pods -l app=$APP_LABEL

echo -e "\n=== PipelineRun ==="
kubectl get pr --sort-by=.metadata.creationTimestamp | tail -3
```

**Ví dụ cụ thể**:

```bash
# Case 1: HeliosApp "nodejs-api", deployment "api", pod label "api"
APP_NAME=nodejs-api
DEPLOYMENT_NAME=api
APP_LABEL=api

# Case 2: HeliosApp "my-nodejs-app", deployment "helios", pod label "helios"
APP_NAME=my-nodejs-app
DEPLOYMENT_NAME=helios
APP_LABEL=helios

echo "=== HeliosApp Status ==="
kubectl get heliosapp $APP_NAME -o jsonpath='{.status.conditions[0]}' | jq '.'
# ...tiếp tục các lệnh trên
```

**Troubleshooting**:

- Nếu `replicas= image=` (trống) → `DEPLOYMENT_NAME` sai

  ```bash
  # Tìm deployment name thực tế
  kubectl get deploy -n default
  ```

- Nếu "NotFound" → `APP_NAME` sai, check `kubectl get heliosapp` để xem tên đúng

### Detailed Checks

```bash
# HeliosApp status
kubectl get heliosapp <your-app-name> -o jsonpath='{.status.conditions[0]}' | jq '.'

# ArgoCD sync (phải Synced/Healthy)
kubectl get application -n argocd <your-app-name>-argocd -o jsonpath='{.status.sync.status}/{.status.health.status}'

# List deployments để tìm tên chính xác
kubectl get deploy -n default

# Deployment details (thay <your-deployment-name> bằng tên thực tế)
kubectl get deploy <your-deployment-name> -o jsonpath='replicas={.spec.replicas} image={.spec.template.spec.containers[0].image} port={.spec.template.spec.containers[0].ports[0].containerPort}'

# EventListener service (cho webhook)
kubectl get svc -l eventlistener | grep el-

# Pods (phải có số lượng = replicas)
kubectl get pods -l app=<your-app-label>

# Xem logs pod (nếu cần debug)
kubectl logs -l app=<your-app-label> --tail=50
```

### Verify Deployment Thành Công

Deployment được coi là thành công khi:

1. ✅ **PipelineRun**: `SUCCEEDED = True, REASON = Succeeded`
2. ✅ **ArgoCD Application**: `Sync: Synced, Health: Healthy`
3. ✅ **Pods**: Số pods Running = replicas trong spec
4. ✅ **Image**: Pods chạy image mới nhất với SHA digest

**Lưu ý**: HeliosApp status có thể vẫn là `"status": "False"` do bug nhỏ trong operator, nhưng không ảnh hưởng nếu ArgoCD và Pods đã OK.

## 🔄 Update

```bash
# Cập nhật replicas
kubectl patch heliosapp <your-app-name> --type merge -p '{"spec":{"replicas":<new-replicas>}}'

# Cập nhật port
kubectl patch heliosapp <your-app-name> --type merge -p '{"spec":{"port":<new-port>}}'

# Hoặc push code → webhook tự trigger
```

**Ví dụ**:

```bash
kubectl patch heliosapp nodejs-api --type merge -p '{"spec":{"replicas":5}}'
```

## 🗑️ Xóa

```bash
kubectl delete heliosapp <your-app-name>
kubectl delete application -n argocd <your-app-name>-argocd
kubectl delete all -l app=<your-app-label>
```

**Ví dụ**:

```bash
kubectl delete heliosapp nodejs-api
kubectl delete application -n argocd nodejs-api-argocd
kubectl delete all -l app=api
```

## 💡 Lưu Ý

- **ArgoCD Sync**: Poll Git mỗi 30s-3min (không instant), xem UI để track real-time
- **Operator Auto-sync**: Tự động sync `replicas` & `port` từ HeliosApp spec vào GitOps manifest
- **contextSubpath**: Set nếu Dockerfile không ở root repo (VD: `backend/`, `services/api/`)
- **Pipeline Behavior**: Chỉ update `image:` tag/digest, không tạo file manifest mới
- **Single-node Cluster**: Phải disable Tekton affinity assistant (xem bước 2)
- **Label Matching**: `APP_LABEL` phải khớp với `spec.template.metadata.labels.app` trong deployment manifest GitOps
- **HeliosApp Status Bug**: Status có thể stuck ở "DeployingWithArgoCD" nhưng deployment vẫn OK nếu ArgoCD Synced/Healthy
- **Image Cache**: Luôn xóa image cũ trong minikube trước khi load image mới để tránh dùng cache

```bash
# Check PipelineRun có succeed không
kubectl get pr --sort-by=.metadata.creationTimestamp | tail -3

# Check operator logs
kubectl logs -n helios-operator-system deployment/helios-operator-controller-manager | grep -i argocd
```

## 📚 Additional Documentation

- [Architecture](./ARCHITECTURE.md)
- [API Reference](./API_REFERENCE.md)
- [Production](./PRODUCTION_DEPLOYMENT.md)
- [Troubleshooting](./TROUBLESHOOTING.md)
