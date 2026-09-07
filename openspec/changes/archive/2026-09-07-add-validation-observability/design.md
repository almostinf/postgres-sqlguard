## Context

See [proposal.md](proposal.md) for motivation and [specs/observability/spec.md](specs/observability/spec.md) for the observable contract.

The root `sqlguard` package currently exposes `NewEngine(rules ...Rule)`, and an initialized `Engine` stores only validated rule registrations. `Validate` has several terminal return paths: pre-parse cancellation, post-parse cancellation, parser failure, cancellation during traversal, first policy violation, and success. The engine is immutable after construction and is safe for concurrent use when its collaborators satisfy their concurrency contracts.

The core currently has no observability dependency. The Prometheus client is present only indirectly through development tooling; the implementation will make the supported client package a direct dependency of the dedicated Prometheus adapter. `log/slog` is in the standard library. The privacy boundary prohibits forwarding SQL, errors, parser diagnostics, or caller-context values to observability implementations.

## Goals / Non-Goals

**Goals:**

- Add minimal, driver-independent `Metrics` and `Logger` contracts to the root package.
- Make observability configuration explicit in the primary `NewEngine` constructor, accepting a deliberate breaking change while the public API is still being established.
- Represent every terminal validation result with one immutable, bounded event.
- Isolate errors and panics from each runtime sink without changing the validation result or preventing the other sink from running.
- Keep Prometheus and `log/slog` dependencies outside the validation core in dedicated public adapter packages.
- Make adapter construction validate all static configuration before it can be attached to an engine.

**Non-Goals:**

- Audit-mode execution or audit-specific outcomes.
- Latency histograms, tracing, arbitrary tags, caller-context propagation, SQL-derived dimensions, or automatic process-global registration.
- Background delivery, buffering, retries, batching, or lifecycle management for custom sinks.
- Reporting a failed observability emission through a second observability channel.
- Changing existing parser, rule traversal, violation, or context-cancellation semantics.

## Decisions

### 1. Change the primary constructor to accept engine options

The root package will change the primary constructor and define its options as:

```go
type EngineOptions struct {
    Metrics Metrics
    Logger  Logger
}

func NewEngine(options EngineOptions, rules ...Rule) (*Engine, error)
```

`EngineOptions` contains only the two backend-neutral sink contracts. Prometheus-specific `Service`, `MetricName`, and `Registerer` values remain in the Prometheus adapter configuration, while a `*slog.Logger` is wrapped by the slog adapter before it is assigned to `EngineOptions.Logger`.

A nil interface means disabled, so `EngineOptions{}` is the explicit no-observability configuration. An interface containing a typed nil is invalid and causes construction to fail without returning an engine, matching the existing fail-early handling for typed-nil rules. The engine copies the resolved sink references into private fields and never mutates them.

Alternatives considered:

- Preserve `NewEngine(rules ...Rule)` and add `NewEngineWithOptions`. Two constructors would retain source compatibility but make the less capable path permanently prominent and duplicate construction entry points at this early API stage.
- Change `NewEngine` to accept mixed functional options and rules. Its existing variadic `Rule` signature cannot accept both without weakening compile-time types, and separate variadic lists are not available in Go.
- Add setters to `Engine`. Setters would violate engine immutability and complicate concurrent use.
- Use package-global sinks. Global state would prevent per-engine configuration and make tests and multi-service processes interfere with each other.

### 2. Use one immutable terminal event with two narrow sink interfaces

The root package will define bounded `ValidationMode` and `ValidationOutcome` string types with exported constants for the values in the spec, plus an immutable `ValidationEvent`. Event fields remain private and are exposed through read-only accessors for mode, outcome, and rule identifier. Only the engine creates events.

The sink contracts will each have one method and no context parameter:

```go
type Metrics interface {
    RecordValidation(ValidationEvent) error
}

type Logger interface {
    LogValidation(ValidationEvent) error
}
```

Not passing `context.Context` is deliberate: a caller context can contain tokens, credentials, or other request-scoped secrets, while this capability explicitly permits only the event's bounded fields. Implementations that need static deployment identity capture it safely at construction.

Separate interfaces allow either concern to be replaced or disabled independently. A combined Observer interface was rejected because it couples metrics and logging lifecycles and forces consumers to implement unused methods. A generic attribute map was rejected because it makes label names and values unbounded and weakens compile-time privacy review.

### 3. Centralize terminal classification and emit synchronously

`Validate` will route every terminal return through a small finalization path that accepts the original result and the already classified event. Classification happens where the engine still knows the cause: cancellation, parser failure, first rejecting rule, or success. It does not inspect error strings or unwrap arbitrary errors to construct dimensions.

Emission is synchronous and occurs after the terminal result is known but before `Validate` returns. Metrics runs before logging for deterministic tests, but ordering is not part of the public contract. Each call is isolated separately so a metrics failure cannot suppress logging and a logging failure cannot affect metrics.

Background goroutines were rejected because they require queues, shutdown semantics, event copying, and overflow policy, and could retain data beyond the validation call. Deferring classification to consumers was rejected because it duplicates engine logic and contradicts engine-owned observability.

### 4. Contain every runtime sink error and panic

