module "cluster" {
  source       = "../../modules/cluster-k3d"
  cluster_name = "helios-dev"
}

module "platform_services" {
  source = "../../modules/platform-services"

  # Ensure the cluster finishes creating before attempting to deploy platform services
  depends_on = [module.cluster]

  # Repository variables
  gitops_repo_url = "https://github.com/helios-platform-team/helios-system-gitops.git"
  gitops_path     = "overlays/dev"

  # ArgoCD resource configurations (scaled down for local)
  argocd_version        = "9.5.15"
  argocd_ha_enabled     = false
  argocd_replicas       = 1
  argocd_cpu_request    = "50m"
  argocd_memory_request = "128Mi"
  argocd_memory_limit   = "1024Mi"

  # CoreDNS and limits configuration
  patch_coredns  = true
  workspace_root = "${path.module}/../../.."

  # Docker registry credentials
  docker_username = var.docker_username
  docker_password = var.docker_password
  docker_server   = var.docker_server
}

module "gitea" {
  source = "../../modules/gitea"

  # Wait for platform services to ensure namespaces (like argocd) exist
  depends_on = [module.platform_services]

  gitea_port         = 3030
  gitea_admin_user   = "helios"
  gitea_admin_pass   = "helios123"
  workspace_root     = "${path.module}/../../.."
  gitops_secret_name = "helios-gitops-bot"
}
