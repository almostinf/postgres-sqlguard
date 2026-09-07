package sqlguard_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	sqlguard "github.com/almostinf/postgres-sqlguard"
)

type metricsStub struct{}

func (*metricsStub) RecordValidation(sqlguard.ValidationEvent) error {
	return nil
}

type loggerStub struct{}

func (*loggerStub) LogValidation(sqlguard.ValidationEvent) error {
	return nil
}

func TestNewEngineOptions(t *testing.T) {
	var typedNilMetrics *metricsStub

	var typedNilLogger *loggerStub

	tests := map[string]struct {
		options    sqlguard.EngineOptions
		rules      []sqlguard.Rule
		wantError  string
		wantEngine bool
	}{
		"accepts_both_implementations": {
			options: sqlguard.EngineOptions{
				Metrics: &metricsStub{},
				Logger:  &loggerStub{},
			},
			wantEngine: true,
		},
		"accepts_logger_with_metrics_disabled": {
			options:    sqlguard.EngineOptions{Logger: &loggerStub{}},
			wantEngine: true,
		},
		"accepts_metrics_with_logger_disabled": {
			options:    sqlguard.EngineOptions{Metrics: &metricsStub{}},
			wantEngine: true,
		},
		"accepts_zero_valued_options": {
			wantEngine: true,
		},
		"rejects_invalid_rules_with_options": {
			options:   sqlguard.EngineOptions{Metrics: &metricsStub{}},
			rules:     []sqlguard.Rule{nil},
			wantError: "sqlguard: rule must not be nil",
		},
		"rejects_typed_nil_logger": {
			options:   sqlguard.EngineOptions{Logger: typedNilLogger},
			wantError: "sqlguard: logger must not be nil",
		},
		"rejects_typed_nil_metrics": {
			options:   sqlguard.EngineOptions{Metrics: typedNilMetrics},
			wantError: "sqlguard: metrics must not be nil",
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			engine, err := sqlguard.NewEngine(testCase.options, testCase.rules...)
			if testCase.wantError != "" {
				require.EqualError(t, err, testCase.wantError)
				require.Nil(t, engine)

				return
			}

			require.NoError(t, err)

			if testCase.wantEngine {
				require.NotNil(t, engine)
			}

			require.NoError(t, engine.Validate(context.Background(), "SELECT 1"))
		})
	}
}
