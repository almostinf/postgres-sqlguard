package slog_test

import (
	stdslog "log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	sqlguard "github.com/almostinf/postgres-sqlguard"
	sqlguardslog "github.com/almostinf/postgres-sqlguard/observability/slog"
)

func TestNew(t *testing.T) {
	tests := map[string]struct {
		logger    *stdslog.Logger
		wantError string
	}{
		"accepts_logger": {
			logger: stdslog.New(stdslog.DiscardHandler),
		},
		"rejects_nil_logger": {
			wantError: "sqlguard slog: logger must not be nil",
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			logger, err := sqlguardslog.New(testCase.logger)
			if testCase.wantError != "" {
				require.EqualError(t, err, testCase.wantError)
				require.Nil(t, logger)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, logger)

			var implementation sqlguard.Logger = logger
			require.NotNil(t, implementation)
		})
	}
}
