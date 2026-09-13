## Why

The library currently depends on the CGO-backed `pg_query_go` parser, which
prevents consumers from building and deploying SQLGuard with `CGO_ENABLED=0`.
This change evaluates and, if a backend can meet the existing safety contract,
adds a no-CGO parsing path without changing the public validation behavior seen
by callers or rules.

## What Changes

- Compare viable no-CGO PostgreSQL parser approaches against the current
  backend for supported PostgreSQL grammar, parser-neutral AST fidelity, rule
  semantics, latency and allocations, binary size, dependency footprint, and
  operational constraints.
- Add a supported no-CGO parser backend selected at build time when
  `CGO_ENABLED=0`, while retaining the current PostgreSQL parser backend for
  CGO-enabled builds.
- Require both build modes to preserve fail-closed complete-input parsing,
  deterministic statement and nested-CTE traversal, prepared-validation
  behavior, typed privacy-safe failures, and the rule-visible statement
  structure needed by built-in and custom rules.
- Add a shared conformance corpus and backend-specific benchmarks so grammar,
  AST, rule-result, performance, and size differences are measured and
  documented rather than assumed.
- Document build selection, supported PostgreSQL grammar compatibility,
  deployment tradeoffs, and a minimal implementation example of the selected
  backend boundary.
- Add CI coverage that builds and tests the supported public behavior with both
  `CGO_ENABLED=1` and `CGO_ENABLED=0`.
- Revise the product roadmap so no-CGO support may be delivered independently
  of audit mode while keeping both capabilities' safety requirements intact.
- Preserve the existing public API; callers do not select a parser backend at
  runtime and rules do not depend on backend-specific AST types.

## Non-Goals

- Add runtime parser selection or expose parser implementation details through
  the public API.
- Replace structural PostgreSQL parsing with regular expressions, keyword
  matching, partial parsing, or another fail-open fallback.
- Promise support for PostgreSQL syntax that neither documented backend accepts,
  or silently accept syntax that cannot be represented with equivalent rule
  semantics.
- Introduce a validation cache or change rule registration, observability, or
  database-driver integration contracts.
- Implement, defer, or weaken audit mode; this change only removes the ordering
  dependency between the separately specified product stages.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `validation-engine`: Extend the validation contract to support build-time
  parser backend selection, no-CGO builds, cross-backend grammar and traversal
  conformance, and backend-specific performance and footprint measurements
  while preserving existing public outcomes.
- `rule-extension`: Require the parser-neutral statement representation and
  rule evaluation results to remain semantically equivalent across supported
  parser backends.

## Impact

This change affects `internal/parser`, parser dependencies and build
constraints, the Makefile and CI build matrix, parser and engine conformance
tests, benchmarks, README consumer requirements, and product documentation. It
may add a pure-Go parser dependency and backend-specific conversion code, with
measurable effects on build time, binary size, grammar-version coverage, and
runtime cost. It also changes roadmap sequencing so audit mode and no-CGO
support may be delivered independently. The public `Validator`, `Engine`,
`Prepared`, `Statement`, rule, and error APIs remain source compatible, and
parser-generated types remain confined to the internal parser boundary.
