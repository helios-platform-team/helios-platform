module "cluster" {
  source       = "../../modules/cluster-k3d"
  cluster_name = "helios-dev"
}

module "platform_services" {
  source = "../../modules/platform-services"

  # Ensure the cluster finishes creating before attempting to deploy Helm charts
  depends_on = [module.cluster]

  # Repository variables
  gitops_repo_url = "https://github.com/helios-platform-team/helios-system-gitops.git"
  gitops_path     = "overlays/dev"

  # ArgoCD resource configurations (scaled down for local)
  argocd_version        = "5.51.6"
  argocd_ha_enabled     = false
  argocd_replicas       = 1
  argocd_cpu_request    = "50m"
  argocd_memory_request = "128Mi"
  argocd_memory_limit   = "256Mi"
}