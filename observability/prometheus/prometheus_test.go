package prometheus_test

import (
	"testing"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	sqlguard "github.com/almostinf/postgres-sqlguard"
	sqlguardprometheus "github.com/almostinf/postgres-sqlguard/observability/prometheus"
)

func TestNew(t *testing.T) {
	var typedNilRegisterer *prometheusclient.Registry

	tests := map[string]struct {
		config    sqlguardprometheus.Config
		wantError string
	}{
		"accepts_default_metric_name": {
			config: sqlguardprometheus.Config{
				Registerer: prometheusclient.NewRegistry(),
				Service:    "payments_api",
			},
		},
		"accepts_valid_custom_metric_name": {
			config: sqlguardprometheus.Config{
				Registerer: prometheusclient.NewRegistry(),
				Service:    "payments_api",
				MetricName: "payments_sql_validations_total",
			},
		},
		"rejects_empty_service": {
			config: sqlguardprometheus.Config{
				Registerer: prometheusclient.NewRegistry(),
			},
			wantError: "sqlguard prometheus: service must not be empty",
		},
		"rejects_metric_name_starting_with_digit": {
			config: sqlguardprometheus.Config{
				Registerer: prometheusclient.NewRegistry(),
				Service:    "payments_api",
				MetricName: "9invalid_metric",
			},
			wantError: "sqlguard prometheus: metric name is invalid",
		},
		"rejects_metric_name_with_invalid_characters": {
			config: sqlguardprometheus.Config{
				Registerer: prometheusclient.NewRegistry(),
				Service:    "payments_api",
				MetricName: "invalid-metric name",
			},
			wantError: "sqlguard prometheus: metric name is invalid",
		},
		"rejects_nil_registerer": {
			config: sqlguardprometheus.Config{
				Service: "payments_api",
			},
			wantError: "sqlguard prometheus: registerer must not be nil",
		},
		"rejects_typed_nil_registerer": {
			config: sqlguardprometheus.Config{
				Registerer: typedNilRegisterer,
				Service:    "payments_api",
			},
			wantError: "sqlguard prometheus: registerer must not be nil",
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			metrics, err := sqlguardprometheus.New(testCase.config)
			if testCase.wantError != "" {
				require.EqualError(t, err, testCase.wantError)
				require.Nil(t, metrics)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, metrics)

			var implementation sqlguard.Metrics = metrics
			require.NotNil(t, implementation)
		})
	}
}

func TestNewReportsRegistrationConflict(t *testing.T) {
	registry := prometheusclient.NewRegistry()
	config := sqlguardprometheus.Config{
		Registerer: registry,
		Service:    "payments_api",
	}

	first, err := sqlguardprometheus.New(config)
	require.NoError(t, err)
	require.NotNil(t, first)

	conflicting, err := sqlguardprometheus.New(config)
	require.ErrorContains(t, err, "sqlguard prometheus: register metric")
	require.Nil(t, conflicting)
}

func TestNewKeepsRegistriesIndependent(t *testing.T) {
	first, err := sqlguardprometheus.New(sqlguardprometheus.Config{
		Registerer: prometheusclient.NewRegistry(),
		Service:    "payments_api",
	})
	require.NoError(t, err)
	require.NotNil(t, first)

	second, err := sqlguardprometheus.New(sqlguardprometheus.Config{
		Registerer: prometheusclient.NewRegistry(),
		Service:    "payments_api",
	})
	require.NoError(t, err)
	require.NotNil(t, second)
}
