param(
    [string]$GiteaPort = "3030",
    [string]$AdminUser = $env:GITEA_ADMIN_USER,
    [string]$AdminPass = $env:GITEA_ADMIN_PASS
)

if ([string]::IsNullOrEmpty($AdminUser)) { $AdminUser = "helios" }
if ([string]::IsNullOrEmpty($AdminPass)) { $AdminPass = "helios123" }

$GiteaBase = "http://localhost:$GiteaPort"

Write-Host "Cleaning up port $GiteaPort..."
Get-NetTCPConnection -LocalPort $GiteaPort -ErrorAction SilentlyContinue | ForEach-Object {
    Stop-Process -Id $_.OwningProcess -Force -ErrorAction SilentlyContinue
}

Write-Host "Port-forwarding Gitea for setup..."
$Job = Start-Job -ScriptBlock {
    param($port)
    kubectl port-forward -n gitea svc/gitea-http "${port}:3000"
} -ArgumentList $GiteaPort

# Wait for Gitea to be available
$maxRetries = 10
$retryCount = 0
$connected = $false

Write-Host "Waiting for Gitea at $GiteaBase..."
while ($retryCount -lt $maxRetries -and -not $connected) {
    try {
        $response = Invoke-RestMethod -Uri "$GiteaBase/api/v1/version" -Method Get -ErrorAction Stop
        $connected = $true
        Write-Host "Connected to Gitea version: $($response.version)"
    } catch {
        $retryCount++
        Write-Host "Retry ${retryCount}/${maxRetries}: Gitea not ready yet..."
        Start-Sleep -Seconds 3
    }
}

if (-not $connected) {
    Stop-Job $Job
    Remove-Job $Job
    Write-Error "Failed to connect to Gitea via port-forward."
    exit 1
}

# Create Auth Header
$Pair = "$($AdminUser):$($AdminPass)"
$Encoded = [System.Convert]::ToBase64String([System.Text.Encoding]::ASCII.GetBytes($Pair))
$Headers = @{
    Authorization = "Basic $Encoded"
    "Content-Type" = "application/json"
}

Write-Host "Creating Gitea organization 'helios-platform'..."
try {
    $body = @{
        username = "helios-platform"
        full_name = "Helios Platform"
        visibility = "public"
    } | ConvertTo-Json
    Invoke-RestMethod -Uri "$GiteaBase/api/v1/orgs" -Method Post -Headers $Headers -Body $body
} catch {
    Write-Host "Organization might already exist or request failed: $_"
}

Write-Host "Creating Gitea API token..."
$timestamp = [DateTimeOffset]::Now.ToUnixTimeSeconds()
$tokenName = "helios-platform-$timestamp"
$body = @{
    name = $tokenName
    scopes = @("all")
} | ConvertTo-Json

try {
    $tokenResp = Invoke-RestMethod -Uri "$GiteaBase/api/v1/users/$AdminUser/tokens" -Method Post -Headers $Headers -Body $body
    $token = $tokenResp.sha1
    if ([string]::IsNullOrEmpty($token)) { $token = $tokenResp.token }
} catch {
    Write-Error "Failed to create Gitea API token: $_"
    Stop-Job $Job
    Remove-Job $Job
    exit 1
}

if (-not $token) {
    Write-Error "Could not extract token from response."
    Stop-Job $Job
    Remove-Job $Job
    exit 1
}

Write-Host "Token created successfully."

# Update .env files
$envFiles = @(".env", "apps/portal/.env")

function Update-EnvVar {
    param($filePath, $key, $value)
    if (Test-Path $filePath) {
        $lines = Get-Content $filePath
        $updated = $false
        $newLines = @()
        foreach ($line in $lines) {
            if ($line -match "^$key=") {
                $newLines += "$key=$value"
                $updated = $true
            } else {
                $newLines += $line
            }
        }
        if (-not $updated) {
            $newLines += "$key=$value"
        }
        $newLines | Set-Content $filePath
        Write-Host "Updated $key in $filePath"
    }
}

foreach ($file in $envFiles) {
    Update-EnvVar -filePath $file -key "GITEA_TOKEN" -value $token
    Update-EnvVar -filePath $file -key "GITEA_USER" -value $AdminUser
    Update-EnvVar -filePath $file -key "GITEA_URL" -value $GiteaBase
    Update-EnvVar -filePath $file -key "GITEA_INTERNAL_URL" -value "http://gitea-http.gitea.svc.cluster.local:3000"
}

Write-Host "============================================="
Write-Host "  Gitea configuration complete on Windows!"
Write-Host "============================================="

# Cleanup
Stop-Job $Job
Remove-Job $Job
