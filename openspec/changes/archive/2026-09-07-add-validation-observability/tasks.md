## 1. Core Observability Contracts

- [x] 1.1 Add table-driven contract tests for the bounded validation modes and outcomes, immutable event accessors, and empty non-violation `rule_id`.
- [x] 1.2 Define the public `ValidationMode`, `ValidationOutcome`, `ValidationEvent`, `Metrics`, `Logger`, and `EngineOptions` contracts with complete GoDoc and no SQL or context-bearing fields.
- [x] 1.3 Add constructor tests for the required `EngineOptions` argument, zero-valued options, independently configured or disabled sinks, typed-nil sink rejection, unchanged rule validation, and no partially configured engine.
- [x] 1.4 Change the primary constructor to `NewEngine(options EngineOptions, rules ...Rule)` and store validated sink references without adding a second constructor.
- [x] 1.5 Migrate all repository tests, benchmarks, examples, and internal call sites from `NewEngine(rules...)` to `NewEngine(EngineOptions{}, rules...)` where observability remains disabled.

## 2. Engine-Owned Emission and Failure Isolation

- [x] 2.1 Add engine tests covering exact-once metrics and logging events for allowed, policy-violation, parser-failure, and cancellation terminal paths, including stable rule identifiers and the enforce mode.
- [x] 2.2 Implement centralized terminal classification and synchronous finalization so every `Validate` return path emits the bounded event through each enabled sink.
- [x] 2.3 Add tests in which either or both sinks return errors or panic, proving the original successful or typed failure result survives, the other sink still runs once, and no retry or recursive emission occurs.
- [x] 2.4 Implement independently recovered sink calls that discard runtime sink errors and panics without changing validation behavior.
- [x] 2.5 Add privacy and concurrency tests proving sinks receive no SQL, errors, parser diagnostics, or caller-context values and that one configured engine remains race-free with concurrency-safe sinks.

## 3. Official Prometheus Integration

- [x] 3.1 Promote the supported Prometheus client to a direct module dependency and add the public `observability/prometheus` package with package documentation.
- [x] 3.2 Add table-driven constructor tests for a required non-nil registerer, required non-empty `service`, the default and valid custom metric names, invalid metric names, registration conflicts, and independent registries.
- [x] 3.3 Implement the Prometheus configuration and constructor, registering one counter with the effective name and exactly the labels `service`, `mode`, `outcome`, and `rule_id` in the caller-supplied registry only.
- [x] 3.4 Add gathered-metric tests proving exact-once increments, fixed construction-time `service`, bounded event values, empty non-violation `rule_id`, and isolation between adapter instances.
- [x] 3.5 Implement the `sqlguard.Metrics` adapter using only its immutable construction-time configuration and `ValidationEvent` accessors.

## 4. Official log/slog Integration

- [x] 4.1 Add the public `observability/slog` package with package documentation and constructor tests for valid and nil `*slog.Logger` inputs.
- [x] 4.2 Add recording-handler tests for the stable `sqlguard validation` message, INFO allowed events, ERROR policy/parser events, DEBUG cancellation events, exact integration-supplied attributes, disabled records, returned handler errors, and privacy sentinels.
- [x] 4.3 Implement the `sqlguard.Logger` adapter by retaining the configured handler, checking `Enabled`, and calling `Handle` with a clean background context so handler errors reach the engine isolation boundary without caller-context values.
- [x] 4.4 Add concurrent adapter tests proving one configured slog adapter is race-free when its supplied handler satisfies the documented concurrency contract.

## 5. Documentation and Performance Coverage

- [x] 5.1 Update root package and exported API documentation with sink concurrency ownership, runtime failure isolation, default-disabled behavior, and the distinction between nil disabling and typed-nil invalid configuration.
- [x] 5.2 Update `README.md` with the breaking `NewEngine` migration plus custom sink, Prometheus, and `log/slog` construction examples; document `service`, metric-name override, exact labels and log levels, privacy restrictions, and the absence of audit-mode behavior.
- [x] 5.3 Extend validation benchmarks with disabled observability and bounded no-op sink cases so synchronous emission overhead remains visible without adding caching or background delivery.
