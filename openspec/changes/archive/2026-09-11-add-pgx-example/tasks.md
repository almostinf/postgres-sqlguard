## 1. pgx Example Foundation

- [x] 1.1 Add a pinned pgx v5 dependency and create the `example/pgx`
  `pgxexample` package with GoDoc that identifies it as illustrative and not a
  stable production adapter.
- [x] 1.2 Add constructor tests for nil and typed-nil validators and executors,
  then define the narrow `Executor`, `GuardedDB`, and fail-fast
  `NewGuardedDB` contracts.
- [x] 1.3 Add compile-time checks that `*pgx.Conn` and `*pgxpool.Pool` satisfy
  `Executor`, while keeping every pgx import outside the core validation
  packages.

## 2. Guarded Operations and Failure Semantics

- [x] 2.1 Add recording-fake tests proving accepted `Exec`, `Query`, and
  `QueryRow` calls validate first, delegate exactly once, and preserve the
  exact context, SQL string, ordered arguments, results, and driver error
  chains.
- [x] 2.2 Implement `GuardedDB.Exec`, `GuardedDB.Query`, and
  `GuardedDB.QueryRow` with validate-before-delegate behavior and no SQL or
  argument rewriting or retention.
- [x] 2.3 Add tests covering policy violations, parser failures, and context
  errors for every supported operation, asserting that the executor is never
  called and that `errors.Is` or `errors.As` discovers the original failure.
- [x] 2.4 Implement the rejected `Exec` and `Query` result paths and a private
  failed-row implementation that exposes rejected `QueryRow` validation
  through `Scan` without retaining SQL, arguments, or caller context.
- [x] 2.5 Add at least one real-Engine integration test demonstrating that the
  example blocks rejected SQL and delegates accepted SQL through the public
  SQLGuard validator contract.
- [x] 2.6 Add matrix tests proving every supported operation rejects a pgx
  `QueryRewriter` in any argument position before invoking the validator or
  executor, with `QueryRow` exposing the rejection through `Scan`.
- [x] 2.7 Add the privacy-safe `ErrQueryRewriterUnsupported` sentinel and
  implement the shared fail-closed argument check while preserving positional
  argument behavior.

## 3. Usage and Boundary Documentation

- [x] 3.1 Add a compiling usage example for wrapping a pgx pool and package
  documentation that distinguishes the guarded methods from access through
  the retained raw pool.
- [x] 3.2 Document in `example/pgx` and the main README that batches, prepared
  statement workflows, transactions, nested transactions, `CopyFrom`, and raw
  connection use are outside the example's validation boundary.
- [x] 3.3 Update the driver-integration usage model, pgx integration
  requirements, and Stage 1 delivery list in `docs/product-requirements.md` to
  promise a maintained compiling example rather than an official
  production-ready adapter.
