# observability Specification

## Purpose

Defines privacy-safe, bounded, and replaceable validation metrics and log events that the engine emits without changing validation behavior or enforce-mode safety.

## Requirements

### Requirement: Replaceable observability implementations
The public engine constructor SHALL have the signature `NewEngine(options EngineOptions, rules ...Rule)`. `EngineOptions` SHALL contain only independently optional `Metrics` and `Logger` implementations; backend-specific settings SHALL remain owned by their integration packages. A nil implementation SHALL behave as a no-op, and replacing or disabling either implementation MUST NOT change validation decisions or returned validation errors.

#### Scenario: Observability is disabled by default
- **WHEN** a caller constructs an engine with `EngineOptions{}`
- **THEN** validation operates without emitting metrics or log events

#### Scenario: Existing callers migrate explicitly
- **WHEN** a caller does not need observability after the constructor signature changes
- **THEN** the caller can preserve disabled behavior by passing `EngineOptions{}` before its registered rules

#### Scenario: Custom implementations are configured independently
- **WHEN** a caller configures a custom metrics implementation and disables logging, or configures a custom logger and disables metrics
- **THEN** the engine emits only through the configured implementation and preserves the validation result

#### Scenario: Configured engine is used concurrently
- **WHEN** multiple goroutines validate through the same engine with concurrency-safe observability implementations
- **THEN** observability emission does not mutate engine configuration or introduce an engine data race

### Requirement: Engine-owned terminal outcome emission
For every validation call that reaches a terminal outcome, the engine SHALL notify each enabled observability implementation exactly once. Consumers MUST NOT need to inspect returned errors or reproduce engine counters to obtain these outcome signals.

#### Scenario: Successful validation is emitted
- **WHEN** parsing and all registered rule evaluations succeed
- **THEN** each enabled implementation receives exactly one `allowed` outcome

#### Scenario: Policy violation is emitted
- **WHEN** a registered rule rejects a statement
- **THEN** each enabled implementation receives exactly one `policy_violation` outcome with that rule's stable identifier

#### Scenario: Parser failure is emitted
- **WHEN** the PostgreSQL parser rejects the complete input
- **THEN** each enabled implementation receives exactly one `parser_failure` outcome

#### Scenario: Cancellation is emitted
- **WHEN** validation terminates because the caller context is canceled or its deadline expires
- **THEN** each enabled implementation receives exactly one `canceled` outcome and the caller still receives the corresponding context error

### Requirement: Stable bounded event dimensions
Every engine-provided observability event SHALL contain only the dimensions `mode`, `outcome`, and `rule_id`. For this change, `mode` SHALL be `enforce`; `outcome` SHALL be one of `allowed`, `policy_violation`, `parser_failure`, or `canceled`; and `rule_id` SHALL be the rejecting rule's construction-time identifier for `policy_violation` and the empty string for every other outcome. The engine MUST NOT derive dimension names or values from SQL or other per-request application data.

#### Scenario: Non-violation leaves the rule identifier empty
- **WHEN** the terminal outcome is `allowed`, `parser_failure`, or `canceled`
- **THEN** the emitted event has `rule_id=""`

#### Scenario: Violation cardinality is bounded by engine rules
- **WHEN** the terminal outcome is `policy_violation`
- **THEN** `rule_id` is selected from the finite set of identifiers accepted when that engine was constructed

#### Scenario: Audit vocabulary is not emitted
- **WHEN** this change is used before an audit-mode capability is implemented
- **THEN** the engine emits only `mode="enforce"` and does not imply audit execution behavior

### Requirement: Official Prometheus integration
The project SHALL provide an official Prometheus integration that records terminal outcomes in a counter named `sqlguard_validations_total` by default and allows the caller to override that name with another valid Prometheus metric name. The counter SHALL have exactly the labels `service`, `mode`, `outcome`, and `rule_id`. The caller SHALL provide a non-empty `service` value once when constructing the integration; that immutable value SHALL be used for every observation from that integration and MUST NOT be derived from validation input or request context. The integration SHALL register with a caller-supplied Prometheus registerer and MUST NOT implicitly register collectors in a process-global registry.

#### Scenario: Prometheus counter is incremented
- **WHEN** an engine using the official Prometheus integration completes validation
- **THEN** the matching counter series is incremented exactly once using the configured `service` and the event's bounded dimensions

