# quotetooldemo — Documentation

Complete documentation set for the quote-to-cash demo project.

## Contents

- **[ARCHITECTURE.md](ARCHITECTURE.md)** — System topology, domain boundaries,
  state machines, deployment surface, observability, testing strategy.
- **[12-FACTOR-AUDIT.md](12-FACTOR-AUDIT.md)** — Per-factor compliance
  verification with concrete code citations for all five services.
- **[architecture-diagram.drawio](architecture-diagram.drawio)** — Editable
  draw.io source. Open at [app.diagrams.net](https://app.diagrams.net) → File
  → Open From → Device.
- **[adr/](adr/)** — Architectural Decision Records.

## ADR index

Each ADR follows the format: Context → Decision → Consequences (positive /
negative / neutral) → Alternatives considered.

| # | Decision |
|---|---|
| [0001](adr/0001-microservices-per-bounded-context.md) | One microservice per bounded context |
| [0002](adr/0002-store-interface-at-consumer.md) | Store interface defined at the consumer, not the persistence package |
| [0003](adr/0003-cross-service-http-not-events.md) | Synchronous HTTP for cross-service calls (not event streaming) — with escape hatch |
| [0004](adr/0004-state-machine-in-code-not-db.md) | State-machine enforcement in Go code, not database triggers |
| [0005](adr/0005-named-constants-no-magic-strings.md) | Every string and number lives in a named constant |
| [0006](adr/0006-mutation-testing-target.md) | Mutation testing to 70%+ kill ratio (industry standard) |

## Reading order

Start with the architecture diagram plus the ADR index. Every non-trivial
design choice carries a documented rationale, including choices reasonable
engineers might disagree with. The consequences and alternatives sections
surface the trade-offs worth discussing.
