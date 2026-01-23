# Monitoring & Observability

Guide to monitor Helios Operator and deployed applications.

## Overview

Helios Operator exposes metrics, logs, and status information for monitoring:

- Operator health & performance
- HeliosApp reconciliation status
- Tekton Pipeline execution
- ArgoCD deployment status
- Resource usage

## Metrics

### Prometheus Metrics

Operator exposes metrics at `:8080/metrics`.

#### Controller Metrics

| Metric                                      | Type      | Description                |
| ------------------------------------------- | --------- | -------------------------- |
| `controller_runtime_reconcile_total`        | Counter   | Total reconciliation calls |
| `controller_runtime_reconcile_errors_total` | Counter   | Failed reconciliations     |
| `controller_runtime_reconcile_time_seconds` | Histogram | Reconciliation duration    |
| `workqueue_depth`                           | Gauge     | Current queue depth        |
| `workqueue_adds_total`                      | Counter   | Total items added to queue |

#### Custom Metrics

| Metric                                | Type      | Description                                |
| ------------------------------------- | --------- | ------------------------------------------ |
| `helios_heliosapp_phase`              | Gauge     | Current phase by HeliosApp (0-6)           |
| `helios_pipelinerun_duration_seconds` | Histogram | PipelineRun execution time                 |
| `helios_pipelinerun_success_total`    | Counter   | Successful PipelineRuns                    |
| `helios_pipelinerun_failure_total`    | Counter   | Failed PipelineRuns                        |
| `helios_argocd_sync_status`           | Gauge     | ArgoCD sync status (0=OutOfSync, 1=Synced) |
| `helios_argocd_health_status`         | Gauge     | ArgoCD health (0=Unhealthy, 1=Healthy)     |

### Setup Prometheus

#### Option 1: Prometheus Operator

```bash
# Install Prometheus Operator
kubectl apply -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/main/bundle.yaml

# Create ServiceMonitor
kubectl apply -f config/prometheus/monitor.yaml
```

`config/prometheus/monitor.yaml`:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: helios-operator-metrics
  namespace: helios-system
spec:
  endpoints:
    - interval: 30s
      path: /metrics
      port: https
      scheme: https
      tlsConfig:
        insecureSkipVerify: true
  selector:
    matchLabels:
      control-plane: controller-manager
```

#### Option 2: Standalone Prometheus

`prometheus.yaml`:

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: "helios-operator"
    kubernetes_sd_configs:
      - role: pod
        namespaces:
          names:
            - helios-system
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_control_plane]
        action: keep
        regex: controller-manager
      - source_labels: [__meta_kubernetes_pod_container_port_name]
        action: keep
        regex: https
```

Deploy:

```bash
kubectl create configmap prometheus-config --from-file=prometheus.yaml -n monitoring
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: prometheus
  namespace: monitoring
spec:
  replicas: 1
  selector:
    matchLabels:
      app: prometheus
  template:
    metadata:
      labels:
        app: prometheus
    spec:
      containers:
        - name: prometheus
          image: prom/prometheus:latest
          args:
            - --config.file=/etc/prometheus/prometheus.yaml
          ports:
            - containerPort: 9090
          volumeMounts:
            - name: config
              mountPath: /etc/prometheus
      volumes:
        - name: config
          configMap:
            name: prometheus-config
EOF
```

### Grafana Dashboards

#### Install Grafana

```bash
helm repo add grafana https://grafana.github.io/helm-charts
helm install grafana grafana/grafana -n monitoring
```

Get admin password:

```bash
kubectl get secret -n monitoring grafana -o jsonpath="{.data.admin-password}" | base64 -d
```

#### Import Helios Dashboard

1. **Access Grafana**: `kubectl port-forward -n monitoring svc/grafana 3000:80`
2. **Add Prometheus datasource**: Configuration → Data Sources → Add Prometheus
3. **Import dashboard**: Dashboard → Import → Upload `dashboards/helios-operator.json`

#### Sample Dashboard Panels

**HeliosApp Phase Distribution**:

```promql
sum by (phase) (helios_heliosapp_phase)
```

**Reconciliation Rate**:

````promql

