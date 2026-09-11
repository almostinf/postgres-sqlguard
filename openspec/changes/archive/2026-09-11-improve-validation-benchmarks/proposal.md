## Why

The current validation benchmark crosses every SQL input with every
observability configuration, obscuring the independent costs of SQL complexity,
validation outcome, and observability. A focused benchmark suite will make
performance results easier to interpret and compare over time.

## What Changes

- Replace the cross-product benchmark with three independent benchmark groups:
  validation by SQL complexity, validation by outcome, and observability
  overhead.
- Cover simple, medium, multi-statement, and nested-CTE inputs in the complexity
  group.
- Cover allowed validation, policy violations, and parser failures in the
  outcome group.
- Cover disabled, metrics-only, logger-only, and combined metrics-and-logger
  configurations in the observability group.
- Report absolute `Validate` latency and allocations as SQLGuard's incremental
  cost, with performance material labeled "Validation latency by SQL
  complexity."
- Remove comparison against a synthetic "without SQLGuard" baseline; consumers
  can compare validation cost with their own database latency.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `validation-engine`: Expand the benchmark baseline requirement so runnable
  benchmarks isolate SQL complexity, validation outcome, and observability as
  separate measurement axes.

## Non-Goals

- Changing validation behavior, public APIs, parser selection, or rule
  semantics.
- Introducing a validation cache or claiming an end-to-end database latency
  comparison.

## Impact

- Restructures `validator_benchmark_test.go` and its benchmark names and cases.
- Updates benchmark documentation and any checked-in performance presentation
  material to describe absolute validation latency accurately.
- Modifies the `validation-engine` capability specification without changing
  runtime behavior, dependencies, or safety and privacy guarantees.
