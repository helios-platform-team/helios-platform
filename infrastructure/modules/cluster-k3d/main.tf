resource "terraform_data" "k3d_cluster" {
  triggers_replace = [
    var.cluster_name
  ]

  # Runs upon 'terraform apply'
  provisioner "local-exec" {
    command = <<-EOT
      if k3d cluster list ${var.cluster_name} >/dev/null 2>&1; then
        echo "Cluster ${var.cluster_name} already exists. Skipping creation."
      else
        k3d cluster create ${var.cluster_name} \
          --image rancher/k3s:v1.36.1-k3s1 \
          --agents 1 \
          --wait \
          --api-port 127.0.0.1:6550 \
          --k3s-arg "--disable=traefik@server:0" \
          --k3s-arg "--disable=metrics-server@server:0"
      fi
      
      kubectl config set-cluster k3d-${var.cluster_name} --server=https://localhost:6550
      kubectl cluster-info
    EOT
  }

  # Runs upon 'terraform destroy'
  provisioner "local-exec" {
    when    = destroy
    command = "k3d cluster delete ${self.triggers_replace[0]}"
  }
}

# Add a delay to wait for kubeconfig context to merge and cluster API server to become ready
resource "time_sleep" "wait_for_cluster" {
  depends_on      = [terraform_data.k3d_cluster]
  create_duration = "10s"
}
