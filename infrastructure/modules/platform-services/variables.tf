variable "gitops_repo_url" {
  type        = string
  description = "GitOps repo URL for ArgoCD to sync with"
}

variable "gitops_path" {
  type        = string
  description = "Directory path inside the GitOps repo"
}

variable "argocd_version" {
  type        = string
  description = "Helm chart version for ArgoCD"
}

variable "argocd_ha_enabled" {
  type        = bool
  description = "Enable High Availability for Redis"
}

variable "argocd_replicas" {
  type        = number
  description = "Number of replicas for ArgoCD components"
}

variable "argocd_cpu_request" {
  type        = string
  description = "CPU request for ArgoCD pods"
}

variable "argocd_memory_request" {
  type        = string
  description = "Memory request for ArgoCD pods"
}

variable "argocd_memory_limit" {
  type        = string
  description = "Memory limit for ArgoCD pods"
}