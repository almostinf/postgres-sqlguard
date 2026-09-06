## 1. Public Rules Package and Constructors

- [x] 1.1 Add `rules/doc.go` with package GoDoc and add black-box contract tests proving that `NewUpdateRequiresWhere` and `NewDeleteRequiresWhere` return `sqlguard.Rule` values with the stable identifiers `update_requires_where` and `delete_requires_where`.
- [x] 1.2 Add the public no-error constructors and private stateless rule types in `rules/mutation.go`, including compile-time interface assertions and private constants for rule IDs, target statement kinds, and `where_clause`.
- [x] 1.3 Add a public Engine test showing that unsafe mutations remain allowed when neither built-in is explicitly registered, covering the "rules are not enabled implicitly" scenario.

## 2. UPDATE Requires WHERE Behavior

- [x] 2.1 Add parallel table-driven black-box cases for `UPDATE` without `WHERE`, with a normal predicate, with `WHERE TRUE`, and for a non-`UPDATE` statement, asserting both validation outcomes and the typed `update_requires_where` violation.
- [x] 2.2 Implement the update rule through `Statement.Kind()` and `Statement.Root().Child("where_clause")`, allowing non-target statements and any syntactically present predicate while rejecting a target statement without that child.

## 3. DELETE Requires WHERE Behavior

- [x] 3.1 Add parallel table-driven black-box cases for `DELETE` without `WHERE`, with a normal predicate, with `WHERE TRUE`, and for a non-`DELETE` statement, asserting both validation outcomes and the typed `delete_requires_where` violation.
- [x] 3.2 Implement the delete rule using the same private structural decision path as the update rule, with no SQL text matching or mutation-specific additions to the public `Statement` API.

## 4. Parsed Structure and Complete-Input Coverage

- [x] 4.1 Add table-driven cases proving that `WHERE` text in comments or literals does not satisfy either policy and that `UPDATE` or `DELETE` text inside a `SELECT` literal does not trigger either rule.
- [x] 4.2 Add Engine-level cases proving that an unsafe later mutation in multi-statement input is rejected by the matching registered rule.
- [x] 4.3 Add Engine-level cases for unsafe `UPDATE` and `DELETE` statements in statement-bearing CTEs at multiple nesting depths, plus a complete-input case in which every mutation has a syntactic `WHERE` clause and validation succeeds.

## 5. Extension Boundary, Concurrency, and Documentation

- [x] 5.1 Add a test-only custom rule through the public `sqlguard.Rule` contract, register it alongside both built-ins, and verify deterministic registration-order evaluation without a built-in-specific Engine or parser path.
- [x] 5.2 Add concurrent validation coverage that reuses rule instances returned by both constructors and remains race-free without rule-owned mutable state.
- [x] 5.3 Update `README.md` with the `rules` package import and explicit constructor registration example, documenting that both policies are opt-in and that `WHERE TRUE` satisfies their syntactic baseline.
