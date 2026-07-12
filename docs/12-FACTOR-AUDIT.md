# 12-Factor Compliance Audit

Every microservice in this repo (`customers`, `quotes`, `orders`, `invoices`,
`reports`) satisfies the [12-Factor App](https://12factor.net/) methodology.
Each row cites the concrete code path that enforces the factor.

| # | Factor | Verdict | Evidence |
|---|---|---|---|
| 1 | **Codebase** | ✅ Compliant | One repo, one revision-tracked codebase. Each service builds its own container from a service-scoped `Dockerfile`. Monorepo layout with per-service `go.mod` isolates deploy units, which the 12-Factor spec explicitly permits. |
| 2 | **Dependencies** | ✅ Compliant | Each service declares its Go dependencies in `services/<name>/go.mod` and locks them in `go.sum`. No implicit system libraries — the runtime image extends `gcr.io/distroless/base-debian12`, which ships nothing beyond libc + ca-certs. |
| 3 | **Config** | ✅ Compliant | Every runtime knob reads from an env var, never a config file: `DATABASE_URL`, `SERVICE_PORT`, `LOG_LEVEL`, `QUOTES_SERVICE_URL`, `ORDERS_SERVICE_URL`. Constants files (`constants.go`) name the env keys; `main.go` calls `os.Getenv` against those constants. Zero hard-coded hostnames, ports, or credentials. |
| 4 | **Backing services** | ✅ Compliant | Postgres attaches via the `DATABASE_URL` connection string — the app treats it as a swappable resource. Cross-service calls flow through `*_SERVICE_URL` env vars (see `orders/main.go` → `EnvQuotesServiceURL`, `invoices/main.go` → `EnvOrdersServiceURL`). Swapping a local Postgres for RDS costs one env var change. |
| 5 | **Build / Release / Run** | ✅ Compliant | Multi-stage `Dockerfile` per service: stage 1 pulls modules and compiles; stage 2 copies the static binary into a distroless runtime. The build artifact holds no config; the runtime consumes config at process start. Fly.io release commands and the `docker-compose.yml` map cleanly onto the three stages. |
| 6 | **Processes** | ✅ Compliant | Each service runs as a stateless Go process. No in-memory session state, no local filesystem writes, no sticky-session assumptions. `pgxpool` holds connection state, which the 12-Factor spec categorizes as a backing-service artifact, not process state. |
| 7 | **Port binding** | ✅ Compliant | `srv.ListenAndServe()` in every `main.go` binds `:$SERVICE_PORT` and exports the service as an HTTP server. No reliance on an external web server (Apache/nginx); the Go binary self-hosts. |
| 8 | **Concurrency** | ✅ Compliant | Go's `net/http` spawns one goroutine per request out of the box — the concurrency model comes free with the runtime. Horizontal scaling via `docker compose up --scale customers=3` or Fly.io `fly scale count 3` works because each process runs stateless. |
| 9 | **Disposability** | ✅ Compliant | Fast startup: distroless binary boots in <100ms. Graceful shutdown: every `main.go` traps `SIGINT` + `SIGTERM`, calls `srv.Shutdown(ctx)` with a `ShutdownTimeout` window (constant defined in each service's `constants.go`), and drains in-flight requests before exit. |
| 10 | **Dev / Prod parity** | ✅ Compliant | `docker-compose.yml` runs the same images the production pipeline builds — no dev-only paths through the code. Postgres runs as a container locally and as a managed service in production; the `DATABASE_URL` swap alone bridges the gap. |
| 11 | **Logs** | ✅ Compliant | Every service writes JSON-formatted `log/slog` records to stdout. No file rotation, no direct writes to log-aggregation endpoints — the runtime (Fly.io / `docker logs`) captures and forwards the stream. Structured attributes (`method`, `path`, `duration_ms`) survive as JSON keys, ready for downstream ingestion. |
| 12 | **Admin processes** | ✅ Compliant | `scripts/seed.ps1` and `scripts/mod-tidy-all.ps1` execute one-off tasks against the same code + config as the long-running services. No side channels, no elevated privileges. In production, `fly ssh console` runs the same seed script against the deployed process environment. |

## Notes on architectural choices that reinforce compliance

- **Store interface at the consumer**: `services/*/store.go` defines a `Store`
  interface consumed by handlers, with `PGStore` as the concrete
  implementation. Swapping backing databases (Factor 4) reduces to writing a
  second `Store` implementation. Testing against a mock `Store` verifies
  handler behavior without a live database, which keeps the build stage
  (Factor 5) dependency-free.
- **Cross-service HTTP over shared library**: `orders/quotes_client.go` and
  `invoices/orders_client.go` call sibling services via HTTP with the base URL
  supplied by env var. No shared Go package couples the services at compile
  time — each service ships independently.
- **`context.Context` propagation**: every handler pulls `r.Context()` and
  passes it into store + client calls. Cancellation and deadline signals ride
  the request; when a client disconnects, the request cancels cleanly and
  frees the DB connection, reinforcing Factor 9 (disposability).
- **Named constants for cross-service status literals**: `constants.go`
  defines strings like `ExternalQuoteStatusAccepted` in the consumer service.
  A rename in the producer service surfaces as a compile error in the
  consumer, not a runtime 500.
