package pgxexample_test

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	sqlguard "github.com/almostinf/postgres-sqlguard"
	pgxexample "github.com/almostinf/postgres-sqlguard/example/pgx"
	"github.com/almostinf/postgres-sqlguard/rules"
)

func ExampleNewGuardedDB() {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, "postgres://postgres@localhost/application")
	if err != nil {
		return
	}
	defer pool.Close()

	engine, err := sqlguard.NewEngine(
		sqlguard.EngineOptions{},
		rules.NewUpdateRequiresWhere(),
		rules.NewDeleteRequiresWhere(),
	)
	if err != nil {
		return
	}

	database, err := pgxexample.NewGuardedDB(engine, pool)
	if err != nil {
		return
	}

	_, err = database.Exec(
		ctx,
		"UPDATE accounts SET active = $1 WHERE id = $2",
		false,
		42,
	)
	if err != nil {
		return
	}
}
