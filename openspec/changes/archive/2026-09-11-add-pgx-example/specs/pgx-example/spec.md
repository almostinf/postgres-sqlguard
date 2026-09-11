## Purpose

Defines a compiling pgx integration example and its explicitly limited safety
boundary without establishing a production adapter commitment.

## ADDED Requirements

### Requirement: Illustrative pgx integration scope
The repository SHALL provide a compiling `example/pgx` integration that is
identified as an illustrative example rather than an official, stable, or
production-ready pgx adapter. The core validation packages MUST remain usable
without importing a pgx API.

#### Scenario: Consumer finds the example scope
- **WHEN** a consumer reads the pgx example and its documentation
- **THEN** the example identifies its supported operations and states that its API has no production compatibility commitment

#### Scenario: Core remains driver-independent
- **WHEN** a consumer imports and uses the core validation package
- **THEN** the consumer is not required to use a pgx type or database connection

### Requirement: Validation precedes supported pgx operations
The example wrapper SHALL support the pgx-style `Exec`, `Query`, and
`QueryRow` operations. It MUST complete SQLGuard validation before delegating
any supported operation to the wrapped executor.

#### Scenario: Allowed Exec delegates after validation
- **WHEN** `Exec` receives SQL that the configured validator accepts
- **THEN** the wrapper delegates the execution exactly once after validation succeeds

#### Scenario: Allowed Query delegates after validation
- **WHEN** `Query` receives SQL that the configured validator accepts
- **THEN** the wrapper delegates the query exactly once after validation succeeds

#### Scenario: Allowed QueryRow delegates after validation
- **WHEN** `QueryRow` receives SQL that the configured validator accepts
- **THEN** the wrapper delegates the row query exactly once after validation succeeds

### Requirement: Query rewriting is rejected before validation
The example wrapper MUST reject any argument that implements pgx
`QueryRewriter` before invoking either the configured validator or the wrapped
executor. This includes pgx named- and struct-argument helpers implemented
through `QueryRewriter`. The returned error MUST be privacy-safe and
discoverable with `errors.Is` against a stable example-package error.

#### Scenario: Exec rejects a query rewriter
- **WHEN** `Exec` receives any argument that implements pgx `QueryRewriter`
- **THEN** it returns the unsupported-rewriter error without invoking the validator or executor

#### Scenario: Query rejects a query rewriter
- **WHEN** `Query` receives any argument that implements pgx `QueryRewriter`
- **THEN** it returns the unsupported-rewriter error without invoking the validator or executor

#### Scenario: QueryRow defers a rejected query rewriter
- **WHEN** `QueryRow` receives any argument that implements pgx `QueryRewriter`
- **THEN** it invokes neither the validator nor executor and its returned row exposes the unsupported-rewriter error through `Scan`

### Requirement: Validation failure prevents delegation
If SQLGuard returns a policy violation, parser failure, or caller-context
error, the example wrapper MUST NOT invoke the wrapped executor. The failure
MUST remain discoverable through Go's standard `errors.Is` or `errors.As`
mechanisms as applicable.

#### Scenario: Policy violation blocks execution
- **WHEN** a supported operation receives SQL rejected by a configured rule
- **THEN** the wrapped executor is not called and the caller can discover the policy violation with `errors.As`

#### Scenario: Parser failure blocks execution
- **WHEN** a supported operation receives SQL rejected by the PostgreSQL parser
- **THEN** the wrapped executor is not called and the caller can discover the parser failure with `errors.As`

#### Scenario: Canceled context blocks execution
- **WHEN** validation returns a caller-context cancellation or deadline error
- **THEN** the wrapped executor is not called and the caller can discover the context error with `errors.Is`

### Requirement: Allowed calls preserve pgx inputs and outcomes
For calls without a pgx `QueryRewriter`, after successful validation the
example wrapper SHALL pass the exact caller context, SQL string, and ordered
argument values to the wrapped executor without modification. It SHALL
preserve the executor's operation result and error behavior.

#### Scenario: Inputs pass through unchanged
- **WHEN** validation accepts a supported operation with a context, SQL string, and arguments
- **THEN** the wrapped executor receives that same context, SQL string, and ordered arguments

#### Scenario: Exec outcome passes through
- **WHEN** the wrapped executor returns an `Exec` command tag or driver error after successful validation
- **THEN** the caller receives that command tag and can inspect that driver error through its original error chain

#### Scenario: Query outcome passes through
- **WHEN** the wrapped executor returns rows or a driver error from `Query` after successful validation
- **THEN** the caller receives those rows and can inspect that driver error through its original error chain

### Requirement: QueryRow exposes deferred validation failures
Because pgx-style `QueryRow` does not return an immediate error, the example
wrapper SHALL expose a validation failure from the returned row's `Scan`
operation. A rejected `QueryRow` MUST NOT invoke the wrapped executor.

#### Scenario: Rejected QueryRow fails on Scan
- **WHEN** `QueryRow` validation fails and the caller invokes `Scan` on the returned row
- **THEN** `Scan` returns an error through which the original validation failure remains discoverable

#### Scenario: Accepted QueryRow preserves row behavior
- **WHEN** `QueryRow` validation succeeds
- **THEN** the wrapper returns the delegated row and its `Scan` behavior is preserved

### Requirement: Unsupported and bypassable paths are explicit
The pgx example documentation MUST state that batch execution, prepared
statement workflows, transactions, nested transactions, `CopyFrom`, and use
of an unwrapped connection are outside the example's guarded boundary. It MUST
NOT imply that these paths have been validated.

#### Scenario: Consumer reviews the safety boundary
- **WHEN** a consumer reads the pgx example documentation
- **THEN** every unsupported or bypassable path is listed as not protected by the example wrapper
