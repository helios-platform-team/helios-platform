# Explicitly delete ArgoCD CRDs on destroy so Helm doesn't emit
# "These resources were kept due to the resource policy" warnings.
# ArgoCD's chart annotates its CRDs with helm.sh/resource-policy=keep,
# which prevents helm uninstall from removing them. We clean them up
# ourselves before the helm release is destroyed.
resource "terraform_data" "argocd_crds_cleanup" {
  depends_on = [helm_release.argocd]

  provisioner "local-exec" {
    when    = destroy
    command = <<-EOT
      kubectl delete crd \
        applications.argoproj.io \
        applicationsets.argoproj.io \
        appprojects.argoproj.io \
        --ignore-not-found=true
    EOT
  }
}

resource "helm_release" "argocd" {
  name             = "argocd"
  repository       = "oci://ghcr.io/argoproj/argo-helm"
  chart            = "argo-cd"
  namespace        = "argocd"
  create_namespace = true
  version          = var.argocd_version

  values = [
    <<-EOF
      redis:
        ha:
          enabled: ${var.argocd_ha_enabled}
      
      controller:
        replicas: ${var.argocd_replicas}
        resources:
          requests:
            memory: "${var.argocd_memory_request}"
            cpu: "${var.argocd_cpu_request}"
          limits:
            memory: "${var.argocd_memory_limit}"

      server:
        replicas: ${var.argocd_replicas}
        resources:
          requests:
            memory: "${var.argocd_memory_request}"
            cpu: "${var.argocd_cpu_request}"

      configs:
        cm:
          resource.customizations.ignoreDifferences.apiextensions.k8s.io_CustomResourceDefinition: |
            jsonPointers:
              - /status
              - /spec/preserveUnknownFields
              - /spec/conversion
              - /spec/names/listKind
          resource.customizations.ignoreDifferences._ConfigMap: |
            jqPathExpressions:
              - select(.metadata.name == "feature-flags") | .data
    EOF
  ]
}

resource "kubectl_manifest" "helios_bootstrap" {
  # Bắt buộc phải đợi ArgoCD cài xong để cụm có sẵn Application CRD
  depends_on = [helm_release.argocd]

  yaml_body = <<-EOF
    apiVersion: argoproj.io/v1alpha1
    kind: Application
    metadata:
      name: helios-bootstrap
      namespace: argocd
    spec:
      project: default
      source:
        repoURL: '${var.gitops_repo_url}'
        targetRevision: HEAD
        path: ${var.gitops_path}
      destination:
        server: https://kubernetes.default.svc
        namespace: argocd
      syncPolicy:
        automated:
          prune: true
          selfHeal: true
  EOF
}