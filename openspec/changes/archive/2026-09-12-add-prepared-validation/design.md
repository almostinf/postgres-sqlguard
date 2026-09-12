## Context

See [proposal.md](proposal.md) for motivation. Today `Engine.Validate` owns a
single synchronous pipeline: check context, parse SQL through
`internal/parser`, check context again, traverse every statement, evaluate
rules, and emit one terminal observability event. The parser already converts
the pg_query protobuf tree into package-owned immutable nodes, so the parsed
result can be shared safely without exposing backend-generated types.

The change crosses the public core API, engine control flow, observability
vocabulary, official integrations, benchmarks, and a new pgx example. It must
preserve these constraints:

- parsing remains synchronous and fail closed;
- rules may depend on caller context or mutable concurrent state;
- `Engine` and parsed state remain immutable and concurrency-safe;
- SQL, AST contents, literals, and parser diagnostics never enter public
  errors or observability;
- the existing `Validator` interface and uncached pgx example remain source
  compatible;
- caching is optional, bounded, and outside the core library.

The behavior contracts are defined by the change specs under
[specs/](specs/).

## Goals / Non-Goals

**Goals:**

- Reuse the current immutable parser result across validation calls without
  re-running the PostgreSQL parser.
- Keep rule evaluation, context checks, and terminal outcome emission inside
  every prepared validation call.
- Preserve exactly one terminal observability event for the existing
  `Validate` path and for each guarded operation in the new example.
- Demonstrate a realistic bounded concurrent cache around pgx-style `Exec`,
  `Query`, and `QueryRow` methods.
- Make parsing savings and cache overhead independently measurable.

**Non-Goals:**

- A cache, cache policy, or cache configuration in the core package.
- Caching policy outcomes, normalizing SQL, hashing cache keys, serializing
  ASTs, or exposing parsed nodes publicly.
- Suppressing concurrent duplicate misses with singleflight.
- Supporting pgx batch, transaction, `CopyFrom`, unwrapped executor, or
  manually named server-prepared-statement execution paths.
- Turning either pgx example into a stable production adapter.

## Decisions

### 1. Add engine-scoped preparation methods without changing `Validator`

The public additions are:

```go
type Prepared struct {
    result *parser.Result
}

func (e *Engine) Prepare(ctx context.Context, sql string) (Prepared, error)

func (e *Engine) ValidatePrepared(
    ctx context.Context,
    prepared Prepared,
) error
```

`Prepare` is engine-scoped because a parser failure or cancellation must be
emitted through that engine's configured observability sinks. The returned
value is nevertheless engine-independent: it contains only the immutable
parser result and may be passed to another engine.

The existing `Validator` interface remains unchanged. Adding methods to it
would break every consumer implementation and test double. Packages that need
prepared behavior, including the new example, define a narrow local interface
containing `Prepare` and `ValidatePrepared`.

Alternatives considered:

- A package-level `Prepare` keeps parsing conceptually independent, but it
  cannot report parser failures through a specific engine and would make the
  guarded example lose or manually duplicate observability.
- Adding prepared methods to `Validator` makes discovery convenient but is an
  avoidable breaking change.
- A separate public preparer abstraction adds indirection without a second
  implementation to justify it.

### 2. Split preparation from policy evaluation

`Engine.Prepare` performs only this flow:

1. Return and emit `canceled` if the context is already done.
2. Call `parser.Parse` synchronously.
3. Check the context again, giving cancellation precedence over a simultaneous
   parser error.
4. On parse error, translate it and emit `parser_failure` exactly once.
5. On success, return `Prepared{result: result}` without invoking rules or
   emitting a terminal outcome.

`Engine.ValidatePrepared` performs this flow:

1. Check context before inspecting the prepared value.
2. Reject `prepared.result == nil` with `ErrInvalidPrepared` and emit
   `invalid_prepared`.
3. Traverse the prepared result with `parser.StatementSequence`.
4. Create the existing short-lived `Statement` facade for each root and run
   rules in registration order, with the existing context checks and
   first-violation behavior.
5. Emit exactly one of `allowed`, `policy_violation`, or `canceled`.

`Engine.Validate` calls `Prepare` and, only after success,
`ValidatePrepared`. This centralizes direct and prepared semantics. A failed
preparation has already emitted its terminal event, while successful
preparation emits nothing; therefore composition cannot double count.

Alternatives considered:

- Evaluating rules inside `Prepare` would make prewarming context-dependent,
  reject reusable syntax because of one caller's state, and invite consumers
  to treat a cached result as a cached allow decision.
- Letting successful `Prepare` emit `allowed` would be inaccurate because no
  rule has run and would double count the composed `Validate` path.

### 3. Keep `Prepared` opaque, immutable, and valid across engines

`Prepared` is a small exported value with only an unexported pointer to the
existing immutable parser result. It has no SQL, AST, mutation, or
serialization accessors. Copying it shares immutable state. The zero value has
a nil result and is deliberately invalid, so there is no public constructor
other than successful preparation.

