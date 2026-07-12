# ADR 0003 — Synchronous HTTP for Cross-Service Calls (Not Event Streaming)

**Status:** Accepted — with a documented escape hatch
**Date:** 2026-07-11

## Context

Two workflow boundaries cross domain lines:

- Creating an order requires reading a quote (to copy total, validate
  status).
- Creating an invoice requires reading an order (to copy amount, validate
  status).

Two shapes fit:

1. **Synchronous HTTP** — `orders` calls `quotes/:id` at create time.
2. **Event streaming** — `quotes` publishes `quote.accepted` events; `orders`
   maintains a local read model of accepted quotes.

## Decision

Use synchronous HTTP for both cross-service edges. Publish events later, once
the workflow demands eventually-consistent replicas (e.g., a search index or
analytics warehouse).

## Consequences

**Positive**

- Simplest possible implementation. Consumers get an immediate 4xx if the
  producer rejects the referenced entity — no eventual-consistency window
  during which an "accepted" quote might still show as "sent" in a stale
  read model.
- The failure mode surfaces as a `502 Bad Gateway` when the producer is down.
  Callers see the failure immediately rather than silently queueing writes
  against stale data.
- Debuggability — a request trace crosses two services and stops. Events
  would sprawl across brokers, dead-letter queues, retry topics.

**Negative**

- Availability coupling — invoices cannot create if orders is down. For a
  back-office workflow with an operator watching the screen when the failure
  surfaces, this trade-off carries lower criticality than a consumer checkout
  flow.
- Latency stacks — invoice-create carries orders-get latency inline. Fine at
  demo scale; would need caching or a shift to events at multi-thousand-QPS.

**Neutral**

- The 5-second `HTTPClientTimeout` constant caps the blast radius. A slow
  producer times out cleanly rather than tying up the consumer indefinitely.

## Alternatives considered

- **Full event-driven architecture (Kafka / NATS).** Right answer for a
  large-scale distributed system. Wrong answer for a 5-service demo where the
  operational overhead of a broker exceeds the value of eventual consistency.
- **gRPC instead of REST.** Type safety wins. Rejected for the demo because
  HTTP+JSON on the wire makes the workflow obvious to a reviewer without a
  protobuf toolchain. A production build would seriously consider gRPC for
  internal service-to-service calls.
- **Direct database reads across services.** Fastest at query time. Rejected
  because it violates the bounded-context ownership boundary from ADR 0001 —
  a schema change in quotes would silently break orders.

## Escape hatch

The one exception lives in the `reports` service, which reads across every
schema. See `services/reports/reports.go`. At production scale, this pattern
moves to materialized views on the individual databases or to a dedicated
read model populated by events. At demo scale, direct SQL joins keep the
code obvious.
