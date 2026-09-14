package rules_test

import (
	"testing"

	"github.com/almostinf/postgres-sqlguard/rules"
)

func TestMutationRulesUseParsedStructure(t *testing.T) {
	engine := mustNewEngine(
		t,
		rules.NewUpdateRequiresWhere(),
		rules.NewDeleteRequiresWhere(),
		rules.NewInsertRequiresColumns(),
		rules.NewDenyTruncate(),
		rules.NewDenyDropTable(),
		rules.NewDenyAlterTable(),
	)

	tests := map[string]validationTestCase{
		"allows_alter_index": {
			input:      "ALTER INDEX account_idx RENAME TO account_idx_old",
			checkError: requireNoValidationError,
		},
		"allows_drop_view": {
			input:      "DROP VIEW account_summary",
			checkError: requireNoValidationError,
		},
		"allows_mutation_text_in_comment": {
			input:      "SELECT 1 /* INSERT DROP TABLE ALTER TABLE TRUNCATE UPDATE DELETE WHERE */",
			checkError: requireNoValidationError,
		},
		"allows_mutation_text_in_select_literal": {
			input:      "SELECT 'INSERT DROP TABLE ALTER TABLE TRUNCATE UPDATE DELETE WHERE'",
			checkError: requireNoValidationError,
		},
		"rejects_delete_with_where_only_in_comment": {
			input:      "DELETE FROM accounts /* WHERE id = 42 */",
			checkError: requireViolation("delete_requires_where"),
		},
		"rejects_update_with_where_only_in_comment": {
			input:      "UPDATE accounts SET active = false /* WHERE id = 42 */",
			checkError: requireViolation("update_requires_where"),
		},
		"rejects_update_with_where_only_in_literal": {
			input:      "UPDATE accounts SET note = 'WHERE id = 42'",
			checkError: requireViolation("update_requires_where"),
		},
	}

	runValidationTests(t, engine, tests)
}

func TestMutationRulesCoverCompleteInput(t *testing.T) {
	engine := mustNewEngine(
		t,
		rules.NewUpdateRequiresWhere(),
		rules.NewDeleteRequiresWhere(),
		rules.NewInsertRequiresColumns(),
		rules.NewDenyTruncate(),
		rules.NewDenyDropTable(),
		rules.NewDenyAlterTable(),
	)

	tests := map[string]validationTestCase{
		"allows_safe_mutations_in_complete_input": {
			input: `
				UPDATE accounts SET active = false WHERE id = 42;
				WITH changed AS (
					WITH removed AS (
						DELETE FROM sessions WHERE expired RETURNING *
					)
					UPDATE accounts SET active = false
					FROM removed
					WHERE accounts.id = removed.account_id
					RETURNING accounts.*
				)
				SELECT * FROM changed;
				DELETE FROM obsolete_rows WHERE archived;
			`,
			checkError: requireNoValidationError,
		},
		"rejects_delete_in_nested_cte": {
			input: `
				WITH changed AS (
					WITH removed AS (
						DELETE FROM sessions RETURNING *
					)
					UPDATE accounts SET active = false
					FROM removed
					WHERE accounts.id = removed.account_id
					RETURNING accounts.*
				)
				SELECT * FROM changed;
			`,
			checkError: requireViolation("delete_requires_where"),
		},
		"rejects_later_delete_statement": {
			input:      "SELECT 1; DELETE FROM accounts",
			checkError: requireViolation("delete_requires_where"),
		},
		"rejects_later_update_statement": {
			input:      "SELECT 1; UPDATE accounts SET active = false",
			checkError: requireViolation("update_requires_where"),
		},
		"rejects_later_alter_table_statement": {
			input:      "SELECT 1; ALTER TABLE accounts DROP COLUMN legacy_id",
			checkError: requireViolation("deny_alter_table"),
		},
		"rejects_later_drop_table_statement": {
			input:      "SELECT 1; DROP TABLE accounts",
			checkError: requireViolation("deny_drop_table"),
		},
		"rejects_later_insert_statement": {
			input:      "SELECT 1; INSERT INTO accounts VALUES (42, false)",
			checkError: requireViolation("insert_requires_columns"),
		},
		"rejects_later_truncate_statement": {
			input:      "SELECT 1; TRUNCATE TABLE accounts",
			checkError: requireViolation("deny_truncate"),
		},
		"rejects_update_in_cte": {
			input: `
				WITH changed AS (
					UPDATE accounts SET active = false RETURNING *
				)
				SELECT * FROM changed;
			`,
			checkError: requireViolation("update_requires_where"),
		},
	}

	runValidationTests(t, engine, tests)
}

func TestMutationRulesSelectFirstBuiltinViolation(t *testing.T) {
	engine := mustNewEngine(
		t,
		rules.NewUpdateRequiresWhere(),
		rules.NewDeleteRequiresWhere(),
		rules.NewInsertRequiresColumns(),
		rules.NewDenyTruncate(),
		rules.NewDenyDropTable(),
		rules.NewDenyAlterTable(),
	)

	tests := map[string]validationTestCase{
		"selects_drop_before_later_truncate": {
			input:      "SELECT 1; DROP TABLE accounts; TRUNCATE TABLE sessions",
			checkError: requireViolation("deny_drop_table"),
		},
		"selects_insert_before_later_update": {
			input:      "INSERT INTO accounts VALUES (42, false); UPDATE accounts SET active = false",
			checkError: requireViolation("insert_requires_columns"),
		},
		"selects_update_before_later_insert": {
			input:      "UPDATE accounts SET active = false; INSERT INTO accounts VALUES (42, false)",
			checkError: requireViolation("update_requires_where"),
		},
	}

	runValidationTests(t, engine, tests)
}
