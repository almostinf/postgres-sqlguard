## Why

Applications need consistent, privacy-safe visibility into validation outcomes without duplicating counters, interpreting returned errors, or weakening the engine's enforcement guarantees. Observability is part of the enforce MVP, so the engine and its supported integrations need an explicit public contract before implementation.

## What Changes

- Add replaceable metrics and logging abstractions that the validation engine updates for every documented validation outcome.
- **BREAKING** Change `NewEngine` from `NewEngine(rules ...Rule)` to `NewEngine(options EngineOptions, rules ...Rule)` so observability is configured through the primary constructor; existing callers must pass `EngineOptions{}` when no sinks are enabled.
- Allow consumers to use defaults, provide custom implementations, or disable metrics and logging independently without changing validation behavior.
- Provide official Prometheus metrics with a required caller-supplied service label and an overridable metric name, plus a `log/slog` logging integration.
- Require bounded metric labels and prohibit SQL text, SQL arguments, literal values, tokens, credentials, and secrets from observability data.
- Define how observability implementation failures are contained so they cannot weaken enforce-mode safety.
- Specify exact metric names, label sets, emitted events, and failure behavior in the capability spec and design.
- Keep audit-mode behavior out of scope until the separate audit capability exists; this change may reserve stable outcome vocabulary without introducing audit execution semantics.

## Capabilities

### New Capabilities

- `observability`: Defines engine-owned validation metrics and log events, configurable implementations, official Prometheus and `log/slog` integrations, bounded labels, privacy constraints, and failure isolation.

### Modified Capabilities

None.

## Impact

- Public engine construction and configuration APIs gain `EngineOptions` with `Metrics` and `Logger` fields and require migration of every existing `NewEngine` call.
- Engine validation paths emit outcome signals while preserving their existing return values, traversal behavior, context handling, and enforce-mode guarantees.
- New official integration packages expose Prometheus and `log/slog` adapters without adding driver dependencies to the core validator.
- The Prometheus integration introduces a Prometheus client dependency; the core remains independent of concrete observability backends.
- Tests, documentation, concurrency guarantees, and privacy verification expand to cover default, custom, disabled, and failing observability implementations.
