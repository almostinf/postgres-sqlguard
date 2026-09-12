## Why

Repeated validation of stable, parameterized SQL pays the PostgreSQL parsing
and AST-conversion cost on every call, which becomes material at high QPS.
Applications need an explicit way to reuse immutable parsed structure without
caching policy decisions or weakening per-call context, rule, observability,
and privacy guarantees.

## What Changes

- Add `Engine.Prepare` and `Engine.ValidatePrepared` around an opaque,
  immutable, concurrency-safe `Prepared` value while keeping the existing
  `Validator` interface compatible.
- Preserve fail-closed behavior by parsing complete input during preparation
  and evaluating every registered rule on every prepared validation call.
- Add a privacy-safe invalid-prepared error and bounded
  `invalid_prepared` observability outcome; preparation parser failures remain
  visible as `parser_failure`.
- Extend validation benchmarks to measure ordinary validation and direct
  prepared validation as distinct absolute-cost paths.
- Add a compiling, tested `example/pgx-prepared` package that demonstrates
  `Exec`, `Query`, and `QueryRow` validation with a bounded concurrent LRU
  cache of parsed representations backed by `github.com/hashicorp/golang-lru/v2`.
- Add cache hit and miss benchmarks plus documentation of cache limits,
  unsupported pgx server-prepared-statement names, and SQL/AST retention risks.
- Keep caching out of the core library and never cache allow/deny outcomes.

## Capabilities

### New Capabilities

- `pgx-prepared-example`: Defines the illustrative pgx wrapper that uses
  prepared validation with a bounded concurrent parsing cache, preserves the
  guarded execution boundary, and documents its privacy and integration
  limitations.

### Modified Capabilities

- `validation-engine`: Adds preparation and prepared-validation behavior,
  concurrency guarantees, per-call rule evaluation, and prepared-path
  benchmark coverage.
- `validation-errors`: Adds the inspectable, privacy-safe failure contract for
  an invalid prepared value.
- `observability`: Adds preparation failure emission and the bounded
  `invalid_prepared` outcome across custom and official integrations.

## Impact

- Public additions in the core `sqlguard` package: `Prepared`,
  `Engine.Prepare`, `Engine.ValidatePrepared`, `ErrInvalidPrepared`, and
  `ValidationOutcomeInvalidPrepared`.
- Internal engine/parser flow will share immutable parsed results between
  `Validate` and prepared validation without exposing parser-backend types.
- Official Prometheus and `log/slog` integrations gain the new bounded outcome;
  `invalid_prepared` is logged at ERROR level.
- A new `example/pgx-prepared` package, tests, benchmarks, README/package
  documentation, and product-requirement clarification are added. The existing
  `example/pgx` package and `Validator` interface remain compatible.
- `github.com/hashicorp/golang-lru/v2 v2.0.7`, already present indirectly in
  the module graph, becomes a direct runtime dependency of the prepared pgx
  example. The example retains responsibility for SQL-size limits, owned cache
  keys, and outcome-aware insertion around the library cache.
