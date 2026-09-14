## 1. Public Rule Contracts

- [x] 1.1 Add black-box constructor contract cases proving that `NewInsertRequiresColumns`, `NewDenyTruncate`, `NewDenyDropTable`, and `NewDenyAlterTable` return `sqlguard.Rule` values with the specified stable identifiers.
- [x] 1.2 Add private immutable rule types, public GoDoc constructors, rule-ID constants, and compile-time `sqlguard.Rule` assertions in focused `rules/insert.go` and `rules/ddl.go` files.
- [x] 1.3 Add a public Engine case proving that none of the four new policies is applied unless its rule is explicitly registered.

## 2. INSERT Requires Explicit Columns

- [x] 2.1 Add parallel table-driven black-box cases for positional `VALUES`, positional `INSERT SELECT`, `DEFAULT VALUES`, an explicit target-column list, and a non-`INSERT` statement, asserting the typed `insert_requires_columns` violation where applicable.
- [x] 2.2 Add private INSERT kind and column-field constants and implement `NewInsertRequiresColumns` using `InsertStmt` root kind and non-empty `cols` children, rejecting a missing or empty list and allowing non-target statements.
- [x] 2.3 Add Engine-level cases for an unsafe `INSERT` inside a statement-bearing CTE and safe explicit-column inserts across complete multi-statement input.

## 3. Destructive DDL Deny Rules

- [x] 3.1 Add parallel table-driven black-box cases proving `NewDenyTruncate` rejects single- and multi-table `TRUNCATE` forms with options while allowing non-target statements.
- [x] 3.2 Add the private TRUNCATE kind constant and implement `NewDenyTruncate` as a stateless `TruncateStmt` kind rejection with no SQL-text or object-name inspection.
- [x] 3.3 Add parallel table-driven black-box cases proving `NewDenyDropTable` rejects ordinary and conditional multi-table drops while allowing `DROP VIEW` and non-drop statement kinds.
- [x] 3.4 Add private DROP kind, object-type field, and table-enum constants and implement `NewDenyDropTable` from `DropStmt.remove_type`, rejecting `OBJECT_TABLE` and missing or unreadable discriminators while allowing present non-table object types.
- [x] 3.5 Add parallel table-driven black-box cases proving `NewDenyAlterTable` rejects ordinary, conditional, and `ALL IN TABLESPACE` forms while allowing `ALTER ROLE` and non-alter statement kinds.
- [x] 3.6 Add private ALTER kind, object-type field, and table-enum constants and implement `NewDenyAlterTable` for both `AlterTableStmt` and `AlterTableMoveAllStmt` with `objtype=OBJECT_TABLE`, rejecting missing or unreadable discriminators and allowing present non-table object types.

## 4. Structural, Ordering, and Concurrency Coverage

- [x] 4.1 Add parsed-structure cases proving mutation keywords in comments and `SELECT` literals do not trigger any new rule and that adjacent DDL object families remain distinguishable.
- [x] 4.2 Add multi-statement cases proving each new rule rejects a matching later statement through the Engine's established traversal.
- [x] 4.3 Register new and existing built-ins together and verify deterministic first-violation behavior without changes to Engine, parser traversal, or typed error handling.
- [x] 4.4 Extend concurrent validation coverage to reuse all new rule instances across allowed and rejected inputs without rule-owned mutable state.

## 5. Consumer Documentation

- [x] 5.1 Expand the README built-in-rules section with explicit registration of the four new constructors and concise descriptions of their stable policies.
- [x] 5.2 Document the syntactic-only INSERT guarantee, operation-family scope of the DDL deny rules, opt-in behavior, and composition with the existing UPDATE/DELETE rules.
