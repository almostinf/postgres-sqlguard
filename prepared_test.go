package sqlguard_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	sqlguard "github.com/almostinf/postgres-sqlguard"
)

type preparedValidator interface {
	Prepare(context.Context, string) (sqlguard.Prepared, error)
	ValidatePrepared(context.Context, sqlguard.Prepared) error
}

var (
	_ sqlguard.Validator = validatorFunc(nil)
	_ preparedValidator  = (*sqlguard.Engine)(nil)
)

func TestEnginePrepare(t *testing.T) {
	tests := map[string]struct {
		ctx         func() context.Context
		input       string
		checkResult func(t *testing.T, prepared sqlguard.Prepared, rule *ruleSpy)
		checkError  func(t *testing.T, err error)
	}{
		"defers_rules_for_valid_input": {
			ctx:   context.Background,
			input: "SELECT ARRAY[1, 2]::int[] @> ARRAY[1]",
			checkResult: func(t *testing.T, prepared sqlguard.Prepared, rule *ruleSpy) {
				t.Helper()

				require.NotEqual(t, sqlguard.Prepared{}, prepared)
				require.Empty(t, rule.Calls())
			},
			checkError: requireNoValidationError,
		},
		"rejects_malformed_input_before_rules": {
			ctx:   context.Background,
			input: "SELECT 'sqlguard_prepare_secret_13c94d' FROM",
			checkResult: func(t *testing.T, prepared sqlguard.Prepared, rule *ruleSpy) {
				t.Helper()

				require.Equal(t, sqlguard.Prepared{}, prepared)
				require.Empty(t, rule.Calls())
			},
			checkError: func(t *testing.T, err error) {
				t.Helper()

				requireParseFailure(t, err)
				requireErrorChainOmits(t, err, "sqlguard_prepare_secret_13c94d")
			},
		},
		"returns_preexisting_cancellation": {
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				return ctx
			},
			input: "SELECT 1",
			checkResult: func(t *testing.T, prepared sqlguard.Prepared, rule *ruleSpy) {
				t.Helper()

				require.Equal(t, sqlguard.Prepared{}, prepared)
				require.Empty(t, rule.Calls())
			},
			checkError: requireContextError(context.Canceled),
		},
		"prioritizes_cancellation_after_malformed_parse": {
			ctx: func() context.Context {
				return &cancelAfterParseContext{Context: context.Background()}
			},
			input: "SELECT * FROM",
			checkResult: func(t *testing.T, prepared sqlguard.Prepared, rule *ruleSpy) {
				t.Helper()

				require.Equal(t, sqlguard.Prepared{}, prepared)
				require.Empty(t, rule.Calls())
			},
			checkError: requireContextError(context.Canceled),
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rule := &ruleSpy{id: "deferred_rule"}
			engine := mustNewEngine(t, rule)
			prepared, err := engine.Prepare(testCase.ctx(), testCase.input)

			testCase.checkError(t, err)
			testCase.checkResult(t, prepared, rule)
		})
	}
}

func TestEnginePrepareTerminalOutcomes(t *testing.T) {
	tests := map[string]struct {
		ctx         func() context.Context
		input       string
		wantEvent   bool
		wantOutcome sqlguard.ValidationOutcome
		checkError  func(t *testing.T, err error)
	}{
		"successful_preparation_emits_nothing": {
			ctx:        context.Background,
			input:      "SELECT 1",
			checkError: requireNoValidationError,
		},
		"parser_failure_emits_once": {
			ctx:         context.Background,
			input:       "SELECT * FROM",
			wantEvent:   true,
			wantOutcome: sqlguard.ValidationOutcomeParserFailure,
			checkError:  requireParseFailure,
		},
		"cancellation_emits_once": {
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				return ctx
			},
			input:       "SELECT 1",
			wantEvent:   true,
			wantOutcome: sqlguard.ValidationOutcomeCanceled,
			checkError:  requireContextError(context.Canceled),
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			metrics := &recordingMetrics{}
			engine, err := sqlguard.NewEngine(sqlguard.EngineOptions{Metrics: metrics})
			require.NoError(t, err)

			_, err = engine.Prepare(testCase.ctx(), testCase.input)
			testCase.checkError(t, err)

			if !testCase.wantEvent {
				require.Empty(t, metrics.Events())

				return
			}

			requireSingleValidationEvent(t, metrics.Events(), validationEventExpectation{
				mode:    sqlguard.ValidationModeEnforce,
				outcome: testCase.wantOutcome,
			})
		})
	}
}

