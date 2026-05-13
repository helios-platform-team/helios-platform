resource "terraform_data" "k3d_cluster" {
  triggers_replace = [
    var.cluster_name
  ]

  # Runs upon 'terraform apply'
  provisioner "local-exec" {
    command = <<-EOT
      k3d cluster create ${var.cluster_name} \
        --agents 1 \
        --wait \
        --api-port 127.0.0.1:6550 \
        --k3s-arg "--disable=metrics-server@server:0"
    EOT
  }

  # Runs upon 'terraform destroy'
  provisioner "local-exec" {
    when    = destroy
    command = "k3d cluster delete ${self.triggers_replace[0]}"
  }
}

# (Tuỳ chọn) Tạo một khoảng trễ nhỏ để đợi Kubeconfig được ghi xuống đĩa trước khi Helm chạy
resource "time_sleep" "wait_for_cluster" {
  depends_on      = [terraform_data.k3d_cluster]
  create_duration = "5s"
}
