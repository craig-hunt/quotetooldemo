# mutation_test.ps1
# Runs `gremlins` (Go mutation-testing tool) across every service and reports the
# per-service mutation kill ratio.
#
# gremlins: https://github.com/go-gremlins/gremlins
# Install once:
#   go install github.com/go-gremlins/gremlins/cmd/gremlins@latest
#
# Target: 70% kill ratio per service (industry standard).
# Exit code non-zero if any service falls under the threshold.
#
# Uses the integration build tag by default so gremlins picks up both unit +
# integration tests. Pass -NoIntegration to run against unit tests only
# (skips Postgres-container spin-up per mutation, much faster).
#
# Usage:
#   .\scripts\mutation_test.ps1
#   .\scripts\mutation_test.ps1 -Threshold 0.70
#   .\scripts\mutation_test.ps1 -Service quotes         # single service
#   .\scripts\mutation_test.ps1 -NoIntegration           # skip integration tests

[CmdletBinding()]
param(
    [double]$Threshold = 0.70,
    [string]$Service = '',
    [switch]$NoIntegration
)

$ErrorActionPreference = 'Stop'

if (-not (Get-Command gremlins -ErrorAction SilentlyContinue)) {
    Write-Error "gremlins not on PATH. install it with: go install github.com/go-gremlins/gremlins/cmd/gremlins@latest"
    exit 1
}

$root = Split-Path -Parent $PSScriptRoot
$servicesPath = Join-Path $root 'services'

$services = @('customers', 'quotes', 'orders', 'invoices', 'reports')
if ($Service) { $services = @($Service) }

$results = @()
$anyBelow = $false

foreach ($svc in $services) {
    $svcPath = Join-Path $servicesPath $svc
    if (-not (Test-Path $svcPath)) {
        Write-Warning "skipping missing service: $svc"
        continue
    }

    Write-Host ""
    Write-Host "==> mutation-testing services/$svc" -ForegroundColor Cyan

    Push-Location $svcPath
    try {
        # gremlins prints a summary like "Killed X, Lived Y, Not viable Z" and a
        # final "efficacy: NN%". We capture stdout, then parse the ratio.
        $gremlinsArgs = @('unleash', '--output', 'json')
        if (-not $NoIntegration) { $gremlinsArgs += @('--tags', 'integration') }
        $out = & gremlins @gremlinsArgs 2>&1
        $exit = $LASTEXITCODE

        # Try JSON first; fall back to text scrape if the JSON schema shifts.
        $killed = 0
        $lived = 0
        $ratio = 0.0

        try {
            $json = $out -join "`n" | ConvertFrom-Json
            $killed = [int]($json.killed)
            $lived  = [int]($json.lived)
            $total  = $killed + $lived
            if ($total -gt 0) { $ratio = $killed / $total }
        }
        catch {
            $killMatch = ($out | Select-String -Pattern '(\d+)\s+killed').Matches
            $liveMatch = ($out | Select-String -Pattern '(\d+)\s+lived').Matches
            if ($killMatch.Count -gt 0) { $killed = [int]$killMatch[0].Groups[1].Value }
            if ($liveMatch.Count -gt 0) { $lived = [int]$liveMatch[0].Groups[1].Value }
            $total = $killed + $lived
            if ($total -gt 0) { $ratio = $killed / $total }
        }

        $meetsThreshold = $ratio -ge $Threshold
        if (-not $meetsThreshold) { $anyBelow = $true }

        $results += [pscustomobject]@{
            Service  = $svc
            Killed   = $killed
            Lived    = $lived
            KillRatio = '{0:P1}' -f $ratio
            MeetsThreshold = $meetsThreshold
            ExitCode = $exit
        }
    }
    finally {
        Pop-Location
    }
}

Write-Host ""
Write-Host "==> summary (threshold: $('{0:P0}' -f $Threshold))" -ForegroundColor Cyan
$results | Format-Table -AutoSize

if ($anyBelow) {
    Write-Warning "one or more services below the $('{0:P0}' -f $Threshold) kill-ratio threshold"
    exit 1
}
Write-Host "all services meet the threshold" -ForegroundColor Green
