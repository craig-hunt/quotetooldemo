# mod-tidy-all.ps1
# Runs `go mod tidy` in each service directory. Produces go.sum files so
# Docker builds can verify module hashes. Run once after cloning or after
# adding dependencies. Requires Go 1.22+ on PATH.

$ErrorActionPreference = 'Stop'

$RepoRoot = Split-Path -Parent $PSScriptRoot
$Services = @('customers', 'quotes', 'orders', 'invoices', 'reports')

foreach ($svc in $Services) {
    $ServicePath = Join-Path $RepoRoot "services\$svc"
    if (-not (Test-Path $ServicePath)) {
        Write-Host "==> skipping missing service directory: services/$svc" `
            -ForegroundColor Yellow
        continue
    }
    Write-Host "==> tidying services/$svc" -ForegroundColor Cyan
    Push-Location $ServicePath
    try {
        go mod tidy
        if ($LASTEXITCODE -ne 0) {
            throw "go mod tidy failed in services/$svc"
        }
    } finally {
        Pop-Location
    }
}

Write-Host "==> done" -ForegroundColor Green
