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

# New configuration variables for Tekton limits, CoreDNS, and Secrets
variable "patch_coredns" {
  type        = bool
  description = "Whether to patch CoreDNS ConfigMap to use robust Google and Cloudflare DNS"
  default     = false
}

variable "workspace_root" {
  type        = string
  description = "Absolute path to the workspace root directory"
  default     = "../../../"
}

variable "docker_username" {
  type        = string
  description = "Username for Docker Registry"
  default     = ""
}

variable "docker_password" {
  type        = string
  description = "Password or Token for Docker Registry"
  default     = ""
  sensitive   = true
}

variable "docker_server" {
  type        = string
  description = "Docker Registry Server"
  default     = "https://index.docker.io/v1/"
}

variable "tekton_limit_namespace" {
  type        = string
  description = "Namespace where Tekton limits are applied"
  default     = "default"
}

variable "tekton_limit_cpu" {
  type        = string
  description = "CPU limit for Tekton containers"
  default     = "2"
}

variable "tekton_limit_memory" {
  type        = string
  description = "Memory limit for Tekton containers"
  default     = "4Gi"
}

variable "tekton_limit_request_cpu" {
  type        = string
  description = "CPU request for Tekton containers"
  default     = "500m"
}

variable "tekton_limit_request_memory" {
  type        = string
  description = "Memory request for Tekton containers"
  default     = "512Mi"
}