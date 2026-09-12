package sqlguard

import (
	"errors"

	"github.com/almostinf/postgres-sqlguard/internal/parser"
)

// ErrInvalidPrepared indicates that prepared validation received a zero or
// otherwise invalid Prepared value. It contains no SQL or parsed structure.
var ErrInvalidPrepared = errors.New("sqlguard: invalid prepared value")

const (
	violationMessage  = "sql validation rejected by rule"
	parseErrorMessage = "sql validation parsing failed"
)

// ParseErrorCategory identifies a bounded class of parsing failure. Categories
// are safe for programmatic handling and never contain parser diagnostics.
type ParseErrorCategory string

const (
	// ParseErrorUnknown indicates a parsing failure without a more specific
	// public category.
	ParseErrorUnknown ParseErrorCategory = "unknown"

	// ParseErrorSyntax indicates that PostgreSQL rejected the input syntax.
	ParseErrorSyntax ParseErrorCategory = "syntax"
)

// Violation reports that a validation rule rejected a statement. It stores
// only the stable identifier of the responsible rule.
type Violation struct {
	ruleID string
}

// newViolation creates a privacy-safe violation for an already validated rule
// identifier. Rule identifier validation belongs to Engine registration.
func newViolation(ruleID string) *Violation {
	return &Violation{ruleID: ruleID}
}

// Error returns a constant message that cannot disclose validation input.
func (*Violation) Error() string {
	return violationMessage
}

// RuleID returns the stable identifier of the rule that rejected a statement.
func (e *Violation) RuleID() string {
	if e == nil {
		return ""
	}

	return e.ruleID
}

// ParseError reports that PostgreSQL parsing failed before rule evaluation. It
// contains only a bounded category and never retains backend diagnostics.
type ParseError struct {
	category ParseErrorCategory
}

// newParseError creates a privacy-safe parser failure from a bounded category.
func newParseError(category ParseErrorCategory) *ParseError {
	return &ParseError{category: category}
}

// Error returns a constant message that cannot disclose parser input.
func (*ParseError) Error() string {
	return parseErrorMessage
}

// Category returns the bounded parser-failure category.
func (e *ParseError) Category() ParseErrorCategory {
	if e == nil {
		return ""
	}

	return e.category
}

// translateParserError converts an internal parser failure into the public
// error contract. It deliberately copies only a bounded category and never
// retains or unwraps the source error.
func translateParserError(source error) error {
	if source == nil {
		return nil
	}

	category := ParseErrorUnknown

	var parserError *parser.Error
	if errors.As(source, &parserError) && parserError.Category() == parser.FailureSyntax {
		category = ParseErrorSyntax
	}

	return newParseError(category)
}
