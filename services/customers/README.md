# customers service

CRUD for customer entities. Every quote / order / invoice references a `customer_id` that resolves here.

## Endpoints

| Method | Path | Purpose |
|---|---|---|
| GET | `/health` | Liveness probe |
| POST | `/customers` | Create a customer |
| GET | `/customers?limit=50&offset=0` | List customers, paginated |
| GET | `/customers/{id}` | Retrieve one |
| PUT | `/customers/{id}` | Update |
| DELETE | `/customers/{id}` | Soft-delete |

## Sample request

```bash
curl -X POST http://localhost:8081/customers \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Blackstone Paving",
    "contact_name": "Ellie Cortez",
    "email": "ellie@blackstonepaving.example",
    "phone": "555-201-4433",
    "billing_address": {
      "street": "1140 Industrial Way",
      "city": "Wilmington",
      "state": "DE",
      "zip": "19801"
    }
  }'
```

## Local run

From the repo root:

```bash
docker-compose up -d postgres customers
curl http://localhost:8081/health
```

Or directly (requires local Postgres + Go 1.22):

```bash
DATABASE_URL="postgres://quotetool:quotetool_dev@localhost:5432/quotetool?sslmode=disable" \
SERVICE_PORT=8081 \
go run .
```

## Files

- `models.go` — data types
- `store.go` — Postgres access, `Store` interface + `PGStore` implementation
- `handlers.go` — HTTP handlers, uses Go 1.22 pattern-based routing on `net/http`
- `main.go` — wire-up: env → pool → store → handlers → server with graceful shutdown
- `Dockerfile` — multi-stage build, distroless runtime
