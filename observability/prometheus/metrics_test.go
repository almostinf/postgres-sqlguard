package prometheus_test

import (
	"context"
	"maps"
	"testing"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"

	sqlguard "github.com/almostinf/postgres-sqlguard"
	sqlguardprometheus "github.com/almostinf/postgres-sqlguard/observability/prometheus"
)

const defaultMetricName = "sqlguard_validations_total"

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

func TestMetricsRecordsEngineOutcomes(t *testing.T) {
	tests := map[string]struct {
		ctx         func() context.Context
		input       string
		rules       []sqlguard.Rule
		wantOutcome string
		wantRuleID  string
		checkError  func(t *testing.T, err error)
	}{
		"allowed": {
			ctx:         context.Background,
			input:       "SELECT 1",
			wantOutcome: "allowed",
			checkError:  requireNoValidationError,
		},
		"canceled": {
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				return ctx
			},
			input:       "SELECT 1",
			wantOutcome: "canceled",
			checkError: func(t *testing.T, err error) {
				t.Helper()

				require.ErrorIs(t, err, context.Canceled)
			},
		},
		"parser_failure": {
			ctx:         context.Background,
			input:       "SELECT * FROM",
			wantOutcome: "parser_failure",
			checkError: func(t *testing.T, err error) {
				t.Helper()

				var parseError *sqlguard.ParseError
				require.ErrorAs(t, err, &parseError)
			},
		},
		"policy_violation": {
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
			wantOutcome: "policy_violation",
			wantRuleID:  "deny_delete",
			checkError: func(t *testing.T, err error) {
				t.Helper()

				var violation *sqlguard.Violation
				require.ErrorAs(t, err, &violation)
				require.Equal(t, "deny_delete", violation.RuleID())
			},
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			registry := prometheusclient.NewRegistry()
			metrics := mustNewMetrics(t, registry, "payments_api", "")
			engine, err := sqlguard.NewEngine(
				sqlguard.EngineOptions{Metrics: metrics},
				testCase.rules...,
			)
			require.NoError(t, err)

			err = engine.Validate(testCase.ctx(), testCase.input)
			require.NotNil(t, testCase.checkError, "test case must define checkError")
			testCase.checkError(t, err)

			requireCounter(t, registry, defaultMetricName, map[string]string{
				"service": "payments_api",
				"mode":    "enforce",
				"outcome": testCase.wantOutcome,
				"rule_id": testCase.wantRuleID,
			}, 1)
		})
	}
}

func TestMetricsUsesCustomMetricName(t *testing.T) {
	registry := prometheusclient.NewRegistry()
	metrics := mustNewMetrics(t, registry, "payments_api", "payments_sql_checks_total")
	engine, err := sqlguard.NewEngine(sqlguard.EngineOptions{Metrics: metrics})
	require.NoError(t, err)
	require.NoError(t, engine.Validate(context.Background(), "SELECT 1"))

	families, err := registry.Gather()
	require.NoError(t, err)
	require.Len(t, families, 1)
	require.Equal(t, "payments_sql_checks_total", families[0].GetName())
}

func TestMetricsKeepsServiceFixedAcrossOutcomes(t *testing.T) {
	registry := prometheusclient.NewRegistry()
	metrics := mustNewMetrics(t, registry, "ledger_worker", "")
	rule := &ruleStub{
		id: "deny_delete",
		evaluate: func(statement sqlguard.Statement) sqlguard.RuleResult {
			if statement.Kind() == sqlguard.Kind("DeleteStmt") {
				return sqlguard.Reject()
			}

			return sqlguard.Allow()
		},
	}
	engine, err := sqlguard.NewEngine(sqlguard.EngineOptions{Metrics: metrics}, rule)
	require.NoError(t, err)
	require.NoError(t, engine.Validate(context.Background(), "SELECT 1"))

	err = engine.Validate(context.Background(), "DELETE FROM accounts")

	var violation *sqlguard.Violation
	require.ErrorAs(t, err, &violation)

	requireCounter(t, registry, defaultMetricName, map[string]string{
		"service": "ledger_worker",
		"mode":    "enforce",
		"outcome": "allowed",
		"rule_id": "",
	}, 1)
	requireCounter(t, registry, defaultMetricName, map[string]string{
		"service": "ledger_worker",
		"mode":    "enforce",
		"outcome": "policy_violation",
		"rule_id": "deny_delete",
	}, 1)
}

func TestMetricsKeepsRegistriesIndependent(t *testing.T) {
	firstRegistry := prometheusclient.NewRegistry()
	firstMetrics := mustNewMetrics(t, firstRegistry, "first_service", "")
	firstEngine, err := sqlguard.NewEngine(sqlguard.EngineOptions{Metrics: firstMetrics})
	require.NoError(t, err)

	secondRegistry := prometheusclient.NewRegistry()
	secondMetrics := mustNewMetrics(t, secondRegistry, "second_service", "")
	secondEngine, err := sqlguard.NewEngine(sqlguard.EngineOptions{Metrics: secondMetrics})
	require.NoError(t, err)

	require.NoError(t, firstEngine.Validate(context.Background(), "SELECT 1"))
	require.NoError(t, secondEngine.Validate(context.Background(), "SELECT 1"))
	require.NoError(t, secondEngine.Validate(context.Background(), "SELECT 2"))

	requireCounter(t, firstRegistry, defaultMetricName, map[string]string{
		"service": "first_service",
		"mode":    "enforce",
		"outcome": "allowed",
		"rule_id": "",
	}, 1)
	requireCounter(t, secondRegistry, defaultMetricName, map[string]string{
		"service": "second_service",
		"mode":    "enforce",
		"outcome": "allowed",
		"rule_id": "",
	}, 2)
}

func mustNewMetrics(
	t *testing.T,
	registerer prometheusclient.Registerer,
	service string,
	metricName string,
) *sqlguardprometheus.Metrics {
	t.Helper()

	metrics, err := sqlguardprometheus.New(sqlguardprometheus.Config{
		Registerer: registerer,
		Service:    service,
		MetricName: metricName,
	})
	require.NoError(t, err)

	return metrics
}

func requireCounter(
	t *testing.T,
	registry *prometheusclient.Registry,
	metricName string,
	wantLabels map[string]string,
	wantValue float64,
) {
	t.Helper()

	families, err := registry.Gather()
	require.NoError(t, err)

	family := findMetricFamily(families, metricName)
	require.NotNil(t, family)

	var matchingMetrics []*dto.Metric

	for _, metric := range family.GetMetric() {
		labels := make(map[string]string, len(metric.GetLabel()))
		for _, label := range metric.GetLabel() {
			labels[label.GetName()] = label.GetValue()
		}

		if maps.Equal(labels, wantLabels) {
			matchingMetrics = append(matchingMetrics, metric)
		}
	}

	require.Len(t, matchingMetrics, 1)
	require.Equal(t, wantValue, matchingMetrics[0].GetCounter().GetValue())
}

func findMetricFamily(families []*dto.MetricFamily, name string) *dto.MetricFamily {
	for _, family := range families {
		if family.GetName() == name {
			return family
		}
	}

	return nil
}

func requireNoValidationError(t *testing.T, err error) {
	t.Helper()

	require.NoError(t, err)
}
