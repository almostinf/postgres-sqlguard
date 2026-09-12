## ADDED Requirements

### Requirement: Reusable prepared validation
The engine SHALL allow a caller to parse complete SQL into an opaque immutable
prepared value and SHALL allow that value to be validated repeatedly without
reparsing it. Preparation MUST NOT evaluate registered rules, and every
prepared validation MUST evaluate the current engine's registered rules
without reusing an earlier allow or reject decision. A successfully prepared
value SHALL be usable with any engine and MUST NOT expose parser-backend types,
SQL text, or mutable parsed state.

#### Scenario: Successful preparation defers rule evaluation
- **WHEN** a caller prepares syntactically valid SQL
- **THEN** preparation returns a prepared value without invoking any registered rule

#### Scenario: Prepared validation evaluates rules on every call
- **WHEN** a caller validates the same prepared value more than once
- **THEN** every call evaluates the registered rules using that call's context and current rule state

#### Scenario: Prepared value is engine-independent
- **WHEN** a prepared value created by one engine is passed to another engine
- **THEN** the receiving engine validates it using only the receiving engine's registered rules and observability configuration

#### Scenario: Existing validator contract remains compatible
- **WHEN** a consumer implements or accepts the existing `Validator` interface
- **THEN** the interface continues to require only `Validate` and existing implementations remain source compatible

#### Scenario: Direct validation preserves behavior
- **WHEN** a caller uses `Validate` instead of the prepared API
- **THEN** the engine prepares and validates the input with the same returned result, traversal, context, and single terminal outcome as before

## MODIFIED Requirements

### Requirement: PostgreSQL grammar parsing
The engine SHALL parse the complete validation input using a PostgreSQL grammar
before rule evaluation. `Prepare` SHALL perform this parsing exactly once for
a successfully returned prepared value, and `ValidatePrepared` MUST NOT parse
that value again. Safety decisions MUST use the parsed structure rather than
regular expressions, keyword searches, or other textual approximations.

#### Scenario: PostgreSQL-specific syntax is parsed structurally
- **WHEN** a caller validates syntactically valid PostgreSQL-specific SQL that is accepted by the supported parser grammar
- **THEN** the engine evaluates its parsed statement structure instead of rejecting it because it is not generic SQL

#### Scenario: Keywords in non-syntax text do not create statements
- **WHEN** SQL keywords appear only inside a string literal or comment in otherwise valid input
- **THEN** the engine does not treat those keywords as statement syntax

#### Scenario: Any malformed statement fails the complete input
- **WHEN** a multi-statement input contains at least one statement that the PostgreSQL parser cannot parse
- **THEN** validation fails closed with a typed parser failure before any statement is reported as accepted

#### Scenario: Malformed input is not prepared
- **WHEN** `Prepare` receives input rejected by the PostgreSQL parser
- **THEN** it returns a typed parser failure and no prepared value usable by `ValidatePrepared`

#### Scenario: Prepared validation does not reparse
- **WHEN** a caller repeatedly validates one successfully prepared value
- **THEN** no prepared validation call invokes the PostgreSQL parser again

### Requirement: Complete statement traversal
The engine SHALL evaluate every top-level statement in input order and SHALL
include statement-bearing CTE entries at every nesting depth in its rule
traversal for both direct and prepared validation. A registered rule MUST NOT
be bypassed by placing a mutation inside a CTE or after another statement in
the same input.

#### Scenario: Every top-level statement is evaluated
- **WHEN** valid input contains multiple top-level statements
- **THEN** each statement is presented to the registered rules unless validation has already stopped on a violation or caller cancellation

#### Scenario: Mutation nested in a CTE is evaluated
- **WHEN** a data-modifying statement appears inside a CTE of another statement
- **THEN** the nested statement is presented to the same registered rules as a top-level statement

#### Scenario: Recursively nested statement-bearing CTEs are evaluated
- **WHEN** statement-bearing CTEs occur at more than one nesting depth
- **THEN** the engine traverses every nested statement and no nesting level bypasses rule evaluation

#### Scenario: Prepared traversal matches direct traversal
- **WHEN** the same multi-statement or nested-CTE input is validated directly and through a prepared value
- **THEN** registered rules observe the same deterministic statement order in both paths

