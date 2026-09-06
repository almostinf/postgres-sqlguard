# Repository Agent Guide Design

## Goal

Add a concise root `AGENTS.md` that helps coding agents orient themselves,
select the authoritative documentation, respect architectural boundaries, and
run the correct repository workflow without duplicating the contributor guide.

## Audience and language

The guide is written in English for automated coding agents and contributors
working through an agent. It assumes familiarity with Go and Git, but not with
this repository.

## Organization

Use a responsibility-first structure:

1. summarize the project and its safety-sensitive purpose;
2. identify authoritative documents and what each one governs;
3. map repository directories and important root files;
4. describe the expected OpenSpec workflow;
5. state architectural, privacy, Go, and testing constraints;
6. list canonical Make targets;
7. define the completion gate for a change.

## Source-of-truth policy

`AGENTS.md` is a navigation and guardrail document, not a duplicate source of
truth. It links to:

- `README.md` for the public API and consumer-facing constraints;
- `CONTRIBUTOR.md` for the complete development and OpenSpec lifecycle;
- `docs/product-requirements.md` for product scope, safety, and privacy;
- `docs/adr/` for accepted engineering decisions;
- `openspec/specs/` for current behavioral contracts;
- `openspec/config.yaml` for artifact and operation rules;
- `Makefile` for executable quality gates.

Where these documents differ in scope, agents follow the most specific
authoritative document and update planning artifacts before intentionally
changing specified behavior.

## Repository map

The guide describes:

- the root `sqlguard` package as the public driver-independent API;
- `internal/parser` as the exclusive PostgreSQL parser-backend boundary;
- `internal/testutil` as the location reserved for genuinely shared test
  helpers;
- root and internal tests, fuzzing, and benchmarks;
- `docs/`, `openspec/`, and `.github/workflows/` responsibilities.

The map stays conceptual rather than listing every file, so it remains useful
as the repository grows.

## Mandatory guardrails

The guide highlights constraints that an agent must not infer only from code:

- repository changes follow the OpenSpec lifecycle; changes without observable
  behavior impact may use `skip_specs: true`, but still require the applicable
  proposal, tasks, and verification artifacts;
- core validator, rule, and statement contracts remain in the root package and
  independent of drivers and parser-generated types; integration-specific
  public APIs may live in dedicated packages;
- only `internal/parser` imports `pg_query_go`;
- enforce mode remains fail-closed and covers complete input and nested CTEs;
  future audit behavior follows its own accepted capability specification;
- public errors, logs, and future observability must not expose SQL, literals,
  arguments, credentials, or raw parser diagnostics;
- Engine state is immutable after construction, and rules own synchronization
  for concurrent calls;
- tests follow `docs/adr/0001-table-driven-test-style.md` where its lifecycle
  applies;
- exported Go declarations and packages follow the repository GoDoc policy.

## Commands and completion

The guide lists `make lint`, `make test`, `make test-race`, `make precommit`,
`make bench`, and `make fuzz`, including `FUZZ_TIME` customization. It identifies
`make precommit` as the required executable gate and points to
`CONTRIBUTOR.md` for detailed verification and archive rules.

A change is complete only when its relevant OpenSpec artifacts match the
implementation, required tests are present, `openspec-verify-change` reports no
critical issues, every warning is resolved or explicitly reviewed and
accepted, `openspec validate <change-name> --strict` succeeds, and
`make precommit` succeeds.

## Non-goals

- Repeat the full OpenSpec command reference from `CONTRIBUTOR.md`.
- Restate every product requirement or capability scenario.
- Document transient implementation details or enumerate every test case.
- Add new development policies beyond the repository's existing documents.
