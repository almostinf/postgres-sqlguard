package sqlguard_test

import (
	"context"
	"testing"

	sqlguard "github.com/almostinf/postgres-sqlguard"
)

func BenchmarkEngineValidate(b *testing.B) {
	engine, err := sqlguard.NewEngine(&ruleStub{id: "benchmark_allow"})
	if err != nil {
		b.Fatal(err)
	}

	benchmarks := map[string]string{
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

	for name, sql := range benchmarks {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				if err := engine.Validate(context.Background(), sql); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