Each sink invocation will run behind a narrow helper that discards the returned error and uses `recover` around only that invocation. The helper recovers every panic value, performs no retry, emits no secondary signal, and then allows the other sink and the original validation return path to proceed.

Static failures remain visible: invalid typed-nil implementations and adapter configuration errors are returned by constructors. Runtime failures are intentionally invisible to the engine's caller. Custom sinks remain free to maintain their own internal health metrics, but those mechanisms are outside this library.

Returning runtime sink errors from `Validate` was rejected because it would replace a successful result or obscure a typed violation/parser failure. Recovering only selected panic types was rejected because arbitrary user implementations can panic with any Go value and the isolation guarantee does not distinguish their origin.

### 5. Provide a dedicated Prometheus adapter with fixed cardinality inputs

The official adapter will live in `observability/prometheus` and implement `sqlguard.Metrics`. Its constructor accepts a configuration containing:

- a required caller-supplied `prometheus.Registerer`;
- a required non-empty `Service` captured once for the adapter lifetime;
- an optional `MetricName`, defaulting to `sqlguard_validations_total`.

Construction validates the service, validates the effective metric name using Prometheus naming rules, creates a `CounterVec` with label names `service`, `mode`, `outcome`, and `rule_id`, and registers it with the supplied registerer. Any descriptor or registration error is returned before the adapter is usable. It never uses the default global registry and never treats an already-registered collector as success implicitly, because doing so could bind to an incompatible collector.

At runtime the adapter calls the counter with the construction-time service and event accessors. For non-violation outcomes it supplies the required Prometheus label value as the empty string; label names cannot vary between series of one metric. The adapter adds no labels and exposes no per-validation label injection API.

Making `service` a caller-applied Prometheus wrapping label was considered, but requiring it in the adapter makes the supported metric contract self-contained and testable. Making the metric name completely fixed was rejected because applications may need an established namespace or collision-free name in a shared registry.

### 6. Drive log/slog through its Handler to retain runtime errors

The official adapter will live in `observability/slog` and implement `sqlguard.Logger`. Its constructor accepts a non-nil `*slog.Logger`, retains the logger's handler, and returns an error for nil input.

For each event the adapter maps outcome to the specified level, constructs a record with message `sqlguard validation`, adds only `mode`, `outcome`, and `rule_id`, checks `Handler.Enabled`, and calls `Handler.Handle` with a clean background context. Calling the handler directly preserves its returned error so the engine can apply the common isolation boundary; `slog.Logger` convenience methods discard handler errors. The clean context prevents caller values from crossing the privacy boundary.

Passing the validation context was rejected because context values are arbitrary and may contain secrets. Logging the validation error was rejected because it is unnecessary for classification and risks propagating future unsafe error details.

### 7. Verify contracts at their owning layers

Core tests will use table-driven recording, erroring, and panicking fakes to cover every terminal outcome, exact-once delivery, independent disabling, typed-nil rejection, sink isolation, deterministic sink invocation, immutability, privacy sentinels, and concurrent use under the race detector. Existing validation tests continue to prove unchanged return behavior.

Prometheus adapter tests will use isolated registries and gathered metric families to assert default/custom names, exact label sets and values, registration conflicts, invalid configuration, independent registries, and counter increments. `log/slog` adapter tests will use a recording handler to assert levels, message, exact integration-supplied attributes, disabled records, returned handler errors, privacy sentinels, and concurrent calls.

## Risks / Trade-offs

- [A custom sink panic is hidden from the validation caller] → Restrict recovery to the sink call, document the isolation contract, and require custom implementations to own health reporting.
- [Changing `NewEngine` breaks every existing caller at compile time] → Make the break explicit in the proposal and documentation, provide the mechanical `EngineOptions{}` migration, and perform it before a stable API release.
- [Synchronous sinks add validation latency] → Keep interfaces minimal and recommend that custom sinks perform only bounded local work; asynchronous delivery remains their responsibility.
- [Recovering panics may mask a severe sink bug] → Preserve enforcement and availability by design; test that the original result survives and leave diagnosis to the sink's own instrumentation.
- [`service` can create excess cardinality if callers construct many adapters dynamically] → Capture it once per adapter, prohibit per-validation replacement, and document adapters as long-lived dependencies.
- [Custom metric names make cross-service dashboards less uniform] → Provide and document a stable default while requiring explicit construction-time override.
- [Calling `slog.Handler` directly omits caller source information normally calculated by `slog.Logger`] → Treat source location as out of scope; the stable message and bounded attributes are the supported record contract.
- [A future audit mode will add new bounded values] → Keep mode and outcome typed and centralized, but add audit values only through its own accepted capability change.

## Migration Plan

1. Add the core event types and sink interfaces, change `NewEngine` to accept `EngineOptions` first, migrate repository call sites to `EngineOptions{}`, and add the isolated finalization path.
2. Add the two official adapter packages and make the Prometheus client a direct dependency used only by its adapter.
3. Document opt-in construction examples, privacy constraints, concurrency ownership, and explicit disabling.
4. Run the full lint, unit, race, strict OpenSpec, and precommit gates before release.

The constructor migration is source-breaking but mechanical: existing `NewEngine(rules...)` calls become `NewEngine(EngineOptions{}, rules...)`. Rollback restores the previous constructor signature and removes the first argument from migrated call sites; no stored data or database migration is involved.
