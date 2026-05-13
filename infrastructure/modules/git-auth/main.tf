resource "gitlab_personal_access_token" "this" {
  count    = var.git_provider_type == "gitlab" ? 1 : 0
  user_id  = "me"
  name     = "terraform-generated-token"
  scopes   = ["api"]
  expires_at = "2026-12-31"
}

# Example for GitHub/General: Generating an SSH Deploy Key
resource "tls_private_key" "ssh_key" {
  count     = var.git_provider_type == "github" || var.git_provider_type == "gitea" ? 1 : 0
  algorithm = "ED25519"
}

# Generating a random string if you just need a generic app key
resource "random_password" "app_key" {
  length  = 32
  special = true
}