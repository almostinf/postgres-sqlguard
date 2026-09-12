package pgxprepared_test

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	sqlguard "github.com/almostinf/postgres-sqlguard"
	pgxprepared "github.com/almostinf/postgres-sqlguard/example/pgx-prepared"
)

type exampleExecutor struct{}

func (exampleExecutor) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func (exampleExecutor) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, nil
}

func (exampleExecutor) QueryRow(context.Context, string, ...any) pgx.Row {
	return nil
}

func ExampleGuardedDB() {
	engine, err := sqlguard.NewEngine(sqlguard.EngineOptions{})
	if err != nil {
		return
	}

	guarded, err := pgxprepared.NewGuardedDB(engine, exampleExecutor{}, pgxprepared.CacheOptions{
		Capacity:    128,
		MaxSQLBytes: 4096,
	})
	if err != nil {
		return
	}

	const sql = "UPDATE accounts SET active = $1 WHERE id = $2"

	_, _ = guarded.Exec(context.Background(), sql, true, 42)
	_, _ = guarded.Exec(context.Background(), sql, false, 84)
}
