# run_tests.ps1
# Runs `go test` across every service. Prints a per-service pass/fail summary and
# exits non-zero if any service fails.
#
# Usage:
#   .\scripts\run_tests.ps1
#   .\scripts\run_tests.ps1 -Verbose    # -v flag on go test
#   .\scripts\run_tests.ps1 -Coverage   # emit per-service coverage.out + %

[CmdletBinding()]
param(
    [switch]$Coverage
)

$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
$servicesPath = Join-Path $root 'services'

$services = @(
    'customers',
    'quotes',
    'orders',
    'invoices',
    'reports'
)

$goFlags = @()
if ($VerbosePreference -eq 'Continue') { $goFlags += '-v' }
if ($Coverage) { $goFlags += @('-covermode=atomic') }

$results = @()
$anyFailed = $false

foreach ($svc in $services) {
    $svcPath = Join-Path $servicesPath $svc
    if (-not (Test-Path $svcPath)) {
        Write-Warning "skipping missing service: $svc"
        continue
    }

    Write-Host ""
    Write-Host "==> testing services/$svc" -ForegroundColor Cyan

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

Write-Host ""
Write-Host "==> summary" -ForegroundColor Cyan
$results | Format-Table -AutoSize

if ($anyFailed) {
    Write-Error "one or more services failed tests"
    exit 1
}

Write-Host "all services passed" -ForegroundColor Green
