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
	)

	tests := map[string]validationTestCase{
		"allows_mutation_text_in_select_literal": {
			input:      "SELECT 'UPDATE accounts SET active = false; DELETE FROM accounts'",
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
