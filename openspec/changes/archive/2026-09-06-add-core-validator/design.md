## Context

See `proposal.md` for motivation and the three capability specs for observable behavior. The repository currently contains only package documentation and tooling. This change introduces the first public API and a CGO-backed parser into a library whose later stages must support built-in rules, pgx integration, audit mode, and potentially a no-CGO backend without weakening validation semantics.

The parser boundary is security-sensitive: the engine must parse the complete input before evaluating policy, traverse statement-bearing CTEs itself, and never expose parser diagnostics that may echo submitted SQL. The engine must also remain immutable after construction so one instance can safely serve concurrent callers.

## Goals / Non-Goals

**Goals:**

- Keep public validation and rule contracts independent of database drivers and parser implementation types.
- Make parsing, traversal, rule ordering, and first-violation selection deterministic and directly testable.
- Restrict public errors to bounded, privacy-safe metadata.
- Establish a benchmark baseline without adding caching or other speculative optimization.

**Non-Goals:**

- Promise compatibility with every PostgreSQL server major; this version targets the grammar supplied by the pinned parser release.
- Expose, mutate, normalize, fingerprint, or deparse the backend parse tree.
- Make unsafe custom rule state safe through locking owned by the engine.
- Add policy configuration, built-in policies, driver wrappers, or observability hooks.

## Decisions

### Use pg_query_go v6.2.2 behind an internal parser adapter

Pin `github.com/pganalyze/pg_query_go/v6` at `v6.2.2`. This release embeds the PostgreSQL 17 parser through libpg_query and returns the raw parse tree as generated Go protobuf structures. It satisfies the real-grammar requirement and the project's current CGO stage.

Only `pg_query.Parse` will be used. The internal `parser` package will own the dependency and translate its result into engine-owned read-only statement views. Parser protobuf types, raw parser errors, deparse, normalize, and fingerprint APIs will not cross the internal boundary.

Using a hand-written or generic SQL parser was rejected because PostgreSQL-specific grammar accuracy is the safety requirement. The WASM-based drop-in parser was rejected for this stage because no-CGO support is intentionally deferred. Exposing pg_query protobuf nodes directly to rules was rejected because that would make a backend schema and its major-version churn part of the public API.

### Keep the public API small and immutable

The root package will define:

- a `Validator` interface with `Validate(context.Context, string) error`;
- an `Engine` that implements `Validator`;
- a constructor that accepts an ordered rule collection and returns either a fully initialized engine or a construction error;
- a `Rule` contract with a stable identifier and an evaluation method over a read-only `Statement` view;
- public typed `Violation` and `ParseError` values.

The constructor will reject nil rules, empty identifiers, and duplicate identifiers. It will copy the supplied rule slice and build immutable registration metadata. There will be no mutating registration method, default rules, package-level registry, or global engine.

A mutable engine with `Register` was rejected because concurrent registration complicates ordering and race guarantees. A concrete-only validator was rejected because driver integrations need a narrow dependency boundary for composition and testing.

### Keep the public API in the root package

The implementation will keep the small public surface in package `sqlguard` and isolate the parser backend under `internal/parser`. The expected project structure after this change is:

```text
postgres-sqlguard/
├── doc.go                         # package-level GoDoc
├── validator.go                   # Validator
├── engine.go                      # Engine, NewEngine, and validation flow
├── rule.go                        # Rule, rule result, and identifier validation
├── statement.go                   # public read-only Statement facade
├── errors.go                      # Violation, ParseError, and error categories
├── engine_test.go                 # ordering, CTE, context, and concurrency tests
├── errors_test.go                 # typed errors and SQL privacy tests
├── statement_test.go              # read-only facade contract tests
├── validator_benchmark_test.go    # single, batch, and nested-CTE benchmarks
│
├── internal/
│   ├── parser/
│   │   ├── doc.go
│   │   ├── parser.go              # parser-neutral result types
│   │   ├── node.go                # parser-neutral node representation
│   │   ├── pgquery.go             # pg_query_go adapter
│   │   ├── traversal.go           # top-level statement and nested CTE traversal
│   │   ├── pgquery_test.go
│   │   ├── traversal_test.go
│   │   └── fuzz_test.go
│   └── testutil/
│       ├── doc.go
│       └── rules.go               # shared concurrency-safe test rules and spies
│
├── go.mod                         # direct pg_query_go/v6 dependency
├── go.sum
└── README.md                      # API, CGO, and PostgreSQL grammar compatibility
```

