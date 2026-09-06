# postgres-sqlguard

`postgres-sqlguard` is a driver-independent Go library for validating SQL with
explicit application rules before it reaches PostgreSQL. It parses the complete
input with a real PostgreSQL grammar and does not require a database connection.

## Requirements

- Go 1.26;
- CGO enabled;
- a working C compiler available through `CC`.

The pinned `pg_query_go` parser embeds the PostgreSQL 17 grammar. Compatibility
with other PostgreSQL major versions is not implied.

## Usage

Implement `Rule`, construct an immutable `Engine`, and call `Validate`:

```go
package main

import (
	"context"
	"errors"
	"log"

	sqlguard "github.com/almostinf/postgres-sqlguard"
)

type denyDelete struct{}

func (denyDelete) ID() string {
	return "deny_delete"
}

func (denyDelete) Evaluate(_ context.Context, statement sqlguard.Statement) sqlguard.RuleResult {
	if statement.Kind() == sqlguard.Kind("DeleteStmt") {
		return sqlguard.Reject()
	}

	return sqlguard.Allow()
}

func main() {
	engine, err := sqlguard.NewEngine(denyDelete{})
	if err != nil {
		log.Fatal(err)
	}

	err = engine.Validate(context.Background(), "DELETE FROM accounts")
	if err == nil {
		return
	}

	var violation *sqlguard.Violation
	if errors.As(err, &violation) {
		log.Printf("SQL rejected by rule %q", violation.RuleID())
	}
}
```

Only explicitly supplied rules are active. Rule identifiers must contain at
most 64 ASCII characters, start with a letter or digit, and otherwise contain
only letters, digits, `.`, `-`, or `_`. Empty and duplicate identifiers are
rejected during construction.

### Built-in mutation rules

The public `rules` package provides opt-in policies for requiring `WHERE` on
`UPDATE` and `DELETE` statements:

```go
import (
	sqlguard "github.com/almostinf/postgres-sqlguard"
	"github.com/almostinf/postgres-sqlguard/rules"
)

engine, err := sqlguard.NewEngine(
	rules.NewUpdateRequiresWhere(),
	rules.NewDeleteRequiresWhere(),
)
```

Neither policy is enabled implicitly. They inspect parsed PostgreSQL structure,
so comments and literals cannot imitate a `WHERE` clause. Any syntactically
present predicate, including `WHERE TRUE`, satisfies these baseline rules;
tautology analysis is outside their scope.

## Validation semantics

The Engine parses the complete SQL string once before running any rule. If any
statement is malformed, validation returns a typed `ParseError` and no rule is
called. Empty, whitespace-only, comment-only, and semicolon-only inputs contain
no executable statements and therefore succeed.

For successfully parsed input, traversal is deterministic:

1. top-level statements are visited in input order;
2. each statement root is visited first;
3. statement-bearing CTEs are then visited depth-first in declaration order;
4. rules run in registration order for each visited statement;
5. the first rejection stops validation and returns a `Violation`.

Rules receive a read-only `Statement` facade rather than parser protobuf values
or source SQL. Custom rules can inspect node kinds and named structural fields
without depending on the parser backend.

## Errors and privacy

Use standard `errors.As` inspection rather than parsing error strings:

```go
var parseError *sqlguard.ParseError
if errors.As(err, &parseError) {
	log.Printf("parse failure category: %s", parseError.Category())
}

var violation *sqlguard.Violation
if errors.As(err, &violation) {
	log.Printf("rejecting rule: %s", violation.RuleID())
}
```

Public parser failures and violations contain only bounded categories or stable
rule identifiers. Their error messages are constant, and they do not retain or
unwrap raw parser diagnostics, SQL text, literals, comments, tokens,
credentials, or secrets.

## Context and concurrency

`Validate` passes the exact caller context to every rule and returns standard
`context.Canceled` or `context.DeadlineExceeded` errors directly. Cancellation
is checked before parsing, after parsing, and before every rule invocation.

The parser is a synchronous CGO call and cannot be interrupted after it starts.
If cancellation occurs during parsing, it is observed immediately after the C
call returns. Parsing is not moved to a background goroutine.

An initialized Engine is safe for concurrent use. A registered Rule instance
can be invoked concurrently by separate validations, so custom rules must be
immutable or synchronize their own mutable state.

## Development

Run the standard checks:

```sh
make precommit
```

Run the validator fuzz target:

```sh
make fuzz
make fuzz FUZZ_TIME=1m
```

Run the allocation-reporting benchmark baseline:

```sh
make bench
```
