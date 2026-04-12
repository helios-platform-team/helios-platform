param(
    [string]$GiteaInternalHost = "gitea-http.gitea.svc.cluster.local:3000"
)

$user = if ($env:GITEA_ADMIN_USER) { $env:GITEA_ADMIN_USER } else { "helios" }
$pass = if ($env:GITEA_ADMIN_PASS) { $env:GITEA_ADMIN_PASS } else { "helios123" }

$secret = @"
apiVersion: v1
kind: Secret
metadata:
  name: gitea-repo-creds
  namespace: argocd
  labels:
    argocd.argoproj.io/secret-type: repo-creds
stringData:
  type: git
  url: http://$GiteaInternalHost
  username: "$user"
  password: "$pass"
"@

$secret | kubectl apply -f -
