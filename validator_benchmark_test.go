package sqlguard_test

import (
	"context"
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

func BenchmarkEngineValidate(b *testing.B) {
	inputs := map[string]string{
		"single_statement": "SELECT * FROM accounts WHERE id = 42",
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

	configurations := map[string]sqlguard.EngineOptions{
		"logger_noop": {
			Logger: benchmarkLogger{},
		},
		"metrics_and_logger_noop": {
			Metrics: benchmarkMetrics{},
			Logger:  benchmarkLogger{},
		},
		"metrics_noop": {
			Metrics: benchmarkMetrics{},
		},
		"observability_disabled": {},
	}

	for configurationName, options := range configurations {
		b.Run(configurationName, func(b *testing.B) {
			engine, err := sqlguard.NewEngine(options, &ruleStub{id: "benchmark_allow"})
			if err != nil {
				b.Fatal(err)
			}

			for inputName, sql := range inputs {
				b.Run(inputName, func(b *testing.B) {
					b.ReportAllocs()
					b.ResetTimer()

					for b.Loop() {
						if err := engine.Validate(context.Background(), sql); err != nil {
							b.Fatal(err)
						}
					}
				})
			}
		})
	}
}
