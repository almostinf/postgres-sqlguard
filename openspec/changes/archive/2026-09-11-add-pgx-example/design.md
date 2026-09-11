## Context

See [proposal.md](proposal.md) for motivation and
[the pgx example delta spec](specs/pgx-example/spec.md) for observable
requirements.

The core already exposes the driver-independent `sqlguard.Validator` interface
and uses a synchronous, fail-closed `Validate(context.Context, string)` call.
The repository has no pgx dependency or integration package today. Its root
module and `make precommit` cover packages matched by `./...`. The new code
must remain an explicitly illustrative example while still being compiled and
tested by the normal repository workflow.

## Goals / Non-Goals

**Goals:**

- Demonstrate one small validation boundary that applications can adapt around
  pgx connection or pool execution.
- Make validation ordering, pass-through behavior, and the `QueryRow` deferred
  error case obvious in code and tests.
- Fail closed when a pgx argument could rewrite SQL after validation.
- Replace product-level promises of an official pgx adapter with an honest
  description of the example and its escape hatches.

**Non-Goals:**

- Provide a supported import path or semantic-version compatibility guarantee
  for the example package.
- Mirror the full pgx API or cover transactions, batches, prepared statements,
  `CopyFrom`, or access through an unwrapped connection.
- Support pgx `QueryRewriter`, including `NamedArgs`, `StrictNamedArgs`,
  `StructArgs`, or `StrictStructArgs`.
- Change the validator, engine, rules, errors, or observability contracts.

## Decisions

### 1. Keep the example in the root module

Create `example/pgx` as package `pgxexample` in the existing module and add a
pinned `github.com/jackc/pgx/v5` dependency. The package name distinguishes the
example wrapper from the upstream `pgx` package when both are imported.

Only the example files import pgx packages. The root `sqlguard` package and
its public contracts retain a driver-independent import graph. Keeping one
module also means `go test ./...`, the race detector, the linter, and dependency
tidiness checks cover the example without a second orchestration layer.

**Alternatives considered:**

- A nested module under `example/pgx` would keep pgx out of the root `go.mod`,
  but `./...` commands from the repository root would skip it and require
  duplicate CI and Make targets.
- A production package such as `integrations/pgx` would imply a support and
  compatibility commitment that this change explicitly removes.

### 2. Wrap a narrow pgx-compatible executor interface

The example defines an `Executor` interface containing the pgx v5 signatures
for `Exec`, `Query`, and `QueryRow`. A `GuardedDB` holds a
`sqlguard.Validator` and an `Executor`; `NewGuardedDB` rejects nil, including
typed-nil, collaborators so a bad example configuration fails at construction
rather than on the first database call.

The narrow interface permits real `*pgx.Conn` and `*pgxpool.Pool` values as
well as deterministic test fakes. Compile-time interface checks for the
connection and pool make upstream signature drift visible during the
repository build.

The example README or package documentation shows construction with a real
pool and warns that callers can bypass the guard by retaining or using the raw
pool directly.

**Alternatives considered:**

- Helper functions such as `ValidateAndExec` require callers to remember the
  validation step at every call site and demonstrate a weaker boundary.
- Mirroring concrete connection and pool types would duplicate a broad pgx API
  and obscure which methods are actually guarded.
- A pgx tracing hook cannot serve as this fail-closed boundary because tracing
  is observational rather than an error-returning replacement for all three
  operations.

### 3. Reject query rewriting, then validate once and delegate

Before validation, every supported method scans its arguments for values that
implement `pgx.QueryRewriter`. The check is deliberately conservative across
the complete argument list rather than duplicating pgx's operation-specific
leading-option parser. If any such value is present, the wrapper returns an
exported, constant `ErrQueryRewriterUnsupported` without invoking the
validator, rewriter, or executor. `QueryRow` exposes the same sentinel through
the private failed-row implementation.

Rejecting before validation avoids recording a successful validation for SQL
that is not necessarily the SQL pgx would execute. Positional arguments and
non-rewriting pgx execution options continue through the normal path.

After the rewriter check, each supported method calls `Validator.Validate`
with the exact caller context and SQL string. On success it calls the
corresponding executor method exactly once with the same context, SQL, and
variadic argument values in their original order. The wrapper does not
normalize, copy, prepare, interpolate, or retain SQL or arguments.

For `Exec` and `Query`, a validation failure is returned in the method's error
position together with the zero or nil result, and the executor is not called.
After successful validation, the executor's result and error are returned
without replacement so its error chain remains inspectable.

For `QueryRow`, whose pgx-style signature has no immediate error result,
validation failure produces a private row implementation whose `Scan` returns
that same failure. Successful validation returns the executor's row directly.
The failure row stores only the privacy-safe validation error, not SQL,
arguments, or caller context.

**Alternative considered:** Panicking or returning nil from rejected
`QueryRow` calls would violate normal pgx deferred-error behavior and make the
failure harder to handle correctly.

Calling `RewriteQuery` inside the wrapper was also rejected: the contract
requires a concrete `*pgx.Conn`, which the generic pool-or-connection executor
boundary cannot supply reliably, and passing the rewriter onward could apply
the transformation twice. Merely documenting rewriting as an escape hatch was
rejected because the example should fail closed for a detectable unsafe path.

### 4. Test at the wrapper boundary with deterministic fakes

Use a recording executor fake to assert call count, context identity, exact SQL
and argument forwarding, returned pgx values, driver error identity, and the
absence of delegation on every validation failure class. Use a validator fake
for wrapper plumbing tests and at least one real Engine case to demonstrate the
example against the public SQLGuard contract.

Tests follow ADR 0001 where cases share a lifecycle: map-backed tables,
snake_case names, parallel independent subtests, and `testify/require`.
Separate tests cover `QueryRow.Scan`, constructor validation, and compile-time
pgx compatibility where their lifecycles differ. A dedicated table verifies
that rewriter arguments in different positions are rejected across all three
operations before either collaborator runs.

### 5. Align product requirements with the example boundary

Update the Driver integration usage model, the Official pgx integration
requirement, and the Stage 1 delivery list in `docs/product-requirements.md`.
They will describe a maintained, compiling example rather than a
production-ready official adapter and will point out that unsupported pgx
paths receive no validation guarantee. The general fail-closed and honest
integration-boundary principles remain unchanged.

## Risks / Trade-offs

- **[Users may treat example code as a complete production adapter]** → Put the
  illustrative status and unsupported paths in package documentation, the
  example usage, and the main README rather than only in tests.
- **[Adding pgx to the root module looks like core driver coupling]** → Keep pgx
  imports confined to `example/pgx` and verify the core public API contains no
  pgx types.
- **[Upstream pgx method signatures can change]** → Pin pgx v5 and keep
  compile-time interface assertions plus normal build coverage.
- **[The narrow wrapper leaves real escape hatches]** → Enumerate them
  prominently and do not claim safety for operations that are not wrapped.
- **[Rejecting query rewriters also rejects pgx named arguments]** → Document
  positional arguments as the supported example path and name each affected
  pgx helper explicitly.

## Migration Plan

1. Add the pinned pgx dependency and compiling example package without changing
   core packages.
2. Add wrapper contract tests and include them in the standard quality gates.
3. Update product requirements and README usage/limitations.
4. Run strict OpenSpec validation and the repository precommit gate.

There is no runtime state or data migration. Rollback consists of removing the
example and pgx dependency and reverting the documentation claims; core
validation behavior is unaffected.
