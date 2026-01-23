# Troubleshooting Guide

Common issues and solutions when using Helios Operator.

## Table of Contents

- [Deployment Issues](#deployment-issues)
- [Tekton Issues](#tekton-issues)
- [ArgoCD Issues](#argocd-issues)
- [Operator Issues](#operator-issues)
- [Network Issues](#network-issues)
- [Permission Issues](#permission-issues)
- [Debugging Tools](#debugging-tools)

## Deployment Issues

### HeliosApp Stuck in Pending

**Symptoms**:

```bash
$ kubectl get heliosapp my-app
NAME     PHASE     AGE
my-app   Pending   10m
```

**Diagnosis**:

```bash
kubectl describe heliosapp my-app
kubectl get events --field-selector involvedObject.name=my-app
```

**Common Causes**:

#### Cause 1: Pipeline does not exist

```bash
# Check
kubectl get pipeline

# Fix
kubectl apply -f tekton/pipeline.yaml
```

#### Cause 2: ServiceAccount missing credentials

```bash
# Check
kubectl get sa helios-tekton-sa -o yaml

# Fix: Add git secret
kubectl create secret generic git-ssh-key \
  --from-file=ssh-privatekey=$HOME/.ssh/id_rsa
kubectl patch sa helios-tekton-sa -p '{"secrets": [{"name": "git-ssh-key"}]}'
```

#### Cause 3: CRD not installed

```bash
# Check
kubectl get crd heliosapps.platform.helios.io

# Fix
make install
```

### HeliosApp Stuck in ManifestGenerationInProgress

**Symptoms**: PipelineRun not running or stuck.

**Diagnosis**:

```bash
# Find PipelineRun
kubectl get pipelinerun -l heliosapp=my-app

# Get logs
tkn pipelinerun logs <pipelinerun-name> -f

# Describe
kubectl describe pipelinerun <pipelinerun-name>
```

**Common Causes**:

#### Cause 1: PipelineRun pending due to insufficient resources

```bash
# Check pod status
kubectl get pods -l tekton.dev/pipelineRun=<name>

# Fix: Add more nodes or adjust resource requests
```

#### Cause 2: Git clone failed

```text
Error: Authentication failed
```

**Fix**: Check SSH key:

```bash
# Test SSH key manually
ssh -T git@github.com

# Recreate secret
kubectl delete secret git-ssh-key
kubectl create secret generic git-ssh-key \
  --from-file=ssh-privatekey=$HOME/.ssh/id_rsa \
  --type=kubernetes.io/ssh-auth

kubectl annotate secret git-ssh-key tekton.dev/git-0=github.com
```

#### Cause 3: Container image build failed

```text
Error: failed to push image
```

**Fix**: Check registry credentials:

```bash
# Create docker-registry secret
kubectl create secret docker-registry docker-creds \
  --docker-server=ghcr.io \
  --docker-username=<username> \
  --docker-password=<token>

# Patch ServiceAccount
kubectl patch sa helios-tekton-sa \
  -p '{"secrets": [{"name": "docker-creds"}]}'
```

### HeliosApp in ManifestGenerationFailed

**Diagnosis**:

```bash
kubectl get heliosapp my-app -o jsonpath='{.status.conditions}'
```

**Fix**: Check PipelineRun logs to find exact error:

```bash
tkn pipelinerun logs <name> -f
```

Common errors:

- **Template not found**: Check `spec.templateRepo` and `spec.templatePath`
- **Values invalid**: Check `spec.values` format
- **Git commit failed**: Check write permissions for gitops repo

## Tekton Issues

### EventListener Not Accessible

**Symptoms**: Webhook does not trigger PipelineRun.

**Diagnosis**:

```bash
# Check EventListener
kubectl get eventlistener

# Check Service
kubectl get svc -l eventlistener=my-app-listener

# Check logs
kubectl logs -l eventlistener=my-app-listener
```

**Fix**:

**Step 1: Expose EventListener**:

```bash
# Port-forward for testing
kubectl port-forward svc/el-my-app-listener 8080:8080

# Test webhook
curl -X POST http://localhost:8080 \
  -H "Content-Type: application/json" \
  -d '{"ref": "refs/heads/main"}'
```

**Step 2: Production: Use Ingress**:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-app-webhook
spec:
  rules:
    - host: webhook.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: el-my-app-listener
                port:
                  number: 8080
```

### PipelineRun Timeout

**Symptoms**: PipelineRun stuck after 1 hour.

**Fix**: Increase timeout:

```yaml
spec:
  pipelineParams:
    timeout: "2h" # Default: 1h
```

### Task Pod Crashes

**Diagnosis**:

```bash
kubectl get pods -l tekton.dev/pipelineRun=<name>
kubectl logs <pod-name> -c step-<step-name>
```

**Common Issues**:

- Out of memory: Increase `resources.limits.memory`
- Disk space: Use larger PersistentVolume
- Network timeout: Check firewall rules

## ArgoCD Issues

### ArgoCD Application Not Created

**Symptoms**: Phase stuck at `DeployingWithArgoCD`.

**Diagnosis**:

```bash
# Check if ArgoCD installed
kubectl get pods -n argocd

# Check Application
kubectl get application -n argocd

# Operator logs
kubectl logs -n helios-system deployment/helios-operator-controller-manager -f
```

**Fix**:

**Step 1: Install ArgoCD**:

```bash
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
```

**Step 2: Check RBAC**:

```bash
# Operator needs permissions to create Applications
kubectl describe clusterrole helios-operator-role
```

### ArgoCD Application OutOfSync

**Symptoms**: `syncStatus: OutOfSync` despite auto-sync enabled.

**Diagnosis**:

```bash
# Check Application status
kubectl get application -n argocd my-app -o yaml

# ArgoCD UI
kubectl port-forward svc/argocd-server -n argocd 8080:443
```

**Causes**:

#### Cause 1: Sync policy incorrect

**Fix**: Verify HeliosApp spec:

```yaml
spec:
  autoSync: true
  prune: true
  selfHeal: true
```

#### Cause 2: Manifest changes not committed

Check GitOps repo:

```bash
cd /path/to/gitops-repo
git pull
ls environments/dev/my-app/
```

#### Cause 3: Manual sync needed

```bash
# Via kubectl
kubectl patch application my-app -n argocd \
  --type merge \
  -p '{"operation": {"sync": {}}}'

# Via argocd CLI
argocd app sync my-app
```

### ArgoCD Application Degraded

**Symptoms**: `healthStatus: Degraded`.

**Diagnosis**:

```bash
# Check Application resources
kubectl get application -n argocd my-app -o jsonpath='{.status.resources}'

# Check deployed resources
kubectl get all -n <target-namespace> -l app.kubernetes.io/instance=my-app
```

**Common Causes**:

- Pod CrashLoopBackOff
- ImagePullBackOff
- Insufficient resources
- Config errors

**Fix**: Check deployed resource logs:

```bash
kubectl logs -n <namespace> deployment/<name>
kubectl describe pod -n <namespace> <pod-name>
```

## Operator Issues

### Operator Not Running

**Diagnosis**:

```bash
kubectl get pods -n helios-system
kubectl logs -n helios-system deployment/helios-operator-controller-manager
```

**Common Causes**:

#### Cause 1: CRD not installed

```bash
make install
```

#### Cause 2: Image pull failed

```bash
kubectl describe pod -n helios-system <pod-name>
```

**Fix**: Check image name and registry credentials.

#### Cause 3: Resource limits

```bash
# Increase limits in config/manager/manager.yaml
resources:
  limits:
    cpu: 500m
    memory: 512Mi
```

### Reconciliation Loop

**Symptoms**: Operator continuously reconciling cùng một HeliosApp.

**Diagnosis**:

```bash
# Check logs
kubectl logs -n helios-system deployment/helios-operator-controller-manager -f | grep "Reconciling"

# Check status
kubectl get heliosapp my-app -o jsonpath='{.status}'
```

**Fix**:

**Step 1: Add pause annotation**:

```bash
kubectl annotate heliosapp my-app helios.io/pause-reconciliation=true
```

**Step 2: Check for stuck conditions**:

```bash
kubectl get heliosapp my-app -o yaml
```

Look for incorrect `observedGeneration` or error conditions.

### Leader Election Issues

**Symptoms**: Multiple operator replicas active simultaneously.

**Diagnosis**:

```bash
kubectl get lease -n helios-system
kubectl logs -n helios-system deployment/helios-operator-controller-manager -f | grep "leader"
```

**Fix**: Ensure only 1 replica:

```bash
kubectl scale deployment -n helios-system helios-operator-controller-manager --replicas=1
```

## Network Issues

### Cannot Reach Git Repository

**Diagnosis**:

```bash
# From PipelineRun pod
kubectl exec -it <pod-name> -c step-git-clone -- sh
git clone <repo-url>
```

**Fix**:

**Step 1: Check network policies**:

```bash
kubectl get networkpolicy
```

**Step 2: Check DNS**:

```bash
kubectl run -it --rm debug --image=busybox --restart=Never -- nslookup github.com
```

**Step 3: Check proxy settings**:

If using corporate proxy:

```yaml
# Add to Pipeline
env:
  - name: HTTP_PROXY
    value: "http://proxy.example.com:8080"
  - name: HTTPS_PROXY
    value: "http://proxy.example.com:8080"
```

### Cannot Reach Container Registry

**Fix**: Add imagePullSecrets:

```bash
kubectl create secret docker-registry regcred \
  --docker-server=<registry> \
  --docker-username=<username> \
  --docker-password=<password>
```

## Permission Issues

### RBAC Errors

**Symptoms**: "forbidden: User cannot..."

**Diagnosis**:

```bash
# Check ServiceAccount
kubectl describe sa helios-tekton-sa

# Check RoleBindings
kubectl get rolebinding,clusterrolebinding -A | grep helios
```

**Fix**: Apply correct RBAC:

```bash
kubectl apply -f config/rbac/
```

### Git Access Denied

**Symptoms**: "Permission denied (publickey)" when git clone/push.

**Fix**:

**Step 1: Generate SSH key**:

```bash
ssh-keygen -t ed25519 -C "helios@example.com"
```

**Step 2: Add public key to GitHub**:

Settings → SSH Keys → Add SSH key

**Step 3: Create secret**:

```bash
kubectl create secret generic git-ssh-key \
  --from-file=ssh-privatekey=$HOME/.ssh/id_ed25519 \
  --type=kubernetes.io/ssh-auth
```

**Step 4: Annotate secret**:

```bash
kubectl annotate secret git-ssh-key tekton.dev/git-0=github.com
```

## Debugging Tools

### Essential Commands

```bash
# Get all Helios resources
kubectl get heliosapp -A

# Describe with events
kubectl describe heliosapp <name>

# Get status
kubectl get heliosapp <name> -o yaml

# Watch status changes
kubectl get heliosapp <name> -w

# Operator logs
kubectl logs -n helios-system deployment/helios-operator-controller-manager -f

# PipelineRun logs
tkn pipelinerun logs <name> -f

# ArgoCD Application status
kubectl get application -n argocd <name> -o yaml
```

### Advanced Debugging

**Enable debug logging**:

```bash
# Edit deployment
kubectl edit deployment -n helios-system helios-operator-controller-manager

# Add arg
args:
  - --zap-log-level=debug
```

**Port-forward for debugging**:

```bash
# Operator metrics
kubectl port-forward -n helios-system deployment/helios-operator-controller-manager 8080:8080

# Access metrics
curl http://localhost:8080/metrics

# ArgoCD UI
kubectl port-forward svc/argocd-server -n argocd 8081:443

# Access UI at https://localhost:8081
```

**Exec into pods**:

```bash
# Into Tekton task pod
kubectl exec -it <pod-name> -c step-<step> -- sh

# Into operator pod
kubectl exec -it -n helios-system <pod-name> -- sh
```

## Getting Help

### Check Logs

Always start with logs:

```bash
# Operator logs
kubectl logs -n helios-system deployment/helios-operator --tail=100

# Events
kubectl get events -A --sort-by='.lastTimestamp'

# Specific resource events
kubectl describe heliosapp <name>
```

### Collect Debug Info

```bash
# Dump all relevant info
kubectl get heliosapp <name> -o yaml > heliosapp.yaml
kubectl get pipelinerun -l heliosapp=<name> -o yaml > pipelinerun.yaml
kubectl get application -n argocd <name> -o yaml > application.yaml
kubectl logs -n helios-system deployment/helios-operator --tail=500 > operator.log
```

### Team Support

- 💬 Team chat: Contact via internal channels
- � Documentation: Check other guides in [docs](./README.md)
- � Team: Reach out to team members directly

## See Also

- [Getting Started](./GETTING_STARTED.md)
- [Architecture](./ARCHITECTURE.md)
- [Development Guide](./DEVELOPMENT.md)
