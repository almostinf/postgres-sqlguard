## Context

See `proposal.md` for motivation. The repository currently contains product and contributor documentation but no Go module or executable quality gates. The product targets Go 1.26, expects public APIs to be documented, requires race-safe code, and allows CGO in the MVP because the future PostgreSQL parser may require it. The initial tooling must therefore work before runtime dependencies exist while remaining suitable for later CGO-backed packages.

## Goals / Non-Goals

**Goals:**

- Provide one reproducible command surface for local development and CI.
- Establish conservative package boundaries without committing future changes to premature public APIs.
- Pin the lint tool through the Go module and make its policy reviewable in the repository.
- Treat formatting and GoDoc rules as executable checks rather than prose-only conventions.
- Exercise the project with CGO and the race detector from its first implementation change.

**Non-Goals:**

- Implement SQL parsing, validation rules, driver adapters, configuration, or observability.
- Select runtime parser, pgx, YAML, Prometheus, or logging dependencies.
- Support `CGO_ENABLED=0`; that remains a later delivery stage.
- Add release, coverage publishing, benchmark, or multi-platform workflows.

## Decisions

### Use a minimal, non-speculative package skeleton

The module path will be `github.com/almostinf/postgres-sqlguard` with `go 1.26`. A root `doc.go` will establish package `sqlguard`, while `internal/parser/doc.go` and `internal/testutil/doc.go` reserve implementation and shared-test boundaries that are already implied by the product requirements. Each package will have a package comment and no placeholder runtime API.

Public feature packages such as built-in rules, pgx integration, configuration, and observability will be introduced by their own behavior changes. This is preferred over creating empty public packages now because package paths become part of the compatibility surface. A root-only skeleton was also considered, but it would not establish the internal separation needed by the first parser-backed implementation.

### Pin golangci-lint as a Go module tool

The Go 1.26 `tool` mechanism will pin the golangci-lint v2 command in `go.mod`, and Make will invoke it with `go tool golangci-lint`. This avoids reliance on whichever binary happens to be installed globally and keeps local and CI resolution identical. A downloaded binary under `.tools/` and a floating global installation were considered; both require a second version-management mechanism and create more drift.

### Make is the canonical command interface

The Makefile will expose at least:

- `lint`: run golangci-lint for all packages with CGO enabled;
- `test`: run the ordinary Go test suite with CGO enabled;
- `test-race`: run all tests with `-race` and CGO enabled;
- `precommit`: run module tidiness verification, lint, ordinary tests, and race tests.

Commands will be overridable through conventional variables such as `GO`, `CGO_ENABLED`, and `CC`, while safe defaults select Go, CGO enabled, and the platform C compiler. Targets will be phony and fail immediately when a constituent check fails. `go mod tidy -diff` will verify module metadata without rewriting it. Although race testing includes test execution, keeping `test` and `test-race` in `precommit` gives both modes first-class visibility and matches the repository's documented gate.

Direct shell scripts and separate CI-only commands were considered. They would duplicate orchestration and make local reproduction less reliable.

### Use an explicit golangci-lint v2 policy

The committed configuration will use the golangci-lint v2 schema and a deliberate allowlist rather than `enable-all`. It will include compiler/type analysis and common correctness checks, enforce `gofmt` plus import organization, and use `revive`'s exported-declaration rule for GoDoc. Package comments and exported declarations must follow Go conventions; exported declaration comments must begin with the declared name, and prose comments must end with punctuation where the selected linter can check that without noisy false positives.

Generated code and narrowly justified test-only patterns may receive path-specific exclusions. Broad exclusions, blanket `nolint`, and disabling documentation checks for production packages are not part of the baseline. Any `nolint` directive must name the linter and explain the exception.

Relying only on `go vet` was considered but does not enforce formatting, import organization, or GoDoc. Enabling every golangci-lint linter was rejected because upgrades would silently expand policy and create low-signal churn.

### Run one reproducible CGO-enabled CI gate

GitHub Actions will run on pull requests and pushes to the default branch. It will check out the repository, install Go from `go.mod`, configure module caching, ensure a C compiler is available, set `CGO_ENABLED=1`, and call `make precommit`. Workflow permissions will be read-only and action versions will be pinned to stable major versions.

A matrix that also runs with CGO disabled was rejected because no-CGO support is explicitly outside the current product stage. Calling a golangci-lint-specific action plus separate test commands was also rejected because it bypasses the canonical Make targets and can diverge from local checks.

### Keep contributor rules synchronized with enforcement

`CONTRIBUTOR.md` will describe prerequisites, each Make target, formatting and import expectations, GoDoc conventions, and the requirement to run `make precommit` before verification. `.gitignore` will cover build, coverage, editor, and local tool artifacts introduced by the workflow. Documentation will describe the enforced rule, while configuration remains the executable source of truth.

## Risks / Trade-offs

- [Go 1.26 or golangci-lint v2 tool support is unavailable on a contributor machine] → State the minimum versions clearly and let setup fail with actionable Go toolchain diagnostics.
- [CGO tests fail because a C compiler is absent] → Document the compiler prerequisite and install or verify it explicitly in CI.
- [A strict lint baseline slows early development] → Start with a reviewed allowlist and require narrow, explained exceptions instead of weakening the project-wide policy.
- [Placeholder internal packages later prove unnecessary] → Keep them internal and API-free so they can be removed without compatibility impact.
- [Tool dependencies enlarge `go.mod` and `go.sum`] → Accept the metadata cost in exchange for reproducible versions; runtime consumers do not link tool dependencies into the library.
- [Local and CI behavior diverge across operating systems] → Route both through Make, keep commands portable, and limit this change's CI promise to the Linux CGO environment.

## Migration Plan

1. Add the module and package skeleton.
2. Add lint policy and canonical Make targets, then run each target locally.
3. Update contributor documentation and ignore rules.
4. Add the CI workflow that invokes the established precommit target.

Before this bootstrap is merged, rollback is a normal revert of the new tooling files; no data or public runtime API migration is required.
