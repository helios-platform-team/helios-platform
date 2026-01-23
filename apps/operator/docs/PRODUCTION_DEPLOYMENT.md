# Production Deployment Guide

Best practices and recommendations for deploying Helios Operator in production environment.

## Prerequisites

### Infrastructure Requirements

**Kubernetes Cluster**:

- Version: 1.34+ (stable)
- Nodes: Minimum 3 nodes for high availability
- Storage: Persistent storage class available
- Network: CNI plugin installed (Calico, Cilium, etc.)

**Resource Requirements**:

| Component        | CPU     | Memory | Storage  |
| ---------------- | ------- | ------ | -------- |
| Helios Operator  | 200m    | 256Mi  | N/A      |
| Tekton Pipelines | 500m    | 512Mi  | 10Gi PVC |
| ArgoCD           | 1000m   | 2Gi    | 10Gi PVC |
| Total (minimum)  | 2 cores | 4Gi    | 50Gi     |

**Network**:

- Ingress controller installed
- TLS certificates for webhooks
- Egress access to Git repositories
- Container registry access

## Installation

### 1. Prepare Namespace

```bash
# Create namespace
kubectl create namespace helios-system

# Label for monitoring
kubectl label namespace helios-system monitoring=enabled
```

### 2. Install Dependencies

**Tekton Pipelines**:

```bash
kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
kubectl apply -f https://storage.googleapis.com/tekton-releases/triggers/latest/release.yaml
kubectl apply -f https://storage.googleapis.com/tekton-releases/triggers/latest/interceptors.yaml
```

**ArgoCD**:

```bash
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# Wait for ready
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=argocd-server -n argocd --timeout=300s
```

### 3. Configure Secrets

**Git SSH Key**:

```bash
# Generate dedicated key for production
ssh-keygen -t ed25519 -f helios-prod-key -C "helios-prod@example.com" -N ""

# Create secret
kubectl create secret generic git-ssh-credentials \
  --from-file=ssh-privatekey=./helios-prod-key \
  --type=kubernetes.io/ssh-auth \
  -n helios-system

# Annotate for Tekton
kubectl annotate secret git-ssh-credentials \
  tekton.dev/git-0=github.com \
  -n helios-system
```

**Container Registry Credentials**:

```bash
kubectl create secret docker-registry registry-credentials \
  --docker-server=ghcr.io \
  --docker-username=$GITHUB_USERNAME \
  --docker-password=$GITHUB_TOKEN \
  -n helios-system
```

### 4. Deploy Operator

**Option A: Helm (Recommended)**:

```bash
helm repo add helios https://charts.helios.io
helm install helios-operator helios/helios-operator \
  --namespace helios-system \
  --set image.tag=v1.0.0 \
  --set resources.limits.cpu=500m \
  --set resources.limits.memory=512Mi \
  --set metrics.enabled=true \
  --set webhook.enabled=true
```

**Option B: Kubectl**:

```bash
kubectl apply -f https://github.com/your-org/helios-operator/releases/download/v1.0.0/install.yaml
```

### 5. Verify Installation

```bash
# Check operator pods
kubectl get pods -n helios-system

# Check CRD
kubectl get crd heliosapps.platform.helios.io

# Check webhooks
kubectl get validatingwebhookconfiguration,mutatingwebhookconfiguration | grep helios
```

## High Availability

### Operator HA

Deploy multiple replicas with leader election:

```yaml
# values.yaml for Helm
podDisruptionBudget:
  enabled: true
  minAvailable: 1

affinity:
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 100
        podAffinityTerm:
          labelSelector:
            matchLabels:
              app: helios-operator
          topologyKey: kubernetes.io/hostname
```

### ArgoCD HA

```bash
# Install HA version
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/ha/install.yaml
```

Configure:

```yaml
# argocd-cm ConfigMap
data:
  application.instanceLabelKey: argocd.argoproj.io/instance
  application.resourceTrackingMethod: annotation
```

### Tekton HA

Tekton controller automatically handles HA. Ensure:

- Multiple worker nodes available
- PersistentVolumes spread across zones

## Security

### RBAC Hardening

**Principle of Least Privilege**:

