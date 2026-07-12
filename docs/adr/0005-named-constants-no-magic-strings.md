# ADR 0005 — Every String and Number Lives in a Named Constant

**Status:** Accepted
**Date:** 2026-07-11 (fixed after initial scaffold pass violated the rule)

## Context

The first scaffold pass shipped with string literals scattered across
handlers, store files, and cross-service clients: env var names like
`"DATABASE_URL"`, HTTP header names like `"Content-Type"`, MIME types, log
levels, error messages, cross-service status literals like `"accepted"` and
`"fulfilled"`, and SQL identifiers like `"orders.orders"`.

Every duplicated literal becomes a silent bug waiting for a production
incident. A rename in the producing service silently breaks the consuming
service. A typo in an error message ships to users without a compiler check.

## Decision

Every string literal and numeric constant with meaning beyond its local
expression must live in a named constant. Each service defines its constants
in `services/<name>/constants.go`. Test files reference the same constants
that production code uses.

Grouped by category:

- **Env var names** — `EnvDatabaseURL`, `EnvServicePort`, `EnvLogLevel`,
  `EnvQuotesServiceURL`, `EnvOrdersServiceURL`
- **Runtime defaults** — `DefaultPort`, `ReadHeaderTimeout`,
  `ShutdownTimeout`, `DefaultPageLimit`, `MaxPageLimit`, `HTTPClientTimeout`,
  `DefaultDueDays`
- **HTTP headers + MIME types** — `HeaderContentType`, `ContentTypeJSON`,
  `HeaderAccessControlAllowOrigin`, `CORSAllowedMethods`, etc.
- **Health / response tokens** — `StatusOK`, `StatusKey`, `ErrorKey`,
  `MetricRequest`
- **Query param names** — `QueryParamLimit`, `QueryParamOffset`,
  `QueryParamStatus`, `QueryParamCustomerID`
- **SQL identifiers** — `SchemaQuotesTable`, `OrderNumberSequence`,
  `InvoiceNumberSequence`, etc.
- **Cross-service status literals** — `ExternalQuoteStatusAccepted`,
  `ExternalOrderStatusFulfilled`
- **Business-domain identifiers** — `OrderNumberPrefix = "SO"`,
  `InvoiceNumberPrefix = "INV"`, `OrderNumberPattern`
- **Report stage + aging bucket labels** — `StageQuotesCreated`,
  `Bucket1To30Days`, etc. (API contract for the frontend charts)
- **User-facing error messages** — `ErrMsgInvalidJSON`,
  `ErrMsgCustomerIDRequired`, `ErrMsgInvalidTransition`, etc.

## Consequences

**Positive**

- Cross-service rename safety. A producer changing a status literal surfaces
  as a compile error in the consumer's `constants.go`, not a silent 500 in
  production.
- Grep-driven refactors. Renaming `OrderNumberPrefix` from `"SO"` to
  `"ORD"` touches one line.
- Test/production parity. Tests assert against the same `ErrMsg*` constants
  that the handlers return — a typo in an error message becomes a compile
  error in the test, not a silent divergence.

**Negative**

- One extra file per service. Amortizes across every subsequent change.

**Neutral**

- The rule extends to test files. Tests reference `HeaderContentType`, not
  the raw `"Content-Type"` string.

## Alternatives considered

- **String literals inline with a comment** — human-eye-only convention.
  Rejected; the compiler must enforce the constant, not a comment reviewer.
- **Constants in a shared package across services.** Rejected because it
  reintroduces the compile-time coupling that ADR 0001 explicitly avoids.
  Each service's constants stay scoped to its own bounded context.
- **Config files (YAML) for the "config-like" constants.** Rejected because
  YAML config lives outside the compiler; a typo in a YAML key surfaces at
  runtime, not build time.
