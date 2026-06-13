terraform {
  required_version = ">= 1.15.5"

  required_providers {
    time = {
      source  = "hashicorp/time"
      version = "~> 0.14.0"
    }
  }
}
