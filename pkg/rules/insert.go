package rules

import (
	"context"

	sqlguard "github.com/almostinf/postgres-sqlguard"
)

const (
	insertRequiresColumnsRuleID = "insert_requires_columns"
	insertStatementKind         = sqlguard.Kind("InsertStmt")
	insertColumnsField          = "cols"
)

type insertRequiresColumns struct{}

var _ sqlguard.Rule = insertRequiresColumns{}

// NewInsertRequiresColumns returns a rule that requires INSERT statements to
// contain an explicit target-column list.
func NewInsertRequiresColumns() sqlguard.Rule {
	return insertRequiresColumns{}
}

func (insertRequiresColumns) ID() string {
	return insertRequiresColumnsRuleID
}

func (insertRequiresColumns) Evaluate(
	_ context.Context,
	statement sqlguard.Statement,
) sqlguard.RuleResult {
	if statement.Kind() != insertStatementKind {
		return sqlguard.Allow()
	}

	if len(statement.Root().Children(insertColumnsField)) > 0 {
		return sqlguard.Allow()
	}

	return sqlguard.Reject()
}
