module "platform_services" {
  source = "../../modules/platform-services"

  # Repository variables
  gitops_repo_url = var.gitops_repo_url
  gitops_path     = "overlays/prod"

  # ArgoCD resource configurations (High Availability enabled for production)
  argocd_version        = "9.5.15"
  argocd_ha_enabled     = true
  argocd_replicas       = 3
  argocd_cpu_request    = "100m"
  argocd_memory_request = "256Mi"
  argocd_memory_limit   = "512Mi"

  # No CoreDNS patching needed in production
  patch_coredns  = false
  workspace_root = "${path.module}/../../.."

  # Docker registry credentials
  docker_username = var.docker_username
  docker_password = var.docker_password
  docker_server   = var.docker_server
}