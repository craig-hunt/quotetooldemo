# ADR 0002 — Store Interface Defined at the Consumer

**Status:** Accepted
**Date:** 2026-07-11

## Context

Every microservice needs a persistence layer. The Go idiom offers two shapes:

1. **Interface with implementation** — the persistence package exports a
   `Store` interface plus a concrete `PGStore` implementation. Callers import
   both.
2. **Interface at consumer** — the handlers define the `Store` interface they
   need; the persistence package exports only concrete types (`PGStore`).

## Decision

Every service defines `Store` at the consumer (in `handlers.go` /
`store.go` within the same `main` package). The `PGStore` struct satisfies
the interface implicitly. Tests substitute a `MockStore` that also implicitly
satisfies the interface.

## Consequences

**Positive**

- Handlers depend on the minimum surface they actually call. If the invoice
  handler needs only `Create`, `Get`, `List`, `Transition`, the interface
  captures exactly those four methods — no `Update` bloat carried along for
  future callers.
- Mock stores stay tiny. Each `MockStore` in `services/*/mockstore_test.go`
  implements the same four-to-five methods and nothing more.
- Persistence-layer changes stay local. Adding a query helper to `PGStore`
  requires zero handler changes.

**Negative**

- Small amount of duplication — every service redefines `Store` for its own
  domain. Sharing a base interface across services would fight the
  bounded-context split from ADR 0001.

**Neutral**

- Go's structural typing makes this pattern free. No `implements` keyword, no
  registration boilerplate.

## Alternatives considered

- **Interface exported from the persistence package.** Classic OOP shape. In
  Go, this couples the consumer to the persistence package's view of the
  world — handlers become import-order dependencies on `store.go`.
- **Repository pattern with generic base.** Adds abstraction the demo does not
  need. Two of the five services (`customers`, `quotes`) have enough
  variation in their store shape that a generic base would leak type
  parameters into every call site.
