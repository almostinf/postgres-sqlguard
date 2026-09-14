# Validation guide

This guide covers engine construction, custom and built-in rules, prepared
validation, deterministic traversal, typed failures, privacy, context, and
concurrency. For the shortest first run, start with the repository
[README](../README.md).

## Construct an engine

An `Engine` is an immutable ordered collection of explicitly registered rules.
No rule is discovered through global state, package initialization, environment
variables, or implicit defaults.

```go
engine, err := sqlguard.NewEngine(
	sqlguard.EngineOptions{},
	rules.NewUpdateRequiresWhere(),
	rules.NewDeleteRequiresWhere(),
)
if err != nil {
	return err
}

if err := engine.Validate(ctx, query); err != nil {
	return err
}
```

Rule identifiers must contain at most 64 ASCII characters, start with a letter
or digit, and otherwise contain only letters, digits, `.`, `-`, or `_`. Engine
construction rejects empty, invalid, and duplicate identifiers without
returning a partially configured engine.

## Built-in mutation rules

Import the opt-in policies from:

```go
import "github.com/almostinf/postgres-sqlguard/pkg/rules"
```

| Constructor | Stable rule identifier | Policy |
| --- | --- | --- |
| `NewUpdateRequiresWhere` | `update_requires_where` | Require a syntactic `WHERE` clause on `UPDATE`. |
| `NewDeleteRequiresWhere` | `delete_requires_where` | Require a syntactic `WHERE` clause on `DELETE`. |
| `NewInsertRequiresColumns` | `insert_requires_columns` | Require an explicit target-column list on `INSERT`. |
| `NewDenyTruncate` | `deny_truncate` | Reject every `TRUNCATE`. |
| `NewDenyDropTable` | `deny_drop_table` | Reject every `DROP TABLE`. |
| `NewDenyAlterTable` | `deny_alter_table` | Reject every `ALTER TABLE`, including tablespace-wide forms. |

Register only the policies the application needs. The package does not enable
defaults, and built-ins use the same `sqlguard.Rule` contract and registration
order as application rules.

These rules inspect parsed PostgreSQL structure rather than searching source
text. Keywords inside comments, identifiers, and literals cannot imitate an
operation or satisfy a required clause. Any syntactically present predicate,
including `WHERE TRUE`, satisfies the UPDATE and DELETE policies; SQLGuard does
not perform tautology analysis.

`NewInsertRequiresColumns` checks only that a target-column list is present.
PostgreSQL remains responsible for validating the names, values, and
completeness of that list. DDL rules reject operation families, not selected
object names: `NewDenyDropTable` permits `DROP VIEW`, and
`NewDenyAlterTable` permits other `ALTER` families such as `ALTER ROLE` and
`ALTER INDEX`.

## Write a custom rule

A rule receives a read-only, parser-neutral `Statement`. It does not receive SQL
source or parser-generated protobuf values.

```go
type denyDelete struct{}

func (denyDelete) ID() string {
	return "deny_delete"
}

func (denyDelete) Evaluate(
	_ context.Context,
	statement sqlguard.Statement,
) sqlguard.RuleResult {
	if statement.Kind() == sqlguard.Kind("DeleteStmt") {
		return sqlguard.Reject()
	}

	return sqlguard.Allow()
}
```

Use `Statement.Root` and the `Node` accessors to inspect named structural
fields. Adding a custom rule does not require changing the engine or detecting
which parser backend is active.

## Prepared validation

Use `Prepare` when an application repeatedly validates the same stable,
parameterized SQL text:

```go
query := "UPDATE accounts SET active = $1 WHERE id = $2"

prepared, err := engine.Prepare(ctx, query)
if err != nil {
	return err
}

if err := engine.ValidatePrepared(requestCtx, prepared); err != nil {
	return err
}
```

Preparation parses the complete input once and returns an opaque immutable
value. It does not evaluate rules or cache an allow/deny decision. Every
`ValidatePrepared` call checks its own context and evaluates the receiving
engine's rules again. A successful value can be copied, shared concurrently,
and validated by another engine. Its zero value is rejected with
`ErrInvalidPrepared`.

`Prepared` is SQLGuard's client-side parsed representation. It is unrelated to
pgx or PostgreSQL server prepared statements, does not execute SQL, and does not
bind arguments. It retains parsed identifiers, literals, and byte values until
unreachable. Prefer placeholders over sensitive literals and keep prepared
values only as long as the application needs them.

Preparation remains synchronous. Malformed input and cancellation are reported
by `Prepare`. Successful preparation emits no terminal observability outcome;
`ValidatePrepared` emits the result of each actual policy decision.

## Complete-input validation semantics

The engine parses the complete SQL string before invoking a rule. If any
statement is malformed, validation fails closed with a typed `ParseError`, and
no rule runs. Empty, whitespace-only, comment-only, and semicolon-only inputs
contain no executable statements and succeed.

For valid input, traversal is deterministic:

1. top-level statements are visited in input order;
2. each statement root is visited first;
3. statement-bearing CTEs are visited depth-first in declaration order;
4. rules run in registration order for each visited statement;
5. the first rejection stops validation and returns a `Violation`.

The same traversal and parser-neutral values are preserved by the supported CGO
and no-CGO backends. See [Parser backends](parser-backends.md) for the grammar,
platform, performance, and maintenance contract.

## Errors and privacy

Inspect failures with standard Go error mechanisms rather than parsing error
strings:

```go
var parseError *sqlguard.ParseError
if errors.As(err, &parseError) {
	log.Printf("parse failure category: %s", parseError.Category())
}

var violation *sqlguard.Violation
if errors.As(err, &violation) {
	log.Printf("rejecting rule: %s", violation.RuleID())
}

if errors.Is(err, sqlguard.ErrInvalidPrepared) {
	log.Print("invalid prepared value")
}
```

Parser failures, violations, and invalid-prepared failures expose only bounded
categories or stable rule identifiers. Their error strings and unwrap chains do
not retain SQL, arguments, parsed structure, literals, comments, tokens,
credentials, secrets, or raw parser diagnostics.

The same privacy boundary applies to built-in observability. See the
[Observability guide](observability.md).

## Context and concurrency

`Validate`, `Prepare`, and `ValidatePrepared` preserve caller cancellation.
Direct and prepared validation pass the exact caller context to rules and return
standard `context.Canceled` or `context.DeadlineExceeded` errors. Cancellation
is checked before parsing, after parsing, and before every rule invocation.

Parsing is synchronous in both build modes and cannot be interrupted after it
starts. Cancellation that occurs during parsing is observed when the selected
backend returns; SQLGuard does not move parsing into an unbounded background
goroutine.

An initialized engine and successful prepared value are safe for concurrent
use. A registered `Rule`, `Metrics`, or `Logger` instance may be called
concurrently, so implementations must be immutable or synchronize their own
mutable state.

## Integration boundary

SQLGuard validates only calls that an application sends through its engine. It
does not intercept driver traffic, execute SQL, bind parameters, open a database
connection, enforce PostgreSQL privileges, or replace transactions and database
constraints. Applications must define and test a boundary that all intended
database paths cross. See [Integrating with pgx](pgx-integration.md) for two
compiling examples and their limitations.
