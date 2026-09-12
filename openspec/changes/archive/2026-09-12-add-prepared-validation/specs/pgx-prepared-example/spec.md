## Purpose

Defines an illustrative pgx integration that reuses SQLGuard prepared values
through a bounded concurrent cache while preserving per-call policy checks,
fail-closed execution, and explicit privacy and driver limitations.

## ADDED Requirements

### Requirement: Illustrative prepared pgx integration scope
The repository SHALL provide a compiling `example/pgx-prepared` package that
uses SQLGuard preparation and prepared validation. The package MUST be
identified as an illustrative example rather than an official, stable, or
production-ready pgx adapter, and the core validation packages MUST remain
usable without importing a pgx API.

#### Scenario: Consumer finds the example scope
- **WHEN** a consumer reads the prepared pgx example and its documentation
- **THEN** the example identifies its supported operations, cache behavior, and lack of a production compatibility commitment

#### Scenario: Core remains cache and driver independent
- **WHEN** a consumer imports and uses the core validation package
- **THEN** the consumer is not required to use pgx or the example cache

### Requirement: Explicit bounded cache configuration
The example SHALL require positive cache capacity and maximum-cacheable-SQL
byte limits when it is constructed. Invalid limits or nil collaborators MUST
fail construction without returning a partially configured wrapper. The cache
MUST be local to the constructed wrapper, MUST bound its entry count by the
configured capacity, and MUST NOT cache SQL whose byte length exceeds the
configured maximum.

#### Scenario: Invalid cache limits are rejected
- **WHEN** construction receives a non-positive capacity or maximum-cacheable-SQL byte limit
- **THEN** construction returns a privacy-safe configuration error and no wrapper

#### Scenario: Oversized SQL bypasses storage
- **WHEN** a guarded call receives SQL larger than the configured cacheable byte limit
- **THEN** the SQL is prepared and validated for that call but no cache entry is retained

#### Scenario: Capacity evicts the least recently used entry
- **WHEN** inserting a prepared value would exceed cache capacity
- **THEN** the least recently used entry is evicted while more recently used entries remain eligible for reuse

#### Scenario: Cache key owns bounded storage
- **WHEN** cacheable SQL is a short substring backed by a larger caller string
- **THEN** the cache retains an exact independently owned key whose size is bounded by the configured SQL byte limit rather than retaining the larger backing string through the key

### Requirement: Cache reuses parsing but not validation decisions
For an exact SQL cache hit, the example SHALL invoke prepared validation for
that call without preparing the SQL again. For a miss, it SHALL prepare the
complete SQL and then invoke prepared validation. It MUST NOT cache parser or
context failures from preparation, MUST NOT invoke prepared validation after
any preparation error, and MUST NOT delegate after any preparation or
validation error. The example MUST evaluate rules through prepared validation
on every hit and MUST NOT cache an allow or reject decision.

#### Scenario: Cache miss prepares and validates
- **WHEN** a guarded call receives cacheable SQL with no exact cache entry and preparation succeeds
- **THEN** the example prepares it once and invokes prepared validation once before any delegation

#### Scenario: Cache hit validates without parsing
- **WHEN** a guarded call receives SQL with an exact prepared cache entry
- **THEN** the example invokes prepared validation without invoking preparation and delegates only if that validation succeeds

#### Scenario: Preparation error fails closed
- **WHEN** preparation returns any non-nil error
- **THEN** the example invokes neither prepared validation nor the executor and does not insert a cache entry

#### Scenario: Policy violation is re-evaluated from cached structure
- **WHEN** prepared validation rejects successfully prepared cacheable SQL with a policy violation and a later call uses the same SQL
- **THEN** the parsed representation may be reused but the rule set is evaluated again for the later call

#### Scenario: Cancellation after preparation avoids insertion
- **WHEN** preparation succeeds but prepared validation returns a caller-context error before a miss is inserted
- **THEN** the executor is not called and the miss does not create a cache entry

