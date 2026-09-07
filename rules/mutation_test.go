package rules_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	sqlguard "github.com/almostinf/postgres-sqlguard"
	"github.com/almostinf/postgres-sqlguard/rules"
)

type validationTestCase struct {
	input       string
	setupTest   func(t *testing.T)
	checkError  func(t *testing.T, err error)
	checkResult func(t *testing.T)
}

func TestMutationRuleConstructors(t *testing.T) {
	tests := map[string]struct {
		newRule     func() sqlguard.Rule
		wantRuleID  string
		setupTest   func(t *testing.T)
		checkResult func(t *testing.T, rule sqlguard.Rule, wantRuleID string)
	}{
		"creates_delete_requires_where": {
			newRule:    rules.NewDeleteRequiresWhere,
			wantRuleID: "delete_requires_where",
			checkResult: func(t *testing.T, rule sqlguard.Rule, wantRuleID string) {
				t.Helper()

				require.NotNil(t, rule)
				require.Equal(t, wantRuleID, rule.ID())
			},
		},
		"creates_update_requires_where": {
			newRule:    rules.NewUpdateRequiresWhere,
			wantRuleID: "update_requires_where",
			checkResult: func(t *testing.T, rule sqlguard.Rule, wantRuleID string) {
				t.Helper()

				require.NotNil(t, rule)
				require.Equal(t, wantRuleID, rule.ID())
			},
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if testCase.setupTest != nil {
				testCase.setupTest(t)
			}

			rule := testCase.newRule()

			testCase.checkResult(t, rule, testCase.wantRuleID)
		})
	}
}

func TestUpdateRequiresWhere(t *testing.T) {
	engine := mustNewEngine(t, rules.NewUpdateRequiresWhere())

	tests := map[string]validationTestCase{
		"allows_delete_statement": {
			input:      "DELETE FROM accounts",
			checkError: requireNoValidationError,
		},
		"allows_update_with_predicate": {
			input:      "UPDATE accounts SET active = false WHERE id = 42",
			checkError: requireNoValidationError,
		},
		"allows_update_with_where_true": {
			input:      "UPDATE accounts SET active = false WHERE TRUE",
			checkError: requireNoValidationError,
		},
		"rejects_update_without_where": {
			input:      "UPDATE accounts SET active = false",
			checkError: requireViolation("update_requires_where"),
		},
	}

	runValidationTests(t, engine, tests)
}

func TestDeleteRequiresWhere(t *testing.T) {
	engine := mustNewEngine(t, rules.NewDeleteRequiresWhere())

	tests := map[string]validationTestCase{
		"allows_delete_with_predicate": {
			input:      "DELETE FROM accounts WHERE id = 42",
			checkError: requireNoValidationError,
		},
		"allows_delete_with_where_true": {
			input:      "DELETE FROM accounts WHERE TRUE",
			checkError: requireNoValidationError,
		},
		"allows_update_statement": {
			input:      "UPDATE accounts SET active = false",
			checkError: requireNoValidationError,
		},
		"rejects_delete_without_where": {
			input:      "DELETE FROM accounts",
			checkError: requireViolation("delete_requires_where"),
		},
	}

	runValidationTests(t, engine, tests)
}

func TestMutationRulesAreOptIn(t *testing.T) {
	engine, err := sqlguard.NewEngine(sqlguard.EngineOptions{})
	require.NoError(t, err)

	tests := map[string]validationTestCase{
		"allows_delete_without_where": {
			input: "DELETE FROM accounts",
			checkError: func(t *testing.T, err error) {
				t.Helper()

				require.NoError(t, err)
			},
		},
		"allows_update_without_where": {
			input: "UPDATE accounts SET active = false",
			checkError: func(t *testing.T, err error) {
				t.Helper()

				require.NoError(t, err)
			},
		},
	}

	runValidationTests(t, engine, tests)
}

func runValidationTests(
	t *testing.T,
	engine *sqlguard.Engine,
	tests map[string]validationTestCase,
) {
	t.Helper()

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if testCase.setupTest != nil {
				testCase.setupTest(t)
			}

			err := engine.Validate(context.Background(), testCase.input)

			require.NotNil(t, testCase.checkError, "test case must define checkError")
			testCase.checkError(t, err)

			if testCase.checkResult != nil {
				testCase.checkResult(t)
			}
		})
	}
}

func mustNewEngine(t *testing.T, registeredRules ...sqlguard.Rule) *sqlguard.Engine {
	t.Helper()

	engine, err := sqlguard.NewEngine(sqlguard.EngineOptions{}, registeredRules...)
	require.NoError(t, err)

	return engine
}

func requireNoValidationError(t *testing.T, err error) {
	t.Helper()

	require.NoError(t, err)
}

func requireViolation(wantRuleID string) func(t *testing.T, err error) {
	return func(t *testing.T, err error) {
		t.Helper()

		require.Error(t, err)

		var violation *sqlguard.Violation
		require.ErrorAs(t, err, &violation)
		require.Equal(t, wantRuleID, violation.RuleID())
	}
}