func TestEngineValidatePrepared(t *testing.T) {
	tests := map[string]struct {
		ctx         func() context.Context
		prepared    func(t *testing.T, engine *sqlguard.Engine) sqlguard.Prepared
		checkResult func(t *testing.T, rule *ruleSpy, metrics *recordingMetrics)
		checkError  func(t *testing.T, err error)
	}{
		"rejects_zero_value_privately": {
			ctx: context.Background,
			prepared: func(t *testing.T, _ *sqlguard.Engine) sqlguard.Prepared {
				t.Helper()

				return sqlguard.Prepared{}
			},
			checkResult: func(t *testing.T, rule *ruleSpy, metrics *recordingMetrics) {
				t.Helper()

				require.Empty(t, rule.Calls())
				requireSingleValidationEvent(t, metrics.Events(), validationEventExpectation{
					mode:    sqlguard.ValidationModeEnforce,
					outcome: sqlguard.ValidationOutcomeInvalidPrepared,
				})
			},
			checkError: func(t *testing.T, err error) {
				t.Helper()

				require.ErrorIs(t, err, sqlguard.ErrInvalidPrepared)
				require.Nil(t, errors.Unwrap(err))
				requireErrorChainOmits(t, err, "sqlguard_invalid_prepared_secret_a1d245")
			},
		},
		"prioritizes_cancellation_over_zero_value": {
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				return ctx
			},
			prepared: func(t *testing.T, _ *sqlguard.Engine) sqlguard.Prepared {
				t.Helper()

				return sqlguard.Prepared{}
			},
			checkResult: func(t *testing.T, rule *ruleSpy, metrics *recordingMetrics) {
				t.Helper()

				require.Empty(t, rule.Calls())
				requireSingleValidationEvent(t, metrics.Events(), validationEventExpectation{
					mode:    sqlguard.ValidationModeEnforce,
					outcome: sqlguard.ValidationOutcomeCanceled,
				})
			},
			checkError: requireContextError(context.Canceled),
		},
		"evaluates_a_valid_value": {
			ctx: context.Background,
			prepared: func(t *testing.T, engine *sqlguard.Engine) sqlguard.Prepared {
				t.Helper()

				prepared, err := engine.Prepare(context.Background(), "SELECT 1")
				require.NoError(t, err)

				return prepared
			},
			checkResult: func(t *testing.T, rule *ruleSpy, metrics *recordingMetrics) {
				t.Helper()

				require.Len(t, rule.Calls(), 1)
				requireSingleValidationEvent(t, metrics.Events(), validationEventExpectation{
					mode:    sqlguard.ValidationModeEnforce,
					outcome: sqlguard.ValidationOutcomeAllowed,
				})
			},
			checkError: requireNoValidationError,
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rule := &ruleSpy{id: "prepared_rule"}
			metrics := &recordingMetrics{}
			engine, err := sqlguard.NewEngine(sqlguard.EngineOptions{Metrics: metrics}, rule)
			require.NoError(t, err)

			prepared := testCase.prepared(t, engine)
			err = engine.ValidatePrepared(testCase.ctx(), prepared)

			testCase.checkError(t, err)
			testCase.checkResult(t, rule, metrics)
		})
	}
}

func TestEngineValidatePreparedReevaluatesRules(t *testing.T) {
	type decisionKey string

	const key decisionKey = "decision"

	rejectFromState := false
	rule := &ruleSpy{
		id: "context_decision",
		decision: func(ctx context.Context, _ sqlguard.Statement) sqlguard.RuleResult {
			rejectFromContext, _ := ctx.Value(key).(bool)
			if rejectFromState || rejectFromContext {
				return sqlguard.Reject()
			}

			return sqlguard.Allow()
		},
	}
	engine := mustNewEngine(t, rule)
	prepared, err := engine.Prepare(context.Background(), "SELECT 1")
	require.NoError(t, err)

	allowContext := context.WithValue(context.Background(), key, false)
	rejectContext := context.WithValue(context.Background(), key, true)

	require.NoError(t, engine.ValidatePrepared(allowContext, prepared))

	rejectFromState = true

	requireViolation("context_decision")(t, engine.ValidatePrepared(allowContext, prepared))

	rejectFromState = false

	requireViolation("context_decision")(t, engine.ValidatePrepared(rejectContext, prepared))

	calls := rule.Calls()
	require.Len(t, calls, 3)
	require.True(t, calls[0].ctx == allowContext)
	require.True(t, calls[1].ctx == allowContext)
	require.True(t, calls[2].ctx == rejectContext)
}

