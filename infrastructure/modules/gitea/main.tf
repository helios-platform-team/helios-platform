# Explicitly delete the Gitea PVC on destroy so Helm doesn't emit
# "These resources were kept due to the resource policy" warnings.
# The Gitea chart annotates gitea-shared-storage with helm.sh/resource-policy=keep.
resource "terraform_data" "gitea_pvc_cleanup" {
  depends_on = [helm_release.gitea]

  provisioner "local-exec" {
    when    = destroy
    # Do not block terraform destroy on PVC finalizers/volume detach delays.
    # We only need to issue the delete request before tearing down the cluster.
    command = "kubectl delete pvc gitea-shared-storage -n gitea --ignore-not-found=true --wait=false || true"
  }
}

resource "helm_release" "gitea" {
  name             = "gitea"
  repository       = "oci://registry-1.docker.io/giteacharts"
  chart            = "gitea"
  namespace        = "gitea"
  create_namespace = true
  wait             = var.gitea_configure
  timeout          = 600

  values = [
    yamlencode({
      image = {
        registry   = "docker.io"
        repository = "gitea/gitea"
      }
      gitea = {
        admin = {
          username = var.gitea_admin_user
          password = var.gitea_admin_pass
        }
        config = {
          database = {
            DB_TYPE = "sqlite3"
          }
          session = {
            PROVIDER = "memory"
          }
          cache = {
            ADAPTER = "memory"
          }
          queue = {
            TYPE = "level"
          }
          server = {
            ROOT_URL  = "http://localhost:${var.gitea_port}"
            HTTP_PORT = "3000"
          }
          webhook = {
            ALLOWED_HOST_LIST = "*"
          }
        }
      }
      postgresql = {
        enabled = false
      }
      "postgresql-ha" = {
        enabled = false
      }
      "redis-cluster" = {
        enabled = false
      }
      redis = {
        enabled = false
      }
      "valkey-cluster" = {
        enabled = false
      }
      valkey = {
        enabled = false
      }
      persistence = {
        size = "1Gi"
      }
      resources = {
        requests = {
          memory = "128Mi"
        }
        limits = {
          memory = "256Mi"
        }
      }
    })
  ]
}

resource "terraform_data" "configure_gitea" {
  count      = var.gitea_configure ? 1 : 0
  depends_on = [helm_release.gitea]

  input = {
    uuid = uuid()
  }

  provisioner "local-exec" {
    command = <<-EOT
      echo "Waiting for Gitea deployment..."
      kubectl rollout status deployment/gitea -n gitea --timeout=600s

      echo "Finding Gitea pod..."
      POD_NAME=$(kubectl get pods -n gitea -l app.kubernetes.io/name=gitea -o jsonpath="{.items[0].metadata.name}")

      echo "Creating Gitea organization 'helios-platform'..."
      kubectl exec -n gitea $POD_NAME -c gitea -- curl -s -X POST "http://localhost:3000/api/v1/orgs" -u "${var.gitea_admin_user}:${var.gitea_admin_pass}" -H "Content-Type: application/json" -d '{"username":"helios-platform", "full_name":"Helios Platform", "visibility":"public"}' || true

      echo "Generating Gitea API Token..."
      TOKEN_NAME="helios-token-${self.input.uuid}"
      TOKEN_OUTPUT=$(kubectl exec -n gitea $POD_NAME -c gitea -- gitea admin user generate-access-token --username ${var.gitea_admin_user} --token-name $TOKEN_NAME --scopes all 2>/dev/null || true)
      
      TOKEN=$(echo "$TOKEN_OUTPUT" | grep -oE "Access token was successfully created: [a-zA-Z0-9_]+" | awk '{print $NF}' || true)
      
      if [ -z "$TOKEN" ]; then
        TOKEN=$(echo "$TOKEN_OUTPUT" | awk '/Access token was successfully created:/ {print $NF}' || true)
      fi

      if [ -z "$TOKEN" ]; then
        echo "Failed to generate token. Output was: $TOKEN_OUTPUT"
        exit 1
      fi

      echo "Generated Gitea token successfully."

      ENV_FILE="${var.workspace_root}/.env"
      PORTAL_ENV="${var.workspace_root}/apps/portal/.env"

      update_env() {
        local file="$1" key="$2" val="$3"
        if [ ! -f "$file" ]; then touch "$file"; fi
        if grep -q "^$key=" "$file"; then
          sed -i "s|^$key=.*|$key=$val|" "$file"
        else
          echo "$key=$val" >> "$file"
        fi
      }

      update_env "$ENV_FILE" "GITEA_TOKEN" "$TOKEN"
      update_env "$ENV_FILE" "GITEA_USER" "${var.gitea_admin_user}"
      update_env "$ENV_FILE" "GITEA_URL" "http://localhost:${var.gitea_port}"
      update_env "$ENV_FILE" "GITEA_INTERNAL_URL" "http://${var.gitea_internal_host}"
      update_env "$ENV_FILE" "GITOPS_SECRET_REF" "${var.gitops_secret_name}"

      update_env "$PORTAL_ENV" "GITEA_TOKEN" "$TOKEN"
      update_env "$PORTAL_ENV" "GITEA_USER" "${var.gitea_admin_user}"
      update_env "$PORTAL_ENV" "GITEA_URL" "http://localhost:${var.gitea_port}"
      update_env "$PORTAL_ENV" "GITEA_INTERNAL_URL" "http://${var.gitea_internal_host}"
      update_env "$PORTAL_ENV" "GITOPS_SECRET_REF" "${var.gitops_secret_name}"

      echo "Environment files updated successfully!"
    EOT
  }
}

resource "kubernetes_secret_v1" "gitea_repo_creds" {
  depends_on = [helm_release.gitea]

  metadata {
    name      = "gitea-repo-creds"
    namespace = "argocd"
    labels = {
      "argocd.argoproj.io/secret-type" = "repo-creds"
    }
  }

  data = {
    type     = "git"
    url      = "http://${var.gitea_internal_host}"
    username = var.gitea_admin_user
    password = var.gitea_admin_pass
  }
}

resource "kubernetes_secret_v1" "gitea_gitops_secret" {
  depends_on = [helm_release.gitea]

  metadata {
    name      = var.gitops_secret_name
    namespace = "default"
    annotations = {
      "tekton.dev/git-0" = "http://${var.gitea_internal_host}"
    }
  }

  type = "kubernetes.io/basic-auth"

  data = {
    username = var.gitea_admin_user
    password = var.gitea_admin_pass
  }
}
