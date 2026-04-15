$dockerUser = $env:DOCKER_USERNAME
$dockerPass = $env:DOCKER_PASSWORD
if ([string]::IsNullOrEmpty($dockerPass)) { $dockerPass = $env:DOCKER_TOKEN }
$dockerServer = $env:DOCKER_SERVER
$dockerEmail = $env:DOCKER_EMAIL

if ([string]::IsNullOrEmpty($dockerUser) -or [string]::IsNullOrEmpty($dockerPass)) {
    Write-Error "DOCKER_USERNAME and either DOCKER_PASSWORD or DOCKER_TOKEN must be set in .env"
    exit 1
}

if ([string]::IsNullOrEmpty($dockerServer)) { $dockerServer = "https://index.docker.io/v1/" }
if ([string]::IsNullOrEmpty($dockerEmail)) { $dockerEmail = "dev@helios.io" }

Write-Host "Creating docker-registry secret..."
kubectl create secret docker-registry docker-credentials `
    --docker-server=$dockerServer `
    --docker-username=$dockerUser `
    --docker-password=$dockerPass `
    --docker-email=$dockerEmail `
    --dry-run=client -o yaml | kubectl apply -f -

Write-Host "Patching pipeline service account..."
$sa = kubectl get sa pipeline -n default -o name 2>$null
if ($LASTEXITCODE -eq 0) {
    kubectl patch sa pipeline -p '{"secrets": [{"name": "docker-credentials"}]}'
    Write-Host "Patched pipeline service account with docker-credentials"
} else {
    Write-Host "pipeline ServiceAccount not found yet; skipping patch (will be created by Tekton)"
}
