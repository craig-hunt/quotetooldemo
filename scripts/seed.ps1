# seed.ps1
# Seeds demo data by hitting the running services via HTTP. Assumes services
# are running (via ./scripts/build_launch.ps1 or docker compose up) at their
# default local ports.

$ErrorActionPreference = 'Stop'

$CustomersUrl = if ($env:CUSTOMERS_URL) { $env:CUSTOMERS_URL } else { 'http://localhost:8081' }
$QuotesUrl    = if ($env:QUOTES_URL)    { $env:QUOTES_URL }    else { 'http://localhost:8082' }
$OrdersUrl    = if ($env:ORDERS_URL)    { $env:ORDERS_URL }    else { 'http://localhost:8083' }
$InvoicesUrl  = if ($env:INVOICES_URL)  { $env:INVOICES_URL }  else { 'http://localhost:8084' }

function Invoke-Api {
    param(
        [string]$Method,
        [string]$Url,
        [object]$Body
    )
    $params = @{
        Method      = $Method
        Uri         = $Url
        ContentType = 'application/json'
        ErrorAction = 'Stop'
    }
    if ($Body) {
        $params['Body'] = ($Body | ConvertTo-Json -Depth 10 -Compress)
    }
    return Invoke-RestMethod @params
}

Write-Host "==> seeding customers" -ForegroundColor Cyan
$cust1 = Invoke-Api -Method POST -Url "$CustomersUrl/customers" -Body @{
    name           = 'Blackstone Paving'
    contact_name   = 'Ellie Cortez'
    email          = 'ellie@blackstonepaving.example'
    phone          = '555-201-4433'
    billing_address = @{
        street = '1140 Industrial Way'
        city   = 'Wilmington'
        state  = 'DE'
        zip    = '19801'
    }
}
Write-Host "    customer 1: $($cust1.id)"

$cust2 = Invoke-Api -Method POST -Url "$CustomersUrl/customers" -Body @{
    name           = 'Ridgeline Contractors'
    contact_name   = 'Marcus Reyes'
    email          = 'marcus@ridgelinecontractors.example'
    phone          = '555-338-1129'
    billing_address = @{
        street = '22 Quarry Road'
        city   = 'Reading'
        state  = 'PA'
        zip    = '19601'
    }
}
Write-Host "    customer 2: $($cust2.id)"

Write-Host "==> seeding quotes" -ForegroundColor Cyan
$quote1 = Invoke-Api -Method POST -Url "$QuotesUrl/quotes" -Body @{
    customer_id     = $cust1.id
    project_name    = 'Warehouse Lot Repave'
    project_address = '1140 Industrial Way, Wilmington, DE 19801'
    tax_rate        = 0.06
    markup_rate     = 0.15
    notes           = 'Full-depth reconstruction on primary lot'
    line_items      = @(
        @{ area_sqft = 20000; depth_inches = 4; mix_type = 'hma_base';    unit_price_per_ton = 88.00 },
        @{ area_sqft = 20000; depth_inches = 2; mix_type = 'hma_surface'; unit_price_per_ton = 105.00 }
    )
}
Write-Host "    quote 1: $($quote1.id)"

$quote2 = Invoke-Api -Method POST -Url "$QuotesUrl/quotes" -Body @{
    customer_id     = $cust2.id
    project_name    = 'Sitework Overlay'
    project_address = '22 Quarry Road, Reading, PA 19601'
    tax_rate        = 0.06
    markup_rate     = 0.12
    line_items      = @(
        @{ area_sqft = 8500; depth_inches = 2; mix_type = 'superpave'; unit_price_per_ton = 115.00 }
    )
}
Write-Host "    quote 2: $($quote2.id)"

Write-Host "==> sending + accepting quote 1" -ForegroundColor Cyan
Invoke-Api -Method POST -Url "$QuotesUrl/quotes/$($quote1.id)/send" | Out-Null
Invoke-Api -Method POST -Url "$QuotesUrl/quotes/$($quote1.id)/accept" | Out-Null

Write-Host "==> creating order from quote 1" -ForegroundColor Cyan
$order1 = Invoke-Api -Method POST -Url "$OrdersUrl/orders" -Body @{
    quote_id = $quote1.id
    notes    = 'Schedule for next week'
}
Write-Host "    order 1: $($order1.id)"

Write-Host "==> fulfilling order 1" -ForegroundColor Cyan
Invoke-Api -Method POST -Url "$OrdersUrl/orders/$($order1.id)/fulfill" | Out-Null

Write-Host "==> creating invoice from order 1" -ForegroundColor Cyan
$invoice1 = Invoke-Api -Method POST -Url "$InvoicesUrl/invoices" -Body @{
    order_id  = $order1.id
    due_days  = 30
}
Write-Host "    invoice 1: $($invoice1.id)"

Write-Host "==> sending invoice 1" -ForegroundColor Cyan
Invoke-Api -Method POST -Url "$InvoicesUrl/invoices/$($invoice1.id)/send" | Out-Null

Write-Host ""
Write-Host "==> seed complete" -ForegroundColor Green
