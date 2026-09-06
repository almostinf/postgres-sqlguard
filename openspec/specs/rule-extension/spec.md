# rule-extension Specification

## Purpose

Defines explicit and deterministic registration and execution of application-provided validation rules over a consistent parsed PostgreSQL representation.

## Requirements

### Requirement: Explicit rule registration
An engine SHALL be constructed from an explicit ordered collection of rules. The library MUST NOT discover rules through global registration, package initialization, environment state, or implicit default policies.

#### Scenario: Only supplied rules are active
- **WHEN** an engine is constructed with a specific collection of rules
- **THEN** validation invokes only those rules and does not silently add another policy

#### Scenario: Separate engines remain independent
- **WHEN** two engines are constructed with different rule collections in the same process
- **THEN** each engine invokes only its own registered rules

### Requirement: Stable and unambiguous rule identifiers
Each rule SHALL expose a stable, non-empty identifier suitable for programmatic error handling. Engine construction MUST reject registrations with an empty identifier or duplicate identifiers rather than creating an ambiguous policy set.

#### Scenario: Empty rule identifier is rejected
- **WHEN** a caller attempts to construct an engine with a rule whose identifier is empty
- **THEN** construction fails with an error that does not create a usable partially configured engine

#### Scenario: Duplicate rule identifier is rejected
- **WHEN** a caller attempts to construct an engine with two rules that expose the same identifier
- **THEN** construction fails with an error that identifies the duplicate identifier without including SQL data

### Requirement: Consistent parsed representation
Rules SHALL evaluate a structured statement representation produced by the engine's PostgreSQL parser. Within one validation call, every rule evaluating the same statement MUST receive a consistent interpretation of that statement, and adding a custom rule MUST NOT require changes to the parser or engine.

#### Scenario: Rules share one statement interpretation
- **WHEN** multiple registered rules evaluate the same parsed statement during one validation call
- **THEN** they observe equivalent statement structure derived from the same successfully parsed input

#### Scenario: Custom rule is added without engine modification
- **WHEN** an application implements the public rule contract and registers the rule explicitly
- **THEN** the engine can execute it without changes to engine or parser implementation

### Requirement: Deterministic rule evaluation
The engine SHALL use a documented deterministic order for statement traversal and SHALL evaluate rules in registration order for each visited statement. Repeated validation of the same input with the same rules MUST select the same first violation.

#### Scenario: Rules run in registration order
- **WHEN** multiple rules are registered and more than one would reject the same visited statement
- **THEN** the violation from the earliest rejecting rule in registration order is returned

#### Scenario: First violation is reproducible
- **WHEN** the same SQL and ordered rule collection are validated repeatedly
- **THEN** the same statement-rule pair determines the first violation each time

### Requirement: Rule concurrency contract
The public rule contract SHALL state that one rule instance can be invoked concurrently by separate validation calls. The engine MUST NOT introduce races when invoking concurrency-safe rules and MUST NOT mutate rule-owned state outside the rule contract.

#### Scenario: Shared concurrency-safe rule instance
- **WHEN** concurrent calls use one engine containing a concurrency-safe rule instance
- **THEN** rule evaluation remains race-free and each call receives its own context and statement input

