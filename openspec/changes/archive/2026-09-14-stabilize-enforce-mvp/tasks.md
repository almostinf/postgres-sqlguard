## 1. Stabilize Public Package Layout

- [x] 1.1 Extend the capability contracts to require `pkg/rules`, `pkg/observability/prometheus`, and `pkg/observability/slog`, and to reject every former `/rules` and `/observability/...` package path.
- [x] 1.2 Move the built-in rules and their unit, structure, extension, and concurrency tests to `pkg/rules` without changing constructors, rule identifiers, evaluation order, or policy behavior.
- [x] 1.3 Move the Prometheus and `log/slog` integrations and their tests to `pkg/observability/...` without changing their APIs, event dimensions, privacy behavior, or failure isolation.
- [x] 1.4 Update all repository imports, examples, GoDoc references, and documentation to the new paths, then confirm module package enumeration contains the new paths and none of the old paths.

## 2. Restructure User Documentation

- [x] 2.1 Extract the README's detailed engine, prepared-validation, built-in-rule, observability, pgx-example, privacy, and development guidance into focused files under `docs/` and the applicable package GoDoc, preserving all safety boundaries and limitations.
- [x] 2.2 Add a tested or compile-checked quick-start example that demonstrates installation, construction with `EngineOptions{}`, opt-in `pkg/rules`, validation, and typed violation handling.
- [x] 2.3 Rewrite `README.md` around the project name, one-sentence value proposition, concise differentiators, installation, quick start, built-in policy summary, explicit limitations, backend choice, benchmark summary, and links to detailed documentation; stage 6 adds the release illustration to this structure.
- [x] 2.4 Audit every README claim and link against current specs, tests, supported Go/PostgreSQL versions, parser-backend behavior, and the new public import paths.

## 3. Produce Reproducible Benchmark Graphics

- [x] 3.1 Define a dated machine-readable benchmark dataset format that records host/toolchain metadata, identical CGO and no-CGO commands, raw result locations, run count, medians, units, SQL-complexity cases, cold RSS, and stripped binary size.
- [x] 3.2 Implement a repository-local deterministic chart generator using no production dependency, with tests for dataset validation, stable ordering, escaping, units, and byte-identical SVG output.
- [x] 3.3 Add Make targets that capture or normalize release benchmark evidence and regenerate the checked-in charts without replacing the existing `bench-*` and `size-probe` interfaces.
- [x] 3.4 Capture at least five comparable CGO and no-CGO runs for direct validation latency and `B/op` across simple, medium, multi-statement, and nested-CTE inputs, retaining the raw outputs and reported medians.
- [x] 3.5 Capture equivalent stripped binary sizes and fixed-workload cold-process maximum RSS for both backends, recording platform-specific measurement commands and limitations.
- [x] 3.6 Generate and commit accessible SVG comparisons for latency, allocated bytes, cold RSS, and binary size, and add a clean-regeneration check that detects drift from the recorded dataset.
- [x] 3.7 Update `docs/parser-backends.md` and the concise README benchmark section with the release dataset, methodology link, units, host/date disclosure, and explicit non-portability and no-database-round-trip caveats.

## 4. Separate CI and Publish Coverage

- [x] 4.1 Add or refine Make coverage targets so the CGO test path emits one deterministic atomic coverage profile while ordinary CGO, no-CGO, and race targets retain their existing semantics.
- [x] 4.2 Replace the compound CI workflow with `lint.yml`, containing independently named module-tidiness, CGO lint, and no-CGO lint jobs that call canonical Make targets.
- [x] 4.3 Add `test.yml` with independently named CGO tests, no-CGO tests, and CGO race jobs, preserving the no-compiler sentinel, timeouts, cancellation, caching, and least-privilege defaults.
- [x] 4.4 Upload the default-branch CGO coverage profile to Codecov through a reviewed immutable action revision and job-scoped OIDC permission, while keeping fork pull-request tests useful when trusted upload identity is unavailable.
- [x] 4.5 Add Codecov configuration and README badges for `Lint`, `Tests`, coverage, Go Reference, and Apache-2.0, using live targets rather than hard-coded status or coverage values.
- [x] 4.6 Document the required-check name migration so repository branch protection is changed only after the replacement workflows have succeeded on `main`.

## 5. Complete License and Attribution Review

- [x] 5.1 Record the decision to retain Apache-2.0 in user-facing project documentation, including its permissive terms, explicit patent grant rationale, and a non-legal-advice qualification where appropriate.
- [x] 5.2 Audit the canonical `LICENSE` text and the final production dependency and embedded-artifact closure against `THIRD_PARTY_NOTICES.md`, correcting missing or stale license and NOTICE material without adding blanket Go source headers.
- [x] 5.3 Determine whether any project-level `NOTICE` or file-specific attribution is required by incorporated content, add only the required material, and document the reviewed conclusion for the release.

## 6. Create Compliant Release Artwork

- [x] 6.1 Generate a compact release illustration of a sword-bearing Go gopher guarding an original blue elephant/database character, avoiding reproduction, tracing, modification, or combination of the official Slonik mark.
- [x] 6.2 Review the illustration at README display size for recognizability, accessibility, unintended text or logos, false endorsement cues, and compliance with the Go and PostgreSQL identity constraints; regenerate or edit it until the review passes.
- [x] 6.3 Commit the optimized raster asset and a provenance record containing its prompt, generation date/tool, edits, Renée French/CC BY 4.0 gopher attribution, PostgreSQL trademark/no-affiliation note, and README alt text.

## 7. Prepare the Stable Release Handoff

- [x] 7.1 Add `v1.0.0` release notes covering the stable import paths, enforce-mode guarantees, supported Go and PostgreSQL grammar versions, CGO/no-CGO trade-offs, migration mapping, and explicit non-goals.
- [x] 7.2 Add a release runbook that identifies the exact post-verification sequence: archive the OpenSpec change, merge to `main`, confirm replacement workflows on the release commit, rerun the documented release gates, create an annotated `v1.0.0` tag, and push only that tag.
- [x] 7.3 Document the immutable-tag recovery policy: discard only an unpushed local candidate tag, never move or delete published `v1.0.0`, and publish corrections as a higher semantic version.
- [x] 7.4 Review the final repository package list, documentation navigation, generated assets, badge targets, release notes, and runbook as one consumer-facing `v1.0.0` candidate, resolving inconsistencies before handing the change to verification.
