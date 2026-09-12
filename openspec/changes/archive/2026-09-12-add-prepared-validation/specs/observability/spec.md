## ADDED Requirements

### Requirement: Prepared API outcome emission
Preparation and prepared validation SHALL preserve engine-owned terminal
outcome emission without double counting a logical validation. Successful
preparation MUST emit no terminal event because rules have not yet produced a
policy decision. A preparation failure or a prepared-validation result SHALL
notify each enabled implementation exactly once, and `Validate` composed from
the two phases MUST still notify each enabled implementation exactly once.

#### Scenario: Successful preparation emits no terminal outcome
- **WHEN** `Prepare` successfully parses complete SQL
- **THEN** neither enabled observability implementation is notified

#### Scenario: Preparation parser failure is emitted
- **WHEN** `Prepare` receives SQL rejected by the PostgreSQL parser
- **THEN** each enabled implementation receives exactly one `parser_failure` outcome and the caller does not need to invoke `ValidatePrepared`

#### Scenario: Preparation cancellation is emitted
- **WHEN** `Prepare` terminates because its caller context is canceled or its deadline expires
- **THEN** each enabled implementation receives exactly one `canceled` outcome

#### Scenario: Prepared validation emits its decision
- **WHEN** `ValidatePrepared` allows input, encounters a policy violation, or observes cancellation
- **THEN** each enabled implementation receives exactly one corresponding terminal outcome

#### Scenario: Invalid prepared value is emitted
- **WHEN** `ValidatePrepared` receives an invalid prepared value with an active context
- **THEN** each enabled implementation receives exactly one `invalid_prepared` outcome with `rule_id=""`

#### Scenario: Direct validation is not double counted
- **WHEN** `Validate` completes through its preparation and prepared-validation phases
- **THEN** each enabled implementation receives exactly one terminal outcome for the call

#### Scenario: Prometheus records invalid prepared input
- **WHEN** an engine using the official Prometheus integration rejects an invalid prepared value
- **THEN** `sqlguard_validations_total` is incremented exactly once with `outcome="invalid_prepared"` and `rule_id=""`

## MODIFIED Requirements

### Requirement: Stable bounded event dimensions
Every engine-provided observability event SHALL contain only the dimensions
`mode`, `outcome`, and `rule_id`. For this change, `mode` SHALL be `enforce`;
`outcome` SHALL be one of `allowed`, `policy_violation`, `parser_failure`,
`canceled`, or `invalid_prepared`; and `rule_id` SHALL be the rejecting rule's
construction-time identifier for `policy_violation` and the empty string for
every other outcome. The engine MUST NOT derive dimension names or values from
SQL, prepared structure, or other per-request application data.

#### Scenario: Non-violation leaves the rule identifier empty
- **WHEN** the terminal outcome is `allowed`, `parser_failure`, `canceled`, or `invalid_prepared`
- **THEN** the emitted event has `rule_id=""`

#### Scenario: Violation cardinality is bounded by engine rules
- **WHEN** the terminal outcome is `policy_violation`
- **THEN** `rule_id` is selected from the finite set of identifiers accepted when that engine was constructed

#### Scenario: Audit vocabulary is not emitted
- **WHEN** this change is used before an audit-mode capability is implemented
- **THEN** the engine emits only `mode="enforce"` and does not imply audit execution behavior

### Requirement: Official log/slog integration
The project SHALL provide an official `log/slog` integration that emits one
record with the stable message `sqlguard validation` for each terminal event.
The record SHALL contain exactly the attributes `mode`, `outcome`, and
`rule_id`; `allowed` SHALL use INFO level; `policy_violation`,
`parser_failure`, and `invalid_prepared` SHALL use ERROR level; and `canceled`
SHALL use DEBUG level.

#### Scenario: Successful validation is logged
- **WHEN** an engine using the official `log/slog` integration produces an `allowed` event
- **THEN** the logger emits one INFO record named `sqlguard validation` with the event's three bounded attributes

#### Scenario: Rejected validation is logged
- **WHEN** the engine produces a `policy_violation`, `parser_failure`, or `invalid_prepared` event
- **THEN** the logger emits one ERROR record named `sqlguard validation` without embedding the returned error, SQL, parsed structure, or parser diagnostic

#### Scenario: Cancellation is logged at debug level
- **WHEN** the engine produces a `canceled` event
- **THEN** the logger emits one DEBUG record named `sqlguard validation` with `rule_id=""`

### Requirement: Runtime observability failure isolation
An error or panic produced while an enabled observability implementation
processes any direct, preparation, or prepared-validation event MUST NOT change
the operation result, weaken an enforce-mode rejection, escape from the
operation, or prevent the other enabled implementation from receiving the
same event. The engine SHALL contain each failure independently, SHALL NOT
retry the failed emission, and SHALL NOT recursively emit another
observability event about that failure.

#### Scenario: Metrics failure preserves an allowed result
- **WHEN** metrics emission returns an error or panics after validation succeeds
- **THEN** validation still succeeds and the enabled logger is invoked once

#### Scenario: Logger failure preserves a policy violation
- **WHEN** logging returns an error or panics while reporting a policy violation
- **THEN** the caller still receives the original typed violation and the enabled metrics implementation is invoked once

#### Scenario: Preparation sink failure preserves parser failure
- **WHEN** either sink returns an error or panics while `Prepare` reports a parser failure
- **THEN** the caller still receives the original typed parser failure and the other enabled sink is invoked once

#### Scenario: Prepared sink failure preserves invalid-prepared result
- **WHEN** either sink returns an error or panics while `ValidatePrepared` reports invalid prepared input
- **THEN** the caller still receives the invalid-prepared error and the other enabled sink is invoked once

#### Scenario: Both implementations fail
- **WHEN** both enabled implementations return errors or panic for the same terminal event
- **THEN** neither failure escapes validation or preparation or replaces its terminal result, and neither implementation is retried