### Requirement: Validation precedes supported pgx operations
The example wrapper SHALL support pgx-style `Exec`, `Query`, and `QueryRow`
operations. It MUST complete prepared SQLGuard validation before delegating any
supported operation to the wrapped executor.

#### Scenario: Allowed Exec delegates after validation
- **WHEN** `Exec` receives SQL that prepared validation accepts
- **THEN** the wrapper delegates execution exactly once after validation succeeds

#### Scenario: Allowed Query delegates after validation
- **WHEN** `Query` receives SQL that prepared validation accepts
- **THEN** the wrapper delegates the query exactly once after validation succeeds

#### Scenario: Allowed QueryRow delegates after validation
- **WHEN** `QueryRow` receives SQL that prepared validation accepts
- **THEN** the wrapper delegates the row query exactly once after validation succeeds

### Requirement: Query rewriting is rejected before guarded work
The example wrapper MUST reject any argument that implements pgx
`QueryRewriter` before cache access, preparation, prepared validation, or
executor delegation. This includes pgx named- and struct-argument helpers
implemented through `QueryRewriter`. The returned error MUST be privacy-safe
and discoverable with `errors.Is` against a stable example-package error.

#### Scenario: Exec rejects a query rewriter
- **WHEN** `Exec` receives any argument that implements pgx `QueryRewriter`
- **THEN** it returns the unsupported-rewriter error without cache access, preparation, validation, or delegation

#### Scenario: Query rejects a query rewriter
- **WHEN** `Query` receives any argument that implements pgx `QueryRewriter`
- **THEN** it returns the unsupported-rewriter error without cache access, preparation, validation, or delegation

#### Scenario: QueryRow defers a rejected query rewriter
- **WHEN** `QueryRow` receives any argument that implements pgx `QueryRewriter`
- **THEN** it performs no guarded work or delegation and its returned row exposes the unsupported-rewriter error through `Scan`

### Requirement: Prepared validation failure prevents delegation
If preparation or prepared validation returns any error, the example wrapper
MUST NOT invoke the wrapped executor. Policy violations, parser failures,
invalid-prepared failures, and caller-context errors MUST remain discoverable
through Go's standard `errors.Is` or `errors.As` mechanisms as applicable.

#### Scenario: Policy violation blocks execution
- **WHEN** a supported operation receives SQL rejected by a configured rule
- **THEN** the wrapped executor is not called and the caller can discover the policy violation with `errors.As`

#### Scenario: Parser failure blocks execution
- **WHEN** a supported operation receives SQL rejected during preparation by the PostgreSQL parser
- **THEN** the wrapped executor is not called and the caller can discover the parser failure with `errors.As`

#### Scenario: Context failure blocks execution
- **WHEN** preparation or prepared validation returns a caller-context cancellation or deadline error
- **THEN** the wrapped executor is not called and the caller can discover the context error with `errors.Is`

#### Scenario: Invalid prepared value blocks execution
- **WHEN** the configured prepared validator returns an invalid prepared value that prepared validation rejects
- **THEN** the wrapped executor is not called and the caller can discover `ErrInvalidPrepared` with `errors.Is`

### Requirement: Allowed calls preserve pgx inputs and outcomes
For calls without a pgx `QueryRewriter`, after successful prepared validation
the example wrapper SHALL pass the exact caller context, SQL string, and
ordered argument values to the wrapped executor without modification. It
SHALL preserve the executor's operation result and error behavior.

#### Scenario: Inputs pass through unchanged
- **WHEN** prepared validation accepts a supported operation with a context, SQL string, and arguments
- **THEN** the wrapped executor receives that same context, SQL string, and ordered arguments

#### Scenario: Exec outcome passes through
- **WHEN** the wrapped executor returns an `Exec` command tag or driver error after successful validation
- **THEN** the caller receives that command tag and can inspect that driver error through its original error chain

#### Scenario: Query outcome passes through
- **WHEN** the wrapped executor returns rows or a driver error after successful `Query` validation
- **THEN** the caller receives those rows and can inspect that driver error through its original error chain

