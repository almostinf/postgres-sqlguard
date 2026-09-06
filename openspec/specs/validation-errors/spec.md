# validation-errors Specification

## Purpose

Defines typed, inspectable, and privacy-safe failures for policy violations and PostgreSQL parser errors returned by the validation boundary.

## Requirements

### Requirement: Typed policy violation
When a rule rejects a statement, validation SHALL return a typed violation that identifies the responsible rule by its stable identifier. Callers MUST be able to discover the violation with Go's standard `errors.As` mechanism without parsing error text.

#### Scenario: Caller inspects a violation
- **WHEN** a registered rule rejects a parsed statement
- **THEN** validation returns an error discoverable as the public violation type and exposes the rejecting rule's identifier

#### Scenario: Wrapped violation remains inspectable
- **WHEN** a validation violation is wrapped with additional application context using standard Go error wrapping
- **THEN** `errors.As` still discovers the underlying violation

### Requirement: Typed parser failure
When the supported PostgreSQL parser rejects validation input, validation SHALL return a typed parser failure distinct from a policy violation. Callers MUST be able to discover the parser failure with `errors.As` without parsing error text.

#### Scenario: Caller distinguishes malformed SQL
- **WHEN** validation input is not accepted by the supported PostgreSQL grammar
- **THEN** the returned error is discoverable as a parser failure and is not discoverable as a policy violation

#### Scenario: Malformed input does not panic
- **WHEN** the validator receives empty, truncated, or otherwise malformed SQL input
- **THEN** it returns either the documented successful empty-input result or a typed parser failure, according to the supported grammar, and does not panic

### Requirement: Privacy-safe validation errors
Violation and parser-failure values, their error strings, and their unwrap chains MUST NOT contain or expose the submitted SQL text, SQL literals, comments, tokens, credentials, or secrets. Error metadata SHALL be limited to bounded structural information needed for programmatic handling, including stable rule identifiers and failure categories.

#### Scenario: Violation omits sensitive input
- **WHEN** rejected SQL contains a unique literal, token, or credential-like value
- **THEN** neither the violation value nor any error reachable through its unwrap chain exposes that value or the submitted SQL

#### Scenario: Parser failure omits sensitive input
- **WHEN** malformed SQL contains a unique literal, token, or credential-like value
- **THEN** neither the parser-failure value nor any error reachable through its unwrap chain exposes that value or the submitted SQL

### Requirement: First violation result
Validation SHALL stop rule evaluation at the first violation according to the documented deterministic statement and rule traversal order and SHALL return that violation unchanged as the validation outcome.

#### Scenario: Later violations are not evaluated
- **WHEN** a rule reports a violation before later statement-rule pairs in the traversal order
- **THEN** validation returns that first violation and does not evaluate the later pairs

