## Why

The built-in `rules` package currently protects only `UPDATE` and `DELETE`
statements without a `WHERE` clause, leaving common high-impact mutation and
schema operations to application-defined rules. A small set of opt-in,
AST-backed policies can cover these recurring safeguards without weakening
explicit registration, parser independence, or privacy guarantees.

## What Changes

- Add an opt-in rule that rejects `INSERT` statements without an explicit
  target-column list, preventing positional inserts that silently follow table
  schema order.
- Add opt-in rules that reject `TRUNCATE`, `DROP TABLE`, and `ALTER TABLE`
  statements respectively.
- Give every new rule a stable identifier and a public constructor in the
  existing `rules` package; no rule is enabled implicitly.
- Apply each policy from parsed PostgreSQL structure across the Engine's
  existing complete statement traversal, including later statements and
  statement-bearing CTEs where PostgreSQL permits them.
- Document intended composition with the existing `UPDATE`- and
  `DELETE`-require-`WHERE` rules and cover the new policies with public,
  backend-independent tests.
- Keep semantic predicate analysis, function-level policies, configurable
  statement allowlists, YAML rule loading, and audit mode out of scope for this
  independently releasable change.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `builtin-mutation-rules`: Extend the public opt-in built-in policy set with
  explicit-column `INSERT` enforcement and focused rejection of destructive
  `TRUNCATE`, `DROP TABLE`, and `ALTER TABLE` statements.

## Impact

- Adds constructors and private immutable implementations under `rules/`, with
  stable violation identifiers exposed through the existing typed error
  contract.
- Expands rule-focused tests and README documentation while reusing the
  existing `sqlguard.Rule`, parser-neutral `Statement` facade, Engine traversal,
  context, concurrency, and observability behavior.
- Does not change Engine defaults, parser backends, root package APIs,
  dependencies, error types, SQL retention behavior, or driver integrations.
