module "helios_base" {
  source = "../../modules/platform-base"
  
  cluster_name = "helios-prod"
  
  # Trỏ ArgoCD vào thư mục chứa TOÀN BỘ hệ thống (Tekton + Operator Pod + Backstage Pod)
  gitops_path  = "overlays/prod" 
}