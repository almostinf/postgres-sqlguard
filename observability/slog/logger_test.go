package slog_test

import (
	"context"
	"errors"
	"fmt"
	stdslog "log/slog"
	"maps"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	sqlguard "github.com/almostinf/postgres-sqlguard"
	sqlguardslog "github.com/almostinf/postgres-sqlguard/observability/slog"
)

type contextKey string

type ruleStub struct {
	id       string
	evaluate func(sqlguard.Statement) sqlguard.RuleResult
}

func (r *ruleStub) ID() string {
	return r.id
}

func (r *ruleStub) Evaluate(_ context.Context, statement sqlguard.Statement) sqlguard.RuleResult {
	if r.evaluate == nil {
		return sqlguard.Allow()
	}

	return r.evaluate(statement)
}

type handlerCall struct {
	ctx    context.Context
	level  stdslog.Level
	record stdslog.Record
}

type recordingHandler struct {
	mutex sync.Mutex

	enabled        bool
	handleError    error
	enabledCalls   []handlerCall
	handledRecords []handlerCall
}

func (h *recordingHandler) Enabled(ctx context.Context, level stdslog.Level) bool {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.enabledCalls = append(h.enabledCalls, handlerCall{
		ctx:   ctx,
		level: level,
	})

	return h.enabled
}

func (h *recordingHandler) Handle(ctx context.Context, record stdslog.Record) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.handledRecords = append(h.handledRecords, handlerCall{ctx: ctx, record: record.Clone()})

	return h.handleError
}

func (h *recordingHandler) WithAttrs([]stdslog.Attr) stdslog.Handler {
	return h
}

func (h *recordingHandler) WithGroup(string) stdslog.Handler {
	return h
}

func (h *recordingHandler) Snapshot() ([]handlerCall, []handlerCall) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	return append([]handlerCall(nil), h.enabledCalls...),
		append([]handlerCall(nil), h.handledRecords...)
}

func TestLoggerEmitsValidationRecords(t *testing.T) {
	tests := map[string]struct {
		ctx         func() context.Context
		input       string
		rules       []sqlguard.Rule
		wantLevel   stdslog.Level
		wantOutcome string
		wantRuleID  string
		checkError  func(t *testing.T, err error)
	}{
		"allowed_at_info": {
			ctx:         context.Background,
			input:       "SELECT 1",
			wantLevel:   stdslog.LevelInfo,
			wantOutcome: "allowed",
			checkError:  requireNoValidationError,
		},
		"canceled_at_debug": {
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				return ctx
			},
			input:       "SELECT 1",
			wantLevel:   stdslog.LevelDebug,
			wantOutcome: "canceled",
			checkError: func(t *testing.T, err error) {
				t.Helper()

				require.ErrorIs(t, err, context.Canceled)
			},
		},
		"parser_failure_at_error": {
			ctx:         context.Background,
			input:       "SELECT * FROM",
			wantLevel:   stdslog.LevelError,
			wantOutcome: "parser_failure",
			checkError: func(t *testing.T, err error) {
				t.Helper()

				var parseError *sqlguard.ParseError
				require.ErrorAs(t, err, &parseError)
			},
		},
		"policy_violation_at_error": {
			ctx:   context.Background,
			input: "DELETE FROM accounts",
			rules: []sqlguard.Rule{
				&ruleStub{
					id: "deny_delete",
					evaluate: func(sqlguard.Statement) sqlguard.RuleResult {
						return sqlguard.Reject()
					},
				},
			},
			wantLevel:   stdslog.LevelError,
			wantOutcome: "policy_violation",
			wantRuleID:  "deny_delete",
			checkError: func(t *testing.T, err error) {
				t.Helper()

				var violation *sqlguard.Violation
				require.ErrorAs(t, err, &violation)
			},
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			handler := &recordingHandler{enabled: true}
			logger := mustNewLogger(t, handler)
			engine, err := sqlguard.NewEngine(
				sqlguard.EngineOptions{Logger: logger},
				testCase.rules...,
			)
			require.NoError(t, err)

			err = engine.Validate(testCase.ctx(), testCase.input)
			require.NotNil(t, testCase.checkError, "test case must define checkError")
			testCase.checkError(t, err)

			enabledCalls, handledRecords := handler.Snapshot()
			require.Len(t, enabledCalls, 1)
			require.Equal(t, testCase.wantLevel, enabledCalls[0].level)
			requireCleanContext(enabledCalls[0].ctx, t)
			require.Len(t, handledRecords, 1)
			requireCleanContext(handledRecords[0].ctx, t)
			require.Equal(t, testCase.wantLevel, handledRecords[0].record.Level)
			require.Equal(t, "sqlguard validation", handledRecords[0].record.Message)
			require.Equal(t, 3, handledRecords[0].record.NumAttrs())
			require.True(t, maps.Equal(map[string]string{
				"mode":    "enforce",
				"outcome": testCase.wantOutcome,
				"rule_id": testCase.wantRuleID,
			}, recordAttributes(handledRecords[0].record)))
		})
	}
}

