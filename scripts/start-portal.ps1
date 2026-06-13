param(
    [string]$ArgocdPort = "8080",
    [string]$GiteaLocalPort = "3030",
    [string]$ProxyPort = "8001"
)

$ErrorActionPreference = "Stop"
$Jobs = @()

function Cleanup {
    Write-Host "[Stop] Stopping background processes..." -ForegroundColor Yellow
    foreach ($job in $Jobs) {
        Stop-Job $job -ErrorAction SilentlyContinue
        Remove-Job $job -ErrorAction SilentlyContinue
    }
    # Also kill any lingering kubectl port-forwards we might have started
    Get-Process -Name "kubectl" -ErrorAction SilentlyContinue | Where-Object { $_.CommandLine -match "port-forward" -or $_.CommandLine -match "proxy" } | Stop-Process -Force -ErrorAction SilentlyContinue
    Write-Host "[Done] Cleanup complete." -ForegroundColor Green
}

# Ensure cleanup on exit
trap { Cleanup; exit }

Write-Host "[Gitea] Starting Gitea Port-Forward (localhost:$GiteaLocalPort)..." -ForegroundColor Yellow
$Jobs += Start-Job -ScriptBlock { kubectl port-forward -n gitea svc/gitea-http "${using:GiteaLocalPort}:3000" }

Write-Host "[Proxy] Starting Kubectl Proxy (localhost:$ProxyPort)..." -ForegroundColor Yellow
$Jobs += Start-Job -ScriptBlock { kubectl proxy --port="${using:ProxyPort}" }

# ArgoCD Token Automation
$argocdToken = ""
$adminSecret = kubectl -n argocd get secret argocd-initial-admin-secret -o json 2>$null | ConvertFrom-Json
if ($null -ne $adminSecret) {
    Write-Host "[ArgoCD] Fetching Admin Password..." -ForegroundColor Yellow
    $passB64 = $adminSecret.data.password
    $pass = [System.Text.Encoding]::UTF8.GetString([System.Convert]::FromBase64String($passB64)).Trim()
    Write-Host "[ArgoCD] Admin Password: $pass" -ForegroundColor Green

    Write-Host "[ArgoCD] Starting Port-Forward (localhost:$ArgocdPort)..." -ForegroundColor Yellow
    $Jobs += Start-Job -ScriptBlock { kubectl port-forward -n argocd svc/argocd-server "${using:ArgocdPort}:443" }
    
    # Wait for PF to be ready
    Start-Sleep -Seconds 3

    try {
        $previousValidationCallback = [System.Net.ServicePointManager]::ServerCertificateValidationCallback
        [System.Net.ServicePointManager]::ServerCertificateValidationCallback = { $true }
        
        $body = @{ username = "admin"; password = $pass } | ConvertTo-Json -Compress
        $parsed = Invoke-RestMethod -Uri "https://127.0.0.1:${ArgocdPort}/api/v1/session" -Method Post -ContentType "application/json" -Body $body -ErrorAction Stop
        
        if ($parsed.token) {
            $env:ARGOCD_AUTH_TOKEN = $parsed.token
            Write-Host "[ArgoCD] Token Generated!" -ForegroundColor Green
        }
    } catch {
        Write-Host "[ArgoCD] Warning: Could not generate token: $_" -ForegroundColor Yellow
    } finally {
        [System.Net.ServicePointManager]::ServerCertificateValidationCallback = $previousValidationCallback
    }
} else {
    Write-Host "[ArgoCD] Info: Admin secret not found. Skipping token generation." -ForegroundColor Yellow
}

# Load Environment Variables from .env
$envFile = "../../.env"
if (Test-Path $envFile) {
    Write-Host "[Env] Loading variables from $envFile" -ForegroundColor Yellow
    Get-Content $envFile | Where-Object { $_ -match "=" -and $_ -notmatch "^#" } | ForEach-Object {
        $parts = $_.Split('=', 2)
        if ($parts.Count -eq 2) {
            $key = $parts[0].Trim()
            $value = $parts[1].Trim()
            
            # Use ASCII codes for quotes to avoid encoding issues: [char]34 is ", [char]39 is '
            $value = $value.Trim([char]34).Trim([char]39)
            
            if ($key) {
                [System.Environment]::SetEnvironmentVariable($key, $value)
            }
        }
    }
}

Write-Host "[Portal] Starting Backstage Portal..." -ForegroundColor Green
try {
    yarn start
} finally {
    Cleanup
}
