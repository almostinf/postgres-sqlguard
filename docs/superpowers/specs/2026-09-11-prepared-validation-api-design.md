# Prepared Validation API and pgx Cache Example Design

## Goal

Extend the core API with a reusable parsed SQL representation so applications
can avoid repeated PostgreSQL parsing without caching validation decisions.
Add a compiling `example/pgx-prepared` package that demonstrates a bounded,
concurrent cache around that API, with tests and benchmarks.

The optimization must preserve fail-closed validation, per-call rule
evaluation, context behavior, observability, driver independence, and the
project's privacy boundary.

## Scope

This change adds:

- `Engine.Prepare` and `Engine.ValidatePrepared`;
- an opaque immutable `Prepared` value;
- a privacy-safe invalid-prepared error and observability outcome;
- direct prepared-validation benchmarks;
- a separate `example/pgx-prepared` wrapper with a bounded LRU cache;
- wrapper tests and cache hit/miss benchmarks;
- documentation of performance and data-retention trade-offs.

The core library will not provide a transparent or automatic cache. The
existing `example/pgx` package remains the simpler uncached example.

## Core API

The public API will have this shape:

```go
type Prepared struct {
    // unexported immutable parser state
}

func (e *Engine) Prepare(ctx context.Context, sql string) (Prepared, error)

func (e *Engine) ValidatePrepared(ctx context.Context, prepared Prepared) error

var ErrInvalidPrepared error

const ValidationOutcomeInvalidPrepared ValidationOutcome = "invalid_prepared"
```

`Prepared` is a value with no exported fields or AST accessors. A successful
value may be copied and used concurrently because the parser-owned tree is
immutable. It is not tied to the `Engine` that created it and may be validated
by another engine. Its zero value is invalid.

The existing `Validator` interface remains unchanged. This avoids breaking
consumer implementations and test doubles. The prepared pgx example defines a
local interface containing `Prepare` and `ValidatePrepared`, which `Engine`
satisfies.

## Validation Semantics

`Prepare` checks the caller context, parses the complete input once with the
existing PostgreSQL grammar, checks the context again after synchronous
parsing, and converts the result to the same immutable parser-neutral tree
used by normal validation. It does not evaluate rules.

If parsing fails, `Prepare` returns the existing typed, privacy-safe
`ParseError` and synchronously emits one `parser_failure` event through the
engine's configured metrics and logger. The caller does not invoke
`ValidatePrepared` after this error. If cancellation wins before or after
parsing, `Prepare` returns the identifiable context error and emits one
`canceled` event. Successful preparation emits no terminal outcome because no
policy decision has occurred yet.

`ValidatePrepared` checks the context and the prepared value, then evaluates
all registered rules in registration order across every top-level statement
and statement-bearing nested CTE. It checks context during traversal and emits
exactly one terminal outcome for every call. Rules are evaluated again on
every call, including cache hits, so caller context and mutable rule state can
affect each decision.

A zero-value `Prepared` fails closed with `ErrInvalidPrepared` and emits the
bounded `invalid_prepared` outcome. The error contains no SQL or parser data.

`Engine.Validate` is expressed internally as `Prepare` followed by
`ValidatePrepared`. It stops after a preparation error, ensuring the existing
API still emits exactly one outcome and preserves its observable behavior.

## Prepared pgx Example

The new directory is `example/pgx-prepared`, with package name `pgxprepared`.
It is an illustrative, compiling package rather than a production adapter or
stable compatibility commitment.

It guards the same pgx-style `Exec`, `Query`, and `QueryRow` paths as the
existing example. Its local validator interface is:

```go
type PreparedValidator interface {
    Prepare(context.Context, string) (sqlguard.Prepared, error)
    ValidatePrepared(context.Context, sqlguard.Prepared) error
}
```

The constructor accepts the validator, executor, and explicit cache options:

```go
type CacheOptions struct {
    Capacity    int
    MaxSQLBytes int
}

func NewGuardedDB(
    validator PreparedValidator,
    executor Executor,
    options CacheOptions,
) (*GuardedDB, error)
```

Both limits must be positive. Invalid collaborators or options prevent partial
construction and return privacy-safe errors.

Each guarded call follows this sequence:

1. Reject any pgx `QueryRewriter` argument before cache access, preparation,
   validation, or delegation.
2. Look up the exact SQL string in the local cache.
3. On a hit, call `ValidatePrepared`.
4. On a miss in which the SQL exceeds `MaxSQLBytes`, call `Prepare` and
   `ValidatePrepared` without inserting the result.
5. On a cacheable miss, call `Prepare`; return immediately on a parser or
   context error; otherwise insert the prepared representation and call
   `ValidatePrepared`.
6. Delegate the unchanged context, SQL, and ordered arguments exactly once
   only after validation succeeds.

