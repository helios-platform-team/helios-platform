terraform {
  required_version = ">= 1.15.5"

  required_providers {
    gitlab = {
      source  = "gitlabhq/gitlab"
      version = "~> 19.0.0"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "~> 4.3.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.9.0"
    }
  }
}
