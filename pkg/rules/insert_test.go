package rules_test

import (
	"testing"

	"github.com/almostinf/postgres-sqlguard/pkg/rules"
)

func TestInsertRequiresColumns(t *testing.T) {
	engine := mustNewEngine(t, rules.NewInsertRequiresColumns())

	tests := map[string]validationTestCase{
		"allows_insert_with_explicit_columns": {
			input:      "INSERT INTO accounts (id, active) VALUES (42, false)",
			checkError: requireNoValidationError,
		},
		"allows_non_insert_statement": {
			input:      "UPDATE accounts SET active = false",
			checkError: requireNoValidationError,
		},
		"rejects_default_values_without_columns": {
			input:      "INSERT INTO accounts DEFAULT VALUES",
			checkError: requireViolation("insert_requires_columns"),
		},
		"rejects_insert_select_without_columns": {
			input:      "INSERT INTO accounts SELECT id, active FROM staged_accounts",
			checkError: requireViolation("insert_requires_columns"),
		},
		"rejects_values_without_columns": {
			input:      "INSERT INTO accounts VALUES (42, false)",
			checkError: requireViolation("insert_requires_columns"),
		},
	}

	runValidationTests(t, engine, tests)
}

func TestInsertRequiresColumnsCoversCompleteInput(t *testing.T) {
	engine := mustNewEngine(t, rules.NewInsertRequiresColumns())

	tests := map[string]validationTestCase{
		"allows_explicit_columns_across_complete_input": {
			input: `
				INSERT INTO accounts (id, active) VALUES (42, false);
				WITH inserted AS (
					INSERT INTO audit_log (message) VALUES ('created') RETURNING *
				)
				SELECT * FROM inserted;
				INSERT INTO archived_accounts (id, active)
				SELECT id, active FROM accounts;
			`,
			checkError: requireNoValidationError,
		},
		"rejects_insert_without_columns_in_cte": {
			input: `
				WITH inserted AS (
					INSERT INTO accounts VALUES (42, false) RETURNING *
				)
				SELECT * FROM inserted;
			`,
			checkError: requireViolation("insert_requires_columns"),
		},
	}

	runValidationTests(t, engine, tests)
}
