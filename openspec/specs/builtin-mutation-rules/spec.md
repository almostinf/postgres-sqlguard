# builtin-mutation-rules Specification

## Purpose

Defines opt-in PostgreSQL-aware rules that prevent broad `UPDATE` and `DELETE`
mutations unless the parsed statement contains a syntactic `WHERE` clause.

## Requirements

### Requirement: Public opt-in mutation rules
The public `rules` package SHALL expose `NewUpdateRequiresWhere` and
`NewDeleteRequiresWhere` constructors that return implementations of the root
package's existing rule contract. The returned rules' stable identifiers SHALL
be `update_requires_where` and `delete_requires_where` respectively. An Engine
MUST apply either rule only when the caller explicitly registers it.

#### Scenario: Rules are not enabled implicitly
- **WHEN** an Engine is constructed without either built-in mutation rule
- **THEN** the Engine does not apply an update-or-delete WHERE policy implicitly

#### Scenario: Violation identifies the update rule
- **WHEN** a rule returned by `rules.NewUpdateRequiresWhere` rejects a statement
- **THEN** the returned typed violation identifies `update_requires_where` as the rejecting rule

#### Scenario: Violation identifies the delete rule
- **WHEN** a rule returned by `rules.NewDeleteRequiresWhere` rejects a statement
- **THEN** the returned typed violation identifies `delete_requires_where` as the rejecting rule

### Requirement: UPDATE statements require a WHERE clause
A rule returned by `rules.NewUpdateRequiresWhere` MUST reject an `UPDATE`
statement whose parsed structure does not contain a `WHERE` clause and MUST
allow an `UPDATE` statement whose parsed structure contains any syntactically
valid `WHERE` clause. It MUST allow statement kinds other than `UPDATE` to
continue to other registered rules.

#### Scenario: UPDATE without WHERE is rejected
- **WHEN** an Engine using `rules.NewUpdateRequiresWhere()` validates `UPDATE accounts SET active = false`
- **THEN** validation returns a violation from `update_requires_where`

#### Scenario: UPDATE with a predicate is allowed
- **WHEN** an Engine using `rules.NewUpdateRequiresWhere()` validates `UPDATE accounts SET active = false WHERE id = 42`
- **THEN** that rule allows validation to continue

#### Scenario: UPDATE with WHERE TRUE is allowed
- **WHEN** an Engine using `rules.NewUpdateRequiresWhere()` validates `UPDATE accounts SET active = false WHERE TRUE`
- **THEN** that rule allows validation to continue without attempting tautology analysis

#### Scenario: UPDATE rule ignores another statement kind
- **WHEN** an Engine using only `rules.NewUpdateRequiresWhere()` validates a syntactically valid `DELETE` statement
- **THEN** the update rule allows validation to continue

### Requirement: DELETE statements require a WHERE clause
A rule returned by `rules.NewDeleteRequiresWhere` MUST reject a `DELETE`
statement whose parsed structure does not contain a `WHERE` clause and MUST
allow a `DELETE` statement whose parsed structure contains any syntactically
valid `WHERE` clause. It MUST allow statement kinds other than `DELETE` to
continue to other registered rules.

#### Scenario: DELETE without WHERE is rejected
- **WHEN** an Engine using `rules.NewDeleteRequiresWhere()` validates `DELETE FROM accounts`
- **THEN** validation returns a violation from `delete_requires_where`

#### Scenario: DELETE with a predicate is allowed
- **WHEN** an Engine using `rules.NewDeleteRequiresWhere()` validates `DELETE FROM accounts WHERE id = 42`
- **THEN** that rule allows validation to continue

#### Scenario: DELETE with WHERE TRUE is allowed
- **WHEN** an Engine using `rules.NewDeleteRequiresWhere()` validates `DELETE FROM accounts WHERE TRUE`
- **THEN** that rule allows validation to continue without attempting tautology analysis

#### Scenario: DELETE rule ignores another statement kind
- **WHEN** an Engine using only `rules.NewDeleteRequiresWhere()` validates a syntactically valid `UPDATE` statement
- **THEN** the delete rule allows validation to continue

### Requirement: Mutation decisions use parsed structure
The built-in mutation rules MUST determine statement kind and `WHERE` presence
from the parsed PostgreSQL structure. Keywords appearing only in comments or
literals MUST NOT create a mutation or satisfy a missing `WHERE` clause.

#### Scenario: Comment does not satisfy UPDATE policy
- **WHEN** the rule from `rules.NewUpdateRequiresWhere()` evaluates `UPDATE accounts SET active = false /* WHERE id = 42 */`
- **THEN** it rejects the statement because the parsed `UPDATE` has no `WHERE` clause

#### Scenario: Literal does not satisfy UPDATE policy
- **WHEN** the rule from `rules.NewUpdateRequiresWhere()` evaluates `UPDATE accounts SET note = 'WHERE id = 42'`
- **THEN** it rejects the statement because literal text is not a parsed `WHERE` clause

#### Scenario: Comment does not satisfy DELETE policy
- **WHEN** the rule from `rules.NewDeleteRequiresWhere()` evaluates `DELETE FROM accounts /* WHERE id = 42 */`
- **THEN** it rejects the statement because the parsed `DELETE` has no `WHERE` clause

#### Scenario: Literal mutation text does not trigger a rule
- **WHEN** both built-in rules evaluate `SELECT 'UPDATE accounts SET active = false; DELETE FROM accounts'`
- **THEN** both rules allow validation to continue because the parsed statement is a `SELECT`

### Requirement: Mutation rules cover complete traversal
When registered with an Engine, each built-in mutation rule MUST apply to every
matching statement presented by the Engine's deterministic traversal,
including later top-level statements and statement-bearing CTEs at any nesting
depth.

#### Scenario: Unsafe later statement is rejected
- **WHEN** input contains a safe first statement followed by an `UPDATE` or `DELETE` without a `WHERE` clause
- **THEN** the matching registered rule rejects the later statement

#### Scenario: Unsafe UPDATE inside a CTE is rejected
- **WHEN** an `UPDATE` without a `WHERE` clause is contained in a statement-bearing CTE
- **THEN** the registered rule from `rules.NewUpdateRequiresWhere()` rejects that nested mutation

#### Scenario: Unsafe DELETE inside a nested CTE is rejected
- **WHEN** a `DELETE` without a `WHERE` clause is contained in a statement-bearing CTE below another CTE
- **THEN** the registered rule from `rules.NewDeleteRequiresWhere()` rejects that mutation regardless of nesting depth

#### Scenario: Safe mutations across complete input are allowed
- **WHEN** every `UPDATE` and `DELETE` across multiple statements and nested CTEs has a syntactic `WHERE` clause
- **THEN** both built-in mutation rules allow validation to continue for the complete input

### Requirement: Built-ins preserve rule extensibility
Rules returned by the public `rules` package and application-defined rules
SHALL use the same root-package registration and evaluation contracts. Adding
another rule MUST NOT require a built-in-specific registration path or changes
to Engine or parser behavior.

#### Scenario: Custom rule is registered alongside built-ins
- **WHEN** a caller constructs an Engine with both built-in mutation rules and a new application-defined rule
- **THEN** the Engine evaluates all explicitly registered rules through the same rule contract in registration order

#### Scenario: New rule requires no Engine modification
- **WHEN** an application implements the public rule contract for an additional policy
- **THEN** the application can register and execute that policy without modifying the Engine or parser
