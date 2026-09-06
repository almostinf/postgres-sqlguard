## Why

The validation engine can execute application-defined rules, but the library
does not yet provide the baseline mutation safeguards promised by the product
requirements. Providing reusable PostgreSQL-aware rules for `UPDATE` and
`DELETE` avoids each consumer having to reimplement this safety-critical logic.

## What Changes

- Add a public `rules` package with `NewUpdateRequiresWhere` and
  `NewDeleteRequiresWhere` constructors. The returned rules reject their
  respective mutation statement when its parsed structure has no `WHERE`
  clause.
- Treat any syntactically present `WHERE` clause, including `WHERE TRUE`, as
  satisfying these baseline rules; detecting tautologies is not part of this
  change.
- Define behavior for PostgreSQL syntax containing comments, literals,
  multiple top-level statements, and mutations inside nested CTEs without
  relying on textual keyword matching.
- Preserve explicit registration: neither rule is enabled implicitly, and the
  Engine remains unchanged when these or future rules are added.
- Demonstrate through tests that additional rules can be implemented and
  registered through the existing public `Rule` contract without modifying the
  Engine or parser.

Configuration through `rules.yaml`, audit mode, driver integrations, and more
advanced predicate analysis are outside this change.

## Capabilities

### New Capabilities

- `builtin-mutation-rules`: Public baseline rules for requiring a syntactically
  present `WHERE` clause on `UPDATE` and `DELETE`, including structurally
  accurate behavior across complete inputs and nested CTEs.

### Modified Capabilities

None. The change uses and verifies the existing `validation-engine` and
`rule-extension` contracts without changing their requirements.

## Impact

- Adds the public `github.com/almostinf/postgres-sqlguard/rules` package while
  keeping concrete built-in implementations private.
- Adds rule-focused unit and integration tests using the root package's
  existing `Rule`, parsed `Statement` facade, and Engine traversal behavior.
- Does not change the root Engine API, Engine implementation, implicit policy
  behavior, parser dependencies, error types, or supported PostgreSQL grammar.
