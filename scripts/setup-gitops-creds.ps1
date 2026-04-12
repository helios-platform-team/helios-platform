param(
    [string]$GiteaInternalHost = "gitea-http.gitea.svc.cluster.local:3000",
    [string]$SecretName = "helios-gitops-bot"
)

$user = if ($env:GITOPS_GIT_USER) { $env:GITOPS_GIT_USER } elseif ($env:GITEA_BOT_USER) { $env:GITEA_BOT_USER } elseif ($env:GITEA_ADMIN_USER) { $env:GITEA_ADMIN_USER } else { "helios" }
$pass = if ($env:GITOPS_GIT_PASSWORD) { $env:GITOPS_GIT_PASSWORD } elseif ($env:GITEA_BOT_PASSWORD) { $env:GITEA_BOT_PASSWORD } elseif ($env:GITEA_ADMIN_PASS) { $env:GITEA_ADMIN_PASS } else { "helios123" }

kubectl create secret generic "$SecretName" --type=kubernetes.io/basic-auth --from-literal=username="$user" --from-literal=password="$pass" --dry-run=client -o yaml | kubectl apply -f -
kubectl annotate secret "$SecretName" "tekton.dev/git-0=http://$GiteaInternalHost" --overwrite

# PowerShell env update logic
$envFiles = @(".env", "apps/portal/.env")
foreach ($file in $envFiles) {
    if (Test-Path $file) {
        $content = Get-Content $file
        $updated = $false
        $newLines = @()
        foreach ($line in $content) {
            if ($line -match "^GITOPS_SECRET_REF=") {
                $newLines += "GITOPS_SECRET_REF=$SecretName"
                $updated = $true
            } else {
                $newLines += $line
            }
        }
        if (-not $updated) {
            $newLines += "GITOPS_SECRET_REF=$SecretName"
        }
        $newLines | Set-Content $file
    }
}
