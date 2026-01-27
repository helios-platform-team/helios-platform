# Helios Platform: CI/CD Automation Guide

Tài liệu duy nhất bạn cần để kích hoạt và vận hành tính năng Automation (GitOps + Tekton + Webhook).

## 1. Kiến Trúc (How it works)
*   **Trigger**: Bạn push code lên GitHub -> Webhook gọi Ingress -> EventListener.
*   **Pipeline**: Tekton clone code -> Build Docker Image (Kaniko) -> Push Docker Hub.
*   **GitOps**: Pipeline update file manifest trong repo GitOps -> ArgoCD sync về cụm.

## 2. Setup Hạ Tầng (Làm 1 lần)

### 2.1. Cài Đặt Tool
```bash
# 1. Tekton
kubectl apply --filename https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
kubectl apply --filename https://storage.googleapis.com/tekton-releases/triggers/latest/release.yaml
kubectl apply --filename https://storage.googleapis.com/tekton-releases/triggers/latest/interceptors.yaml

# 2. ArgoCD
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# 3. Helios Operator Resources
cd apps/operator
kubectl apply -f tekton/pvc.yaml
kubectl apply -f tekton/
```

### 2.2. Setup Secret (Quan Trọng Nhất)
Bạn chỉ cần tạo 2 secret. Hãy dùng lệnh trực tiếp, **không dùng script**.

**A. Docker Hub Secret** (Để push image)
```bash
# Thay USERNAME và ACCESS_TOKEN
kubectl create secret docker-registry docker-credentials \
  --docker-server=https://index.docker.io/v1/ \
  --docker-username=YOUR_DOCKER_USERNAME \
  --docker-password=YOUR_ACCESS_TOKEN \
  --docker-email=your-email@example.com
```

**B. GitHub Secret** (Để clone/push code)
```bash
# Thay USERNAME và PAT (Scope: repo)
# Lưu ý: secretToken là mật khẩu bạn tự nghĩ ra để bảo vệ Webhook
kubectl create secret generic github-credentials \
  --from-literal=username=YOUR_GITHUB_USERNAME \
  --from-literal=password=YOUR_GITHUB_PAT \
  --from-literal=secretToken=YOUR_SECRET_TOKEN \
  --type=kubernetes.io/basic-auth

# Annotate để Tekton nhận diện
kubectl annotate secret github-credentials tekton.dev/git-0=https://github.com --overwrite

# Link vào ServiceAccount
# Cách 1: PowerShell (Khuyên dùng trên Windows)
@"
secrets:
  - name: github-credentials
"@ | kubectl patch serviceaccount default --patch-file /dev/stdin

# Cách 2: Git Bash / Linux / Mac
```bash
 kubectl patch serviceaccount default -p '{"secrets": [{"name": "github-credentials"}]}'
```

### 2.3. Cấp Quyền cho Tekton Triggers (Bắt buộc)
Mặc định ServiceAccount của Triggers không đọc được Secret, dẫn đến lỗi 400. Chạy lệnh sau để fix:

```bash
kubectl create clusterrolebinding tekton-triggers-sa-admin \
  --clusterrole=cluster-admin \
  --serviceaccount=default:tekton-triggers-sa
```

## 3. Vận Hành (Production Flow)

### 3.1. Chạy Operator
```bash
make install
make run
```

### 3.2. Deploy App
Update file `config/samples/app_v1alpha1_heliosapp.yaml` với repo thật của bạn, sau đó:
```bash
kubectl apply -f config/samples/app_v1alpha1_heliosapp.yaml
```

### 3.3. Expose Webhook ra Internet (Dùng Ngrok)
Do chạy Minikube trên Windows, Ingress rất khó config. Hãy dùng cách **Port-Forward** này cho ổn định:

**Bước 1: Port-Forward Service ra localhost** (Giữ terminal này chạy)
```bash
kubectl port-forward svc/el-sample-app-listener 8080:8080
```

**Bước 2: Chạy Ngrok** (Ở terminal khác)
```bash
ngrok http 8080
```
-> Copy domain ngrok (Ví dụ: `https://abcd-123.ngrok-free.app`).

### 3.4. Config Webhook trên GitHub
Vào repo GitHub (Source Code) -> Settings -> Webhooks -> Add:
*   **Payload URL**: `https://[YOUR-NGROK-DOMAIN]/hooks/sample-app`
*   **Content type**: `application/json`
*   **Secret**: `YOUR_SECRET_TOKEN` (Giá trị secretToken trong secret `github-credentials`, ví dụ: `123456`)
*   **Event**: Just the push event

### 3.5. Test
Push code lên branch `main`.
Kiểm tra Pipeline chạy:
```bash
kubectl get pr -w
```


## 4. Troubleshooting
| Lỗi | Nguyên Nhân | Cách Fix |
|---|---|---|
| Pipeline `401 Unauthorized` | Sai Token Docker | Re-create secret `docker-credentials` |
| Git Clone `Auth failed` | Thiếu Annotation | Chạy lệnh `kubectl annotate` cho secret git |
| Operator GitOps fail | Sai tên secret | Đảm bảo `app_v1alpha1_heliosapp.yaml` trỏ đúng secret name |
