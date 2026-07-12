# Architecture

## System overview

A quote-to-cash tool for a paving contractor, decomposed into five
domain-aligned Go microservices behind a single React SPA. Each service owns
its data, exposes a REST-ish HTTP interface, and communicates with siblings
via HTTP calls when a workflow spans domain boundaries.

## Runtime topology

```
┌────────────────────────────────────────────────────────────────┐
│  React SPA (Vite + TypeScript + Tailwind)                      │
│  hosted on Cloudflare Pages                                    │
└──────────────────┬─────────────────────────────────────────────┘
                   │  fetch / JSON
                   ▼
┌────────────────────────────────────────────────────────────────┐
│  API Gateway (Cloudflare Worker or direct browser calls)       │
└──────────────────┬─────────────────────────────────────────────┘
                   │
   ┌───────────────┼───────────────┬───────────────┬─────────────┐
   ▼               ▼               ▼               ▼             ▼
┌────────┐    ┌────────┐      ┌────────┐      ┌────────┐    ┌────────┐
│customers│   │ quotes │      │ orders │      │invoices│    │reports │
│  :8080  │   │  :8080 │      │  :8080 │      │  :8080 │    │  :8080 │
└────┬────┘   └───┬────┘      └───┬────┘      └───┬────┘    └───┬────┘
     │            │      ┌─────── │       ┌────── │             │
     │            │      │        │       │       │             │
     │            │      ▼        │       ▼       │             │
     │            │   quotes───┐  │    orders──┐  │             │
     │            │            │  │            │  │             │
     │            │            │  ▼            │  ▼             │
     │            │            (calls quotes)   (calls orders)  │
     │            │                                             │
     └────────────┼─────────────────────────────────────────────┘
                  │
                  ▼
     ┌─────────────────────────────────────────┐
     │  PostgreSQL 15                          │
     │  ────────────────────────────────────── │
     │  schemas: customers, quotes, orders,    │
     │           invoices                       │
     │  (reports reads across all four)         │
     └─────────────────────────────────────────┘
```

## Domain boundaries

Each service owns exactly one bounded context and its persistent state.

| Service | Domain | Owns |
|---|---|---|
| **customers** | Customer master data | `customers.customers` — name, contact, billing address |
| **quotes** | Quote generation + calc | `quotes.quotes`, `quotes.quote_line_items` — line items, tonnage math, tax/markup |
| **orders** | Fulfillment scheduling | `orders.orders` + `orders.order_number_seq` — schedule dates, fulfilled/cancelled state |
| **invoices** | Billing + collections | `invoices.invoices` + `invoices.invoice_number_seq` — due dates, sent/paid/overdue state |
| **reports** | Analytics + KPIs | (no schema; reads from every other schema) |

## Cross-service coupling

Only three edges cross domain boundaries. Every edge is HTTP; no service reads
another's tables directly (reports is the sole reader-across-schemas exception,
which the ADR discusses).

| Edge | Purpose | Guard |
|---|---|---|
| `orders → quotes` | On order create, fetch the quote by ID. | Reject if quote status ≠ `accepted`. |
| `invoices → orders` | On invoice create, fetch the order by ID. | Reject if order status ≠ `fulfilled`. |
| `reports → all schemas` | Read-only cross-schema joins for funnel + aging + cycle-time reports. | Read-only; no writes back. |

## State machines

Both `orders` and `quotes` and `invoices` enforce state transitions in code
(see `services/*/store.go` → `allowedTransition`). Attempting an illegal
transition returns `409 Conflict` with the `ErrMsgInvalidTransition` payload.

### Quote lifecycle

```
draft ──> sent ──┬──> accepted (terminal)
                 ├──> rejected (terminal)
                 └──> expired  (terminal)
```

### Order lifecycle

```
open ──┬──> in_progress ──┬──> fulfilled (terminal)
       │                  └──> cancelled (terminal)
       ├──> fulfilled (terminal — skip-ahead permitted)
       └──> cancelled (terminal)
```

### Invoice lifecycle

```
draft ──┬──> sent ──┬──> paid    (terminal)
        │           ├──> overdue ──┬──> paid    (terminal)
        │           │              └──> cancelled (terminal)
        │           └──> cancelled (terminal)
        └──> cancelled (terminal)
```

## Deployment surface

| Component | Host | Config source |
|---|---|---|
| Frontend | Cloudflare Pages | Build-time env at Cloudflare, no secrets |
| Every Go service | Fly.io | Fly secrets → env vars (`DATABASE_URL`, `*_SERVICE_URL`, `LOG_LEVEL`) |
| PostgreSQL 15 | Fly Postgres | Managed volume, connection string via Fly secret |

Development uses `docker-compose.yml` to spin the same images against a local
Postgres container. Dev and prod share build artifacts (Factor 10).

## Observability

- **Structured logs** — every service writes JSON `log/slog` records to
  stdout. Fly + `docker logs` capture the stream; a downstream aggregator
  (e.g., Grafana Loki) parses attributes without regex.
- **Request tracing** — each HTTP request logs `method`, `path`, and
  `duration_ms`. Extending to full OTel tracing needs one `otelhttp.Handler`
  wrap in each `main.go`.
- **Health endpoints** — every service exposes `GET /health` returning
  `{"status":"ok"}` for use by load balancers, Fly health checks, and the
  `build_launch.ps1` smoke script.

## Testing

- **Unit tests** — handlers exercised against a `MockStore`; state machines
  tested via full-transition matrices; cross-service clients tested against
  `httptest.NewServer` fakes.
- **Mutation tests** — `gremlins` runs across each service, holding a 70%+
  kill-ratio threshold on covered mutants (see `docs/adr/0006-mutation-testing.md`).
- **Integration tests** (not yet written for demo scope) — would exercise
  `store.go` SQL paths against an ephemeral Postgres via `dockertest`.
