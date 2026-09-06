package rules

import (
	"context"

	sqlguard "github.com/almostinf/postgres-sqlguard"
)

const (
	updateRequiresWhereRuleID = "update_requires_where"
	deleteRequiresWhereRuleID = "delete_requires_where"
	updateStatementKind       = sqlguard.Kind("UpdateStmt")
	deleteStatementKind       = sqlguard.Kind("DeleteStmt")
	whereClauseField          = "where_clause"
)

type updateRequiresWhere struct{}

type deleteRequiresWhere struct{}

var (
	_ sqlguard.Rule = updateRequiresWhere{}
	_ sqlguard.Rule = deleteRequiresWhere{}
)

// NewUpdateRequiresWhere returns a rule that requires UPDATE statements to
// contain a syntactic WHERE clause.
func NewUpdateRequiresWhere() sqlguard.Rule {
	return updateRequiresWhere{}
}

// NewDeleteRequiresWhere returns a rule that requires DELETE statements to
// contain a syntactic WHERE clause.
func NewDeleteRequiresWhere() sqlguard.Rule {
	return deleteRequiresWhere{}
}

func (updateRequiresWhere) ID() string {
	return updateRequiresWhereRuleID
}

func (updateRequiresWhere) Evaluate(_ context.Context, statement sqlguard.Statement) sqlguard.RuleResult {
	return requireWhereClause(statement, updateStatementKind)
}

func (deleteRequiresWhere) ID() string {
	return deleteRequiresWhereRuleID
}

func (deleteRequiresWhere) Evaluate(_ context.Context, statement sqlguard.Statement) sqlguard.RuleResult {
	return requireWhereClause(statement, deleteStatementKind)
}

func requireWhereClause(statement sqlguard.Statement, targetKind sqlguard.Kind) sqlguard.RuleResult {
	if statement.Kind() != targetKind {
		return sqlguard.Allow()
	}

	if _, ok := statement.Root().Child(whereClauseField); ok {
		return sqlguard.Allow()
	}

	return sqlguard.Reject()
}