Rules still receive a `Statement` only for the duration of `Evaluate` and must
not retain it. The longer lifetime of the underlying prepared tree does not
change the rule contract or expose mutable access.

Alternatives considered:

- Binding a prepared value to its creating engine is unnecessary because it
  contains no rule or observability state and would prevent useful reuse.
- Exporting parsed nodes would couple consumers to internal representation,
  enlarge the stable API, and make mutation and privacy guarantees harder to
  enforce.
- Returning a pointer to `Prepared` introduces a nil case in addition to the
  unavoidable zero-value case without providing useful identity semantics.

### 4. Represent invalid prepared input explicitly

The core exports `ErrInvalidPrepared` for `errors.Is` inspection and
`ValidationOutcomeInvalidPrepared` with the fixed value `invalid_prepared`.
The outcome always has an empty rule identifier. It is recorded by the
official Prometheus integration like other bounded outcomes and logged by the
official `log/slog` integration at ERROR level.

An already-canceled context wins over invalid prepared input. This matches the
existing engine preference for caller cancellation and prevents rules or
structural validation from starting after cancellation.

Alternatives considered:

- Reporting invalid prepared input as `parser_failure` would conflate malformed
  SQL with API misuse and make operational diagnosis misleading.
- Panicking on a zero value conflicts with fail-closed behavior for malformed
  public input.

### 5. Keep caching in a separate `example/pgx-prepared` package

The new directory has import path `example/pgx-prepared` and package name
`pgxprepared`. It owns local `Executor` and `PreparedValidator` interfaces
rather than depending on the existing example package. Small duplication keeps
both examples independently understandable and avoids treating one unstable
example as another example's API dependency.

The constructor is:

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

Both limits and both collaborators are validated before construction returns.
The cache is per wrapper; there is no global state or implicit default.

Alternatives considered:

- A transparent cache inside `Engine.Validate` would change retention for all
  consumers, add policy to the stable core API, and obscure whether a call
  reparses SQL.
- Extending the existing pgx example would mix the simplest validation pattern
  with the more consequential retention and concurrency trade-offs of caching.

### 6. Use HashiCorp's exact-key, thread-safe, bounded LRU

The example uses `github.com/hashicorp/golang-lru/v2 v2.0.7`, which provides a
generic fixed-size thread-safe LRU cache:

```go
cache, err := lru.New[string, sqlguard.Prepared](options.Capacity)
```

The module is already present indirectly in the repository's module graph and
becomes a direct runtime dependency of the example. The wrapper stores the
constructed `*lru.Cache[string, sqlguard.Prepared]` and keeps
`MaxSQLBytes` as its own policy because the cache library deliberately governs
entry capacity rather than SQL-specific retention rules.

`Get(sql)` performs exact lookup and updates recency. Before `Add`, the wrapper
copies an eligible key with `strings.Clone(sql)`. Cloning prevents a short
substring from retaining an arbitrarily large caller-owned backing string.
The library owns synchronization, LRU ordering, and fixed-capacity eviction;
the wrapper owns key lifetime, admission, and validation semantics.

Only `len(sql) <= MaxSQLBytes` is eligible for insertion. The byte limit bounds
owned key size; capacity bounds retained entry count. The parsed tree size is
not measured precisely, so the example documents that these are retention
controls rather than a strict heap-byte budget.

Exact keys are chosen because digest-only keys introduce a collision path that
is unacceptable for a safety boundary. SQL normalization is excluded because
safe PostgreSQL normalization itself requires parsing and changes the cache's
privacy and correctness model.

Alternatives considered:

- A local `map[string]*list.Element` plus `container/list` would avoid a direct
  dependency but would duplicate concurrency and eviction code in an example
  intended to demonstrate realistic integration.
- 2Q, adaptive, probabilistic, or TTL caches add admission, ghost entries,
  background lifecycle, or expiration semantics that are not required by the
  specified exact LRU behavior.

### 7. Parse outside the cache lock and insert only selected outcomes

The shared wrapper validation helper follows this flow:

```text
reject QueryRewriter in operation method
  -> cache hit: ValidatePrepared
  -> cache miss: Prepare
       -> any error: return immediately
       -> ValidatePrepared
       -> cacheable and (allowed or policy violation): put
       -> any validation error: return without delegation
  -> delegate unchanged inputs
```

Parsing and rule evaluation occur outside calls into the cache. `Get` and
`Add` synchronize only their own bounded cache operations. Simultaneous misses
cannot create duplicate logical keys or exceed capacity, although multiple
goroutines may still parse the same SQL and later add equivalent immutable
prepared values. Avoiding singleflight prevents one caller's context or
failure from being shared with another caller.

