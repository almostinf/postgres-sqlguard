## 1. Core Prepared Validation API

- [x] 1.1 Add the opaque `Prepared` value, exported `ErrInvalidPrepared`, and `ValidationOutcomeInvalidPrepared` with complete GoDoc and compile-time coverage that the existing `Validator` interface remains unchanged.
- [x] 1.2 Refactor the engine pipeline so `Prepare` performs synchronous complete parsing with the existing pre/post-parse context precedence, while successful preparation invokes no rules and emits no terminal outcome.
- [x] 1.3 Implement `ValidatePrepared` over the immutable parser result with context-before-validity checks, first-violation behavior, deterministic top-level and nested-CTE traversal, and exactly one terminal outcome.
- [x] 1.4 Compose `Engine.Validate` from `Prepare` and `ValidatePrepared` without changing its returned errors, traversal order, context behavior, or single-event semantics.
- [x] 1.5 Add table-driven core tests for successful and malformed preparation, zero-value rejection with `errors.Is`, cancellation precedence, rule deferral, repeated stateful/context-dependent rule evaluation, and privacy-safe errors.
- [x] 1.6 Add regression tests comparing direct and prepared validation for allowed input, policy violations, parser failures, cancellation, multi-statement traversal, and nested CTEs.
- [x] 1.7 Add concurrency tests that share one prepared value across goroutines and engines with concurrency-safe rules and verify independent outcomes without parsed-state mutation.

## 2. Prepared Observability

- [x] 2.1 Extend core event tests for `invalid_prepared`, empty `rule_id`, parser and cancellation outcomes from `Prepare`, no event after successful preparation, and exactly one event from composed `Validate` calls.
- [x] 2.2 Cover metrics/logger error and panic isolation for preparation failures and invalid prepared values without changing the original returned result or preventing the other sink from running once.
- [x] 2.3 Update the official Prometheus integration tests to record one bounded `outcome="invalid_prepared"` series with an empty rule identifier.
- [x] 2.4 Update the official `log/slog` integration to classify `invalid_prepared` at ERROR level and test the exact message, bounded attributes, and absence of SQL or parsed data.

## 3. Core Prepared Benchmarks

- [x] 3.1 Reuse the existing `simple`, `medium`, `multi_statement`, and `nested_cte` fixtures across direct and prepared complexity benchmarks without introducing a cross-product.
- [x] 3.2 Add `BenchmarkEngineValidatePreparedByComplexity` with preparation and outcome preflight outside the timed loop and rule evaluation inside it.

## 4. Prepared pgx Example Foundation

- [x] 4.1 Promote `github.com/hashicorp/golang-lru/v2 v2.0.7` to a direct module dependency and initialize the example cache with its generic fixed-size thread-safe LRU implementation.
- [x] 4.2 Create the compiling `example/pgx-prepared` package with local `PreparedValidator` and `Executor` contracts, `CacheOptions`, constructor validation, privacy-safe errors, and illustrative-package GoDoc.
- [x] 4.3 Implement exact cache lookup, `MaxSQLBytes` admission, cache-owned cloned keys, and insertion only after an allowed result or an error discoverable as a policy violation.
- [x] 4.4 Implement the shared miss flow so every preparation error short-circuits prepared validation, insertion, and delegation; cancellation, invalid-prepared, and unknown validation errors do not create entries.
- [x] 4.5 Add cache-focused tests for hits, misses, LRU recency and eviction, capacity, oversized bypass, parser-error exclusion, cancellation exclusion, policy-violation reuse with rule re-evaluation, and independently owned key storage.
- [x] 4.6 Add concurrent cache tests covering exact hits and duplicate identical misses without races, capacity overflow, or shared validation decisions.

## 5. Prepared pgx Guarded Operations

- [x] 5.1 Implement guarded `Exec` and `Query` methods that reject every `pgx.QueryRewriter` before cache access and preserve exact context, SQL, arguments, results, and driver error chains after successful validation.
- [x] 5.2 Implement guarded `QueryRow` with deferred preparation and validation failures exposed through `Scan`, without invoking the executor on rejection.
- [x] 5.3 Add table-driven and concurrent operation tests proving preparation and validation precede each call's delegation, all guarded failures block the executor and remain inspectable, and allowed driver outcomes pass through unchanged.
- [x] 5.4 Add query-rewriter tests for `Exec`, `Query`, and `QueryRow` proving cache, validator, and executor are not touched.
- [x] 5.5 Add a compiling usage example demonstrating stable parameterized SQL with changing positional arguments through the prepared cache wrapper.

## 6. Example Benchmarks and Documentation

- [x] 6.1 Add `BenchmarkGuardedDBCache/hit` with a warmed exact key, no-op executor, and validation inside the timed loop.
- [x] 6.2 Add `BenchmarkGuardedDBCache/miss` using precomputed valid inputs larger than cache capacity so timed calls force parsing and eviction without timed SQL formatting.
- [x] 6.3 Document the `Prepare`/`ValidatePrepared` lifecycle, absolute benchmark interpretation, and the distinction between parsing savings and database latency in the README.
- [x] 6.4 Document in the new package that exact SQL keys and AST literals are retained, parameterized SQL is preferred, limits are not a heap quota, and eviction or garbage collection does not guarantee zeroization.
- [x] 6.5 Update product requirements and example documentation to distinguish SQLGuard prepared values from unsupported pgx/server statement-name execution and list batch, transaction, `CopyFrom`, and unwrapped-executor bypasses.