```yaml
# Operator ServiceAccount
apiVersion: v1
kind: ServiceAccount
metadata:
  name: helios-operator
  namespace: helios-system

---
# Minimal ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: helios-operator-role
rules:
  # Only what's needed
  - apiGroups: ["platform.helios.io"]
    resources: ["heliosapps"]
    verbs: ["get", "list", "watch", "update", "patch"]
  - apiGroups: ["tekton.dev"]
    resources: ["pipelineruns"]
    verbs: ["get", "list", "watch", "create"]
  - apiGroups: ["triggers.tekton.dev"]
    resources: ["eventlisteners", "triggerbindings", "triggertemplates"]
    verbs: ["get", "list", "watch", "create", "update", "patch"]
  - apiGroups: ["argoproj.io"]
    resources: ["applications"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
```

### Network Policies

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: helios-operator-netpol
  namespace: helios-system
spec:
  podSelector:
    matchLabels:
      app: helios-operator
  policyTypes:
    - Ingress
    - Egress
  ingress:
    # Allow metrics scraping
    - from:
        - namespaceSelector:
            matchLabels:
              name: monitoring
      ports:
        - protocol: TCP
          port: 8080
    # Allow webhooks
    - from:
        - namespaceSelector: {}
      ports:
        - protocol: TCP
          port: 9443
  egress:
    # Kubernetes API
    - to:
        - namespaceSelector: {}
          podSelector:
            matchLabels:
              component: apiserver
      ports:
        - protocol: TCP
          port: 6443
    # DNS
    - to:
        - namespaceSelector:
            matchLabels:
              name: kube-system
          podSelector:
            matchLabels:
              k8s-app: kube-dns
      ports:
        - protocol: UDP
          port: 53
```

### Pod Security Standards

```yaml
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: helios-operator-psp
spec:
  privileged: false
  allowPrivilegeEscalation: false
  requiredDropCapabilities:
    - ALL
  volumes:
    - "configMap"
    - "emptyDir"
    - "projected"
    - "secret"
    - "downwardAPI"
  hostNetwork: false
  hostIPC: false
  hostPID: false
  runAsUser:
    rule: "MustRunAsNonRoot"
  seLinux:
    rule: "RunAsAny"
  fsGroup:
    rule: "RunAsAny"
  readOnlyRootFilesystem: true
```

### Secret Management

**Use External Secret Managers**:

```yaml
# Example with External Secrets Operator
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: git-credentials
  namespace: helios-system
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: vault-backend
    kind: SecretStore
  target:
    name: git-ssh-credentials
    template:
      type: kubernetes.io/ssh-auth
  data:
    - secretKey: ssh-privatekey
      remoteRef:
        key: helios/git-ssh-key
```

## Monitoring

### Prometheus Setup

```yaml
# ServiceMonitor
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: helios-operator
  namespace: helios-system
spec:
  selector:
    matchLabels:
      app: helios-operator
  endpoints:
    - port: metrics
      interval: 30s
```

### Alerts

```yaml
# PrometheusRule
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: helios-operator-alerts
  namespace: helios-system
spec:
  groups:
    - name: helios-operator
      interval: 30s
      rules:
        - alert: HeliosOperatorDown
          expr: up{job="helios-operator"} == 0
          for: 5m
          labels:
            severity: critical
          annotations:
            summary: Helios Operator is down
```

See [Monitoring Guide](./MONITORING.md) for complete setup.

## Backup & Disaster Recovery

### Backup Strategy

**What to Backup**:

1. HeliosApp CRs
2. Tekton Pipelines & Tasks
3. ArgoCD Applications
4. Secrets (encrypted)
5. Operator configuration

**Using Velero**:

```bash
# Install Velero
velero install \
  --provider aws \
  --bucket helios-backups \
  --secret-file ./credentials-velero

# Create backup schedule
velero schedule create helios-daily \
  --schedule="0 2 * * *" \
  --include-namespaces helios-system,argocd \
  --ttl 720h
```

### Restore Procedure

```bash
# List backups
velero backup get

# Restore
velero restore create --from-backup helios-daily-20240115

# Verify
kubectl get heliosapp -A
```

## Scaling

### Horizontal Scaling

**Operator**: Automatically scales via leader election (max 2-3 replicas recommended).

**Tekton**: Scales based on PipelineRun load:

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: tekton-pipelines-controller
  namespace: tekton-pipelines
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: tekton-pipelines-controller
  minReplicas: 2
  maxReplicas: 5
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
```

### Vertical Scaling

Adjust resources based on load:

```yaml
resources:
  requests:
    cpu: 200m
    memory: 256Mi
  limits:
    cpu: 1000m
    memory: 1Gi
```

### Cluster Scaling

Use Cluster Autoscaler:

```yaml
apiVersion: autoscaling.k8s.io/v1
kind: ClusterAutoscaler
metadata:
  name: cluster-autoscaler
spec:
  scaleDown:
    enabled: true
    delayAfterAdd: 10m
  scaleUp:
    enabled: true
```

