# Parser backends

SQLGuard selects its PostgreSQL parser backend at build time. Ordinary
`CGO_ENABLED=1` builds use `github.com/pganalyze/pg_query_go/v6`; ordinary
`CGO_ENABLED=0` builds use the embedded WebAssembly parser from
`github.com/wasilibs/go-pgquery`. There is no runtime switch, custom build tag,
native sidecar, or fail-open fallback.

Both implementations are private to `internal/parser`. The public API, parser
failure categories, rule semantics, traversal order, prepared values, and
observability outcomes are identical in both build modes.

## Candidate comparison

| Candidate | PostgreSQL grammar and AST | Performance and size | Operational constraints | Decision |
| --- | --- | --- | --- | --- |
| `pganalyze/pg_query_go/v6` | PostgreSQL 17 `libpg_query` grammar and generated protobuf AST | Fastest measured warm path and smallest linked binary | Requires CGO and a working C compiler; cross-compilation inherits CGO constraints | Default when CGO is enabled |
| `wasilibs/go-pgquery` | The same PostgreSQL 17-derived `libpg_query` parser compiled to WASM; returns the same `pg_query_go/v6` protobuf messages | Slower cold and warm parsing; embedded WASM and wazero add binary and startup-memory cost | No consumer C compiler; embedded wazero runtime; no TinyGo support; upstream currently requires a pinned pseudo-version | Selected when CGO is disabled |
| `auxten/postgresql-parser` | CockroachDB-derived PostgreSQL-style grammar and a different AST, not the PostgreSQL 17 server grammar/protobuf contract | No comparable maintained project measurement was available | Documented grammar and built-in-function gaps would require a second semantic adapter and still could not establish PostgreSQL 17 parity | Rejected |

The no-CGO dependency is pinned to
`github.com/wasilibs/go-pgquery@v0.0.0-20260908021017-318158a2ab67`
(commit `318158a2ab6787a92a4624ab23f3401a04ee6525`). It depends on exactly
`github.com/pganalyze/pg_query_go/v6@v6.2.2`, the same protobuf and PostgreSQL
17 parser version used by the CGO backend.

## Compatibility and public behavior

The shared boundary converts backend protobuf messages immediately into
package-owned immutable nodes. Rules never receive generated protobuf values.
A checked-in corpus covers PostgreSQL-specific syntax, PostgreSQL 17 additions,
comments, quoted forms, parameters, expressions, DML, DDL, batches, nested
data-modifying CTEs, empty input, and malformed input. Each build mode must
match the same parser-neutral golden snapshots and engine-level expectations.

The common boundary also rejects embedded NUL bytes before either backend's
NUL-terminated C ABI can truncate the input. Parser failures are mapped to the
same bounded typed error without retaining SQL or raw backend diagnostics.
Preparation parses once, and repeated prepared validation reuses only the
immutable parser-neutral representation.

Compatibility is with the pinned PostgreSQL 17 parser grammar. Compatibility
with every older or newer PostgreSQL major version is not implied.

## Supported build matrix

| Target | `CGO_ENABLED=1` | `CGO_ENABLED=0` |
| --- | --- | --- |
| Linux/amd64 | Supported and exercised in CI; GCC-compatible compiler required | Supported and exercised in CI with `CC` deliberately unusable |
| Darwin/arm64 | Supported and measured with Apple Clang | Supported and measured without invoking a compiler |
| Other standard Go targets | Not claimed until added to the tested matrix | Not claimed until added to the tested matrix |
| TinyGo | Not supported | Not supported by the selected WASM dependency |

No-CGO means that building and running SQLGuard does not require a C compiler
or native parser library. It does not mean that the parser is pure Go: the
selected backend embeds `libpg_query` as WASM and executes it with wazero.

## Reproduced measurements

These measurements are evidence for the backend tradeoff, not release limits
or predictions for another machine.

- Date: 2026-09-13.
- Host: Apple M4 Pro, Darwin/arm64.
- Go: `go1.26.0 darwin/arm64`.
- CGO compiler: Apple Clang 17.0.0 (`arm64-apple-darwin24.6.0`).
- CGO parser: `pg_query_go/v6@v6.2.2`, commit
  `6a1adb4a50b66ca95f2bb34409697c586e18fa28`.
- no-CGO parser: `go-pgquery@v0.0.0-20260908021017-318158a2ab67`, commit
  `318158a2ab6787a92a4624ab23f3401a04ee6525`.
- WASM runtime: `wazero@v1.12.0` and
  `wazero-helpers@v0.0.0-20250123031827-cd30c44769bb`.
- Benchmark settings: five runs, one-second benchtime, 14 logical benchmark
  workers; the table reports the median run.

The parser benchmark input is:

```sql
WITH recent_orders AS (
    SELECT account_id, count(*) AS order_count
    FROM orders
    WHERE created_at >= CURRENT_DATE - INTERVAL '30 days'
    GROUP BY account_id
)
SELECT accounts.id, recent_orders.order_count
FROM accounts
JOIN recent_orders ON recent_orders.account_id = accounts.id
WHERE accounts.active
ORDER BY accounts.id
LIMIT 100
```

