variable "gitops_repo_url" {
  type        = string
  description = "GitOps repository URL for ArgoCD to sync with"
  default     = "https://github.com/helios-platform-team/helios-system-gitops.git"
}

variable "docker_username" {
  type        = string
  description = "Docker Registry Username"
  default     = ""
}

variable "docker_password" {
  type        = string
  description = "Docker Registry Password or Token"
  default     = ""
  sensitive   = true
}

variable "docker_server" {
  type        = string
  description = "Docker Registry Server"
  default     = "https://index.docker.io/v1/"
}

variable "kubeconfig_path" {
  type        = string
  description = "Path to the kubeconfig file for the production cluster"
  default     = "~/.kube/config"
}

variable "kubeconfig_context" {
  type        = string
  description = "Kubeconfig context name for the production cluster"
  default     = "helios-prod"
}