## Performance Tuning

### Operator Tuning

```yaml
# Increase reconciliation workers
args:
  - --max-concurrent-reconciles=5
# Adjust requeue delays
# In code: ctrl.Result{RequeueAfter: 30 * time.Second}
```

### Tekton Tuning

```yaml
# tekton-config ConfigMap
data:
  default-timeout-minutes: "60"
  default-service-account: "helios-tekton-sa"
  default-task-run-workspace-binding: |
    emptyDir: {}
```

### ArgoCD Tuning

```yaml
# argocd-cm ConfigMap
data:
  timeout.reconciliation: "180s"
  application.instanceLabelKey: argocd.argoproj.io/instance
```

## Upgrades

### Pre-Upgrade Checklist

- [ ] Backup all resources
- [ ] Review changelog
- [ ] Test in staging environment
- [ ] Schedule maintenance window
- [ ] Notify stakeholders

### Upgrade Procedure

```bash
# Backup
velero backup create pre-upgrade-backup --include-namespaces helios-system,argocd

# Upgrade CRD (if changed)
kubectl apply -f https://github.com/your-org/helios-operator/releases/download/v1.1.0/crd.yaml

# Upgrade operator
helm upgrade helios-operator helios/helios-operator \
  --namespace helios-system \
  --version 1.1.0 \
  --reuse-values

# Verify
kubectl rollout status deployment/helios-operator -n helios-system
kubectl get heliosapp -A
```

### Rollback

```bash
# Rollback Helm release
helm rollback helios-operator -n helios-system

# Or restore from backup
velero restore create --from-backup pre-upgrade-backup
```

## Cost Optimization

### Resource Optimization

- Set appropriate resource limits
- Use node selectors for cheaper nodes
- Enable cluster autoscaling
- Use spot instances for non-critical workloads

### Pipeline Optimization

- Cache Docker layers
- Reuse workspaces
- Parallel task execution
- Prune old PipelineRuns

```yaml
# Cleanup CronJob
apiVersion: batch/v1
kind: CronJob
metadata:
  name: cleanup-pipelineruns
  namespace: helios-system
spec:
  schedule: "0 0 * * *"
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: cleanup-sa
          containers:
            - name: cleanup
              image: bitnami/kubectl:latest
              command:
                - /bin/sh
                - -c
                - |
                  kubectl delete pipelinerun \
                    --field-selector status.completionTime<$(date -d '7 days ago' -u +"%Y-%m-%dT%H:%M:%SZ") \
                    -A
          restartPolicy: OnFailure
```

## Compliance

### Audit Logging

Enable Kubernetes audit logging:

```yaml
# audit-policy.yaml
apiVersion: audit.k8s.io/v1
kind: Policy
rules:
  - level: RequestResponse
    resources:
      - group: "platform.helios.io"
        resources: ["heliosapps"]
```

### Image Scanning

Use admission controllers:

```yaml
# PolicyException using Kyverno
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: require-signed-images
spec:
  validationFailureAction: enforce
  rules:
    - name: verify-signature
      match:
        any:
          - resources:
              kinds:
                - Pod
      verifyImages:
        - imageReferences:
            - "ghcr.io/your-org/*"
          attestors:
            - entries:
                - keys:
                    publicKeys: |-
                      -----BEGIN PUBLIC KEY-----
                      ...
                      -----END PUBLIC KEY-----
```

## Troubleshooting in Production

See [Troubleshooting Guide](./TROUBLESHOOTING.md) for detailed instructions.

**Quick Commands**:

```bash
# Check operator health
kubectl get pods -n helios-system
kubectl logs -n helios-system deployment/helios-operator -f --tail=100

# Check HeliosApps
kubectl get heliosapp -A -o wide

# Check resources
kubectl top nodes
kubectl top pods -n helios-system
```

## Best Practices Summary

1. ✅ Use namespace isolation
2. ✅ Enable RBAC and Network Policies
3. ✅ Configure resource limits
4. ✅ Setup monitoring & alerting
5. ✅ Regular backups
6. ✅ Test upgrades in staging
7. ✅ Document runbooks
8. ✅ Enable audit logging
9. ✅ Use GitOps for configuration
10. ✅ Regular security scans

## See Also

- [Monitoring Guide](./MONITORING.md)
- [Troubleshooting Guide](./TROUBLESHOOTING.md)
- [Security Best Practices](https://kubernetes.io/docs/concepts/security/)
