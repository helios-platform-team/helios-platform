variable "cluster_name" {
  type        = string
  description = "Name of the local k3d Kubernetes Cluster"
}

variable "registry_name" {
  type        = string
  description = "Name of the local k3d registry to use as a Docker Hub pull-through cache"
  default     = "helios-registry"
}

variable "registry_port" {
  type        = number
  description = "Host port the local registry listens on"
  default     = 5001
}