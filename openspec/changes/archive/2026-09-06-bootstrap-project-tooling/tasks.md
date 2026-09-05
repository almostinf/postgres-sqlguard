## 1. Go Module and Package Skeleton

- [x] 1.1 Initialize `github.com/almostinf/postgres-sqlguard` with Go 1.26 and pin a Go-1.26-compatible golangci-lint v2 command through the module tool mechanism.
- [x] 1.2 Add documented, API-free `doc.go` skeletons for the root `sqlguard`, `internal/parser`, and `internal/testutil` packages.
- [x] 1.3 Add repository ignore rules for Go build outputs, coverage files, local tools, and common editor artifacts without masking source or module files.

## 2. Local Quality Gates

- [x] 2.1 Add a golangci-lint v2 configuration with an explicit correctness allowlist, `gofmt` and import-format enforcement, GoDoc checks for packages and exported declarations, and strict validation of explained `nolint` directives.
- [x] 2.2 Add phony Make targets `lint`, `test`, and `test-race` that cover `./...`, default to `CGO_ENABLED=1`, accept overridable Go/compiler variables, and propagate failures.
- [x] 2.3 Add a `precommit` target that verifies `go mod tidy -diff` and runs lint, ordinary tests, and race tests through the canonical targets.
- [x] 2.4 Exercise each Make target against the package skeleton and adjust only narrowly scoped lint settings until all local gates pass.

## 3. Contributor Documentation

- [x] 3.1 Document Go 1.26, Make, CGO, and C compiler prerequisites plus the purpose and usage of every canonical Make target in `CONTRIBUTOR.md`.
- [x] 3.2 Document formatting, import grouping, package comments, exported-declaration GoDoc, and named/explained `nolint` exception rules in `CONTRIBUTOR.md`.

## 4. Continuous Integration

- [x] 4.1 Add a least-privilege GitHub Actions workflow for pull requests and default-branch pushes that reads the Go version from `go.mod`, enables Go module caching, and provides a C compiler.
- [x] 4.2 Set `CGO_ENABLED=1` explicitly in CI and run `make precommit` so automation uses the same pinned tools and quality gates as local development.
