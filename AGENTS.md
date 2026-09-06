# Repository Agent Guide

## Project overview

`postgres-sqlguard` is a safety-sensitive Go library that validates PostgreSQL
SQL before execution. The public validation core is driver-independent, parses
SQL with a real PostgreSQL grammar, applies explicitly registered rules, and
must preserve fail-closed and privacy-safe behavior.

Keep this file concise. It is a navigation and guardrail document, not a
replacement for the repository's requirements, specifications, or contributor
guide. When guidance overlaps, follow the most specific authoritative document.

## Read first

- [README.md](README.md) describes the public API, consumer requirements, and
  current validation semantics.
- [CONTRIBUTOR.md](CONTRIBUTOR.md) defines the complete development workflow,
  OpenSpec lifecycle, formatting policy, and quality gates.
- [Product requirements](docs/product-requirements.md) define product scope,
  safety guarantees, privacy constraints, and delivery stages.
- [Architecture decisions](docs/adr/) record accepted engineering conventions;
  ADR 0001 governs table-driven Go tests.
- [Current capability specs](openspec/specs/) are the source of truth for
  observable behavior.
- [OpenSpec configuration](openspec/config.yaml) defines artifact and operation
  rules.
- [Makefile](Makefile) is the canonical interface for local checks.

## Repository map

- Root Go files form the public `sqlguard` package and contain the
  driver-independent validator, engine, rule, statement, and error contracts.
- `internal/parser/` is the PostgreSQL parser-backend boundary and the only
  package that may directly import `pg_query_go` or expose its generated types
  internally.
- `internal/testutil/` is reserved for genuinely shared, non-production test
  helpers.
- Root and `internal/` `*_test.go` files cover unit and concurrency behavior;
  dedicated `*_fuzz_test.go` and `*_benchmark_test.go` files cover fuzzing and
  benchmarks.
- `docs/` owns product requirements and architectural decisions.
- `openspec/specs/` contains current capability contracts;
  `openspec/changes/` contains active and archived change artifacts.
- `.github/workflows/` owns continuous-integration configuration.

Keep this map conceptual. Do not turn it into an inventory of every source
file as the repository grows.

## Working with changes

All repository changes follow the OpenSpec lifecycle documented in
[CONTRIBUTOR.md](CONTRIBUTOR.md). Read the relevant current capability specs
before changing behavior, and update the proposal, specs, design, or tasks
before implementing an intentional divergence.

Changes without observable behavior impact, such as pure documentation,
tooling, or internal refactoring, may set `skip_specs: true`. This skips delta
specs only; the applicable proposal, task, validation, apply, and verification
steps still remain.

Do not treat implementation code as an undocumented source of truth. Any
future audit-mode behavior requires its own accepted capability specification
and must not implicitly weaken enforce-mode guarantees.

## Architecture and safety boundaries

- Keep the core validator, rule, and statement contracts independent of
  database drivers and parser-generated types. Put integration-specific public
  APIs in dedicated packages; keep `internal/` packages non-public.
- Only `internal/parser` may depend directly on `pg_query_go`.
- In enforce mode, parse the complete input and fail closed on parser errors.
  Validate every top-level statement and every statement-bearing nested CTE.
- Never expose or retain SQL text, SQL arguments, literal values, comments,
  tokens, credentials, secrets, or raw parser diagnostics in public errors,
  logs, metrics, or future observability output.
- Preserve typed, privacy-safe failures that callers can inspect with standard
  Go error mechanisms instead of parsing messages.
- Treat a constructed `Engine` as immutable. A registered `Rule` may be called
  concurrently and therefore owns synchronization for any mutable state.
- Preserve the caller's context, including cancellation and request-scoped
  values. Do not move synchronous CGO parsing into an unbounded background
  goroutine.

## Go and test conventions

- Target Go 1.26 with CGO enabled and a working C compiler.
- Format Go source with `gofmt` and imports with `goimports` using the grouping
  enforced by the linter configuration.
- Give every exported package a `Package <name>` comment. Give every exported
  declaration a complete GoDoc sentence beginning with its name.
- Use narrow, explained `//nolint:<linter>` directives only; blanket
  suppressions are not accepted.
- Follow [ADR 0001](docs/adr/0001-table-driven-test-style.md) when cases share a
  table-driven lifecycle: use a `map[string]struct{...}`, `snake_case` case
  names, parallel independent subtests, helper-based setup/checks where useful,
  and `testify/require` assertions.
- Keep materially different test lifecycles separate. Do not repeat identical
  assertions to simulate determinism; test ordering and concurrency at the
  layer that owns those behaviors.

## Commands

Use the Make targets rather than reconstructing their underlying commands:

```bash
make lint
make test
make test-race
make precommit
make bench
make fuzz
```

`make precommit` is the required executable gate. Override fuzz duration with
`FUZZ_TIME`, for example `make fuzz FUZZ_TIME=1m`.

## Definition of done

A change is complete only when:

- implementation, tests, and relevant OpenSpec artifacts agree;
- required scenarios and safety boundaries have test coverage;
- `openspec-verify-change` reports no critical issues;
- every verification warning is resolved or explicitly reviewed and accepted;
- `openspec validate <change-name> --strict` succeeds; and
- `make precommit` succeeds.

Do not archive a change with failed or skipped verification checks.