**Error Rate**:

```promql
rate(controller_runtime_reconcile_errors_total{controller="heliosapp"}[5m])
````

**PipelineRun Success Rate**:

```promql
sum(rate(helios_pipelinerun_success_total[5m])) /
sum(rate(helios_pipelinerun_success_total[5m]) + rate(helios_pipelinerun_failure_total[5m]))
```

**ArgoCD Sync Status**:

```promql
helios_argocd_sync_status
```

## Logging

### Structured Logging

Operator uses structured logging with `logr`:

```json
{
  "level": "info",
  "ts": "2024-01-15T10:30:00.000Z",
  "logger": "heliosapp-controller",
  "msg": "Creating Tekton Triggers",
  "heliosapp": "default/my-app",
  "namespace": "default",
  "name": "my-app"
}
```

### Log Levels

- **debug**: Detailed troubleshooting
- **info**: Normal operations (default)
- **error**: Errors requiring attention

Change log level:

```bash
# Edit deployment
kubectl edit deployment -n helios-system helios-operator-controller-manager

# Add arg
args:
  - --zap-log-level=debug
```

### Log Aggregation

#### Option 1: EFK Stack (Elasticsearch + Fluentd + Kibana)

**Deploy Fluentd**:

```bash
kubectl apply -f https://raw.githubusercontent.com/fluent/fluentd-kubernetes-daemonset/master/fluentd-daemonset-elasticsearch.yaml
```

**Configure Elasticsearch endpoint**:

```yaml
env:
  - name: FLUENT_ELASTICSEARCH_HOST
    value: "elasticsearch.logging.svc"
  - name: FLUENT_ELASTICSEARCH_PORT
    value: "9200"
```

**Query logs in Kibana**:

```text
kubernetes.namespace_name: "helios-system" AND kubernetes.labels.control-plane: "controller-manager"
```

#### Option 2: Loki + Promtail

**Install Loki Stack**:

```bash
helm repo add grafana https://grafana.github.io/helm-charts
helm install loki grafana/loki-stack -n logging \
  --set promtail.enabled=true \
  --set grafana.enabled=true
```

**Query in Grafana**:

```logql
{namespace="helios-system", app="helios-operator"}
```

**Filter by log level**:

```logql
{namespace="helios-system"} |= "level=error"
```

### Useful Log Queries

**Find all errors**:

```bash
kubectl logs -n helios-system deployment/helios-operator-controller-manager | grep "level=error"
```

**Track specific HeliosApp**:

```bash
kubectl logs -n helios-system deployment/helios-operator-controller-manager | grep "heliosapp=default/my-app"
```

**Watch logs live**:

```bash
kubectl logs -n helios-system deployment/helios-operator-controller-manager -f
```

## Status Monitoring

### HeliosApp Status

```bash
# Get phase
kubectl get heliosapp my-app -o jsonpath='{.status.phase}'

# Get conditions
kubectl get heliosapp my-app -o jsonpath='{.status.conditions[*].type}'

# Watch status changes
kubectl get heliosapp my-app -w -o yaml
```

### Status Dashboard Script

`monitor-heliosapp.sh`:

```bash
#!/bin/bash
while true; do
  clear
  echo "=== HeliosApp Status ==="
  kubectl get heliosapp -A -o custom-columns=\
NAME:.metadata.name,\
NAMESPACE:.metadata.namespace,\
PHASE:.status.phase,\
SYNC:.status.syncStatus,\
HEALTH:.status.healthStatus,\
AGE:.metadata.creationTimestamp
  sleep 5
