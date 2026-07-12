# build_launch.ps1
# Tears down any running containers + volumes, prunes docker cache, rebuilds
# all images without cache, brings the full stack up, waits for health checks.
# Run from the repo root.

$ErrorActionPreference = 'Stop'

$RepoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $RepoRoot

Write-Host "==> tearing down existing stack" -ForegroundColor Cyan
docker compose down -v --remove-orphans

Write-Host "==> pruning docker build cache" -ForegroundColor Cyan
docker builder prune -af | Out-Null

Write-Host "==> pruning dangling images + volumes" -ForegroundColor Cyan
docker system prune -f --volumes | Out-Null

Write-Host "==> building all services (no cache)" -ForegroundColor Cyan
docker compose build --no-cache

Write-Host "==> starting stack" -ForegroundColor Cyan
docker compose up -d

Write-Host "==> waiting for health checks (up to 60s)" -ForegroundColor Cyan
$Services = @(
    @{ Name = 'customers'; Port = 8081 },
    @{ Name = 'quotes';    Port = 8082 },
    @{ Name = 'orders';    Port = 8083 },
    @{ Name = 'invoices';  Port = 8084 },
    @{ Name = 'reports';   Port = 8085 }
)

$Deadline = (Get-Date).AddSeconds(60)
$Ready = @{}
foreach ($svc in $Services) { $Ready[$svc.Name] = $false }

while ((Get-Date) -lt $Deadline) {
    $AllReady = $true
    foreach ($svc in $Services) {
        if ($Ready[$svc.Name]) { continue }
        try {
            $resp = Invoke-WebRequest -Uri "http://localhost:$($svc.Port)/health" `
                -TimeoutSec 2 -UseBasicParsing -ErrorAction Stop
            if ($resp.StatusCode -eq 200) {
                $Ready[$svc.Name] = $true
                Write-Host "    $($svc.Name) OK" -ForegroundColor Green
            }
        } catch {
            $AllReady = $false
        }
    }
    if ($AllReady) { break }
    Start-Sleep -Seconds 2
}

$Failed = @()
foreach ($svc in $Services) {
    if (-not $Ready[$svc.Name]) { $Failed += $svc.Name }
}

if ($Failed.Count -gt 0) {
    Write-Host "==> services failed health check: $($Failed -join ', ')" -ForegroundColor Red
    Write-Host "    check logs with: docker compose logs <service>" -ForegroundColor Yellow
    exit 1
}

Write-Host ""
Write-Host "==> stack up and healthy" -ForegroundColor Green
Write-Host ""
Write-Host "    customers: http://localhost:8081"
Write-Host "    quotes:    http://localhost:8082"
Write-Host "    orders:    http://localhost:8083"
Write-Host "    invoices:  http://localhost:8084"
Write-Host "    reports:   http://localhost:8085"
Write-Host ""
Write-Host "    seed demo data: ./scripts/seed.ps1"
Write-Host "    tear down:      docker compose down -v"
Write-Host ""
