package rules_test

import (
	"testing"

	"github.com/almostinf/postgres-sqlguard/rules"
)

func TestDenyTruncate(t *testing.T) {
	engine := mustNewEngine(t, rules.NewDenyTruncate())

	tests := map[string]validationTestCase{
		"allows_non_truncate_statement": {
			input:      "DELETE FROM accounts",
			checkError: requireNoValidationError,
		},
		"rejects_multi_table_truncate_with_options": {
			input:      "TRUNCATE accounts, sessions RESTART IDENTITY CASCADE",
			checkError: requireViolation("deny_truncate"),
		},
		"rejects_single_table_truncate": {
			input:      "TRUNCATE TABLE accounts",
			checkError: requireViolation("deny_truncate"),
		},
	}

	runValidationTests(t, engine, tests)
}

func TestDenyDropTable(t *testing.T) {
	engine := mustNewEngine(t, rules.NewDenyDropTable())

	tests := map[string]validationTestCase{
		"allows_drop_view": {
			input:      "DROP VIEW account_summary",
			checkError: requireNoValidationError,
		},
		"allows_non_drop_statement": {
			input:      "TRUNCATE TABLE accounts",
			checkError: requireNoValidationError,
		},
		"rejects_conditional_multi_table_drop": {
			input:      "DROP TABLE IF EXISTS accounts, sessions CASCADE",
			checkError: requireViolation("deny_drop_table"),
		},
		"rejects_table_drop": {
			input:      "DROP TABLE accounts",
			checkError: requireViolation("deny_drop_table"),
		},
	}

	runValidationTests(t, engine, tests)
}

func TestDenyAlterTable(t *testing.T) {
	engine := mustNewEngine(t, rules.NewDenyAlterTable())

	tests := map[string]validationTestCase{
		"allows_alter_role": {
			input:      "ALTER ROLE app_user SET statement_timeout = '5s'",
			checkError: requireNoValidationError,
		},
		"allows_non_alter_statement": {
			input:      "DROP TABLE accounts",
			checkError: requireNoValidationError,
		},
		"rejects_add_column": {
			input:      "ALTER TABLE accounts ADD COLUMN archived_at timestamptz",
			checkError: requireViolation("deny_alter_table"),
		},
		"rejects_conditional_alteration": {
			input:      "ALTER TABLE IF EXISTS accounts DROP COLUMN legacy_id",
			checkError: requireViolation("deny_alter_table"),
		},
		"rejects_tablespace_wide_alteration": {
			input:      "ALTER TABLE ALL IN TABLESPACE old_space SET TABLESPACE new_space",
			checkError: requireViolation("deny_alter_table"),
		},
	}

	runValidationTests(t, engine, tests)
}
