## MODIFIED Requirements

### Requirement: Public opt-in mutation rules
The public `rules` package SHALL expose `NewUpdateRequiresWhere`,
`NewDeleteRequiresWhere`, `NewInsertRequiresColumns`, `NewDenyTruncate`,
`NewDenyDropTable`, and `NewDenyAlterTable` constructors that return
implementations of the root package's existing rule contract. The returned
rules' stable identifiers SHALL be `update_requires_where`,
`delete_requires_where`, `insert_requires_columns`, `deny_truncate`,
`deny_drop_table`, and `deny_alter_table` respectively. An Engine MUST apply a
built-in rule only when the caller explicitly registers it.

#### Scenario: Rules are not enabled implicitly
- **WHEN** an Engine is constructed without built-in mutation rules
- **THEN** the Engine does not apply any built-in mutation policy implicitly

#### Scenario: Violation identifies the update rule
- **WHEN** a rule returned by `rules.NewUpdateRequiresWhere` rejects a statement
- **THEN** the returned typed violation identifies `update_requires_where` as the rejecting rule

#### Scenario: Violation identifies the delete rule
- **WHEN** a rule returned by `rules.NewDeleteRequiresWhere` rejects a statement
- **THEN** the returned typed violation identifies `delete_requires_where` as the rejecting rule

#### Scenario: Violation identifies the insert rule
- **WHEN** a rule returned by `rules.NewInsertRequiresColumns` rejects a statement
- **THEN** the returned typed violation identifies `insert_requires_columns` as the rejecting rule

#### Scenario: Violation identifies the truncate rule
- **WHEN** a rule returned by `rules.NewDenyTruncate` rejects a statement
- **THEN** the returned typed violation identifies `deny_truncate` as the rejecting rule

#### Scenario: Violation identifies the drop-table rule
- **WHEN** a rule returned by `rules.NewDenyDropTable` rejects a statement
- **THEN** the returned typed violation identifies `deny_drop_table` as the rejecting rule

#### Scenario: Violation identifies the alter-table rule
- **WHEN** a rule returned by `rules.NewDenyAlterTable` rejects a statement
- **THEN** the returned typed violation identifies `deny_alter_table` as the rejecting rule

### Requirement: Mutation decisions use parsed structure
The built-in mutation rules MUST determine statement kind, `WHERE` presence,
explicit `INSERT` target columns, and the object type of applicable schema
statements from parsed PostgreSQL structure. Keywords appearing only in
comments, identifiers, or literals MUST NOT create a matching operation or
satisfy a required structural clause.

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
- **WHEN** all built-in rules evaluate `SELECT 'INSERT DROP TABLE ALTER TABLE TRUNCATE UPDATE DELETE WHERE'`
- **THEN** every built-in rule allows validation to continue because the parsed statement is a `SELECT`

#### Scenario: Drop-table policy distinguishes object type
- **WHEN** the rule from `rules.NewDenyDropTable()` evaluates a `DROP VIEW` statement
- **THEN** it allows validation to continue because the parsed drop operation does not target tables

### Requirement: Mutation rules cover complete traversal
When registered with an Engine, each built-in mutation rule MUST apply to every
matching statement presented by the Engine's deterministic traversal,
including later top-level statements and statement-bearing CTEs at any nesting
depth where PostgreSQL permits the matching operation.

#### Scenario: Unsafe later statement is rejected
- **WHEN** input contains a safe first statement followed by an operation rejected by a registered built-in rule
- **THEN** the matching registered rule rejects the later statement

#### Scenario: Unsafe UPDATE inside a CTE is rejected
- **WHEN** an `UPDATE` without a `WHERE` clause is contained in a statement-bearing CTE
- **THEN** the registered rule from `rules.NewUpdateRequiresWhere()` rejects that nested mutation

#### Scenario: Unsafe DELETE inside a nested CTE is rejected
- **WHEN** a `DELETE` without a `WHERE` clause is contained in a statement-bearing CTE below another CTE
- **THEN** the registered rule from `rules.NewDeleteRequiresWhere()` rejects that mutation regardless of nesting depth

#### Scenario: Unsafe INSERT inside a CTE is rejected
- **WHEN** an `INSERT` without an explicit target-column list is contained in a statement-bearing CTE
- **THEN** the registered rule from `rules.NewInsertRequiresColumns()` rejects that nested mutation

#### Scenario: Safe mutations across complete input are allowed
- **WHEN** every `UPDATE` and `DELETE` has a syntactic `WHERE` clause, every `INSERT` has an explicit target-column list, and no statement matches a registered deny rule
- **THEN** all registered built-in mutation rules allow validation to continue for the complete input

## ADDED Requirements

### Requirement: INSERT statements require explicit target columns
A rule returned by `rules.NewInsertRequiresColumns` MUST reject an `INSERT`
statement that omits its target-column list and MUST allow an `INSERT` statement
that contains at least one explicit target column. It MUST allow statement
kinds other than `INSERT` to continue to other registered rules.

