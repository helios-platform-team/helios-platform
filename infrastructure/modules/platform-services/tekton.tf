resource "kubernetes_limit_range_v1" "tekton_limits" {
  metadata {
    name      = "tekton-resource-limits"
    namespace = var.tekton_limit_namespace
  }

  spec {
    limit {
      type = "Container"
      default = {
        cpu    = var.tekton_limit_cpu
        memory = var.tekton_limit_memory
      }
      default_request = {
        cpu    = var.tekton_limit_request_cpu
        memory = var.tekton_limit_request_memory
      }
    }
  }
}

resource "kubernetes_namespace_v1" "tekton_pipelines" {
  metadata {
    name = "tekton-pipelines"
    labels = {
      "app.kubernetes.io/part-of"          = "tekton-pipelines"
      "pod-security.kubernetes.io/enforce" = "restricted"
    }
  }
}

resource "kubernetes_config_map_v1" "tekton_pruner_config" {
  metadata {
    name      = "tekton-pruner-default-spec"
    namespace = kubernetes_namespace_v1.tekton_pipelines.metadata[0].name
    labels = {
      "app.kubernetes.io/part-of"     = "tekton-pruner"
      "pruner.tekton.dev/config-type" = "global"
    }
  }

  data = {
    global-config = <<-EOT
      enforcedConfigLevel: global
      ttlSecondsAfterFinished: 7200
      successfulHistoryLimit: 3
      failedHistoryLimit: 3
    EOT
  }
}

resource "kubernetes_role_v1" "tekton_triggers_runner" {
  metadata {
    name      = "tekton-triggers-pipelinerun-runner"
    namespace = "default"
  }

  rule {
    api_groups = ["tekton.dev"]
    resources  = ["pipelineruns", "taskruns"]
    verbs      = ["get", "list", "watch", "create", "update", "patch"]
  }
}

resource "kubernetes_role_binding_v1" "tekton_triggers_runner_binding" {
  metadata {
    name      = "tekton-triggers-pipelinerun-runner"
    namespace = "default"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = kubernetes_role_v1.tekton_triggers_runner.metadata[0].name
  }

  subject {
    kind      = "ServiceAccount"
    name      = "tekton-triggers-sa"
    namespace = "default"
  }
}

resource "kubernetes_secret_v1" "docker_credentials" {
  count = var.docker_username != "" ? 1 : 0

  metadata {
    name      = "docker-credentials"
    namespace = "default"
  }

  type = "kubernetes.io/dockerconfigjson"

  data = {
    ".dockerconfigjson" = jsonencode({
      auths = {
        (var.docker_server) = {
          username = var.docker_username
          password = var.docker_password
          auth     = base64encode("${var.docker_username}:${var.docker_password}")
        }
      }
    })
  }
}

resource "terraform_data" "patch_coredns" {
  count = var.patch_coredns ? 1 : 0

  provisioner "local-exec" {
    command = <<-EOT
      echo "Patching CoreDNS ConfigMap..."
      kubectl get configmap coredns -n kube-system -o yaml > coredns.yaml
      sed -i 's/forward . \/etc\/resolv.conf/forward . 8.8.8.8 1.1.1.1/' coredns.yaml
      kubectl apply -f coredns.yaml
      rm coredns.yaml
      kubectl rollout restart -n kube-system deployment/coredns
    EOT
  }
}

resource "terraform_data" "patch_pipeline_sa" {
  count      = var.docker_username != "" ? 1 : 0
  depends_on = [kubernetes_secret_v1.docker_credentials]

  provisioner "local-exec" {
    command = <<-EOT
      echo "Patching pipeline ServiceAccount with docker-credentials..."
      # Wait up to 120s for the pipeline SA to be created by Tekton installation
      for i in {1..24}; do
        if kubectl get sa pipeline -n default >/dev/null 2>&1; then
          kubectl patch sa pipeline -n default -p '{"secrets": [{"name": "docker-credentials"}]}'
          echo "Successfully patched pipeline SA"
          break
        fi
        echo "Waiting for pipeline SA to be created..."
        sleep 5
        if [ $i -eq 24 ]; then
          echo "Warning: pipeline SA was not found. Skipping patch."
        fi
      done
      kubectl patch sa default -n default -p '{"imagePullSecrets": [{"name": "docker-credentials"}]}'
      echo "Successfully patched default SA"
    EOT
  }
}

resource "terraform_data" "install_crds" {
  # Always run to ensure CRDs match operator definition
  input = {
    crd_dir = "${var.workspace_root}/apps/operator/config/crd"
  }

  provisioner "local-exec" {
    command = "kubectl kustomize ${self.input.crd_dir} | kubectl apply -f -"
  }
}
