#!/usr/bin/env bash
# Seeds demo data by hitting the running services via HTTP. Assumes services
# are running (docker-compose up) at their default local ports.

set -euo pipefail

CUSTOMERS_URL="${CUSTOMERS_URL:-http://localhost:8081}"
QUOTES_URL="${QUOTES_URL:-http://localhost:8082}"
ORDERS_URL="${ORDERS_URL:-http://localhost:8083}"
INVOICES_URL="${INVOICES_URL:-http://localhost:8084}"

echo "==> seeding customers"
CUST_1=$(curl -s -X POST "${CUSTOMERS_URL}/customers" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Blackstone Paving",
    "contact_name": "Ellie Cortez",
    "email": "ellie@blackstonepaving.example",
    "phone": "555-201-4433",
    "billing_address": {"street": "1140 Industrial Way", "city": "Wilmington", "state": "DE", "zip": "19801"}
  }' | jq -r .id)
echo "customer 1: ${CUST_1}"

CUST_2=$(curl -s -X POST "${CUSTOMERS_URL}/customers" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Ridgeline Contractors",
    "contact_name": "Marcus Reyes",
    "email": "marcus@ridgelinecontractors.example",
    "phone": "555-338-1129",
    "billing_address": {"street": "22 Quarry Road", "city": "Reading", "state": "PA", "zip": "19601"}
  }' | jq -r .id)
echo "customer 2: ${CUST_2}"

echo "==> seeding quotes"
QUOTE_1=$(curl -s -X POST "${QUOTES_URL}/quotes" \
  -H "Content-Type: application/json" \
  -d "{
    \"customer_id\": \"${CUST_1}\",
    \"project_name\": \"Warehouse Lot Repave\",
    \"project_address\": \"1140 Industrial Way, Wilmington, DE 19801\",
    \"tax_rate\": 0.06,
    \"markup_rate\": 0.15,
    \"notes\": \"Full-depth reconstruction on primary lot\",
    \"line_items\": [
      {\"area_sqft\": 20000, \"depth_inches\": 4, \"mix_type\": \"hma_base\", \"unit_price_per_ton\": 88.00},
      {\"area_sqft\": 20000, \"depth_inches\": 2, \"mix_type\": \"hma_surface\", \"unit_price_per_ton\": 105.00}
    ]
  }" | jq -r .id)
echo "quote 1: ${QUOTE_1}"

QUOTE_2=$(curl -s -X POST "${QUOTES_URL}/quotes" \
  -H "Content-Type: application/json" \
  -d "{
    \"customer_id\": \"${CUST_2}\",
    \"project_name\": \"Sitework Overlay\",
    \"project_address\": \"22 Quarry Road, Reading, PA 19601\",
    \"tax_rate\": 0.06,
    \"markup_rate\": 0.12,
    \"line_items\": [
      {\"area_sqft\": 8500, \"depth_inches\": 2, \"mix_type\": \"superpave\", \"unit_price_per_ton\": 115.00}
    ]
  }" | jq -r .id)
echo "quote 2: ${QUOTE_2}"

echo "==> sending + accepting quote 1"
curl -s -X POST "${QUOTES_URL}/quotes/${QUOTE_1}/send" > /dev/null
curl -s -X POST "${QUOTES_URL}/quotes/${QUOTE_1}/accept" > /dev/null

echo "==> creating order from quote 1"
ORDER_1=$(curl -s -X POST "${ORDERS_URL}/orders" \
  -H "Content-Type: application/json" \
  -d "{\"quote_id\": \"${QUOTE_1}\", \"notes\": \"Schedule for next week\"}" | jq -r .id)
echo "order 1: ${ORDER_1}"

echo "==> fulfilling order 1"
curl -s -X POST "${ORDERS_URL}/orders/${ORDER_1}/fulfill" > /dev/null

echo "==> creating invoice from order 1"
INVOICE_1=$(curl -s -X POST "${INVOICES_URL}/invoices" \
  -H "Content-Type: application/json" \
  -d "{\"order_id\": \"${ORDER_1}\", \"due_days\": 30}" | jq -r .id)
echo "invoice 1: ${INVOICE_1}"

echo "==> sending invoice 1"
curl -s -X POST "${INVOICES_URL}/invoices/${INVOICE_1}/send" > /dev/null

echo "==> seed complete"