A successfully parsed representation is inserted after an allowed result or a
typed policy violation. The violation itself is never stored; later hits run
the rules again. Parser/preparation errors, invalid-prepared errors,
cancellation after preparation, and unknown custom-validator errors do not
create entries. Existing entries are not removed merely because a later
context or rule state rejects them.

Oversized SQL follows the same `Prepare` then `ValidatePrepared` flow but is
never inserted. Any non-nil preparation error short-circuits validation and
delegation, including errors returned by custom `PreparedValidator`
implementations.

### 8. Preserve the pgx guard boundary and make name-based execution explicit

`Exec`, `Query`, and `QueryRow` reject every `pgx.QueryRewriter` argument
before cache access. After successful validation they forward the original
context, SQL string, and ordered arguments unchanged. `QueryRow` returns a
local failed-row value whose `Scan` exposes preparation or validation errors.

The wrapper cannot reliably detect a manually registered server-prepared
statement name passed in the `sql` position: a name can itself be valid SQL
text while pgx resolves it to different SQL. Package documentation therefore
marks name-based execution unsupported and distinguishes it from SQLGuard's
client-side `Prepared` AST representation. Batch, transactions, `CopyFrom`,
and unwrapped executor use remain outside the boundary.

No operation, error, benchmark name, or diagnostic exposes a cache key, SQL
text, arguments, AST values, or raw parser errors.

### 9. Benchmark parsing and cache paths independently

The core adds `BenchmarkEngineValidatePreparedByComplexity` using the same
`simple`, `medium`, `multi_statement`, and `nested_cte` inputs as
`BenchmarkEngineValidateByComplexity`. Preparation and outcome preflight occur
outside the timed loop; only `ValidatePrepared` and its rule evaluation are
timed.

The new example adds `BenchmarkGuardedDBCache/hit` and
`BenchmarkGuardedDBCache/miss` with a no-op executor. Hit setup warms the exact
key before timing. Miss setup precomputes more valid SQL strings than cache
capacity, and the timed loop cycles through them in order so each lookup misses
and forces eviction without formatting SQL inside the timed section.

The existing direct, outcome, and observability benchmark groups remain
unchanged. Results are documented as absolute validation/example costs rather
than a synthetic no-library baseline.

## Risks / Trade-offs

- **[Prepared values retain sensitive AST data longer]** → Keep the type
  opaque, expose no SQL/AST accessors, recommend parameterized SQL, and document
  that callers control prepared-value lifetime.
- **[The example retains exact SQL keys]** → Clone keys into cache-owned
  storage, require entry and input-size limits, recommend parameterized SQL,
  and document that eviction does not guarantee memory zeroization.
- **[Capacity and key length are not an exact heap-byte bound]** → Describe
  them as bounded entry/key controls, not a memory quota; keep a precise AST
  size accounting API out of this example.
- **[The library's mutex adds cache-hit contention]** → Keep parsing and rule
  evaluation outside cache calls and measure the end-to-end hit path. A
  different cache or sharding can be considered later only if profiles show
  contention.
- **[A runtime dependency expands maintenance and license surface]** → Pin the
  already-present `github.com/hashicorp/golang-lru/v2 v2.0.7` module, use only
  its small generic LRU API, and keep SQLGuard-specific policy in the example.
- **[Concurrent misses duplicate expensive parsing]** → Accept bounded
  temporary work rather than sharing caller context or adding singleflight;
  document and race-test the behavior.
- **[Policy-rejected SQL can occupy the cache]** → Keep the cache bounded and
  retain only parsed structure, which makes repeated rejections cheaper while
  still re-running every rule.
- **[A server-prepared statement name can bypass text validation]** → Declare
  name-based execution unsupported and keep it outside the example's claimed
  safety boundary.
- **[Adding an outcome expands metric series]** → Use one fixed bounded value,
  preserve the existing label set, and assign an explicit ERROR log level.
- **[Refactoring `Validate` could change event count or precedence]** → Add a
  direct-versus-composed regression matrix for results, traversal, context
  precedence, and custom/official observability.

## Migration Plan

1. Add the opaque prepared type, invalid-prepared error/outcome, and split the
   engine pipeline while keeping `Validate` behavior covered by regression
   tests.
2. Update official observability integrations and their tests for the new
   bounded outcome.
3. Add prepared validation concurrency tests and core benchmarks.
4. Promote `github.com/hashicorp/golang-lru/v2 v2.0.7` from indirect to direct,
   then add the independent pgx-prepared example, integration tests, race
   coverage, and cache benchmarks.
5. Update README, package documentation, and product requirements with the
   new API lifecycle, retention warning, and explicit integration boundary.
6. Run strict OpenSpec validation, the full precommit gate, race tests, and all
   benchmarks before verification.

No consumer migration is required because all core API changes are additive
and `Validator` is unchanged. Before release, rollback consists of removing
the additive API, outcome, example, and documentation together. After release,
normal semantic-version compatibility rules apply to the exported additions.
