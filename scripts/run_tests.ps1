# run_tests.ps1
# Runs `go test` across every service and (optionally) integration tests plus
# the frontend Vitest suite. Prints a per-service pass/fail summary and exits
# non-zero if any suite fails.
#
# Usage:
#   .\scripts\run_tests.ps1                       # unit tests only
#   .\scripts\run_tests.ps1 -Verbose              # -v flag on go test
#   .\scripts\run_tests.ps1 -Coverage             # emit per-service coverage.out + %
#   .\scripts\run_tests.ps1 -Integration          # unit + Postgres-container integration
#   .\scripts\run_tests.ps1 -Integration -Frontend # everything, including Vitest
#
# Integration tests require Docker Desktop running (testcontainers-go spins
# ephemeral Postgres containers).

[CmdletBinding()]
param(
    [switch]$Coverage,
    [switch]$Integration,
    [switch]$Frontend
)

$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
$servicesPath = Join-Path $root 'services'
$frontendPath = Join-Path $root 'frontend'

$services = @('customers', 'quotes', 'orders', 'invoices', 'reports')

$goFlags = @()
if ($VerbosePreference -eq 'Continue') { $goFlags += '-v' }
if ($Coverage) { $goFlags += @('-covermode=atomic') }
if ($Integration) { $goFlags += @('-tags=integration') }

$results = @()
$anyFailed = $false

foreach ($svc in $services) {
    $svcPath = Join-Path $servicesPath $svc
    if (-not (Test-Path $svcPath)) {
        Write-Warning "skipping missing service: $svc"
        continue
    }

    $label = if ($Integration) { "$svc (unit + integration)" } else { $svc }
    Write-Host ""
    Write-Host "==> testing services/$label" -ForegroundColor Cyan

    Push-Location $svcPath
    try {
        $args = @('test') + $goFlags
        if ($Coverage) {
            $covFile = Join-Path $svcPath 'coverage.out'
            $args += @("-coverprofile=$covFile")
        }
        $args += './...'

        & go @args
        $exit = $LASTEXITCODE

        $pctLine = ''
        if ($Coverage -and $exit -eq 0) {
            $covFile = Join-Path $svcPath 'coverage.out'
            if (Test-Path $covFile) {
                $covOut = & go tool cover -func=$covFile | Select-String -Pattern 'total:' | Select-Object -Last 1
                if ($covOut) { $pctLine = $covOut.ToString().Trim() }
            }
        }

        $results += [pscustomobject]@{
            Service  = $svc
            ExitCode = $exit
            Coverage = $pctLine
        }
        if ($exit -ne 0) { $anyFailed = $true }
    }
    finally {
        Pop-Location
    }
}

if ($Frontend) {
    Write-Host ""
    Write-Host "==> testing frontend (Vitest)" -ForegroundColor Cyan
    Push-Location $frontendPath
    try {
        if ($Coverage) {
            & npm run test:coverage
        } else {
            & npm test
        }
        $feExit = $LASTEXITCODE
        $results += [pscustomobject]@{
            Service  = 'frontend'
            ExitCode = $feExit
            Coverage = ''
        }
        if ($feExit -ne 0) { $anyFailed = $true }
    }
    finally {
        Pop-Location
    }
}

Write-Host ""
Write-Host "==> summary" -ForegroundColor Cyan
$results | Format-Table -AutoSize

if ($anyFailed) {
    Write-Error "one or more suites failed"
    exit 1
}

Write-Host "all suites passed" -ForegroundColor Green
