# quotetooldemo

Quote-to-cash workflow demo modeled on a paving contractor's back-office
process. Go microservices backend, React + TypeScript + Tailwind frontend,
Docker Compose for local development, Fly.io + Cloudflare Pages for
deployment.

## What it does

Walks a paving contractor's workflow through eight steps:

1. Create a **customer**
2. Draft a **quote** (multi-line-item, area × depth × mix-type → tons → price calculation)
3. Send the quote, watch for acceptance
4. Convert the accepted quote into a **sales order**
5. Fulfill the order
6. Convert the fulfilled order into an **invoice**
7. Mark the invoice paid
8. Watch the **quote-to-cash reports** roll up

## Architecture

Five Go microservices — each a separate deployable, each owning its own
Postgres schema:

| Service | Purpose |
|---|---|
| `customers` | Customer entity CRUD |
| `quotes` | Quote CRUD + line-item calculations + status transitions |
| `orders` | Sales order derived from an accepted quote |
| `invoices` | Invoice derived from a fulfilled order |
| `reports` | Cross-service quote-to-cash reporting |

The React frontend consumes each service through a Cloudflare Worker gateway.
A single shared Postgres instance backs the stack. No auth (intentional demo
scoping — do not deploy against untrusted networks without adding one).

See [`docs/`](docs/) for the architecture deep-dive, ADRs, 12-factor audit,
and importable draw.io diagram.

## Repo layout

```
quotetooldemo/
├── README.md
├── docker-compose.yml           # local dev: Postgres + all services
├── docs/                        # architecture, ADRs, 12-factor audit
├── frontend/                    # React + TypeScript + Tailwind (Cloudflare Pages)
├── services/
│   ├── customers/
│   ├── quotes/
│   ├── orders/
│   ├── invoices/
│   └── reports/
├── migrations/                  # Postgres schemas per service
└── scripts/                     # deploy helpers, seed data, mutation testing
```

## Local dev

```bash
# start Postgres + all services locally
docker-compose up -d

# start the frontend
cd frontend
npm install
npm run dev
```

Services listen on ports 8081-8085. Frontend serves at http://localhost:5173.

Alternative on Windows: `./scripts/build_launch.ps1` tears down containers,
prunes the docker cache, rebuilds, and health-checks every service.

## Testing

```bash
# unit tests, all services
./scripts/run_tests.ps1 -Verbose -Coverage

# mutation testing (requires gremlins:
#   go install github.com/go-gremlins/gremlins/cmd/gremlins@latest)
./scripts/mutation_test.ps1
```

Every service holds a 70%+ mutation kill ratio on covered code. See
[`docs/adr/0006-mutation-testing-target.md`](docs/adr/0006-mutation-testing-target.md).

## Deployment

Services deploy to Fly.io:

```bash
cd services/customers
fly deploy
```

Frontend deploys to Cloudflare Pages via a `frontend/` build. A Cloudflare
Worker gateway routes `/api/<service>/*` to the corresponding Fly app.

## License

Public demo repo. Fork freely.
