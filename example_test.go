package sqlguard_test

import (
	"context"
	"errors"
	"fmt"

	sqlguard "github.com/almostinf/postgres-sqlguard"
	"github.com/almostinf/postgres-sqlguard/pkg/rules"
)

func Example() {
	engine, err := sqlguard.NewEngine(
		sqlguard.EngineOptions{},
		rules.NewUpdateRequiresWhere(),
	)
	if err != nil {
		fmt.Println("configure sqlguard:", err)

		return
	}

	err = engine.Validate(
		context.Background(),
		"UPDATE accounts SET active = false",
	)

	var violation *sqlguard.Violation
	if errors.As(err, &violation) {
		fmt.Println("rejected by", violation.RuleID())
	}

	// Output:
	// rejected by update_requires_where
}
