package parser

import (
	"io"
	"os"
	"os/exec"
	"testing"
)

const parserBenchmarkSQL = `
	WITH recent_orders AS (
		SELECT account_id, count(*) AS order_count
		FROM orders
		WHERE created_at >= CURRENT_DATE - INTERVAL '30 days'
		GROUP BY account_id
	)
	SELECT accounts.id, recent_orders.order_count
	FROM accounts
	JOIN recent_orders ON recent_orders.account_id = accounts.id
	WHERE accounts.active
	ORDER BY accounts.id
	LIMIT 100
`

const parserFreshProcessHelperEnvironment = "SQLGUARD_PARSER_FRESH_PROCESS_HELPER=1"

func TestParserFreshProcessHelper(t *testing.T) {
	if os.Getenv("SQLGUARD_PARSER_FRESH_PROCESS_HELPER") != "1" {
		return
	}

	result, err := Parse(parserBenchmarkSQL)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Statements()) != 1 {
		t.Fatalf("unexpected fresh-process statement count: %d", len(result.Statements()))
	}
}

func BenchmarkParserFreshProcessFirstParse(b *testing.B) {
	for b.Loop() {
		command := exec.Command(os.Args[0], "-test.run=^TestParserFreshProcessHelper$", "-test.count=1")

		command.Env = append(os.Environ(), parserFreshProcessHelperEnvironment)
		command.Stdout = io.Discard
		command.Stderr = io.Discard

		if err := command.Run(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParserWarmSteadyState(b *testing.B) {
	requireParserBenchmarkPreflight(b)
	b.ReportAllocs()

	for b.Loop() {
		_, _ = Parse(parserBenchmarkSQL)
	}
}

func BenchmarkParserParallel(b *testing.B) {
	requireParserBenchmarkPreflight(b)
	b.ReportAllocs()

	b.RunParallel(func(parallel *testing.PB) {
		for parallel.Next() {
			_, _ = Parse(parserBenchmarkSQL)
		}
	})
}

func requireParserBenchmarkPreflight(b *testing.B) {
	b.Helper()

	result, err := Parse(parserBenchmarkSQL)
	if err != nil {
		b.Fatal(err)
	}

	if len(result.Statements()) != 1 {
		b.Fatalf("unexpected benchmark statement count: %d", len(result.Statements()))
	}
}
