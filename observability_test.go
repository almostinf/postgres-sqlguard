package sqlguard

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidationEvent(t *testing.T) {
	tests := map[string]struct {
		mode        ValidationMode
		outcome     ValidationOutcome
		ruleID      string
		wantMode    string
		wantOutcome string
		wantRuleID  string
	}{
		"allowed_has_no_rule_identifier": {
			mode:        ValidationModeEnforce,
			outcome:     ValidationOutcomeAllowed,
			ruleID:      "ignored_rule",
			wantMode:    "enforce",
			wantOutcome: "allowed",
		},
		"cancellation_has_no_rule_identifier": {
			mode:        ValidationModeEnforce,
			outcome:     ValidationOutcomeCanceled,
			ruleID:      "ignored_rule",
			wantMode:    "enforce",
			wantOutcome: "canceled",
		},
		"parser_failure_has_no_rule_identifier": {
			mode:        ValidationModeEnforce,
			outcome:     ValidationOutcomeParserFailure,
			ruleID:      "ignored_rule",
			wantMode:    "enforce",
			wantOutcome: "parser_failure",
		},
		"invalid_prepared_has_no_rule_identifier": {
			mode:        ValidationModeEnforce,
			outcome:     ValidationOutcomeInvalidPrepared,
			ruleID:      "ignored_rule",
			wantMode:    "enforce",
			wantOutcome: "invalid_prepared",
		},
		"policy_violation_preserves_rule_identifier": {
			mode:        ValidationModeEnforce,
			outcome:     ValidationOutcomePolicyViolation,
			ruleID:      "deny_delete",
			wantMode:    "enforce",
			wantOutcome: "policy_violation",
			wantRuleID:  "deny_delete",
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			event := newValidationEvent(testCase.mode, testCase.outcome, testCase.ruleID)

			require.Equal(t, testCase.mode, event.Mode())
			require.Equal(t, testCase.outcome, event.Outcome())
			require.Equal(t, testCase.wantMode, string(event.Mode()))
			require.Equal(t, testCase.wantOutcome, string(event.Outcome()))
			require.Equal(t, testCase.wantRuleID, event.RuleID())
		})
	}
}

func TestValidationEventHasNoExportedFields(t *testing.T) {
	eventType := reflect.TypeFor[ValidationEvent]()

	for index := range eventType.NumField() {
		field := eventType.Field(index)
		require.NotEmpty(t, field.PkgPath, "field %q must not be exported", field.Name)
	}
}
