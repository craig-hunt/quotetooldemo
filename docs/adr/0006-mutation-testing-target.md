# ADR 0006 — Mutation Testing to 70%+ Kill Ratio (Industry Standard)

**Status:** Accepted
**Date:** 2026-07-11

## Context

Line-coverage percentages lie. A test that executes a line without asserting
on its output claims coverage without providing any guarantee. Mutation
testing measures whether the assertions are load-bearing: it mutates the
production code (flipping `>` to `>=`, `&&` to `||`, negating conditionals,
etc.) and checks whether any test fails. A surviving mutant means either the
test was cosmetic or the mutation is semantically equivalent.

The team target: 70% mutation kill ratio on covered code — the widely-cited
industry standard.

## Decision

Every service ships with a mutation-test pass via
[`gremlins`](https://github.com/go-gremlins/gremlins). The
`scripts/mutation_test.ps1` runner enforces a 70% floor per service and
exits non-zero if any service drops below.

## Consequences

**Positive**

- Tests earn their keep. A cosmetic assertion (e.g., `if err == nil`) that
  never distinguishes real success from silent failure gets caught the first
  time a mutation slips through.
- Boundary conditions get explicit coverage. Mutations like `>=` → `>` force
  tests that pin the exact edge value.
- Documents the "why" behind every assertion — if I can't kill the mutation,
  the test was there for the wrong reason.

**Negative**

- Mutation testing is slow. Each mutation forks a test run. On this repo
  (~5-second test suite per service), each service takes 3-6 seconds for a
  full pass — acceptable. On larger codebases, mutation runs move to CI-only,
  not every-commit.
- SQL-layer coverage stays outside the mutation frame. `store.go` mutants
  land under "not covered" until integration tests exist against a live
  Postgres. The demo defers that; production would add `dockertest`-driven
  integration passes.

**Neutral**

- Some equivalent mutants (e.g., `n >= 0` mutated to `n > 0` where `n` can
  only be 0 or positive after the guard) survive by construction. Recording
  them as known survivors preserves signal quality.

## Current status (2026-07-11)

| Service | Killed | Lived | Not Covered | Efficacy |
|---|---|---|---|---|
| customers | 19 | 1 (equivalent) | 21 (SQL) | 95.00% |
| quotes | 55 | 1 (equivalent) | 37 (SQL) | 98.21% |
| orders | 26 | 1 (equivalent) | 27 (SQL) | 96.30% |
| invoices | 26 | 1 (equivalent) | 25 (SQL) | 96.30% |
| reports | 1 | 0 | 21 (SQL) | 100.00% |

Every service clears the 70% target. The single-mutant survivors across
customers/quotes/orders/invoices are the same class of equivalent mutant:
`n >= 0` on the offset-parse guard mutated to `n > 0`. At `n = 0`, both
original and mutation route to the same outcome (`p.Offset` stays 0). No
kill possible without a distinguishable observation, which the API contract
does not offer.

## Alternatives considered

- **Line coverage alone.** Faster; misleading. Rejected — the point of tests
  is to catch regressions, not to hit a color in a report.
- **Property-based testing** via [`gopter`](https://github.com/leanovate/gopter).
  Higher signal for numeric code (the `quotes/calc.go` paving-math file
  would benefit). Could complement mutation testing later; not a replacement.