### Requirement: Caller context preservation
The validator SHALL accept a caller-provided context in direct validation,
preparation, and prepared validation; preserve its values when invoking rules;
and stop the active operation when that context is canceled or its deadline
expires. The resulting error MUST remain identifiable with Go's standard
context error values. Cancellation detected after synchronous parsing MUST
take precedence over a parser failure, and an already-canceled prepared
validation MUST take precedence over an invalid prepared value.

#### Scenario: Rule receives caller context values
- **WHEN** a caller validates SQL with a context containing a request-scoped value
- **THEN** each invoked rule can observe the same value from the context supplied to that validation call

#### Scenario: Canceled validation stops
- **WHEN** the caller context is canceled before or during direct or prepared validation
- **THEN** validation stops and the caller can identify the corresponding context cancellation error with `errors.Is`

#### Scenario: Canceled preparation stops
- **WHEN** the caller context is canceled before parsing or is observed as canceled after synchronous parsing
- **THEN** preparation returns the identifiable context error and does not return a usable prepared value

#### Scenario: Cancellation wins after malformed parsing
- **WHEN** parsing reports malformed SQL and the context is observed as canceled immediately after parsing
- **THEN** preparation returns the identifiable context error rather than the parser failure

#### Scenario: Cancellation wins for invalid prepared input
- **WHEN** `ValidatePrepared` receives both an already-canceled context and an invalid prepared value
- **THEN** it returns the identifiable context error

### Requirement: Concurrent validator use
An initialized validator SHALL support concurrent direct, preparation, and
prepared-validation calls without races or mutation of shared parsed state,
provided registered rules satisfy the documented rule concurrency contract. A
successfully prepared value SHALL support concurrent validation by multiple
engines under the same rule constraint.

#### Scenario: Concurrent calls are race-free
- **WHEN** multiple goroutines use the same validator concurrently with concurrency-safe rules
- **THEN** all calls produce results corresponding to their own inputs and the Go race detector reports no engine race

#### Scenario: Prepared value is shared concurrently
- **WHEN** multiple goroutines validate the same prepared value through one or more engines with concurrency-safe rules
- **THEN** each call receives its own rule result and no parsed state is mutated or raced

### Requirement: Validation benchmark baseline
The repository SHALL provide runnable validation benchmarks that report
absolute latency and allocations for ordinary `Validate` and direct
`ValidatePrepared` calls without adding a validation cache to the core
library. Preparation for a direct prepared-validation benchmark MUST occur
outside its timed loop. The benchmark suite SHALL continue to isolate SQL
complexity, validation outcome, and observability configuration as separate
measurement axes rather than combining them into a cross-product. It MUST NOT
include a synthetic comparison path that bypasses SQLGuard.

#### Scenario: Core benchmarks are runnable
- **WHEN** a maintainer runs the documented Go benchmark command
- **THEN** benchmarks execute for the representative validation paths and report standard Go benchmark measurements

#### Scenario: SQL complexity is measured independently
- **WHEN** a maintainer runs `BenchmarkEngineValidateByComplexity`
- **THEN** the benchmark reports separate `simple`, `medium`, `multi_statement`, and `nested_cte` results with the validation outcome and observability configuration held constant

#### Scenario: Prepared validation is measured independently
- **WHEN** a maintainer runs `BenchmarkEngineValidatePreparedByComplexity`
- **THEN** the benchmark reports the same SQL-complexity cases with preparation excluded from the timed loop and all rules still evaluated inside it

#### Scenario: Validation outcomes are measured independently
- **WHEN** a maintainer runs `BenchmarkEngineValidateByOutcome`
- **THEN** the benchmark reports separate `allowed`, `policy_violation`, and `parser_failure` results with SQL complexity and observability configuration held constant

#### Scenario: Observability overhead is measured independently
- **WHEN** a maintainer runs `BenchmarkEngineObservability`
- **THEN** the benchmark reports separate `disabled`, `metrics`, `logger`, and `metrics_and_logger` results using the same SQL input and validation outcome

#### Scenario: Results describe validation cost
- **WHEN** benchmark results are documented or presented as a latency comparison
- **THEN** they report absolute SQLGuard validation measurements and identify the complexity comparison as "Validation latency by SQL complexity" without presenting a synthetic "without SQLGuard" baseline
