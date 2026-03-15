param(
    [string]$ArgocdPort = "8080"
)

# Generate ArgoCD auth token
try {
    $passB64 = kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}'
    $pass = [System.Text.Encoding]::UTF8.GetString([System.Convert]::FromBase64String($passB64)).Trim()

    $body = @{ username = "admin"; password = $pass } | ConvertTo-Json -Compress
    
    # PowerShell 5.1 compatibility for skipping SSL checks
    [System.Net.ServicePointManager]::ServerCertificateValidationCallback = { $true }
    
    $parsed = Invoke-RestMethod -Uri "https://127.0.0.1:${ArgocdPort}/api/v1/session" `
        -Method Post `
        -ContentType "application/json" `
        -Body $body
    if ($parsed.token) {
        $env:ARGOCD_AUTH_TOKEN = $parsed.token
        Write-Host "ArgoCD token generated."
    } else {
        Write-Host "WARNING: Could not generate ArgoCD token. ArgoCD features may not work."
    }
} catch {
    Write-Host "WARNING: Could not generate ArgoCD token. ArgoCD features may not work."
    Write-Host "Error: $_"
}

# Start Backstage
yarn start