func TestLoggerHonorsDisabledHandler(t *testing.T) {
	handler := &recordingHandler{}
	logger := mustNewLogger(t, handler)
	engine, err := sqlguard.NewEngine(sqlguard.EngineOptions{Logger: logger})
	require.NoError(t, err)
	require.NoError(t, engine.Validate(context.Background(), "SELECT 1"))

	enabledCalls, handledRecords := handler.Snapshot()
	require.Len(t, enabledCalls, 1)
	require.Empty(t, handledRecords)
}

func TestLoggerReturnsHandlerError(t *testing.T) {
	handlerError := errors.New("handler unavailable")
	handler := &recordingHandler{enabled: true, handleError: handlerError}
	logger := mustNewLogger(t, handler)

	err := logger.LogValidation(sqlguard.ValidationEvent{})
	require.ErrorIs(t, err, handlerError)

	_, handledRecords := handler.Snapshot()
	require.Len(t, handledRecords, 1)
}

func TestLoggerOmitsSensitiveInputAndCallerContext(t *testing.T) {
	const (
		contextSecret = "context_secret_198ab4"
		literalSecret = "literal_secret_b48c72"
		tokenSecret   = "token_secret_71ef53"
	)

	ctx := context.WithValue(context.Background(), contextKey("secret"), contextSecret)
	input := "SELECT '" + literalSecret + "' /* " + tokenSecret + " */ FROM"
	handler := &recordingHandler{enabled: true}
	logger := mustNewLogger(t, handler)
	engine, err := sqlguard.NewEngine(sqlguard.EngineOptions{Logger: logger})
	require.NoError(t, err)

	err = engine.Validate(ctx, input)

	var parseError *sqlguard.ParseError
	require.ErrorAs(t, err, &parseError)

	enabledCalls, handledRecords := handler.Snapshot()
	require.Len(t, enabledCalls, 1)
	require.Nil(t, enabledCalls[0].ctx.Value(contextKey("secret")))
	require.Len(t, handledRecords, 1)
	require.Nil(t, handledRecords[0].ctx.Value(contextKey("secret")))

	representation := fmt.Sprintf("%s %#v", handledRecords[0].record.Message, recordAttributes(handledRecords[0].record))
	for _, sensitive := range []string{contextSecret, literalSecret, tokenSecret, input, err.Error()} {
		require.NotContains(t, representation, sensitive)
	}
}

func TestLoggerConcurrentUse(t *testing.T) {
	const callCount = 64

	handler := &recordingHandler{enabled: true}
	logger := mustNewLogger(t, handler)
	engine, err := sqlguard.NewEngine(sqlguard.EngineOptions{Logger: logger})
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

	for _, validationError := range errors {
		require.NoError(t, validationError)
	}

	enabledCalls, handledRecords := handler.Snapshot()
	require.Len(t, enabledCalls, callCount)
	require.Len(t, handledRecords, callCount)

	for _, call := range handledRecords {
		require.Equal(t, stdslog.LevelInfo, call.record.Level)
		require.Equal(t, "sqlguard validation", call.record.Message)
		require.Equal(t, map[string]string{
			"mode":    "enforce",
			"outcome": "allowed",
			"rule_id": "",
		}, recordAttributes(call.record))
	}
}

func mustNewLogger(t *testing.T, handler stdslog.Handler) *sqlguardslog.Logger {
	t.Helper()

	logger, err := sqlguardslog.New(stdslog.New(handler))
	require.NoError(t, err)

	return logger
}

func recordAttributes(record stdslog.Record) map[string]string {
	attributes := make(map[string]string, record.NumAttrs())
	record.Attrs(func(attribute stdslog.Attr) bool {
		attributes[attribute.Key] = attribute.Value.String()

		return true
	})

	return attributes
}

func requireCleanContext(ctx context.Context, t *testing.T) {
	t.Helper()

	require.NoError(t, ctx.Err())
	_, hasDeadline := ctx.Deadline()
	require.False(t, hasDeadline)
}

func requireNoValidationError(t *testing.T, err error) {
	t.Helper()

	require.NoError(t, err)
}
