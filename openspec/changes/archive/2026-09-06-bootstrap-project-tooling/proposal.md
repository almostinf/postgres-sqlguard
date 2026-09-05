## Why

The repository defines Go 1.26 and precommit quality expectations but does not yet contain a Go module, package layout, repeatable developer commands, lint policy, or CI enforcement. Establishing that baseline now keeps subsequent safety-sensitive features consistent across local development and automation.

## What Changes

- Initialize the project as a Go 1.26 module and add a minimal package structure that future validator, rule, configuration, integration, and observability changes can extend without premature implementation.
- Add a Makefile with canonical `lint`, `test`, `test-race`, and `precommit` targets.
- Add a versioned `golangci-lint` configuration that enforces Go formatting, import ordering, static analysis, and documentation for exported declarations.
- Add GitHub Actions CI that runs the same lint and test gates with CGO enabled.
- Document the formatting, GoDoc, tooling, and contributor command conventions.
- Keep generated files, build outputs, and local tool artifacts out of version control.

## Capabilities

### New Capabilities

None. This change establishes repository tooling and does not add observable library behavior.

### Modified Capabilities

None. The change opts out of delta specs with `skip_specs: true`.

## Impact

The change affects the repository root, initial package directories, contributor documentation, and GitHub Actions configuration. It introduces the Go module identity and a pinned golangci-lint toolchain, requires a Go 1.26 environment, and makes CGO-enabled lint and race tests the shared local and CI acceptance gate. No public runtime API or SQL-validation behavior is introduced.
