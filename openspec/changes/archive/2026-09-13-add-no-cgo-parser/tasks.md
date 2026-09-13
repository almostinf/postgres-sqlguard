## 1. Roadmap and Dependency Gate

- [x] 1.1 Update `docs/product-requirements.md` so audit mode and no-CGO support may be delivered independently while both remain required stages with unchanged safety and privacy guarantees.
- [x] 1.2 Review and pin an immutable `github.com/wasilibs/go-pgquery` revision whose `pg_query_go/v6` protobuf dependency matches the repository's PostgreSQL 17 parser version.
- [x] 1.3 Review the selected dependency graph and add all required MIT, BSD-3-Clause, PostgreSQL, Apache-2.0, and transitive license notices without weakening the existing dependency boundary.

## 2. Build-Selected Parser Backends

- [x] 2.1 Refactor `internal/parser` so parsing, privacy-safe error mapping, protobuf conversion, and traversal stay shared while a narrow raw-parse function becomes the only backend-specific seam.
- [x] 2.2 Add the `//go:build cgo` backend that calls the existing `pg_query_go/v6` parser and prove the current CGO parser tests remain unchanged in behavior.
- [x] 2.3 Add the `//go:build !cgo` backend that calls the pinned `wasilibs/go-pgquery` WASM parser and returns the same protobuf result type to the shared converter.
- [x] 2.4 Add a shared public-API compile contract and build-matrix checks proving `CGO_ENABLED=0` needs neither a C compiler nor a custom build tag and `CGO_ENABLED=1` remains source compatible, without empty build-tag marker tests.
- [x] 2.5 Extend dependency-boundary checks so both parser entry-point imports remain confined to `internal/parser` and the no-CGO package graph contains no selected native source, without depending on upstream source filenames.

## 3. Cross-Backend Conformance

- [x] 3.1 Add a compact checked-in, non-secret PostgreSQL 17 conformance corpus covering PostgreSQL-specific syntax, comments, quoted forms, parameters, expressions, DML, DDL, empty input, batches, nested data-modifying CTEs, and representative malformed input without duplicating equivalent empty or malformed placements.
- [x] 3.2 Add a canonical parser-neutral snapshot helper and golden expectations covering every rule-visible node kind, field name, value kind, semantic value, and list order represented by the corpus.
- [x] 3.3 Run the same table-driven parser corpus against the automatically selected backend in CGO and no-CGO builds, requiring identical parse classification and canonical snapshots.
- [x] 3.4 Run the existing engine/rule/traversal suite in both build modes and add only a PostgreSQL 17 end-to-end case, covering allowed input, typed failures, built-in and custom rules, first-violation selection, ordering, and nested CTE traversal without duplicating the existing assertions.
- [x] 3.5 Run the existing direct/prepared suite in both modes, preserving the single `Engine.Prepare` parser call site and repeated validation, context precedence, and immutable prepared-state behavior without a source-scanning contract test.
- [x] 3.6 Reuse existing public error and observability sentinel tests in both modes and add only the raw-WASM-boundary case proving backend diagnostics expose neither SQL nor raw parser details.
- [x] 3.7 Run concurrency, race, and fuzz coverage in both modes and resolve any WASM-specific initialization, shared-runtime, malformed-input, or panic/trap behavior without adding a fail-open fallback.

## 4. Repository Build Matrix

- [x] 4.1 Add explicit Make targets for CGO and no-CGO lint, unit-test, race-test, benchmark, and supported aggregate workflows while preserving `CGO_ENABLED=1` as the default.
- [x] 4.2 Extend CI with CGO and no-CGO jobs that exercise the same public packages, set the no-CGO job's compiler setting to an unusable value, and verify no native parser compilation is attempted.
- [x] 4.3 Ensure module-tidiness and lint configuration accept the build-tag split and allow parser dependencies only at the documented internal boundary.

## 5. Performance and Footprint Evidence

- [x] 5.1 Add parser-focused fresh-process first-parse, warm steady-state, and parallel benchmarks while keeping the existing engine benchmark names, inputs, and outcome preflights comparable across build modes.
- [x] 5.2 Add a representative size-probe command under test tooling that exercises validation and can be built reproducibly for both modes with identical target and linker settings.
- [x] 5.3 Run and record CGO and no-CGO latency, allocation, cold-start, linked-binary-size, and module-footprint measurements with the exact Go version, GOOS/GOARCH, compiler, dependency revisions, date, inputs, and commands.

## 6. Consumer and Maintainer Documentation

- [x] 6.1 Create `docs/parser-backends.md` with the reviewed candidate comparison, grammar and AST compatibility rationale, reproduced measurements, license/dependency impact, supported platform matrix, maintenance constraints, and repeatable benchmark and size commands.
- [x] 6.2 Update README consumer requirements to document automatic `CGO_ENABLED` selection, PostgreSQL grammar compatibility, the faster CGO default, no-CGO WASM startup/footprint tradeoffs, and the TinyGo limitation.
