# ADR 0001 — One Microservice Per Bounded Context

**Status:** Accepted
**Date:** 2026-07-11

## Context

The quote-to-cash workflow spans five distinct concerns: customer master data,
quote generation with pricing math, order scheduling, invoicing/collections,
and analytics. A monolithic Go application could serve all five. A single-page
demo-project scaffold would probably reach for that shape.

The architectural question sits at whether the system serves the shape of the
business, not the shape of the current sprint. Paving companies grow into
quote-lots + fleet-scheduling + AP/AR concerns that live in different teams
and evolve on different cadences.

## Decision

Split the system into five Go microservices — one per bounded context:
`customers`, `quotes`, `orders`, `invoices`, `reports`. Each service owns its
own database schema, ships in its own container, and deploys independently.

## Consequences

**Positive**

- Independent deploy cadence — the quotes team ships a pricing-rule change
  without a coordinated cross-team release.
- Smaller blast radius — a bug in invoice-status logic cannot corrupt customer
  records.
- Clear team boundaries — Conway's Law rewards codebases that mirror
  ownership, not the other way around.
- Language flexibility — a future ML-driven pricing engine could replace the
  quotes service with a Python implementation without disturbing siblings.

**Negative**

- Higher operational floor — five services means five sets of logs, five
  deploy pipelines, five sets of health checks.
- Cross-service consistency requires deliberate design — see ADR 0003.
- Local development needs `docker-compose` or a service-orchestration layer;
  a monolith would `go run .` and finish.

**Neutral**

- The database still sits in one Postgres instance with schema-per-service
  isolation. A future move to schema-per-database happens by changing the
  `DATABASE_URL` for each service — no code change.

## Alternatives considered

- **Monolith with package-level modularity.** Faster to build. Rejected
  because it hides the organizational-scale coupling patterns that a
  multi-team production system eventually has to manage.
- **Modular monolith with in-process module boundaries.** Middle-ground.
  Rejected because it hides the actual coupling patterns — cross-service HTTP
  calls with status validation surface exactly the kind of API-contract
  discussion the design deserves.
- **Serverless functions per endpoint.** Overkill for a synchronous
  transactional workflow. Cold starts would harm the quote-creation flow.
