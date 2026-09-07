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

type recordingMetrics struct {
	mutex      sync.Mutex
	events     []sqlguard.ValidationEvent
	failure    error
	panicValue any
}

func (m *recordingMetrics) RecordValidation(event sqlguard.ValidationEvent) error {
	m.mutex.Lock()
	m.events = append(m.events, event)
	failure := m.failure
	panicValue := m.panicValue
	m.mutex.Unlock()

	if panicValue != nil {
		panic(panicValue)
	}

	return failure
}

func (m *recordingMetrics) Events() []sqlguard.ValidationEvent {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return append([]sqlguard.ValidationEvent(nil), m.events...)
}

type recordingLogger struct {
	mutex      sync.Mutex
	events     []sqlguard.ValidationEvent
	failure    error
	panicValue any
}

func (l *recordingLogger) LogValidation(event sqlguard.ValidationEvent) error {
	l.mutex.Lock()
	l.events = append(l.events, event)
	failure := l.failure
	panicValue := l.panicValue
	l.mutex.Unlock()

	if panicValue != nil {
		panic(panicValue)
	}

	return failure
}

func TestEngineIsolatesObservabilityFailures(t *testing.T) {
	sinkFailure := errors.New("observability unavailable")

	tests := map[string]struct {
		metrics    *recordingMetrics
		logger     *recordingLogger
		input      string
		rules      []sqlguard.Rule
		want       validationEventExpectation
		checkError func(t *testing.T, err error)
	}{
		"both_errors_preserve_parser_failure": {
			metrics: &recordingMetrics{failure: sinkFailure},
			logger:  &recordingLogger{failure: sinkFailure},
			input:   "SELECT * FROM",
			want: validationEventExpectation{
				mode:    sqlguard.ValidationModeEnforce,
				outcome: sqlguard.ValidationOutcomeParserFailure,
			},
			checkError: requireParseFailure,
		},
		"both_panics_preserve_policy_violation": {
			metrics: &recordingMetrics{panicValue: "metrics panic"},
			logger:  &recordingLogger{panicValue: "logger panic"},
			input:   "DELETE FROM accounts",
			rules: []sqlguard.Rule{
				&ruleStub{
					id: "deny_delete",
					evaluate: func(context.Context, sqlguard.Statement) sqlguard.RuleResult {
						return sqlguard.Reject()
					},
				},
			},
			want: validationEventExpectation{
				mode:    sqlguard.ValidationModeEnforce,
				outcome: sqlguard.ValidationOutcomePolicyViolation,
				ruleID:  "deny_delete",
			},
			checkError: requireViolation("deny_delete"),
		},
		"logger_error_preserves_policy_violation": {
			metrics: &recordingMetrics{},
			logger:  &recordingLogger{failure: sinkFailure},
			input:   "DELETE FROM accounts",
			rules: []sqlguard.Rule{
				&ruleStub{
					id: "deny_delete",
					evaluate: func(context.Context, sqlguard.Statement) sqlguard.RuleResult {
						return sqlguard.Reject()
					},
				},
			},
			want: validationEventExpectation{
				mode:    sqlguard.ValidationModeEnforce,
				outcome: sqlguard.ValidationOutcomePolicyViolation,
				ruleID:  "deny_delete",
			},
			checkError: requireViolation("deny_delete"),
		},
		"logger_panic_preserves_policy_violation": {
			metrics: &recordingMetrics{},
			logger:  &recordingLogger{panicValue: "logger panic"},
			input:   "DELETE FROM accounts",
			rules: []sqlguard.Rule{
				&ruleStub{
					id: "deny_delete",
					evaluate: func(context.Context, sqlguard.Statement) sqlguard.RuleResult {
						return sqlguard.Reject()
					},
				},
			},
			want: validationEventExpectation{
				mode:    sqlguard.ValidationModeEnforce,
				outcome: sqlguard.ValidationOutcomePolicyViolation,
				ruleID:  "deny_delete",
			},
			checkError: requireViolation("deny_delete"),
		},
		"metrics_error_preserves_success": {
			metrics: &recordingMetrics{failure: sinkFailure},
			logger:  &recordingLogger{},
			input:   "SELECT 1",
			want: validationEventExpectation{
				mode:    sqlguard.ValidationModeEnforce,
				outcome: sqlguard.ValidationOutcomeAllowed,
			},
			checkError: requireNoValidationError,
		},
		"metrics_panic_preserves_success": {
			metrics: &recordingMetrics{panicValue: "metrics panic"},
			logger:  &recordingLogger{},
			input:   "SELECT 1",
			want: validationEventExpectation{
				mode:    sqlguard.ValidationModeEnforce,
				outcome: sqlguard.ValidationOutcomeAllowed,
			},
			checkError: requireNoValidationError,
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			engine, err := sqlguard.NewEngine(sqlguard.EngineOptions{
				Metrics: testCase.metrics,
				Logger:  testCase.logger,
			}, testCase.rules...)
			require.NoError(t, err)

			err = engine.Validate(context.Background(), testCase.input)
			require.NotNil(t, testCase.checkError, "test case must define checkError")
			testCase.checkError(t, err)

			requireSingleValidationEvent(t, testCase.metrics.Events(), testCase.want)
			requireSingleValidationEvent(t, testCase.logger.Events(), testCase.want)
		})
	}
}