File boundaries may be combined when an implementation remains small, but package boundaries are intentional: consumers import only the module root, the root package owns all public contracts, and only `internal/parser` imports `pg_query_go`. `internal/testutil/rules.go` will be added only if rule doubles are shared by multiple test files; otherwise those helpers remain local to their tests.

Separate public `validator`, `rule`, or `statement` subpackages were rejected because they would fragment a deliberately small API and force consumers to coordinate types across imports. Splitting parser conversion and traversal into additional internal packages was rejected until their responsibilities require independent reuse or testing.

### Give rules a library-owned read-only statement view

`Statement` will be a library-owned facade representing one statement root. It will expose stable structural inspection needed by custom rules—statement kind plus read-only traversal and typed access to named child fields—without exposing the submitted SQL, source slices, mutable protobuf objects, or pg_query types. Returned collections will be snapshots or iterators over immutable per-call data.

The parser adapter will convert or wrap backend nodes before rule evaluation. The engine owns the representation for the duration of a validation call; rules cannot mutate it. This preserves one consistent interpretation across all rules and leaves room for a future parser backend to produce the same facade.

An opaque statement exposing only a command enum was rejected because third-party rules could not inspect arbitrary PostgreSQL structure. A raw JSON tree was rejected because each rule would need reparsing and unchecked field handling. The facade should stay minimal: add semantic convenience methods only in the changes that need them.

### Rules return a narrow decision and the engine creates violations

Rule evaluation will communicate pass or reject through a narrow library-owned result rather than an arbitrary error. On rejection, the engine constructs the public `Violation` from the registered rule identifier. A violation exposes the stable rule ID through an accessor and uses a constant safe error message; it has no SQL, free-form message, source location, or arbitrary cause field.

This prevents custom rule error text from leaking SQL through `Error` or `Unwrap` and makes the engine the owner of error privacy. Allowing rules to return arbitrary errors was rejected because the engine could not guarantee the `validation-errors` privacy contract. Additional bounded finding metadata, if needed, requires a later capability change.

### Parse first, then traverse in deterministic pre-order

`Validate` will execute these stages:

1. Check the caller context.
2. Parse the complete SQL string once through the internal adapter.
3. Reject the whole call with `ParseError` if parsing fails; no rule runs for a partially parsed batch.
4. Build a deterministic sequence in top-level input order. For each top-level statement, visit its root first, then recursively visit statement-bearing CTE entries depth-first in their declaration order.
5. For every visited statement, check context and evaluate rules in registration order.
6. Stop immediately on context cancellation or the first rule rejection; otherwise succeed after all statement-rule pairs are evaluated.

The root-before-CTE order is deterministic and keeps each submitted top-level statement as the primary unit. A nested mutation cannot bypass a targeted rule: if earlier pairs pass, the engine reaches every nested statement; if an earlier pair rejects, execution has already failed closed.

Empty, whitespace-only, comment-only, and semicolon-only input that the PostgreSQL parser represents as zero statements will succeed because there is no executable statement to evaluate. Malformed non-empty input remains a parser failure.

Rule-driven recursive statement discovery was rejected because individual rules could omit nested CTEs. Interleaving parse and evaluation was rejected because a later malformed statement could otherwise be hidden behind an earlier accepted or rejected statement.

### Sanitize parser failures at the adapter boundary

`ParseError` will contain only a bounded failure category and a constant privacy-safe message. It will not retain or unwrap the pg_query error because upstream diagnostics may include tokens or fragments derived from the input. `Violation` follows the same closed metadata model. Pointer forms of both public types will support `errors.As`; application wrapping remains inspectable through the standard wrapping chain outside the library.

