package sqlguard_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	sqlguard "github.com/almostinf/postgres-sqlguard"
)

type validateTestCase struct {
	engine      *sqlguard.Engine
	input       string
	setupTest   func(t *testing.T)
	checkResult func(t *testing.T)
	checkError  func(t *testing.T, err error)
}

func TestEngineValidate(t *testing.T) {
	acceptRule := &ruleSpy{id: "allow_all"}
	malformedRule := &ruleSpy{id: "must_not_run"}
	traversalRule := &ruleSpy{id: "record_statements"}
	firstViewRule := &ruleSpy{id: "first_view"}
	secondViewRule := &ruleSpy{id: "second_view"}

	callOrder := make([]string, 0)
	firstRule := &ruleSpy{
		id: "first_rule",
		decision: func(context.Context, sqlguard.Statement) sqlguard.RuleResult {
			callOrder = append(callOrder, "first_rule")

			return sqlguard.Allow()
		},
	}
	secondRule := &ruleSpy{
		id: "second_rule",
		decision: func(context.Context, sqlguard.Statement) sqlguard.RuleResult {
			callOrder = append(callOrder, "second_rule")

			return sqlguard.Reject()
		},
	}
	laterRule := &ruleSpy{
		id: "later_rule",
		decision: func(context.Context, sqlguard.Statement) sqlguard.RuleResult {
			callOrder = append(callOrder, "later_rule")

			return sqlguard.Reject()
		},
	}
	privacyRule := &ruleSpy{
		id: "deny_sensitive_query",
		decision: func(context.Context, sqlguard.Statement) sqlguard.RuleResult {
			return sqlguard.Reject()
		},
	}

	tests := map[string]validateTestCase{
		"accepts_postgresql_sql_without_a_driver": {
			engine: mustNewEngine(t, acceptRule),
			input:  "SELECT ARRAY[1, 2]::int[] @> ARRAY[1]",
			checkResult: func(t *testing.T) {
				t.Helper()

				require.Len(t, acceptRule.Calls(), 1)
			},
			checkError: requireNoValidationError,
		},
		"fails_a_malformed_batch_before_rules_run": {
			engine: mustNewEngine(t, malformedRule),
			input:  "SELECT 1; SELECT * FROM",
			checkResult: func(t *testing.T) {
				t.Helper()

				require.Empty(t, malformedRule.Calls())
			},
			checkError: requireParseFailure,
		},
		"visits_top_level_and_nested_cte_statements": {
			engine: mustNewEngine(t, traversalRule),
			input: `
				SELECT 1;
				WITH inserted AS (
					INSERT INTO audit_log (message) VALUES ('created') RETURNING *
				), changed AS (
					WITH removed AS (
						DELETE FROM sessions WHERE expired RETURNING *
					)
					UPDATE accounts SET active = false
					FROM removed
					WHERE accounts.id = removed.account_id
					RETURNING accounts.*
				)
				SELECT * FROM inserted, changed;
				DELETE FROM obsolete_rows;
			`,
			checkResult: func(t *testing.T) {
				t.Helper()

				calls := traversalRule.Calls()

				kinds := make([]sqlguard.Kind, 0, len(calls))

				for _, call := range calls {
					kinds = append(kinds, call.kind)
				}

				require.Equal(t, []sqlguard.Kind{
					"SelectStmt",
					"SelectStmt",
					"InsertStmt",
					"UpdateStmt",
					"DeleteStmt",
					"DeleteStmt",
				}, kinds)
			},
			checkError: requireNoValidationError,
		},
		"shares_a_consistent_statement_view_between_rules": {
			engine: mustNewEngine(t, firstViewRule, secondViewRule),
			input:  "UPDATE public.accounts SET active = false",
			checkResult: func(t *testing.T) {
				t.Helper()

				firstCalls := firstViewRule.Calls()
				secondCalls := secondViewRule.Calls()

				require.Len(t, firstCalls, 1)
				require.Len(t, secondCalls, 1)
				require.Equal(t, firstCalls[0].kind, secondCalls[0].kind)
				require.Equal(t, firstCalls[0].structure, secondCalls[0].structure)
			},
			checkError: requireNoValidationError,
		},
		"stops_at_the_first_violation_in_registration_order": {
			engine: mustNewEngine(t, firstRule, secondRule, laterRule),
			input:  "SELECT 1; SELECT 2",
			checkResult: func(t *testing.T) {
				t.Helper()

				require.Equal(t, []string{"first_rule", "second_rule"}, callOrder)
				require.Empty(t, laterRule.Calls())
			},
			checkError: requireViolation("second_rule"),
		},
		"violation_omits_the_submitted_sql": {
			engine: mustNewEngine(t, privacyRule),
			input:  "SELECT 'sqlguard_secret_literal_f320ac'",
			checkResult: func(t *testing.T) {
				t.Helper()
			},
			checkError: func(t *testing.T, err error) {
				t.Helper()

				requireViolation("deny_sensitive_query")(t, err)
				requireErrorChainOmits(t, err, "sqlguard_secret_literal_f320ac")
			},
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if testCase.setupTest != nil {
				testCase.setupTest(t)
			}

			err := testCase.engine.Validate(context.Background(), testCase.input)

			require.NotNil(t, testCase.checkError, "test case must define checkError")
			testCase.checkError(t, err)

			if testCase.checkResult != nil {
				testCase.checkResult(t)
			}
		})
	}
}

func TestEngineFirstViolationIsReproducible(t *testing.T) {
	firstRule := &ruleSpy{
		id: "first_rejecting_rule",
		decision: func(context.Context, sqlguard.Statement) sqlguard.RuleResult {
			return sqlguard.Reject()
		},
	}
	laterRule := &ruleSpy{
		id: "later_rejecting_rule",
		decision: func(context.Context, sqlguard.Statement) sqlguard.RuleResult {
			return sqlguard.Reject()
		},
	}
	engine := mustNewEngine(t, firstRule, laterRule)

	tests := map[string]struct {
		input       string
		runs        int
		setupTest   func(t *testing.T)
		checkResult func(t *testing.T)
		checkError  func(t *testing.T, errors []error)
	}{
		"selects_the_same_first_violation_on_repeated_validation": {
			input: "SELECT 1; DELETE FROM accounts",
			runs:  2,
			checkResult: func(t *testing.T) {
				t.Helper()

				require.Len(t, firstRule.Calls(), 2)
				require.Empty(t, laterRule.Calls())
			},
			checkError: func(t *testing.T, errors []error) {
				t.Helper()

				require.Len(t, errors, 2)

				for _, err := range errors {
					requireViolation("first_rejecting_rule")(t, err)
				}
			},
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if testCase.setupTest != nil {
				testCase.setupTest(t)
			}

			errors := make([]error, 0, testCase.runs)
			for range testCase.runs {
				errors = append(errors, engine.Validate(context.Background(), testCase.input))
			}

			require.NotNil(t, testCase.checkError, "test case must define checkError")
			testCase.checkError(t, errors)

			if testCase.checkResult != nil {
				testCase.checkResult(t)
			}
		})
	}
}