### Requirement: QueryRow exposes deferred guarded failures
Because pgx-style `QueryRow` does not return an immediate error, the example
wrapper SHALL expose a preparation or validation failure from the returned
row's `Scan` operation. A rejected `QueryRow` MUST NOT invoke the wrapped
executor.

#### Scenario: Rejected QueryRow fails on Scan
- **WHEN** `QueryRow` preparation or validation fails and the caller invokes `Scan`
- **THEN** `Scan` returns an error through which the original failure remains discoverable

#### Scenario: Accepted QueryRow preserves row behavior
- **WHEN** prepared validation for `QueryRow` succeeds
- **THEN** the wrapper returns the delegated row and its `Scan` behavior is preserved

### Requirement: Concurrent cache use is safe
One initialized example wrapper SHALL support concurrent guarded calls without
data races, cache corruption, exceeding the configured retained entry count,
or sharing validation decisions. Concurrent misses for identical SQL MAY
perform duplicate preparation, but each call MUST receive its own prepared
validation result.

#### Scenario: Concurrent calls are race-free
- **WHEN** multiple goroutines use the same wrapper with concurrency-safe validator and executor implementations
- **THEN** calls preserve their own outcomes and the Go race detector reports no wrapper or cache race

#### Scenario: Concurrent identical misses remain fail closed
- **WHEN** concurrent calls miss the cache for the same SQL
- **THEN** each call completes prepared validation before its own delegation even if preparation is duplicated

### Requirement: Prepared cache privacy is explicit
The example MUST document that cache keys retain exact SQL and prepared values
retain parsed identifiers, literals, and byte values. It MUST recommend
parameterized SQL, MUST state that cache limits do not erase retained data,
and MUST state that eviction and Go garbage collection do not guarantee prompt
zeroization. Errors, benchmark labels, and example-provided diagnostics MUST
NOT expose SQL, arguments, literals, cache keys, parsed structure, secrets, or
raw parser diagnostics.

#### Scenario: Consumer reviews retention risk
- **WHEN** a consumer reads the prepared pgx example documentation
- **THEN** the consumer is told what data the cache retains, how it is bounded, and why parameterized SQL reduces retention risk

#### Scenario: Failures do not expose cached data
- **WHEN** preparation, validation, configuration, or delegation fails for input containing a unique secret
- **THEN** no example-provided error or diagnostic exposes that secret, SQL input, cache key, or parsed value

### Requirement: Unsupported and bypassable paths are explicit
The example documentation MUST state that pgx batch execution, transactions,
nested transactions, `CopyFrom`, direct use of an unwrapped connection, and
execution of manually registered pgx or server prepared statements by name are
outside its guarded boundary. It MUST distinguish SQLGuard's client-side
prepared representation from pgx and server prepared statements and MUST NOT
imply that unsupported paths are validated.

#### Scenario: Consumer reviews the safety boundary
- **WHEN** a consumer reads the prepared pgx example documentation
- **THEN** every unsupported or bypassable path is listed as not protected by the wrapper

#### Scenario: Server prepared statement names are not presented as supported
- **WHEN** a consumer reviews how the wrapper accepts its SQL argument
- **THEN** documentation warns that passing a server prepared statement name can execute SQL different from the validated string and is unsupported

### Requirement: Prepared cache benchmarks are runnable
The repository SHALL provide runnable example benchmarks that report absolute
latency and allocations for cache-hit and forced cache-miss guarded calls with
a no-op executor. Inputs needed to force misses MUST be prepared outside the
timed loop so query generation does not contaminate the measurements.

#### Scenario: Cache paths are measured separately
- **WHEN** a maintainer runs the documented benchmark command for `BenchmarkGuardedDBCache`
- **THEN** separate `hit` and `miss` results report standard Go benchmark measurements

#### Scenario: Miss benchmark forces eviction without timed formatting
- **WHEN** the cache-miss benchmark runs
- **THEN** it cycles through precomputed SQL inputs exceeding cache capacity so each timed call misses without formatting SQL inside the timed loop