Successfully parsed representations remain cacheable even when later rule
evaluation rejects them. The cache stores parsing work, never an allow/deny
decision, and every future hit re-runs the rules.

`QueryRow` continues to expose preparation or validation failures from
`Scan`, without calling the executor.

## Cache Design

The cache is owned by one `GuardedDB` and is not global. It is an LRU bounded
by entry count, while `MaxSQLBytes` prevents a single accepted key from being
arbitrarily large. It uses exact SQL strings as keys, avoiding correctness
risk from digest collisions.

A mutex protects the map, LRU list, and recency updates. Parsing occurs outside
the mutex. Cache insertion performs a second lookup so concurrent misses do
not corrupt state or create duplicate entries. Concurrent callers may parse
the same missing SQL more than once; this is an accepted simplicity trade-off
and does not affect correctness. The design introduces no singleflight
dependency or lock spanning CGO parsing and rule execution.

Parser failures are never cached. Oversized inputs are validated normally but
never cached. Eviction removes the cache's references but cannot guarantee
immediate memory erasure under Go's garbage collector.

## Privacy and Safety

The immutable parser representation contains structural strings and bytes,
including identifiers and literal values. Caching therefore extends the heap
lifetime of potentially sensitive query contents. The example documentation
must state that:

- `Prepared` values must not be logged, serialized, or retained indefinitely;
- parameterized SQL should be used so changing values remain driver arguments
  rather than cache keys or AST literals;
- the cache retains exact SQL keys and parsed AST contents;
- entry and input-size limits reduce retention but do not erase memory;
- eviction and garbage collection do not guarantee prompt zeroization.

No errors, observability events, cache diagnostics, or benchmark labels may
contain SQL text, literals, arguments, cache keys, or raw parser diagnostics.

## Error and Observability Behavior

Preparation reuses the existing parser and cancellation error contracts.
Invalid prepared values use `ErrInvalidPrepared`; callers inspect it with
`errors.Is`. The `invalid_prepared` outcome is a fixed bounded label and is
added consistently to the core observability contract and official sinks.

Observability sink errors and panics remain isolated from returned validation
results. Preparation failures, invalid prepared values, policy violations,
cancellation, and allowed prepared validations each emit no more than one
terminal event for their logical operation.

## Tests

Core tests cover:

- successful preparation without rule calls or terminal events;
- typed parser failure and exactly one `parser_failure` event;
- context cancellation before and after parsing;
- zero-value rejection with `ErrInvalidPrepared` and `invalid_prepared`;
- complete top-level and nested-CTE traversal;
- per-call evaluation of context-dependent and stateful rules;
- use of one prepared value across engines and goroutines;
- unchanged `Validate` error and observability behavior;
- malformed inputs and race-detector coverage.

The pgx-prepared example tests cover:

- constructor validation;
- exact cache hits, misses, recency updates, eviction, and oversized bypass;
- parser failures not being cached;
- policy violations retaining only the parsed representation and being
  re-evaluated on later calls;
- concurrent cache access;
- `QueryRewriter` rejection before all guarded work;
- rejected operations never reaching the executor;
- unchanged context, SQL, argument, result, and driver-error forwarding;
- deferred `QueryRow.Scan` failures.

## Benchmarks

The benchmark suite reports four distinct paths:

- ordinary `Engine.Validate`;
- direct `Engine.ValidatePrepared`, with preparation outside the timed loop;
- pgx-prepared wrapper cache hit;
- pgx-prepared wrapper cache miss.

Direct benchmarks reuse the established SQL complexity cases so maintainers
can compare absolute latency and allocations. Wrapper benchmarks use a no-op
executor and precomputed inputs. The miss benchmark cycles through more unique
precomputed SQL strings than the configured cache capacity, avoiding timed
query formatting while forcing misses and eviction.

Results remain absolute SQLGuard/example costs rather than a synthetic
"without SQLGuard" comparison. Documentation distinguishes parsing savings
from database latency and warns that single-threaded benchmarks do not measure
production p99 latency or contention.

## Documentation and Specification Updates

The OpenSpec change updates the `validation-engine` and `observability`
capabilities and introduces a `pgx-prepared-example` capability. Product
requirements will distinguish the existing uncached pgx example's unsupported
prepared workflows from the separately documented prepared-validation
example. README and package documentation will show the API lifecycle, cache
limits, supported execution paths, benchmarks, and privacy trade-offs.

## Non-Goals

This change does not add:

- a core-library cache, implicit caching, or global cache;
- cached rule outcomes or bypassed per-call policy evaluation;
- SQL normalization, redaction, hashing, or AST serialization;
- singleflight miss suppression;
- a production-supported pgx adapter;
- protection for pgx batch, transaction, nested transaction, `CopyFrom`, or
  unwrapped connection paths;
- changes to audit-mode or no-CGO delivery stages.
