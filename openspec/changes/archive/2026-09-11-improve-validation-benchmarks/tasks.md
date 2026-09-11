## 1. Benchmark Harness

- [x] 1.1 Refactor `validator_benchmark_test.go` with shared benchmark setup that constructs engines and verifies expected outcomes before timing, invokes only `Validate` in the measured loop, and reports allocations for every case.

## 2. Isolated Benchmark Groups

- [x] 2.1 Add `BenchmarkEngineValidateByComplexity` with `simple`, `medium`, `multi_statement`, and `nested_cte` cases using one allow rule and disabled observability.
- [x] 2.2 Add `BenchmarkEngineValidateByOutcome` with minimal `allowed`, `policy_violation`, and `parser_failure` cases using disabled observability and verified typed outcomes.
- [x] 2.3 Add `BenchmarkEngineObservability` with `disabled`, `metrics`, `logger`, and `metrics_and_logger` cases using the same allowed SQL input and rule configuration.
- [x] 2.4 Remove the old complexity-by-observability cross-product and ensure the suite contains no synthetic validation-bypass baseline.

## 3. Benchmark Documentation

- [x] 3.1 Update the benchmark documentation with the three filterable benchmark groups and explain that results report absolute SQLGuard validation latency and allocations.
- [x] 3.2 Label any documented complexity comparison "Validation latency by SQL complexity" and explain that consumers should compare it with latency from their own database path rather than a synthetic "without SQLGuard" result.
