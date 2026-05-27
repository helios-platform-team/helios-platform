terraform {
  required_version = ">= 1.15.4"

  required_providers {
    kubectl = {
      source  = "alekc/kubectl"
      version = "~> 2.4.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 3.1.0"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 3.1.2"
    }
  }
}

provider "helm" {
  kubernetes = {
    config_path    = "~/.kube/config"
    config_context = "k3d-helios-dev"
    insecure       = true
  }
}

provider "kubernetes" {
  config_path    = "~/.kube/config"
  config_context = "k3d-helios-dev"
  insecure       = true
}

provider "kubectl" {
  config_path      = "~/.kube/config"
  config_context   = "k3d-helios-dev"
  insecure         = true
  load_config_file = true
}
