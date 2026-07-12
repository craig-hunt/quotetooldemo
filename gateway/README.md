# quotedemo-gateway

Single Cloudflare Worker that routes `/api/<service>/*` requests to the
corresponding Fly.io service upstream.

## Routing

| Incoming path | Forwarded to |
|---|---|
| `/api/customers/*` | `https://quotedemo-customers.fly.dev/*` |
| `/api/quotes/*` | `https://quotedemo-quotes.fly.dev/*` |
| `/api/orders/*` | `https://quotedemo-orders.fly.dev/*` |
| `/api/invoices/*` | `https://quotedemo-invoices.fly.dev/*` |
| `/api/reports/*` | `https://quotedemo-reports.fly.dev/*` |

The `/api/<service>` prefix strips off before the upstream call. Query
strings and request headers pass through unchanged.

## Deploy

One-time install:

```bash
npm install -g wrangler
wrangler login
```

Deploy or update the Worker:

```bash
cd gateway
wrangler deploy
```

The route pattern in `wrangler.toml` only takes effect once
`quotedemo.sagecrestsolutions.com` exists as a Cloudflare Pages custom
domain (step 5 of the deploy sequence).
