## Why

The project now has a reproducible Go development baseline but does not yet provide the driver-independent validation boundary at the center of the product. This change establishes that core so later built-in policies, configuration, observability, and driver integrations can depend on one PostgreSQL-accurate and fail-closed contract.

## What Changes

- Introduce a driver-independent `Validator` contract and an `Engine` constructed with an explicit set of `Rule` implementations.
- Parse validation input with a real PostgreSQL grammar rather than text matching, and evaluate every statement plus relevant statements nested inside CTEs so a mutation cannot bypass registered rules.
- Fail closed when parsing fails, when input is malformed, or when a registered rule reports a violation.
- Add typed, privacy-safe policy violations and parser failures that remain discoverable through Go's standard error inspection mechanisms.
- Preserve caller context behavior and make the validator safe for concurrent use.
- Add representative correctness, concurrency, malformed-input, multi-statement, nested-CTE, and baseline benchmark coverage.

## Non-Goals

- Provide built-in `UPDATE` or `DELETE` policies, YAML configuration, audit mode, observability adapters, or database-driver integrations.
- Add a no-CGO parser backend, validation cache, or performance optimization beyond establishing benchmark baselines.
- Load custom rule implementations dynamically or through global registration.

## Capabilities

### New Capabilities

- `validation-engine`: Driver-independent validation orchestration, PostgreSQL parsing, complete statement traversal, context propagation, concurrency safety, and benchmark baselines.
- `validation-errors`: Typed and privacy-safe policy violations and parser failures with standard Go error inspection behavior.
- `rule-extension`: Explicit rule registration and a stable rule contract that can evaluate the engine's consistent parsed representation without global discovery.

### Modified Capabilities

None. No existing capability requirements are changed.

## Impact

This change introduces the first public validation API in the root Go package, internal parser and traversal implementation, and an external PostgreSQL parser dependency that may require CGO. It adds core tests and benchmarks and becomes a dependency for future built-in-rule, configuration, observability, and pgx-integration changes. It does not contact PostgreSQL or add a dependency on pgx or another database driver.
