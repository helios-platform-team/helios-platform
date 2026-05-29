variable "gitea_port" {
  type        = number
  description = "Local port to expose Gitea on"
  default     = 3030
}

variable "gitea_admin_user" {
  type        = string
  description = "Admin username for Gitea"
  default     = "helios"
}

variable "gitea_admin_pass" {
  type        = string
  description = "Admin password for Gitea"
  default     = "helios123"
  sensitive   = true
}

variable "gitea_configure" {
  type        = bool
  description = "Whether to wait for Gitea and configure org/token"
  default     = true
}

variable "gitea_internal_host" {
  type        = string
  description = "Gitea internal DNS name within Kubernetes"
  default     = "gitea-http.gitea.svc.cluster.local:3000"
}

variable "gitops_secret_name" {
  type        = string
  description = "Kubernetes Secret name for GitOps repository"
  default     = "helios-gitops-bot"
}

variable "workspace_root" {
  type        = string
  description = "Absolute path to the workspace root directory"
}
