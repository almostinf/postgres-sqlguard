package sqlguard_test

import (
	"context"
	"errors"
	"testing"

	sqlguard "github.com/almostinf/postgres-sqlguard"
)

type benchmarkMetrics struct{}

func (benchmarkMetrics) RecordValidation(sqlguard.ValidationEvent) error {
	return nil
}

type benchmarkLogger struct{}

func (benchmarkLogger) LogValidation(sqlguard.ValidationEvent) error {
	return nil
}

type benchmarkOutcome uint8

const (
	benchmarkOutcomeAllowed benchmarkOutcome = iota
	benchmarkOutcomePolicyViolation
	benchmarkOutcomeParserFailure
)

type validationBenchmark struct {
	sql             string
	options         sqlguard.EngineOptions
	rules           []sqlguard.Rule
	expectedOutcome benchmarkOutcome
}

func runValidationBenchmark(b *testing.B, benchmark validationBenchmark) {
	b.Helper()

	engine, err := sqlguard.NewEngine(benchmark.options, benchmark.rules...)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	if err := engine.Validate(ctx, benchmark.sql); !matchesBenchmarkOutcome(benchmark.expectedOutcome, err) {
		b.Fatalf("unexpected validation outcome: want %d, got %T", benchmark.expectedOutcome, err)
	}

	b.ReportAllocs()

	for b.Loop() {
		_ = engine.Validate(ctx, benchmark.sql)
	}
}

func matchesBenchmarkOutcome(expected benchmarkOutcome, err error) bool {
	switch expected {
	case benchmarkOutcomeAllowed:
		return err == nil
	case benchmarkOutcomePolicyViolation:
		var violation *sqlguard.Violation

		return errors.As(err, &violation)
	case benchmarkOutcomeParserFailure:
		var parseError *sqlguard.ParseError

		return errors.As(err, &parseError)
	default:
		return false
	}
}

func BenchmarkEngineValidateByComplexity(b *testing.B) {
	inputs := map[string]string{
		"simple": "SELECT 1",
		"medium": `
			SELECT accounts.id, count(orders.id)
			FROM accounts
			JOIN orders ON orders.account_id = accounts.id
			WHERE accounts.active AND orders.created_at >= CURRENT_DATE - INTERVAL '30 days'
			GROUP BY accounts.id
			HAVING count(orders.id) > 1
			ORDER BY accounts.id
			LIMIT 100
		`,
		"multi_statement": `
			SELECT 1;
			UPDATE accounts SET active = false WHERE id = 42;
			INSERT INTO audit_log (message) VALUES ('updated');
			DELETE FROM sessions WHERE expired;
		`,
		"nested_cte": `
			WITH changed AS (
				WITH removed AS (
					DELETE FROM sessions WHERE expired RETURNING account_id
				)
				UPDATE accounts SET active = false
				FROM removed
				WHERE accounts.id = removed.account_id
				RETURNING accounts.*
			)
			SELECT * FROM changed;
		`,
	}

	for name, sql := range inputs {
		b.Run(name, func(b *testing.B) {
			runValidationBenchmark(b, validationBenchmark{
				sql:             sql,
				rules:           []sqlguard.Rule{&ruleStub{id: "benchmark_allow"}},
				expectedOutcome: benchmarkOutcomeAllowed,
			})
		})
	}
}

func BenchmarkEngineValidateByOutcome(b *testing.B) {
	allowRule := &ruleStub{id: "benchmark_allow"}
	rejectRule := &ruleStub{
		id: "benchmark_reject",
		evaluate: func(context.Context, sqlguard.Statement) sqlguard.RuleResult {
			return sqlguard.Reject()
		},
	}

	benchmarks := map[string]validationBenchmark{
		"allowed": {
			sql:             "SELECT 1",
			rules:           []sqlguard.Rule{allowRule},
			expectedOutcome: benchmarkOutcomeAllowed,
		},
		"policy_violation": {
			sql:             "SELECT 1",
			rules:           []sqlguard.Rule{rejectRule},
			expectedOutcome: benchmarkOutcomePolicyViolation,
		},
		"parser_failure": {
			sql:             "SELECT (",
			rules:           []sqlguard.Rule{allowRule},
			expectedOutcome: benchmarkOutcomeParserFailure,
		},
	}

	for name, benchmark := range benchmarks {
		b.Run(name, func(b *testing.B) {
			runValidationBenchmark(b, benchmark)
		})
	}
}

func BenchmarkEngineObservability(b *testing.B) {
	configurations := map[string]sqlguard.EngineOptions{
		"disabled": {},
		"metrics": {
			Metrics: benchmarkMetrics{},
		},
		"logger": {
			Logger: benchmarkLogger{},
		},
		"metrics_and_logger": {
			Metrics: benchmarkMetrics{},
			Logger:  benchmarkLogger{},
		},
	}

	for name, options := range configurations {
		b.Run(name, func(b *testing.B) {
			runValidationBenchmark(b, validationBenchmark{
				sql:             "SELECT 1",
				options:         options,
				rules:           []sqlguard.Rule{&ruleStub{id: "benchmark_allow"}},
				expectedOutcome: benchmarkOutcomeAllowed,
			})
		})
	}
}
