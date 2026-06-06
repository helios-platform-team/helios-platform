variable "git_provider" {
  type        = string
  description = "Git provider (gitea, github, gitlab)"
  default     = "gitea"
}

variable "git_organization" {
  type        = string
  description = "Git organization name"
  default     = "helios-platform"
}

variable "gitea_configure" {
  type        = bool
  description = "Whether to wait for Gitea and configure org/token"
  default     = true
}

variable "docker_username" {
  type        = string
  description = "Docker Hub username"
  default     = ""
}

variable "docker_password" {
  type        = string
  description = "Docker Hub access token or password"
  default     = ""
  sensitive   = true
}

variable "docker_server" {
  type        = string
  description = "Docker Registry Server"
  default     = "https://index.docker.io/v1/"
}
