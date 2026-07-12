# ADR 0004 — State-Machine Enforcement in Application Code, Not the Database

**Status:** Accepted
**Date:** 2026-07-11

## Context

Three services (`quotes`, `orders`, `invoices`) carry lifecycle state
machines. Illegal transitions must fail with a clear error, and the transition
graph must live somewhere authoritative.

Two shapes:

1. **In the database** — a `CHECK` constraint per table, or a `BEFORE UPDATE`
   trigger that reads old + new status and rejects illegal pairs.
2. **In application code** — an `allowedTransition` function in each service's
   `store.go` that runs before the SQL `UPDATE`.

## Decision

Enforce transitions in application code. See
`services/*/store.go` → `allowedTransition`. The database constraint layer
enforces enum membership (`status IN ('draft','sent',...)`); the transition
graph itself lives in Go.

## Consequences

**Positive**

- Readable, testable, and mutation-testable. Every transition matrix has a
  dedicated `state_test.go` that walks every (from, to) pair and asserts the
  expected boolean. Mutation testing (see ADR 0006) confirms the assertions
  actually distinguish valid from invalid transitions.
- Illegal transitions surface as a typed error (`ErrInvalidStatus`) that
  handlers translate into a `409 Conflict` with the exact reason. A DB trigger
  would surface as a raw SQL error that the handler layer has to sniff and
  translate.
- Transition rules evolve with domain logic. A `BEFORE UPDATE` trigger written
  in PL/pgSQL evolves on a separate release cadence, needs a DBA to review,
  and lives in a language most Go engineers do not write.

**Negative**

- A rogue SQL client (a support engineer running an ad-hoc `UPDATE`) can
  bypass the transition guard. Mitigated by:
  1. Only the service holds write credentials for its own schema.
  2. Any operational tooling routes through the HTTP API, not raw SQL.

**Neutral**

- The `CHECK` constraint on the enum type still catches truly invalid values
  (typos, misspelled statuses). Application-level transition guards catch
  illegal *combinations* of otherwise-valid values.

## Alternatives considered

- **Database triggers.** See "negative" above. Not chosen.
- **Third-party state-machine library** (e.g., `looplab/fsm`). Adds a
  dependency for what is a 15-line hand-rolled function. Rejected for
  simplicity.
- **Declarative state graph in YAML config.** Overkill; the state graphs are
  small and stable and belong next to the code that uses them.
