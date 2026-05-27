variable "git_provider_type" {
  type        = string
  description = "The Git provider to use (github, gitlab, or gitea)"

  validation {
    condition     = contains(["github", "gitlab", "gitea"], var.git_provider_type)
    error_message = "The git_provider must be one of: github, gitlab, gitea."
  }
}