func TestEngineValidatePreparedStopsAtFirstViolation(t *testing.T) {
	callOrder := make([]string, 0, 2)
	firstRule := &ruleStub{
		id: "first_prepared_rule",
		evaluate: func(context.Context, sqlguard.Statement) sqlguard.RuleResult {
			callOrder = append(callOrder, "first_prepared_rule")

			return sqlguard.Reject()
		},
	}
	laterRule := &ruleStub{
		id: "later_prepared_rule",
		evaluate: func(context.Context, sqlguard.Statement) sqlguard.RuleResult {
			callOrder = append(callOrder, "later_prepared_rule")

			return sqlguard.Reject()
		},
	}
	engine := mustNewEngine(t, firstRule, laterRule)
	prepared, err := engine.Prepare(context.Background(), "SELECT 1; SELECT 2")
	require.NoError(t, err)

	err = engine.ValidatePrepared(context.Background(), prepared)
	requireViolation("first_prepared_rule")(t, err)
	require.Equal(t, []string{"first_prepared_rule"}, callOrder)
}

func TestEngineValidatePreparedStopsAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancelingRule := &ruleSpy{
		id: "cancel_prepared_validation",
		decision: func(context.Context, sqlguard.Statement) sqlguard.RuleResult {
			cancel()

			return sqlguard.Allow()
		},
	}
	laterRule := &ruleSpy{id: "unreached_prepared_rule"}
	engine := mustNewEngine(t, cancelingRule, laterRule)
	prepared, err := engine.Prepare(context.Background(), "SELECT 1")
	require.NoError(t, err)

	err = engine.ValidatePrepared(ctx, prepared)
	require.ErrorIs(t, err, context.Canceled)
	require.Len(t, cancelingRule.Calls(), 1)
	require.Empty(t, laterRule.Calls())
}

func TestDirectAndPreparedValidationMatch(t *testing.T) {
	tests := map[string]struct {
		input        string
		decision     func(context.Context, sqlguard.Statement) sqlguard.RuleResult
		setupContext func(t *testing.T) context.Context
		checkError   func(t *testing.T, direct error, prepared error)
	}{
		"allowed_input": {
			input: "SELECT 1",
			checkError: func(t *testing.T, direct error, prepared error) {
				t.Helper()

				require.NoError(t, direct)
				require.NoError(t, prepared)
			},
		},
		"policy_violation": {
			input: "DELETE FROM accounts",
			decision: func(context.Context, sqlguard.Statement) sqlguard.RuleResult {
				return sqlguard.Reject()
			},
			checkError: func(t *testing.T, direct error, prepared error) {
				t.Helper()

				requireViolation("comparison_rule")(t, direct)
				requireViolation("comparison_rule")(t, prepared)
			},
		},
		"parser_failure": {
			input: "SELECT * FROM",
			checkError: func(t *testing.T, direct error, prepared error) {
				t.Helper()

				requireParseFailure(t, direct)
				requireParseFailure(t, prepared)
			},
		},
		"cancellation": {
			input: "SELECT 1",
			setupContext: func(t *testing.T) context.Context {
				t.Helper()

				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				return ctx
			},
			checkError: func(t *testing.T, direct error, prepared error) {
				t.Helper()

				require.ErrorIs(t, direct, context.Canceled)
				require.ErrorIs(t, prepared, context.Canceled)
			},
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			directRule := &ruleSpy{id: "comparison_rule", decision: testCase.decision}
			preparedRule := &ruleSpy{id: "comparison_rule", decision: testCase.decision}
			directEngine := mustNewEngine(t, directRule)
			preparedEngine := mustNewEngine(t, preparedRule)

			ctx := context.Background()
			if testCase.setupContext != nil {
				ctx = testCase.setupContext(t)
			}

			directErr := directEngine.Validate(ctx, testCase.input)
			prepared, prepareErr := preparedEngine.Prepare(ctx, testCase.input)
			preparedErr := prepareErr

			if prepareErr == nil {
				preparedErr = preparedEngine.ValidatePrepared(ctx, prepared)
			}

			testCase.checkError(t, directErr, preparedErr)
		})
	}
}

