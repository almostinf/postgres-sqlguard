# Contributing to postgres-sqlguard

This project uses OpenSpec to define, implement, verify, and archive changes.
Every behavior change must be described by requirements before implementation
starts. The implementation is considered complete only after it has been
verified against those requirements and has passed the project quality gates.

## Command conventions

Commands beginning with `$openspec-` are workflow commands entered in the
Codex chat. Commands in shell blocks are run in a terminal.

Use kebab-case names for changes, for example:

```text
add-core-enforce-validator
add-pgx-wrapper
add-audit-mode
```

## Development prerequisites

Local development requires:

- Go 1.26 or newer;
- GNU Make;
- CGO support and a working C compiler such as GCC or Clang.

The module pins developer tools through Go's `tool` directive. Go downloads
the required toolchain and the pinned golangci-lint version automatically when
the local Go installation permits toolchain downloads.

The Makefile is the canonical interface for repository checks:

```bash
make lint       # Run static analysis, formatting, import, and GoDoc checks.
make test       # Run all tests with CGO enabled.
make test-race  # Run all tests with CGO and the race detector enabled.
make precommit  # Verify module metadata and run every project quality gate.
```

`GO`, `CGO_ENABLED`, and `CC` can be overridden when needed. The repository
defaults to `go`, `CGO_ENABLED=1`, and `cc` respectively.

### Go formatting and documentation

All Go source must be formatted with `gofmt` and imports must follow
`goimports`, with standard-library, third-party, and
`github.com/almostinf/postgres-sqlguard` imports grouped consistently. The
golangci-lint configuration enforces both rules.

Every package must have a package comment beginning with `Package <name>`.
Every exported declaration must have a complete GoDoc sentence beginning with
the declaration's name. Declaration comments must start with a capital letter
and end with punctuation.

Suppress a linter only for a narrow, reviewed exception. Every directive must
name the affected linter and include an explanation, for example:

```go
//nolint:gosec // The test intentionally uses a fixed local credential.
```

Blanket `//nolint` directives are not accepted. Run `make precommit` before
requesting verification of an OpenSpec change.

## Development lifecycle

The normal lifecycle is:

```text
explore -> new -> plan -> validate -> apply -> verify -> archive
```

### 1. Explore the change

Exploration is optional, but recommended for security-sensitive,
cross-cutting, or insufficiently defined work:

```text
$openspec-explore
```

Exploration does not create implementation artifacts. Its purpose is to
clarify scope, constraints, alternatives, and open questions.

### 2. Create the change

Create an empty change scaffold:

```text
$openspec-new-change <change-name>
```

This command selects the configured schema and shows the first artifact that
can be created. It does not create the proposal itself.

### 3. Create and review planning artifacts

Create the next available artifact:

```text
$openspec-continue-change <change-name>
```

For the `spec-driven` schema, repeated calls create the following artifacts:

```text
proposal.md
    -> specs/<capability>/spec.md
    -> design.md
    -> tasks.md
```

Review each artifact before continuing:

- `proposal.md` explains why the change is needed, its scope, non-goals, and
  affected capabilities.
- Delta specs describe observable behavior, requirements, and testable
  scenarios.
- `design.md` records implementation decisions, alternatives, risks, and
  limitations. It may be skipped only when the schema instructions consider it
  unnecessary.
- `tasks.md` contains ordered and verifiable implementation tasks.

Requirements belong in specs. Implementation choices belong in the design.
Do not weaken or silently defer a requirement in order to simplify the task
list.

### 4. Validate the plan

Before implementation, validate the change strictly:

```bash
openspec validate <change-name> --strict
```

Resolve all validation errors before starting implementation. This command
validates OpenSpec artifacts; it does not prove that code implements them.

### 5. Apply the change

Implement the planned tasks:

```text
$openspec-apply-change <change-name>
```

The workflow reads all planning artifacts, implements pending tasks in order,
and marks a task complete only after its specified behavior is implemented.

If implementation reveals that the proposal, requirements, design, or task
breakdown is wrong, update the artifacts first:

```text
$openspec-update-change <change-name>
```

Then resume implementation with `$openspec-apply-change`. The implementation
must not become the undocumented source of truth.

### 6. Verify the result

Verification is a separate post-implementation gate, not another
implementation task.

First, verify completeness, correctness, and coherence against the OpenSpec
artifacts:

```text
$openspec-verify-change <change-name>
```

The verification must establish that:

- every task is complete;
- every requirement is implemented;
- every scenario has implementation and test coverage;
- the implementation follows the accepted design;
- any divergence is reflected in updated OpenSpec artifacts.

Next, rerun strict artifact validation and the executable project checks:

```bash
openspec validate <change-name> --strict
make precommit
```

Verification passes only when:

- strict OpenSpec validation succeeds;
- `openspec-verify-change` reports no critical issues;
- all warnings have been resolved or explicitly reviewed and accepted;
- `make lint` succeeds;
- `go test -race ./...` succeeds.

Do not archive a change with failed or skipped verification checks.

### 7. Archive the change

After successful verification, archive the change:

```text
$openspec-archive-change <change-name>
```

The archive workflow checks artifact and task completion, synchronizes delta
specs into the main specs, verifies the synchronization, and moves the change
under `openspec/changes/archive/`.

Archiving warnings are not permission to bypass the project verification gate.

## Fast path

When a small change is already well-defined, all planning artifacts can be created in one workflow:

```text
$openspec-ff-change <change-name>
```

The remaining lifecycle is unchanged:

```text
$openspec-apply-change <change-name>
$openspec-verify-change <change-name>
$openspec-archive-change <change-name>
```

Run the terminal validation, lint, and race-test commands during the verify
stage. Prefer the guided `new` plus `continue` flow for safety-critical,
architectural, or externally visible changes.

## Changes without behavior impact

Pure documentation, tooling, or internal refactoring changes may set
`skip_specs: true` in the change's `.openspec.yaml`. Use this only when no
observable requirement changes. Such changes still require a proposal, an
appropriate task list, and the applicable verification checks.

## Useful inspection commands

List active changes:

```bash
openspec list
```

Inspect artifact status:

```bash
openspec status --change <change-name>
```

Validate all active changes and main specs:

```bash
openspec validate --all --strict
```

Inspect a change:

```bash
openspec show <change-name>
```

## Recommended command sequences

Guided workflow:

```text
$openspec-explore
$openspec-new-change <change-name>
$openspec-continue-change <change-name>  # repeat until planning is complete
openspec validate <change-name> --strict
$openspec-apply-change <change-name>
$openspec-verify-change <change-name>
openspec validate <change-name> --strict
make precommit
$openspec-archive-change <change-name>
```

Fast workflow:

```text
$openspec-ff-change <change-name>
openspec validate <change-name> --strict
$openspec-apply-change <change-name>
$openspec-verify-change <change-name>
openspec validate <change-name> --strict
make lint
go test -race ./...
$openspec-archive-change <change-name>
```
