package sqlguard_test

import (
	"context"

	sqlguard "github.com/almostinf/postgres-sqlguard"
)

type compileContractRule struct{}

func (compileContractRule) ID() string {
	return "compile-contract"
}

func (compileContractRule) Evaluate(context.Context, sqlguard.Statement) sqlguard.RuleResult {
	return sqlguard.Allow()
}

type compileContractMetrics struct{}

func (compileContractMetrics) RecordValidation(sqlguard.ValidationEvent) error {
	return nil
}

type compileContractLogger struct{}

func (compileContractLogger) LogValidation(sqlguard.ValidationEvent) error {
	return nil
}

var (
	_ sqlguard.Rule      = compileContractRule{}
	_ sqlguard.Metrics   = compileContractMetrics{}
	_ sqlguard.Logger    = compileContractLogger{}
	_ sqlguard.Validator = (*sqlguard.Engine)(nil)
	_ error              = (*sqlguard.Violation)(nil)
	_ error              = (*sqlguard.ParseError)(nil)
	_                    = sqlguard.EngineOptions{Metrics: compileContractMetrics{}, Logger: compileContractLogger{}}

	_ *sqlguard.Engine
	_ sqlguard.EngineOptions
	_ sqlguard.Prepared
	_ sqlguard.Statement
	_ sqlguard.Node
	_ sqlguard.Kind
	_ sqlguard.RuleResult
	_ *sqlguard.Violation
	_ *sqlguard.ParseError
	_ sqlguard.ParseErrorCategory
	_ sqlguard.ValidationEvent
	_ sqlguard.ValidationMode
	_ sqlguard.ValidationOutcome

	_ func(sqlguard.EngineOptions, ...sqlguard.Rule) (*sqlguard.Engine, error)   = sqlguard.NewEngine
	_ func(*sqlguard.Engine, context.Context, string) error                      = (*sqlguard.Engine).Validate
	_ func(*sqlguard.Engine, context.Context, string) (sqlguard.Prepared, error) = (*sqlguard.Engine).Prepare
	_ func(*sqlguard.Engine, context.Context, sqlguard.Prepared) error           = (*sqlguard.Engine).ValidatePrepared

	_ func(sqlguard.Validator, context.Context, string) error                      = sqlguard.Validator.Validate
	_ func(sqlguard.Rule) string                                                   = sqlguard.Rule.ID
	_ func(sqlguard.Rule, context.Context, sqlguard.Statement) sqlguard.RuleResult = sqlguard.Rule.Evaluate
	_ func(sqlguard.Metrics, sqlguard.ValidationEvent) error                       = sqlguard.Metrics.RecordValidation
	_ func(sqlguard.Logger, sqlguard.ValidationEvent) error                        = sqlguard.Logger.LogValidation

	_ func() sqlguard.RuleResult                         = sqlguard.Allow
	_ func() sqlguard.RuleResult                         = sqlguard.Reject
	_ func(sqlguard.RuleResult) bool                     = sqlguard.RuleResult.Rejected
	_ func(sqlguard.Statement) sqlguard.Kind             = sqlguard.Statement.Kind
	_ func(sqlguard.Statement) sqlguard.Node             = sqlguard.Statement.Root
	_ func(sqlguard.Statement, func(sqlguard.Node) bool) = sqlguard.Statement.Walk
	_ func(sqlguard.Node) sqlguard.Kind                  = sqlguard.Node.Kind
	_ func(sqlguard.Node, string) (sqlguard.Node, bool)  = sqlguard.Node.Child
	_ func(sqlguard.Node, string) []sqlguard.Node        = sqlguard.Node.Children
	_ func(sqlguard.Node, string) (bool, bool)           = sqlguard.Node.Bool
	_ func(sqlguard.Node, string) (int64, bool)          = sqlguard.Node.Int
	_ func(sqlguard.Node, string) (uint64, bool)         = sqlguard.Node.Uint
	_ func(sqlguard.Node, string) (float64, bool)        = sqlguard.Node.Float
	_ func(sqlguard.Node, string) (string, bool)         = sqlguard.Node.String
	_ func(sqlguard.Node, string) ([]byte, bool)         = sqlguard.Node.Bytes
	_ func(sqlguard.Node, string) (string, bool)         = sqlguard.Node.Enum
	_ func(sqlguard.Node, func(sqlguard.Node) bool)      = sqlguard.Node.Walk

	_ func(*sqlguard.Violation) string                          = (*sqlguard.Violation).Error
	_ func(*sqlguard.Violation) string                          = (*sqlguard.Violation).RuleID
	_ func(*sqlguard.ParseError) string                         = (*sqlguard.ParseError).Error
	_ func(*sqlguard.ParseError) sqlguard.ParseErrorCategory    = (*sqlguard.ParseError).Category
	_ func(sqlguard.ValidationEvent) sqlguard.ValidationMode    = sqlguard.ValidationEvent.Mode
	_ func(sqlguard.ValidationEvent) sqlguard.ValidationOutcome = sqlguard.ValidationEvent.Outcome
	_ func(sqlguard.ValidationEvent) string                     = sqlguard.ValidationEvent.RuleID

	_ error                       = sqlguard.ErrInvalidPrepared
	_ sqlguard.ParseErrorCategory = sqlguard.ParseErrorUnknown
	_ sqlguard.ParseErrorCategory = sqlguard.ParseErrorSyntax
	_ sqlguard.ValidationMode     = sqlguard.ValidationModeEnforce
	_ sqlguard.ValidationOutcome  = sqlguard.ValidationOutcomeAllowed
	_ sqlguard.ValidationOutcome  = sqlguard.ValidationOutcomePolicyViolation
	_ sqlguard.ValidationOutcome  = sqlguard.ValidationOutcomeParserFailure
	_ sqlguard.ValidationOutcome  = sqlguard.ValidationOutcomeCanceled
	_ sqlguard.ValidationOutcome  = sqlguard.ValidationOutcomeInvalidPrepared
)
