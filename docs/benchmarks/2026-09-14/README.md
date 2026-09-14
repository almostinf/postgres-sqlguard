# Release benchmark: 2026-09-14

This directory contains the benchmark evidence used by the `v1.0.0` README
charts. It compares the automatically selected CGO and no-CGO parser backends
with the same SQL inputs, benchmark names, Go version, host, benchtime, and run
count.

These measurements characterize one machine. They are not performance
guarantees, regression thresholds, or estimates of database round-trip time.
`ns/op` is benchmark execution time for in-process validation; it is not
sampled host CPU utilization.

## Environment

- Date: 2026-09-14.
- Host: Apple M4 Pro, 14 logical CPUs.
- OS and architecture: Darwin 24.6.0, arm64.
- Go: `go1.26.0 darwin/arm64`.
- CGO compiler: Apple Clang 17.0.0, target
  `arm64-apple-darwin24.6.0`.
- Benchmark settings: five runs per case, one-second benchtime, `-benchmem`.
- Binaries: identical target and `-trimpath -ldflags '-s -w -buildid='` flags.

The complete metadata, commands, units, raw-file locations, and reported
medians are in [`dataset.json`](dataset.json). The raw `go test` and
`/usr/bin/time -l` output is retained under [`raw/`](raw/).

## Direct validation results

Each value is the median of five runs of
`BenchmarkEngineValidateByComplexity`. The benchmark constructs its engine
outside the timed loop, parses the complete SQL input on every operation, and
applies one allow rule. `B/op` reports heap bytes allocated per validation.

| SQL complexity | CGO ns/op | no-CGO ns/op | CGO B/op | no-CGO B/op |
| --- | ---: | ---: | ---: | ---: |
| Simple | 5,821 | 13,742 | 5,544 | 87,568 |
| Medium | 75,779 | 164,908 | 41,441 | 123,472 |
| Multi-statement | 34,075 | 68,819 | 25,032 | 107,048 |
| Nested CTE | 48,213 | 99,737 | 29,048 | 209,368 |

The four cases are defined in
[`validator_benchmark_test.go`](../../../validator_benchmark_test.go), which is
the workload source of truth.

## Process footprint

The size probe validates one `SELECT`, one guarded `UPDATE`, and one guarded
`DELETE`. Maximum RSS is the median of five fresh executions. Binary size is
measured once after equivalent stripped builds because the output is
deterministic for the same source, toolchain, target, and flags.

| Measurement | CGO | no-CGO |
| --- | ---: | ---: |
| Cold maximum RSS, median | 11,403,264 bytes (10.88 MiB) | 313,917,440 bytes (299.38 MiB) |
| Stripped linked binary | 7,307,938 bytes (6.97 MiB) | 10,478,434 bytes (9.99 MiB) |

The five maximum-RSS observations were:

- CGO: `11665408`, `11386880`, `11403264`, `11403264`, `11403264` bytes.
- no-CGO: `313458688`, `313917440`, `299368448`, `314048512`,
  `314163200` bytes.

RSS uses the macOS-specific `/usr/bin/time -l` maximum-resident-set field.
Other operating systems expose different commands and may account for memory
differently. Cold results include process and parser-backend initialization;
warm, long-running services amortize that cost.

## Reproduce and verify

Capture the benchmark and footprint evidence on macOS:

```sh
make release-bench-capture
```

This overwrites the dated raw files but does not silently rewrite
`dataset.json`; review the raw runs and their medians before changing published
data. To use another date or destination, override `RELEASE_BENCH_DATE` or
`RELEASE_BENCH_DIR`.

Regenerate the checked-in SVG files using only the Go standard library:

```sh
make release-bench-charts
```

Verify that the checked-in charts are byte-identical to a clean regeneration:

```sh
make release-bench-check
```

The existing `bench-*` and `size-probe` Make interfaces remain unchanged.
