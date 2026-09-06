package sqlguard_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	sqlguard "github.com/almostinf/postgres-sqlguard"
)

type customRule struct {
	result sqlguard.RuleResult
}

func (customRule) ID() string {
	return "custom-rule"
}

func (r customRule) Evaluate(context.Context, sqlguard.Statement) sqlguard.RuleResult {
	return r.result
}

type validatorFunc func(context.Context, string) error

func (validate validatorFunc) Validate(ctx context.Context, sql string) error {
	return validate(ctx, sql)
}

var (
	_ sqlguard.Rule      = customRule{}
	_ sqlguard.Validator = validatorFunc(nil)
)

func TestRuleResult(t *testing.T) {
	tests := map[string]struct {
		result      sqlguard.RuleResult
		wantReject  bool
		setupTest   func(t *testing.T)
		checkResult func(t *testing.T, result sqlguard.RuleResult, wantReject bool)
	}{
		"allow_decision": {
			result:     sqlguard.Allow(),
			wantReject: false,
			checkResult: func(t *testing.T, result sqlguard.RuleResult, wantReject bool) {
				t.Helper()

				require.Equal(t, wantReject, result.Rejected())
			},
		},
		"reject_decision": {
			result:     sqlguard.Reject(),
			wantReject: true,
			checkResult: func(t *testing.T, result sqlguard.RuleResult, wantReject bool) {
				t.Helper()

				require.Equal(t, wantReject, result.Rejected())
			},
		},
		"zero_value_rejects_closed": {
			result:     sqlguard.RuleResult{},
			wantReject: true,
			checkResult: func(t *testing.T, result sqlguard.RuleResult, wantReject bool) {
				t.Helper()

				require.Equal(t, wantReject, result.Rejected())
			},
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if testCase.setupTest != nil {
				testCase.setupTest(t)
			}

			testCase.checkResult(t, testCase.result, testCase.wantReject)
		})
	}
}

func TestCustomRuleUsesPublicContract(t *testing.T) {
	tests := map[string]struct {
		rule        sqlguard.Rule
		setupTest   func(t *testing.T)
		checkResult func(t *testing.T, rule sqlguard.Rule)
	}{
		"implements_rule_without_parser_backend": {
			rule: customRule{result: sqlguard.Allow()},
			checkResult: func(t *testing.T, rule sqlguard.Rule) {
				t.Helper()

				require.Equal(t, "custom-rule", rule.ID())
				require.False(t, rule.Evaluate(context.Background(), sqlguard.Statement{}).Rejected())
			},
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if testCase.setupTest != nil {
				testCase.setupTest(t)
			}

			testCase.checkResult(t, testCase.rule)
		})
	}
}
