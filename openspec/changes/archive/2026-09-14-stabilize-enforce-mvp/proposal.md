## Why

The enforce-mode MVP now has the safety, parser-backend, rule, and observability
foundations needed for a stable release, but its public package layout, release
documentation, evidence, and CI presentation are not yet ready for a clear and
credible `v1.0.0`. This change establishes the final public surface and release
assets before the first compatibility-bearing tag is published.

## What Changes

- **BREAKING**: Move the public `rules` package to `pkg/rules` and the official
  observability integrations to `pkg/observability/...`; remove the old public
  import paths without compatibility wrappers.
- Rewrite the root README as a concise, benefit-led introduction with a focused
  quick start, clear backend choice, prominent links to detailed package and
  topic documentation, and explicit safety and privacy boundaries.
- Add reproducible benchmark evidence and checked-in README graphics comparing
  CGO and no-CGO builds across representative SQL complexity levels for CPU
  time, memory allocations, and linked binary size.
- Evaluate Apache-2.0 versus MIT for the stable release, record the decision and
  trade-offs, and bring license, notice, attribution, and source headers into
  compliance with the selected license without making unsupported legal claims.
- Add README badges for test coverage and the independently visible CI checks.
- Split the combined CI precommit jobs into separately reported lint and test
  jobs for CGO and no-CGO coverage while preserving the repository's canonical
  local quality gates.
- Add a release image of a sword-bearing Go gopher protecting the PostgreSQL
  elephant, with repository-owned source/output assets and documented
  attribution or generation provenance.
- Publish the first stable `v1.0.0` tag only after the OpenSpec verification,
  strict validation, license review, and full precommit gates succeed.
- Non-goals: changing enforce-mode validation decisions, weakening privacy or
  fail-closed guarantees, adding audit mode, or promising compatibility with
  PostgreSQL grammar versions beyond the currently documented support.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `builtin-mutation-rules`: Change the required public import path for the
  built-in rule constructors from `rules` to `pkg/rules` while preserving their
  opt-in policies, identifiers, and evaluation behavior.
- `observability`: Change the required public import paths for the official
  Prometheus and `log/slog` integrations to `pkg/observability/...` while
  preserving the root engine contracts, bounded event model, privacy boundary,
  and runtime failure isolation.

## Impact

- Public API consumers must update imports from
  `github.com/almostinf/postgres-sqlguard/rules` and
  `github.com/almostinf/postgres-sqlguard/observability/...` to their new
  `pkg/...` paths before adopting `v1.0.0`.
- Source, tests, examples, package documentation, README snippets, capability
  specs, and Go package discovery will be updated for the new layout.
- README content and supporting documentation will be reorganized; benchmark
  collection and chart-generation assets will be added or refined so published
  claims remain reproducible.
- GitHub Actions, coverage reporting, and README badges will expose distinct
  quality signals without reducing the checks enforced by `make precommit`.
- License and third-party attribution files may change according to the
  recorded license decision; dependency licenses remain independently binding.
- The release workflow gains an explicit final gate for creating `v1.0.0`; no
  tag is created until implementation and verification are complete.
