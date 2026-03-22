param(
    [string]$ArgocdPort = "8080"
)

# Generate ArgoCD auth token
try {
    $passB64 = kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}'
    $pass = [System.Text.Encoding]::UTF8.GetString([System.Convert]::FromBase64String($passB64)).Trim()

    $body = @{ username = "admin"; password = $pass } | ConvertTo-Json -Compress

    # Limit TLS bypass scope to the local ArgoCD login request.
    $previousValidationCallback = [System.Net.ServicePointManager]::ServerCertificateValidationCallback
    try {
        [System.Net.ServicePointManager]::ServerCertificateValidationCallback = { $true }

        $parsed = Invoke-RestMethod -Uri "https://127.0.0.1:${ArgocdPort}/api/v1/session" `
            -Method Post `
            -ContentType "application/json" `
            -Body $body
    }
    finally {
        [System.Net.ServicePointManager]::ServerCertificateValidationCallback = $previousValidationCallback
    }

    if ($parsed.token) {
        $env:ARGOCD_AUTH_TOKEN = $parsed.token
        Write-Output "ArgoCD token generated."
    } else {
        Write-Warning "Could not generate ArgoCD token. ArgoCD features may not work."
    }
} catch {
    Write-Warning "Could not generate ArgoCD token. ArgoCD features may not work."
    Write-Error "ArgoCD token request failed: $_"
}

# Start Backstage
yarn start
