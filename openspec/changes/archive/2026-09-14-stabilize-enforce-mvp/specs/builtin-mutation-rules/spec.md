## MODIFIED Requirements

### Requirement: Public opt-in mutation rules
The public package at
`github.com/almostinf/postgres-sqlguard/pkg/rules` SHALL expose
`NewUpdateRequiresWhere`, `NewDeleteRequiresWhere`,
`NewInsertRequiresColumns`, `NewDenyTruncate`, `NewDenyDropTable`, and
`NewDenyAlterTable` constructors that return implementations of the root
package's existing rule contract. The module MUST NOT retain the former public
package at `github.com/almostinf/postgres-sqlguard/rules`. The returned rules'
stable identifiers SHALL be `update_requires_where`, `delete_requires_where`,
`insert_requires_columns`, `deny_truncate`, `deny_drop_table`, and
`deny_alter_table` respectively. An Engine MUST apply a built-in rule only when
the caller explicitly registers it.

#### Scenario: Rules are imported from the stable package path
- **WHEN** a consumer imports `github.com/almostinf/postgres-sqlguard/pkg/rules`
- **THEN** all six built-in rule constructors are available through the root package's existing rule contract

#### Scenario: Former rules package is removed
- **WHEN** the module packages for the stable release are enumerated
- **THEN** `github.com/almostinf/postgres-sqlguard/rules` is not present

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