func TestDirectAndPreparedTraversalMatch(t *testing.T) {
	inputs := map[string]string{
		"multi_statement": "SELECT 1; UPDATE accounts SET active = false; DELETE FROM sessions",
		"nested_cte": `
			WITH changed AS (
				WITH removed AS (
					DELETE FROM sessions WHERE expired RETURNING *
				)
				UPDATE accounts SET active = false FROM removed RETURNING accounts.*
			)
			SELECT * FROM changed
		`,
	}

	for name, input := range inputs {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			directRule := &ruleSpy{id: "direct_traversal"}
			preparedRule := &ruleSpy{id: "prepared_traversal"}
			directEngine := mustNewEngine(t, directRule)
			preparedEngine := mustNewEngine(t, preparedRule)

			require.NoError(t, directEngine.Validate(context.Background(), input))
			prepared, err := preparedEngine.Prepare(context.Background(), input)
			require.NoError(t, err)
			require.NoError(t, preparedEngine.ValidatePrepared(context.Background(), prepared))

			directCalls := directRule.Calls()
			preparedCalls := preparedRule.Calls()
			require.Len(t, preparedCalls, len(directCalls))

			for index := range directCalls {
				require.Equal(t, directCalls[index].kind, preparedCalls[index].kind)
				require.Equal(t, directCalls[index].structure, preparedCalls[index].structure)
			}
		})
	}
}

func TestPreparedValueConcurrentCrossEngineUse(t *testing.T) {
	const callCount = 64

	preparingEngine := mustNewEngine(t)
	prepared, err := preparingEngine.Prepare(context.Background(), "SELECT * FROM accounts")
	require.NoError(t, err)

	allowRule := &isolationRule{seen: make(map[string]int)}
	allowEngine := mustNewEngine(t, allowRule)
	rejectEngine := mustNewEngine(t, &ruleStub{
		id: "cross_engine_reject",
		evaluate: func(context.Context, sqlguard.Statement) sqlguard.RuleResult {
			return sqlguard.Reject()
		},
	})

	errorsByCall := make([]error, callCount)

	var waitGroup sync.WaitGroup
	waitGroup.Add(callCount)

	for index := range callCount {
		go func() {
			defer waitGroup.Done()

			if index%2 == 0 {
				ctx := context.WithValue(context.Background(), expectedRelationKey, "accounts")
				errorsByCall[index] = allowEngine.ValidatePrepared(ctx, prepared)

				return
			}

			errorsByCall[index] = rejectEngine.ValidatePrepared(context.Background(), prepared)
		}()
	}

	waitGroup.Wait()

	for index, validationErr := range errorsByCall {
		if index%2 == 0 {
			require.NoError(t, validationErr)
		} else {
			requireViolation("cross_engine_reject")(t, validationErr)
		}
	}

	seen, mismatches := allowRule.Snapshot()
	require.Empty(t, mismatches)
	require.Equal(t, map[string]int{"accounts": callCount / 2}, seen)

	// A final validation after concurrent reuse guards against accidental parsed-state mutation.
	require.NoError(t, allowEngine.ValidatePrepared(
		context.WithValue(context.Background(), expectedRelationKey, "accounts"),
		prepared,
	))

	seen, mismatches = allowRule.Snapshot()
	require.Empty(t, mismatches)
	require.Equal(t, map[string]int{"accounts": callCount/2 + 1}, seen)
}

func TestValidateCompositionEmitsOneTerminalEvent(t *testing.T) {
	tests := map[string]struct {
		input      string
		want       sqlguard.ValidationOutcome
		checkError func(t *testing.T, err error)
	}{
		"allowed": {
			input:      "SELECT 1",
			want:       sqlguard.ValidationOutcomeAllowed,
			checkError: requireNoValidationError,
		},
		"parser_failure": {
			input:      "SELECT * FROM",
			want:       sqlguard.ValidationOutcomeParserFailure,
			checkError: requireParseFailure,
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			metrics := &recordingMetrics{}
			engine, err := sqlguard.NewEngine(sqlguard.EngineOptions{Metrics: metrics})
			require.NoError(t, err)

			err = engine.Validate(context.Background(), testCase.input)
			testCase.checkError(t, err)
			requireSingleValidationEvent(t, metrics.Events(), validationEventExpectation{
				mode:    sqlguard.ValidationModeEnforce,
				outcome: testCase.want,
			})
		})
	}
}

func TestErrInvalidPreparedWrapping(t *testing.T) {
	err := fmt.Errorf("validation failed: %w", sqlguard.ErrInvalidPrepared)

	require.ErrorIs(t, err, sqlguard.ErrInvalidPrepared)
}