func (l *recordingLogger) Events() []sqlguard.ValidationEvent {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	return append([]sqlguard.ValidationEvent(nil), l.events...)
}

func TestEngineEmitsTerminalValidationEvents(t *testing.T) {
	tests := map[string]struct {
		ctx        func() context.Context
		input      string
		rules      []sqlguard.Rule
		want       validationEventExpectation
		checkError func(t *testing.T, err error)
	}{
		"emits_allowed": {
			ctx:   context.Background,
			input: "SELECT 1",
			want: validationEventExpectation{
				mode:    sqlguard.ValidationModeEnforce,
				outcome: sqlguard.ValidationOutcomeAllowed,
			},
			checkError: requireNoValidationError,
		},
		"emits_canceled": {
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				return ctx
			},
			input: "SELECT 1",
			want: validationEventExpectation{
				mode:    sqlguard.ValidationModeEnforce,
				outcome: sqlguard.ValidationOutcomeCanceled,
			},
			checkError: requireContextError(context.Canceled),
		},
		"emits_parser_failure": {
			ctx:   context.Background,
			input: "SELECT * FROM",
			want: validationEventExpectation{
				mode:    sqlguard.ValidationModeEnforce,
				outcome: sqlguard.ValidationOutcomeParserFailure,
			},
			checkError: requireParseFailure,
		},
		"emits_policy_violation_with_stable_rule_identifier": {
			ctx:   context.Background,
			input: "DELETE FROM accounts",
			rules: []sqlguard.Rule{
				&ruleStub{
					id: "deny_delete",
					evaluate: func(context.Context, sqlguard.Statement) sqlguard.RuleResult {
						return sqlguard.Reject()
					},
				},
			},
			want: validationEventExpectation{
				mode:    sqlguard.ValidationModeEnforce,
				outcome: sqlguard.ValidationOutcomePolicyViolation,
				ruleID:  "deny_delete",
			},
			checkError: requireViolation("deny_delete"),
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			metrics := &recordingMetrics{}
			logger := &recordingLogger{}
			engine, err := sqlguard.NewEngine(sqlguard.EngineOptions{
				Metrics: metrics,
				Logger:  logger,
			}, testCase.rules...)
			require.NoError(t, err)

			err = engine.Validate(testCase.ctx(), testCase.input)
			require.NotNil(t, testCase.checkError, "test case must define checkError")
			testCase.checkError(t, err)

			requireSingleValidationEvent(t, metrics.Events(), testCase.want)
			requireSingleValidationEvent(t, logger.Events(), testCase.want)
		})
	}
}

func TestEngineEmitsCancellationAtEachCheckpoint(t *testing.T) {
	tests := map[string]struct {
		setupTest func(t *testing.T) (context.Context, []sqlguard.Rule)
	}{
		"canceled_during_rule_traversal": {
			setupTest: func(t *testing.T) (context.Context, []sqlguard.Rule) {
				t.Helper()

				ctx, cancel := context.WithCancel(context.Background())
				rule := &ruleStub{
					id: "cancel_validation",
					evaluate: func(context.Context, sqlguard.Statement) sqlguard.RuleResult {
						cancel()

						return sqlguard.Allow()
					},
				}

				return ctx, []sqlguard.Rule{rule, &ruleStub{id: "unreached_rule"}}
			},
		},
		"canceled_immediately_after_parsing": {
			setupTest: func(t *testing.T) (context.Context, []sqlguard.Rule) {
				t.Helper()

				return &cancelAfterParseContext{Context: context.Background()}, nil
			},
		},
		"canceled_before_parsing": {
			setupTest: func(t *testing.T) (context.Context, []sqlguard.Rule) {
				t.Helper()

				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				return ctx, nil
			},
		},
	}

	want := validationEventExpectation{
		mode:    sqlguard.ValidationModeEnforce,
		outcome: sqlguard.ValidationOutcomeCanceled,
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx, rules := testCase.setupTest(t)
			metrics := &recordingMetrics{}
			logger := &recordingLogger{}
			engine, err := sqlguard.NewEngine(sqlguard.EngineOptions{
				Metrics: metrics,
				Logger:  logger,
			}, rules...)
			require.NoError(t, err)

			err = engine.Validate(ctx, "SELECT 1")
			require.ErrorIs(t, err, context.Canceled)
			requireSingleValidationEvent(t, metrics.Events(), want)
			requireSingleValidationEvent(t, logger.Events(), want)
		})
	}
}

