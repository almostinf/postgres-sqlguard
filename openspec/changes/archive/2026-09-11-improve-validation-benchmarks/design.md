## Context

See `proposal.md` for motivation and
`specs/validation-engine/spec.md` for the benchmark contract. The current
`BenchmarkEngineValidate` nests observability configurations around SQL inputs,
so every result changes two dimensions at once and the suite runs an unnecessary
cross-product. Engine construction is already outside the timed loop, and that
property must be preserved.

The benchmark remains an external-package benchmark and may reuse the existing
`ruleStub`, no-op metrics, and no-op logger implementations. `make bench` is the
canonical entry point and already requests allocation reporting.

## Goals / Non-Goals

**Goals:**

- Give each reported benchmark path one clear interpretation by varying only
  complexity, outcome, or observability.
- Keep setup, correctness checks, and engine construction outside timed work so
  results measure `Engine.Validate`.
- Retain stable, filterable Go benchmark names and standard `ns/op`, `B/op`, and
  `allocs/op` output.

**Non-Goals:**

- Establishing performance thresholds or making results comparable across
  machines without a controlled benchmark environment.
- Benchmarking database calls, network latency, concurrent validation, or
  production observability integrations.
- Optimizing the engine or changing validation semantics as part of the
  benchmark rewrite.

## Decisions

### Use three top-level benchmark functions

Replace the nested cross-product with
`BenchmarkEngineValidateByComplexity`, `BenchmarkEngineValidateByOutcome`, and
`BenchmarkEngineObservability`. Each function owns a single table of named
sub-benchmarks matching the delta spec.

This structure makes `go test -bench` filters useful and prevents a case name
from implying that two independent dimensions were measured together. A single
benchmark with flattened combination names was rejected because it would retain
the cross-product and make comparisons easier to misuse.

### Hold non-target dimensions constant

The complexity group uses one allow rule and disabled observability for every
case. `simple` is a minimal statement, `medium` is a structurally richer single
statement, and the remaining cases exercise multiple top-level statements and
recursively nested statement-bearing CTEs.

The outcome group uses disabled observability and deliberately selects inputs
and rules that produce allowed, policy-violation, and parser-failure results.
The cases should use the smallest representative input for their outcome so SQL
shape does not dominate the comparison.

The observability group uses one fixed allowed input and the same allow rule for
all four configurations. The existing no-op sink implementations isolate the
synchronous dispatch overhead of disabled, metrics-only, logger-only, and
combined sinks without introducing third-party integration behavior.

Allowing each case to choose realistic but different surrounding inputs was
rejected because the reported deltas would no longer isolate the named axis.

### Validate fixtures before starting the timer

Each sub-benchmark constructs its engine and performs an untimed validation to
confirm the intended outcome before `ResetTimer`. The timed loop invokes
`Validate`, consumes its result, and treats an unexpected outcome as a benchmark
failure. Every sub-benchmark calls `ReportAllocs`, independent of the
`-benchmem` flag used by `make bench`.

Preflight validation catches a malformed fixture or incorrectly configured
rule without charging setup to the measurement. Constructing the engine inside
the timed loop was rejected because engine initialization is not part of the
public validation call being measured.

### Present absolute validation measurements

Documentation continues to direct maintainers to `make bench`. Any table or
chart derived from the complexity group uses the title "Validation latency by
SQL complexity" and reports the benchmark's absolute validation measurements.
No empty loop, parser-only path, or direct executor call is subtracted or shown
as a "without SQLGuard" baseline.

Subtractive baselines were rejected because they introduce their own overhead
and can be mistaken for a portable end-to-end database comparison. Consumers
can instead compare absolute validation latency with latency measured in their
own execution path.

## Risks / Trade-offs

- [Outcome cases necessarily use different inputs or rule decisions] → Keep
  inputs minimal, document the fixed setup, and interpret the group as terminal
  path cost rather than a mathematically pure branch comparison.
- [No-op sinks understate real observability cost] → Name them as dispatch
  overhead fixtures and avoid claims about Prometheus, `slog`, or user-provided
  implementations.
- [Changing benchmark names breaks saved `-bench` filters] → Document all three
  new top-level names; the old ambiguous name is intentionally not retained as
  an alias because duplicate measurements would confuse reports.
- [Benchmark noise can obscure small differences] → Preserve isolated cases and
  standard Go benchmark output so maintainers can use repeated runs and
  statistical comparison tools when needed.

## Migration Plan

1. Replace the existing cross-product benchmark with the three benchmark
   groups and shared setup/check helpers where they reduce duplication.
2. Update benchmark documentation and performance labels to use the new names
   and absolute-latency interpretation.
3. Run the focused benchmarks to verify names, outcomes, and allocation output,
   then run the repository's required quality gates.

Rollback consists of reverting the benchmark and documentation changes; no
runtime code, public API, persisted data, or consumer migration is involved.
