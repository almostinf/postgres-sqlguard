## MODIFIED Requirements

### Requirement: Validation benchmark baseline
The repository SHALL provide runnable validation benchmarks that report
absolute `Validate` latency and allocations without adding a validation cache.
The benchmark suite SHALL isolate SQL complexity, validation outcome, and
observability configuration as separate measurement axes rather than combining
them into a cross-product. It MUST NOT include a synthetic comparison path that
bypasses SQLGuard.

#### Scenario: Core benchmarks are runnable
- **WHEN** a maintainer runs the documented Go benchmark command
- **THEN** benchmarks execute for the representative validation paths and report standard Go benchmark measurements

#### Scenario: SQL complexity is measured independently
- **WHEN** a maintainer runs `BenchmarkEngineValidateByComplexity`
- **THEN** the benchmark reports separate `simple`, `medium`, `multi_statement`, and `nested_cte` results with the validation outcome and observability configuration held constant

#### Scenario: Validation outcomes are measured independently
- **WHEN** a maintainer runs `BenchmarkEngineValidateByOutcome`
- **THEN** the benchmark reports separate `allowed`, `policy_violation`, and `parser_failure` results with SQL complexity and observability configuration held constant

#### Scenario: Observability overhead is measured independently
- **WHEN** a maintainer runs `BenchmarkEngineObservability`
- **THEN** the benchmark reports separate `disabled`, `metrics`, `logger`, and `metrics_and_logger` results using the same SQL input and validation outcome

#### Scenario: Results describe validation cost
- **WHEN** benchmark results are documented or presented as a latency comparison
- **THEN** they report absolute SQLGuard validation measurements and identify the complexity comparison as "Validation latency by SQL complexity" without presenting a synthetic "without SQLGuard" baseline