#### Scenario: Default metric name is used
- **WHEN** a caller constructs the Prometheus integration without overriding the metric name
- **THEN** the integration registers the counter as `sqlguard_validations_total`

#### Scenario: Metric name is overridden
- **WHEN** a caller constructs the Prometheus integration with a valid custom metric name
- **THEN** the integration registers and updates the counter under that custom name instead of `sqlguard_validations_total`

#### Scenario: Service is fixed at construction
- **WHEN** an integration configured with a non-empty `service` records multiple validation outcomes
- **THEN** every resulting series uses that configured `service` value without accepting a per-validation replacement

#### Scenario: Registration conflict is reported during configuration
- **WHEN** the official integration cannot register its collector with the supplied registerer
- **THEN** construction of that integration fails before it is attached to an engine

#### Scenario: Independent registries do not share state
- **WHEN** separate integrations are constructed with separate Prometheus registries
- **THEN** each registry contains only the outcomes emitted through its own integration

### Requirement: Official log/slog integration
The project SHALL provide an official `log/slog` integration that emits one record with the stable message `sqlguard validation` for each terminal event. The record SHALL contain exactly the attributes `mode`, `outcome`, and `rule_id`; `allowed` SHALL use INFO level, `policy_violation` and `parser_failure` SHALL use ERROR level, and `canceled` SHALL use DEBUG level.

#### Scenario: Successful validation is logged
- **WHEN** an engine using the official `log/slog` integration produces an `allowed` event
- **THEN** the logger emits one INFO record named `sqlguard validation` with the event's three bounded attributes

#### Scenario: Rejected validation is logged
- **WHEN** the engine produces a `policy_violation` or `parser_failure` event
- **THEN** the logger emits one ERROR record named `sqlguard validation` without embedding the returned error or parser diagnostic

#### Scenario: Cancellation is logged at debug level
- **WHEN** the engine produces a `canceled` event
- **THEN** the logger emits one DEBUG record named `sqlguard validation` with `rule_id=""`

### Requirement: Privacy-safe observability boundary
The engine and official integrations MUST NOT include or pass SQL text, SQL arguments, literal values, comments, tokens, credentials, secrets, raw parser diagnostics, returned errors, or arbitrary caller-context values in observability events. Custom implementations SHALL receive only the stable bounded event dimensions defined by this capability.

#### Scenario: Sensitive rejected input is not observable
- **WHEN** rejected or malformed input contains unique SQL text, a literal, token, credential, or secret
- **THEN** no engine-provided metric label, log record, or custom observability event exposes that value

#### Scenario: Caller context is not exposed
- **WHEN** the validation context contains request-scoped values
- **THEN** custom and official observability implementations receive none of those values from the engine

### Requirement: Runtime observability failure isolation
An error or panic produced while an enabled observability implementation processes an event MUST NOT change the validation result, weaken an enforce-mode rejection, escape from validation, or prevent the other enabled implementation from receiving the same event. The engine SHALL contain each failure independently, SHALL NOT retry the failed emission, and SHALL NOT recursively emit another observability event about that failure.

#### Scenario: Metrics failure preserves an allowed result
- **WHEN** metrics emission returns an error or panics after validation succeeds
- **THEN** validation still succeeds and the enabled logger is invoked once

#### Scenario: Logger failure preserves a policy violation
- **WHEN** logging returns an error or panics while reporting a policy violation
- **THEN** the caller still receives the original typed violation and the enabled metrics implementation is invoked once

#### Scenario: Both implementations fail
- **WHEN** both enabled implementations return errors or panic for the same terminal event
- **THEN** neither failure escapes validation or replaces its terminal result, and neither implementation is retried

### Requirement: Configuration failures are reported eagerly
Invalid observability configuration and failures that can be detected while constructing an engine or official integration SHALL be returned synchronously before validation can begin. Such errors MUST be privacy-safe and MUST NOT contain runtime SQL or request data.

#### Scenario: Invalid implementation is rejected
- **WHEN** a caller supplies an invalid observability implementation during construction
- **THEN** construction fails without returning a partially configured engine

#### Scenario: Invalid Prometheus identity is rejected
- **WHEN** a caller supplies an empty `service` or an invalid custom metric name to the official Prometheus integration
- **THEN** integration construction fails before registering a collector

#### Scenario: Valid disabled configuration succeeds
- **WHEN** metrics, logging, or both are explicitly disabled
- **THEN** construction succeeds and the disabled implementation requires no runtime initialization