| Parser benchmark | CGO median | no-CGO median | Interpretation |
| --- | ---: | ---: | --- |
| Fresh-process first parse | 6,172,276 ns/op | 473,990,597 ns/op | Fresh test process per operation; includes process startup and backend initialization |
| Warm steady state | 73,942 ns/op; 43,057 B/op; 367 allocs/op | 177,787 ns/op; 223,392 B/op; 375 allocs/op | One parse per operation after preflight |
| Parallel | 25,693 ns/op; 43,060 B/op; 367 allocs/op | 33,261 ns/op; 223,398 B/op; 375 allocs/op | Aggregate throughput under `RunParallel` on this 14-worker host |

The fresh-process benchmark's allocation counters describe the parent process
spawning the helper, not allocations inside the child parser. Use the warm and
parallel rows for parser-visible Go allocation comparisons. On this host the
no-CGO median was about 77x slower for cold process initialization, 2.4x slower
when warm, and 1.3x slower under parallel load.

The size probe validates one `SELECT`, one guarded `UPDATE`, and one guarded
`DELETE`. Both binaries used the same target and flags:
`-trimpath -ldflags '-s -w -buildid='`.

| Size-probe measurement | CGO | no-CGO |
| --- | ---: | ---: |
| Linked executable | 7,307,938 bytes | 10,478,434 bytes |
| One cold execution, maximum RSS | 11,419,648 bytes | 306,413,568 bytes |
| One cold execution, elapsed wall time | below the command's 0.01 s display resolution | 0.46 s |

Maximum RSS and elapsed time are single `/usr/bin/time -l` observations and
are more variable than the five-run benchmark. The no-CGO process pays for
compiling and instantiating the embedded parser module on first use.

Parser-runtime module download ZIP sizes from the Go module cache were:

| Module | ZIP bytes | Used by |
| --- | ---: | --- |
| `pg_query_go/v6@v6.2.2` | 3,369,960 | Both; CGO implementation and shared protobuf types |
| `go-pgquery@v0.0.0-20260908021017-318158a2ab67` | 4,177,109 | no-CGO |
| `wazero@v1.12.0` | 8,045,923 | no-CGO |
| `wazero-helpers@v0.0.0-20250123031827-cd30c44769bb` | 14,111 | no-CGO |
| `google.golang.org/protobuf@v1.36.12` | 2,354,385 | Both |
| `golang.org/x/sys@v0.44.0` | 2,012,379 | no-CGO runtime dependency |

ZIP sizes are download/cache footprint, not linked-binary contribution. The
no-CGO graph still includes `pg_query_go/v6` because both backends share its
generated protobuf types.

## Reproducing the evidence

Run all existing engine benchmarks with identical names and inputs in both
modes:

```sh
make bench-cgo
make bench-no-cgo
```

Run only the parser measurements used above:

```sh
CGO_ENABLED=1 CC=cc go test ./internal/parser -run '^$' -bench '^BenchmarkParser' -benchmem -benchtime=1s -count=5
CGO_ENABLED=0 CC=/definitely/missing/postgres-sqlguard-cc go test ./internal/parser -run '^$' -bench '^BenchmarkParser' -benchmem -benchtime=1s -count=5
```

Build equivalent stripped size probes:

```sh
make size-probe
stat -f '%N %z bytes' /tmp/postgres-sqlguard-size-probe/sqlguard-cgo /tmp/postgres-sqlguard-size-probe/sqlguard-no-cgo
/usr/bin/time -l /tmp/postgres-sqlguard-size-probe/sqlguard-cgo
/usr/bin/time -l /tmp/postgres-sqlguard-size-probe/sqlguard-no-cgo
```

The `stat` and `time` flags above are for macOS. Use the platform-equivalent
commands without changing the built binaries when reproducing elsewhere.
Module ZIP paths can be obtained with `go mod download -json <module>@<version>`
and measured without unpacking them.

## Licensing and maintenance

The CGO path carries the `pg_query_go` BSD-3-Clause and PostgreSQL notices. The
no-CGO path additionally carries the `go-pgquery` and `wazero-helpers` MIT
licenses, wazero's Apache-2.0 notice, and notices inherited by the embedded
`libpg_query` artifact. The complete reviewed texts are in
[`THIRD_PARTY_NOTICES.md`](../THIRD_PARTY_NOTICES.md).

When upgrading either backend, maintainers must keep the PostgreSQL grammar and
`pg_query_go/v6` protobuf versions aligned, review the complete production
license closure, regenerate goldens from the reviewed CGO baseline, and pass
`make precommit-all` plus `make fuzz-all`. Because `go-pgquery` currently has
no tagged release, its pseudo-version and commit must remain immutable and be
reviewed explicitly. WASM initialization remains synchronous; applications
that choose no-CGO should account for its cold-start memory and latency rather
than moving validation into an unbounded background goroutine.
