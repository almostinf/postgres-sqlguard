## Purpose

Defines the driver-independent validation boundary, PostgreSQL-accurate parsing, complete statement coverage, context behavior, concurrency guarantees, and benchmark baseline.

## ADDED Requirements

### Requirement: Driver-independent validation API
The library SHALL expose a `Validator` contract and an `Engine` implementation that validate SQL without requiring a database connection or importing a database-driver API. A validation that parses successfully and produces no rule violation SHALL succeed without contacting PostgreSQL.

#### Scenario: Valid statement is accepted without a driver
- **WHEN** a caller validates a syntactically valid PostgreSQL statement and every registered rule accepts it
- **THEN** validation succeeds without using a database connection or driver

### Requirement: PostgreSQL grammar parsing
The engine SHALL parse the complete validation input using a PostgreSQL grammar before rule evaluation. Safety decisions MUST use the parsed structure rather than regular expressions, keyword searches, or other textual approximations.

#### Scenario: PostgreSQL-specific syntax is parsed structurally
- **WHEN** a caller validates syntactically valid PostgreSQL-specific SQL that is accepted by the supported parser grammar
- **THEN** the engine evaluates its parsed statement structure instead of rejecting it because it is not generic SQL

#### Scenario: Keywords in non-syntax text do not create statements
- **WHEN** SQL keywords appear only inside a string literal or comment in otherwise valid input
- **THEN** the engine does not treat those keywords as statement syntax

#### Scenario: Any malformed statement fails the complete input
- **WHEN** a multi-statement input contains at least one statement that the PostgreSQL parser cannot parse
- **THEN** validation fails closed with a typed parser failure before any statement is reported as accepted

### Requirement: Complete statement traversal
The engine SHALL evaluate every top-level statement in input order and SHALL include statement-bearing CTE entries at every nesting depth in its rule traversal. A registered rule MUST NOT be bypassed by placing a mutation inside a CTE or after another statement in the same input.

#### Scenario: Every top-level statement is evaluated
- **WHEN** valid input contains multiple top-level statements
- **THEN** each statement is presented to the registered rules unless validation has already stopped on a violation or caller cancellation

#### Scenario: Mutation nested in a CTE is evaluated
- **WHEN** a data-modifying statement appears inside a CTE of another statement
- **THEN** the nested statement is presented to the same registered rules as a top-level statement

#### Scenario: Recursively nested statement-bearing CTEs are evaluated
- **WHEN** statement-bearing CTEs occur at more than one nesting depth
- **THEN** the engine traverses every nested statement and no nesting level bypasses rule evaluation

### Requirement: Caller context preservation
The validator SHALL accept a caller-provided context, preserve its values when invoking rules, and stop validation when that context is canceled or its deadline expires. The resulting error MUST remain identifiable with Go's standard context error values.

#### Scenario: Rule receives caller context values
- **WHEN** a caller validates SQL with a context containing a request-scoped value
- **THEN** each invoked rule can observe the same value from the provided context

#### Scenario: Canceled validation stops
- **WHEN** the caller context is canceled before or during validation
- **THEN** validation stops and the caller can identify the corresponding context cancellation error with `errors.Is`

### Requirement: Concurrent validator use
An initialized validator SHALL support concurrent validation calls without races or mutation of shared parsed state, provided registered rules satisfy the documented rule concurrency contract.

#### Scenario: Concurrent calls are race-free
- **WHEN** multiple goroutines use the same validator concurrently with concurrency-safe rules
- **THEN** all calls produce results corresponding to their own inputs and the Go race detector reports no engine race

### Requirement: Validation benchmark baseline
The repository SHALL provide runnable benchmarks for representative single-statement, multi-statement, and nested-CTE validation paths without adding a validation cache.

#### Scenario: Core benchmarks are runnable
- **WHEN** a maintainer runs the documented Go benchmark command
- **THEN** benchmarks execute for the representative validation paths and report standard Go benchmark measurements
