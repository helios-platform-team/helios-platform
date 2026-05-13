terraform {
  required_providers {
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.12.1"
    }
    kubectl = {
      source  = "alekc/kubectl"
      version = ">= 2.0.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.24.0"
    }
  }
}

provider "helm" {
  kubernetes {
    config_path    = "~/.kube/config"
    config_context = "k3d-helios-dev"
  }
}

provider "kubernetes" {
  config_path    = "~/.kube/config"
  config_context = "k3d-helios-dev"
}

provider "kubectl" {
  config_path      = "~/.kube/config"
  config_context   = "k3d-helios-dev"
  load_config_file = true
}