#### Scenario: Positional VALUES insert is rejected
- **WHEN** an Engine using `rules.NewInsertRequiresColumns()` validates `INSERT INTO accounts VALUES (42, false)`
- **THEN** validation returns a violation from `insert_requires_columns`

#### Scenario: Positional INSERT SELECT is rejected
- **WHEN** an Engine using `rules.NewInsertRequiresColumns()` validates `INSERT INTO accounts SELECT id, active FROM staged_accounts`
- **THEN** validation returns a violation from `insert_requires_columns`

#### Scenario: DEFAULT VALUES without columns is rejected
- **WHEN** an Engine using `rules.NewInsertRequiresColumns()` validates `INSERT INTO accounts DEFAULT VALUES`
- **THEN** validation returns a violation from `insert_requires_columns`

#### Scenario: Insert with explicit columns is allowed
- **WHEN** an Engine using `rules.NewInsertRequiresColumns()` validates `INSERT INTO accounts (id, active) VALUES (42, false)`
- **THEN** that rule allows validation to continue

#### Scenario: Insert rule ignores another statement kind
- **WHEN** an Engine using only `rules.NewInsertRequiresColumns()` validates a syntactically valid `UPDATE` statement
- **THEN** the insert rule allows validation to continue

### Requirement: TRUNCATE statements can be denied
A rule returned by `rules.NewDenyTruncate` MUST reject every `TRUNCATE`
statement and MUST allow statement kinds other than `TRUNCATE` to continue to
other registered rules.

#### Scenario: Single-table truncate is rejected
- **WHEN** an Engine using `rules.NewDenyTruncate()` validates `TRUNCATE TABLE accounts`
- **THEN** validation returns a violation from `deny_truncate`

#### Scenario: Multi-table truncate is rejected
- **WHEN** an Engine using `rules.NewDenyTruncate()` validates `TRUNCATE accounts, sessions RESTART IDENTITY CASCADE`
- **THEN** validation returns a violation from `deny_truncate`

#### Scenario: Truncate rule ignores another statement kind
- **WHEN** an Engine using only `rules.NewDenyTruncate()` validates a syntactically valid `DELETE` statement
- **THEN** the truncate rule allows validation to continue

### Requirement: DROP TABLE statements can be denied
A rule returned by `rules.NewDenyDropTable` MUST reject every PostgreSQL
`DROP TABLE` statement, including conditional and multi-table forms. It MUST
allow drop operations for other object types and non-drop statement kinds to
continue to other registered rules.

#### Scenario: Table drop is rejected
- **WHEN** an Engine using `rules.NewDenyDropTable()` validates `DROP TABLE accounts`
- **THEN** validation returns a violation from `deny_drop_table`

#### Scenario: Conditional multi-table drop is rejected
- **WHEN** an Engine using `rules.NewDenyDropTable()` validates `DROP TABLE IF EXISTS accounts, sessions CASCADE`
- **THEN** validation returns a violation from `deny_drop_table`

#### Scenario: Drop of another object type is allowed
- **WHEN** an Engine using only `rules.NewDenyDropTable()` validates `DROP VIEW account_summary`
- **THEN** the drop-table rule allows validation to continue

#### Scenario: Drop-table rule ignores another statement kind
- **WHEN** an Engine using only `rules.NewDenyDropTable()` validates a syntactically valid `TRUNCATE` statement
- **THEN** the drop-table rule allows validation to continue

### Requirement: ALTER TABLE statements can be denied
A rule returned by `rules.NewDenyAlterTable` MUST reject every PostgreSQL
`ALTER TABLE` statement regardless of the selected subcommand or table target.
It MUST allow other `ALTER` families and non-alter statement kinds to continue
to other registered rules.

#### Scenario: Add-column alteration is rejected
- **WHEN** an Engine using `rules.NewDenyAlterTable()` validates `ALTER TABLE accounts ADD COLUMN archived_at timestamptz`
- **THEN** validation returns a violation from `deny_alter_table`

#### Scenario: Conditional alteration is rejected
- **WHEN** an Engine using `rules.NewDenyAlterTable()` validates `ALTER TABLE IF EXISTS accounts DROP COLUMN legacy_id`
- **THEN** validation returns a violation from `deny_alter_table`

#### Scenario: Tablespace-wide alteration is rejected
- **WHEN** an Engine using `rules.NewDenyAlterTable()` validates `ALTER TABLE ALL IN TABLESPACE old_space SET TABLESPACE new_space`
- **THEN** validation returns a violation from `deny_alter_table`

#### Scenario: Other ALTER family is allowed
- **WHEN** an Engine using only `rules.NewDenyAlterTable()` validates `ALTER ROLE app_user SET statement_timeout = '5s'`
- **THEN** the alter-table rule allows validation to continue

#### Scenario: Alter-table rule ignores another statement kind
- **WHEN** an Engine using only `rules.NewDenyAlterTable()` validates a syntactically valid `DROP TABLE` statement
- **THEN** the alter-table rule allows validation to continue
