package pgxprepared

import (
	"context"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	sqlguard "github.com/almostinf/postgres-sqlguard"
)

const benchmarkCacheCapacity = 8

type benchmarkExecutor struct{}

func (benchmarkExecutor) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (benchmarkExecutor) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, nil
}

func (benchmarkExecutor) QueryRow(context.Context, string, ...any) pgx.Row {
	return benchmarkRow{}
}

type benchmarkRow struct{}

func (benchmarkRow) Scan(...any) error {
	return nil
}

func BenchmarkGuardedDBCache(b *testing.B) {
	b.Run("hit", benchmarkGuardedDBCacheHit)
	b.Run("miss", benchmarkGuardedDBCacheMiss)
}

func benchmarkGuardedDBCacheHit(b *testing.B) {
	guarded := newBenchmarkGuardedDB(b)
	ctx := context.Background()
	query := "SELECT $1::integer"

	if _, err := guarded.Exec(ctx, query, 42); err != nil {
		b.Fatalf("warm cache: %v", err)
	}

	b.ReportAllocs()

	for b.Loop() {
		if _, err := guarded.Exec(ctx, query, 42); err != nil {
			b.Fatalf("execute cached query: %v", err)
		}
	}
}

func benchmarkGuardedDBCacheMiss(b *testing.B) {
	engine := newBenchmarkEngine(b)
	queries := make([]string, benchmarkCacheCapacity+1)
	ctx := context.Background()

	for index := range queries {
		queries[index] = "SELECT " + strconv.Itoa(index)

		if _, err := engine.Prepare(ctx, queries[index]); err != nil {
			b.Fatalf("preflight query %d: %v", index, err)
		}
	}

	guarded := newBenchmarkGuardedDBWithEngine(b, engine)
	queryIndex := 0

	b.ReportAllocs()

	for b.Loop() {
		if _, err := guarded.Exec(ctx, queries[queryIndex]); err != nil {
			b.Fatalf("execute cache miss: %v", err)
		}

		queryIndex++
		if queryIndex == len(queries) {
			queryIndex = 0
		}
	}
}

func newBenchmarkGuardedDB(b *testing.B) *GuardedDB {
	b.Helper()

	return newBenchmarkGuardedDBWithEngine(b, newBenchmarkEngine(b))
}

func newBenchmarkGuardedDBWithEngine(b *testing.B, engine *sqlguard.Engine) *GuardedDB {
	b.Helper()

	guarded, err := NewGuardedDB(engine, benchmarkExecutor{}, CacheOptions{
		Capacity:    benchmarkCacheCapacity,
		MaxSQLBytes: 128,
	})
	if err != nil {
		b.Fatalf("construct guarded database: %v", err)
	}

	return guarded
}

func newBenchmarkEngine(b *testing.B) *sqlguard.Engine {
	b.Helper()

	engine, err := sqlguard.NewEngine(sqlguard.EngineOptions{})
	if err != nil {
		b.Fatalf("construct engine: %v", err)
	}

	return engine
}