Construction errors for invalid rule registration will also use fixed messages plus the bounded offending rule identifier where applicable. Identifiers will be validated for a conservative maximum length and character set so they remain safe operational metadata.

Wrapping the raw parser error was rejected despite its debugging value because privacy is a stronger product boundary. Detailed diagnostics can be added later only through an explicitly privacy-reviewed mechanism.

### Preserve context without claiming preemptible C parsing

The exact caller context is passed to each rule. The engine checks `ctx.Err()` before parsing, after parsing, and before each rule invocation. It returns the context error directly so `errors.Is` works for cancellation and deadlines.

The CGO parser call is synchronous and does not accept a Go context, so cancellation cannot interrupt it mid-call; cancellation is observed immediately after control returns. This limitation will be documented. Running every parse in a goroutine was rejected because it would not stop the C work, would complicate resource ownership, and could accumulate abandoned parser calls.

### Make concurrency safety an immutability contract

After construction, `Engine` contains only immutable registration data. Every parse result, statement sequence, and traversal cursor belongs to one validation call. The engine adds no shared cache or mutable counters.

The public Rule documentation states that the same rule instance may be invoked concurrently by different validation calls; custom rules must therefore be immutable or synchronize their own state. Engine tests will use concurrency-safe spies and `go test -race` to verify that the engine and parser adapter introduce no races.

Serializing all validations with an engine mutex was rejected because it hides unsafe rule design, reduces throughput, and is unnecessary for immutable engine state.

### Test at public and internal boundaries

Public black-box tests will cover successful driver-free validation, explicit registration, invalid identifiers, deterministic rule order, first violation, context propagation and cancellation, typed error inspection, privacy sentinels, and concurrent use. Parser/traversal tests will cover PostgreSQL-specific syntax, comment and literal keywords, malformed batches, multiple top-level statements, data-modifying CTEs, and recursive CTE nesting.

Fuzz tests seeded with valid and malformed SQL will assert that validation never panics and never places input sentinels in public errors. Test doubles will implement the real public Rule contract rather than mocking parser internals.

Benchmarks will include single-statement, multi-statement, and nested-CTE inputs with a no-op rule, call `ReportAllocs`, and avoid pass/fail performance thresholds. The baseline records current cost; it does not justify a cache.

## Risks / Trade-offs

- [CGO dependency increases first-build time and requires a C compiler] → Keep the dependency internal, retain the existing CGO-enabled CI gate, and document the supported build requirement.
- [PostgreSQL 17 grammar differs from another server major] → Document the parser grammar version and update it only through a reviewed dependency change with compatibility fixtures.
- [Backend parser can crash the process on an upstream C defect] → Call only the parse API, maintain malformed-input and fuzz coverage, pin the dependency, and avoid deparse paths with known unsafe-input warnings.
- [A generic read-only statement facade may be less convenient than raw protobuf nodes] → Keep traversal lossless enough for custom rules and add focused semantic helpers with future rule capabilities rather than leaking backend types.
- [Parser diagnostics are less detailed after sanitization] → Prefer safe typed categorization; never trade SQL confidentiality for richer default errors.
- [Cancellation cannot preempt an in-progress CGO parse] → Check context around the parser boundary and document the limitation instead of creating leaking goroutines.
- [A custom rule can introduce races] → Make concurrent invocation explicit in the Rule contract and verify engine safety using concurrency-safe rules under the race detector.
- [Depth-first traversal order becomes observable through first violation] → Specify and test the order so future changes require an intentional capability update.

## Migration Plan

1. Add and pin the parser dependency, confirming the existing CGO build and race-test gates on supported development and CI environments.
2. Implement the internal parser adapter, immutable statement facade, and deterministic traversal with focused internal tests.
3. Implement public errors, rule contract, constructor, and engine through public black-box tests.
4. Add concurrency, fuzz, privacy, multi-statement, nested-CTE, and benchmark coverage.
5. Update package and contributor-facing documentation for the new public API and its CGO, grammar-version, context, and rule-concurrency boundaries.

Rollback removes the new public files and parser dependency before a release. After publication, incompatible public API or traversal changes require normal semantic-versioning treatment rather than silent replacement.
