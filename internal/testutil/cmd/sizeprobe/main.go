// Package main provides the representative executable used to compare parser-backend footprint.
package main

import (
	"context"

	sqlguard "github.com/almostinf/postgres-sqlguard"
	"github.com/almostinf/postgres-sqlguard/pkg/rules"
)

func main() {
	engine, err := sqlguard.NewEngine(sqlguard.EngineOptions{},
		rules.NewUpdateRequiresWhere(),
		rules.NewDeleteRequiresWhere(),
	)
	if err != nil {
		panic(err)
	}

	queries := []string{
		"SELECT account_id FROM accounts WHERE active",
		"UPDATE accounts SET active = false WHERE account_id = 42",
		"DELETE FROM sessions WHERE expired",
	}

	for _, query := range queries {
		if err := engine.Validate(context.Background(), query); err != nil {
			panic(err)
		}
	}
}
