# ADR 0001: Table-Driven Go Test Style

- Status: Accepted
- Date: 2026-09-06

## Context

The project needs one predictable style for tests that exercise the same behavior with multiple inputs. Consistent test structure makes cases easier to scan, run independently, and extend without duplicating setup and assertion flow.

Go map iteration and parallel subtests do not provide execution ordering. Test cases using this style must therefore be independent and must not rely on shared mutable fixtures.

## Decision

Use table-driven tests when cases share one execution flow. Declare the cases as a map keyed by a descriptive test name:

```go
tests := map[string]struct {
	input       string
	setupTest   func(t *testing.T)
	checkResult func(t *testing.T, result Result)
	checkError  func(t *testing.T, err error)
}{
	"accepts_valid_input": {
		input:       "valid input",
		checkResult: requireValidResult,
		checkError:  requireNoError,
	},
}
```

Apply these conventions:

1. Use `snake_case` case names matching `^[a-z0-9]+(?:_[a-z0-9]+)*$`.
2. Run each case with `t.Run(name, ...)` and call `t.Parallel()` at the start of the subtest.
3. Perform case-specific setup through an optional `setupTest` callback after `t.Parallel()` so no fixture is created in the parent test and shared accidentally.
4. Keep `checkError` explicit for every case. Use `checkResult` when the result also needs inspection.
5. Use `github.com/stretchr/testify/require` for assertions. Invoke assertion callbacks directly from the subtest goroutine because `require` may stop that goroutine with `FailNow`.
6. Call `t.Helper()` in reusable setup and assertion helpers.
7. Prefer data fields over callbacks when a value fully describes the expectation. Use callbacks for structured results or case-specific checks that would otherwise make the runner branch on test semantics.
8. Keep a scenario as a standalone test when it does not share the table's setup, execution, and assertion lifecycle.
9. Do not repeat an identical assertion a fixed number of times as evidence of determinism. Assert the exact order once and cover concurrency or repeated validation at the layer where those behaviors originate.

A shared named case type and runner may replace an anonymous `struct` when multiple test files genuinely reuse the same lifecycle. The resulting case collection remains a `map[string]...` and follows the same naming and parallelism rules.

## Consequences

- Individual scenarios have stable `go test -run` paths and can execute concurrently.
- Undefined map order helps expose accidental coupling between cases, but failures must never depend on which case starts first.
- Callback-based checks allow precise AST and error assertions, at the cost of more indirection than purely declarative expected values.
- `testify/require` becomes the standard assertion dependency for table-driven tests.
- Tests with materially different lifecycles remain separate instead of being forced into an oversized table.
