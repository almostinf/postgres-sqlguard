package rules_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	sqlguard "github.com/almostinf/postgres-sqlguard"
	"github.com/almostinf/postgres-sqlguard/rules"
)

type customRejectRule struct {
	calls *atomic.Int64
}

func (customRejectRule) ID() string {
	return "custom_mutation_guard"
}

func (r customRejectRule) Evaluate(context.Context, sqlguard.Statement) sqlguard.RuleResult {
	r.calls.Add(1)

	return sqlguard.Reject()
}

func TestMutationRulesPreserveExtensionBoundary(t *testing.T) {
	customFirstCalls := &atomic.Int64{}
	builtinFirstCalls := &atomic.Int64{}

	tests := map[string]struct {
		registeredRules []sqlguard.Rule
		input           string
		customCalls     *atomic.Int64
		wantCustomCalls int64
		setupTest       func(t *testing.T)
		checkError      func(t *testing.T, err error)
		checkResult     func(t *testing.T, calls *atomic.Int64, wantCalls int64)
	}{
		"builtin_rejects_before_custom_rule": {
			registeredRules: []sqlguard.Rule{
				rules.NewUpdateRequiresWhere(),
				customRejectRule{calls: builtinFirstCalls},
				rules.NewDeleteRequiresWhere(),
			},
			input:           "UPDATE accounts SET active = false",
			customCalls:     builtinFirstCalls,
			wantCustomCalls: 0,
			checkError:      requireViolation("update_requires_where"),
			checkResult: func(t *testing.T, calls *atomic.Int64, wantCalls int64) {
				t.Helper()

				require.Equal(t, wantCalls, calls.Load())
			},
		},
		"custom_rule_rejects_before_builtins": {
			registeredRules: []sqlguard.Rule{
				customRejectRule{calls: customFirstCalls},
				rules.NewUpdateRequiresWhere(),
				rules.NewDeleteRequiresWhere(),
			},
			input:           "UPDATE accounts SET active = false",
			customCalls:     customFirstCalls,
			wantCustomCalls: 1,
			checkError:      requireViolation("custom_mutation_guard"),
			checkResult: func(t *testing.T, calls *atomic.Int64, wantCalls int64) {
				t.Helper()

				require.Equal(t, wantCalls, calls.Load())
			},
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if testCase.setupTest != nil {
				testCase.setupTest(t)
			}

			engine := mustNewEngine(t, testCase.registeredRules...)
			err := engine.Validate(context.Background(), testCase.input)

			require.NotNil(t, testCase.checkError, "test case must define checkError")
			testCase.checkError(t, err)

			if testCase.checkResult != nil {
				testCase.checkResult(t, testCase.customCalls, testCase.wantCustomCalls)
			}
		})
	}
}