done
```

Run:

```bash
chmod +x monitor-heliosapp.sh
./monitor-heliosapp.sh
```

## Alerting

### Prometheus Alerts

`alerts.yaml`:

```yaml
groups:
  - name: helios-operator
    interval: 30s
    rules:
      # Operator down
      - alert: HeliosOperatorDown
        expr: up{job="helios-operator"} == 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Helios Operator is down"
          description: "Operator has been down for 5 minutes"

      # High error rate
      - alert: HighReconciliationErrorRate
        expr: rate(controller_runtime_reconcile_errors_total[5m]) > 0.1
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "High reconciliation error rate"
          description: "Error rate is {{ $value }} errors/sec"

      # PipelineRun failures
      - alert: HighPipelineRunFailureRate
        expr: |
          sum(rate(helios_pipelinerun_failure_total[15m])) /
          sum(rate(helios_pipelinerun_success_total[15m]) + rate(helios_pipelinerun_failure_total[15m])) > 0.5
        for: 15m
        labels:
          severity: warning
        annotations:
          summary: "High PipelineRun failure rate"
          description: "{{ $value | humanizePercentage }} of PipelineRuns failing"

      # ArgoCD sync issues
      - alert: ArgoApplicationOutOfSync
        expr: helios_argocd_sync_status == 0
        for: 30m
        labels:
          severity: warning
        annotations:
          summary: "ArgoCD Application out of sync"
          description: "Application {{ $labels.name }} has been out of sync for 30m"

      # ArgoCD health issues
      - alert: ArgoApplicationUnhealthy
        expr: helios_argocd_health_status == 0
        for: 15m
        labels:
          severity: critical
        annotations:
          summary: "ArgoCD Application unhealthy"
          description: "Application {{ $labels.name }} is unhealthy"
```

### Alertmanager Configuration

`alertmanager.yaml`:

```yaml
global:
  resolve_timeout: 5m

route:
  group_by: ["alertname", "namespace"]
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 12h
  receiver: "slack-notifications"

receivers:
  - name: "slack-notifications"
    slack_configs:
      - api_url: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
        channel: "#helios-alerts"
        title: "{{ .GroupLabels.alertname }}"
        text: "{{ range .Alerts }}{{ .Annotations.description }}{{ end }}"
```

Deploy:

```bash
kubectl create configmap alertmanager-config --from-file=alertmanager.yaml -n monitoring
kubectl apply -f alertmanager-deployment.yaml
```

## Health Checks

### Operator Health

Operator expose health endpoints:

```bash
# Readiness probe
kubectl exec -n helios-system deployment/helios-operator-controller-manager -- \
  curl http://localhost:8081/readyz

# Liveness probe
kubectl exec -n helios-system deployment/helios-operator-controller-manager -- \
  curl http://localhost:8081/healthz
```

### HeliosApp Health

Check via conditions:

```bash
kubectl get heliosapp my-app -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'
```

Expected: `True`

## Tracing (Optional)

### OpenTelemetry Integration

Coming soon: Distributed tracing for troubleshooting.

## Dashboards

### CLI Dashboard with k9s

```bash
# Install k9s
brew install k9s  # macOS
# or
curl -sS https://webinstall.dev/k9s | bash

# Launch
k9s -n helios-system
```

Navigate:

- `:heliosapp` - View HeliosApps
- `:pipelinerun` - View PipelineRuns
- `:application` - View ArgoCD Applications (in argocd namespace)

### Web Dashboard with Kubernetes Dashboard

```bash
# Install
kubectl apply -f https://raw.githubusercontent.com/kubernetes/dashboard/v2.7.0/aio/deploy/recommended.yaml

# Create admin user
kubectl create serviceaccount dashboard-admin -n kubernetes-dashboard
kubectl create clusterrolebinding dashboard-admin \
  --clusterrole=cluster-admin \
  --serviceaccount=kubernetes-dashboard:dashboard-admin

# Get token
kubectl -n kubernetes-dashboard create token dashboard-admin

# Access
kubectl proxy
# Open http://localhost:8001/api/v1/namespaces/kubernetes-dashboard/services/https:kubernetes-dashboard:/proxy/
```

## Best Practices

1. **Set up alerts** for operator down & high error rates
2. **Monitor PipelineRun success rate** to detect infrastructure issues
3. **Track ArgoCD sync status** to ensure deployments work
4. **Aggregate logs** for easier debugging
5. **Create Grafana dashboards** for team visibility
6. **Set retention policies** for metrics & logs
7. **Regular reviews** of metrics to optimize

## See Also

- [Troubleshooting Guide](./TROUBLESHOOTING.md)
- [Architecture](./ARCHITECTURE.md)
- [API Reference](./API_REFERENCE.md)