func TestEngineSupportsIndependentlyDisabledObservability(t *testing.T) {
	tests := map[string]struct {
		options          func(*recordingMetrics, *recordingLogger) sqlguard.EngineOptions
		wantMetricEvents int
		wantLogEvents    int
	}{
		"logger_disabled": {
			options: func(metrics *recordingMetrics, _ *recordingLogger) sqlguard.EngineOptions {
				return sqlguard.EngineOptions{Metrics: metrics}
			},
			wantMetricEvents: 1,
		},
		"metrics_disabled": {
			options: func(_ *recordingMetrics, logger *recordingLogger) sqlguard.EngineOptions {
				return sqlguard.EngineOptions{Logger: logger}
			},
			wantLogEvents: 1,
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			metrics := &recordingMetrics{}
			logger := &recordingLogger{}
			engine, err := sqlguard.NewEngine(testCase.options(metrics, logger))
			require.NoError(t, err)
			require.NoError(t, engine.Validate(context.Background(), "SELECT 1"))

			require.Len(t, metrics.Events(), testCase.wantMetricEvents)
			require.Len(t, logger.Events(), testCase.wantLogEvents)
		})
	}
}

func TestEngineObservabilityBoundaryOmitsSensitiveData(t *testing.T) {
	const (
		contextSecret = "context_secret_a49b36"
		literalSecret = "literal_secret_e87c12"
		tokenSecret   = "token_secret_3d92af"
	)

	ctx := context.WithValue(context.Background(), observabilityContextKey("secret"), contextSecret)
	input := "SELECT '" + literalSecret + "' /* " + tokenSecret + " */ FROM"
	metrics := &recordingMetrics{}
	logger := &recordingLogger{}
	engine, err := sqlguard.NewEngine(sqlguard.EngineOptions{
		Metrics: metrics,
		Logger:  logger,
	})
	require.NoError(t, err)

	err = engine.Validate(ctx, input)
	requireParseFailure(t, err)

	for _, events := range [][]sqlguard.ValidationEvent{metrics.Events(), logger.Events()} {
		require.Len(t, events, 1)

		representation := fmt.Sprintf("%#v", events[0])
		for _, sensitive := range []string{contextSecret, literalSecret, tokenSecret, input, err.Error()} {
			require.NotContains(t, representation, sensitive)
		}
	}
}

func TestEngineConcurrentObservability(t *testing.T) {
	const callCount = 64

	metrics := &recordingMetrics{}
	logger := &recordingLogger{}
	engine, err := sqlguard.NewEngine(sqlguard.EngineOptions{
		Metrics: metrics,
		Logger:  logger,
	})
	require.NoError(t, err)

	errors := make([]error, callCount)

	var waitGroup sync.WaitGroup
	waitGroup.Add(callCount)

	for index := range callCount {
		go func() {
			defer waitGroup.Done()

			errors[index] = engine.Validate(context.Background(), fmt.Sprintf("SELECT %d", index))
		}()
	}

	waitGroup.Wait()
	requireNoValidationErrors(t, errors)

	want := validationEventExpectation{
		mode:    sqlguard.ValidationModeEnforce,
		outcome: sqlguard.ValidationOutcomeAllowed,
	}
	requireValidationEvents(t, metrics.Events(), callCount, want)
	requireValidationEvents(t, logger.Events(), callCount, want)
}

type validationEventExpectation struct {
	mode    sqlguard.ValidationMode
	outcome sqlguard.ValidationOutcome
	ruleID  string
}

type observabilityContextKey string

func requireSingleValidationEvent(
	t *testing.T,
	events []sqlguard.ValidationEvent,
	want validationEventExpectation,
) {
	t.Helper()

	require.Len(t, events, 1)
	require.Equal(t, want.mode, events[0].Mode())
	require.Equal(t, want.outcome, events[0].Outcome())
	require.Equal(t, want.ruleID, events[0].RuleID())
}

func requireValidationEvents(
	t *testing.T,
	events []sqlguard.ValidationEvent,
	wantCount int,
	want validationEventExpectation,
) {
	t.Helper()

	require.Len(t, events, wantCount)

	for _, event := range events {
		require.Equal(t, want.mode, event.Mode())
		require.Equal(t, want.outcome, event.Outcome())
		require.Equal(t, want.ruleID, event.RuleID())
	}
}
