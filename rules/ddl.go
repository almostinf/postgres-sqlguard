package rules

import (
	"context"

	sqlguard "github.com/almostinf/postgres-sqlguard"
)

const (
	denyTruncateRuleID      = "deny_truncate"
	denyDropTableRuleID     = "deny_drop_table"
	denyAlterTableRuleID    = "deny_alter_table"
	truncateStatementKind   = sqlguard.Kind("TruncateStmt")
	dropStatementKind       = sqlguard.Kind("DropStmt")
	alterTableStatementKind = sqlguard.Kind("AlterTableStmt")
	alterTableMoveAllKind   = sqlguard.Kind("AlterTableMoveAllStmt")
	dropObjectTypeField     = "remove_type"
	alterObjectTypeField    = "objtype"
	tableObjectType         = "OBJECT_TABLE"
)

type denyTruncate struct{}

type denyDropTable struct{}

type denyAlterTable struct{}

var (
	_ sqlguard.Rule = denyTruncate{}
	_ sqlguard.Rule = denyDropTable{}
	_ sqlguard.Rule = denyAlterTable{}
)

// NewDenyTruncate returns a rule that rejects TRUNCATE statements.
func NewDenyTruncate() sqlguard.Rule {
	return denyTruncate{}
}

// NewDenyDropTable returns a rule that rejects DROP TABLE statements.
func NewDenyDropTable() sqlguard.Rule {
	return denyDropTable{}
}

// NewDenyAlterTable returns a rule that rejects ALTER TABLE statements.
func NewDenyAlterTable() sqlguard.Rule {
	return denyAlterTable{}
}

func (denyTruncate) ID() string {
	return denyTruncateRuleID
}

func (denyTruncate) Evaluate(_ context.Context, statement sqlguard.Statement) sqlguard.RuleResult {
	if statement.Kind() == truncateStatementKind {
		return sqlguard.Reject()
	}

	return sqlguard.Allow()
}

func (denyDropTable) ID() string {
	return denyDropTableRuleID
}

func (denyDropTable) Evaluate(_ context.Context, statement sqlguard.Statement) sqlguard.RuleResult {
	if statement.Kind() != dropStatementKind {
		return sqlguard.Allow()
	}

	objectType, ok := statement.Root().Enum(dropObjectTypeField)
	if !ok || objectType == tableObjectType {
		return sqlguard.Reject()
	}

	return sqlguard.Allow()
}

func (denyAlterTable) ID() string {
	return denyAlterTableRuleID
}

func (denyAlterTable) Evaluate(_ context.Context, statement sqlguard.Statement) sqlguard.RuleResult {
	switch statement.Kind() {
	case alterTableStatementKind, alterTableMoveAllKind:
	default:
		return sqlguard.Allow()
	}

	objectType, ok := statement.Root().Enum(alterObjectTypeField)
	if !ok || objectType == tableObjectType {
		return sqlguard.Reject()
	}

	return sqlguard.Allow()
}
