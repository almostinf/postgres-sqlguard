## 1. PostgreSQL Parser Boundary

- [x] 1.1 Add and pin `github.com/pganalyze/pg_query_go/v6` at `v6.2.2` as a direct module dependency.
- [x] 1.2 Define parser-neutral immutable result and node types in `internal/parser`, and implement the CGO-backed adapter using only `pg_query.Parse`.
- [x] 1.3 Add parser adapter tests for PostgreSQL-specific syntax, keywords inside comments and literals, empty-input forms, truncated SQL, and a malformed statement within a multi-statement input.
- [x] 1.4 Implement deterministic top-level and statement-bearing CTE discovery in root-first, depth-first declaration order at every nesting level.
- [x] 1.5 Add traversal tests covering multiple top-level statements, data-modifying CTEs, recursively nested CTEs, and stable visit order.

## 2. Public Validation and Rule Contracts

- [x] 2.1 Define the driver-independent `Validator`, immutable `Engine`, `Rule`, and narrow rule-result contracts in the root `sqlguard` package.
- [x] 2.2 Implement the library-owned read-only `Statement` facade over parser-neutral nodes, including stable statement kinds, traversal, and typed access to named child fields without exposing SQL text or parser protobuf values.
- [x] 2.3 Document all exported contracts, including explicit registration, statement lifetime and immutability, and the requirement that rule instances support concurrent invocation.
- [x] 2.4 Add external-package compile-time contract tests showing that custom rules depend only on the public API and require no parser-backend imports or engine modifications.

## 3. Typed and Privacy-Safe Errors

- [x] 3.1 Implement public `Violation` and `ParseError` types with bounded metadata, constant safe messages, accessors for rule identifiers or parser-failure categories, and pointer compatibility with `errors.As`.
- [x] 3.2 Translate backend parser errors at the internal boundary without retaining or unwrapping raw diagnostics, SQL fragments, tokens, literals, comments, or credentials.
- [x] 3.3 Add error contract tests that distinguish parser failures from violations, preserve inspectability after application wrapping, and verify that values, messages, and unwrap chains omit unique input sentinels.

## 4. Engine Construction and Evaluation

- [x] 4.1 Implement engine construction with a copied ordered rule collection and reject nil rules, empty or invalid identifiers, and duplicate identifiers without returning a partially usable engine.
- [x] 4.2 Add constructor tests proving that only explicitly supplied rules are active, separate engines remain independent, and invalid or duplicate identifiers fail with privacy-safe errors.
- [x] 4.3 Implement `Engine.Validate` to parse the complete input once before rule execution, visit every discovered statement in deterministic order, evaluate rules in registration order, and stop at the first violation.
- [x] 4.4 Add validation tests for driver-free success, malformed-batch fail-closed behavior before any rule runs, all top-level and nested-CTE coverage, consistent parsed representation across custom rules, registration order, reproducible first violations, and suppression of later evaluations.
- [x] 4.5 Add context checks before and after parsing and before each rule call, pass the exact caller context to rules, and return standard context errors directly.
- [x] 4.6 Add context tests for value propagation, pre-canceled calls, cancellation during rule traversal, and `errors.Is` compatibility.
- [x] 4.7 Add concurrent engine tests with concurrency-safe rule spies to prove per-call statement isolation and race-free reuse of one initialized engine.

## 5. Robustness, Benchmarks, and Documentation

- [x] 5.1 Add seeded fuzz tests for valid and malformed SQL that assert validation does not panic and public errors never disclose submitted input sentinels.
- [x] 5.2 Add allocation-reporting benchmarks for representative single-statement, multi-statement, and nested-CTE validation with a no-op rule and no validation cache.
- [x] 5.3 Update the package documentation and README with a minimal API example, CGO build requirement, PostgreSQL 17 grammar compatibility, parser cancellation limitation, deterministic traversal order, error privacy guarantees, and rule concurrency contract.
