package parser

// Result is an immutable parser-neutral PostgreSQL parse result. It owns the
// top-level statement order produced by PostgreSQL.
type Result struct {
	statements []*Node
}

// Statements returns a snapshot of the top-level statements in input order.
func (r *Result) Statements() []*Node {
	if r == nil {
		return nil
	}

	return append([]*Node(nil), r.statements...)
}

// FailureCategory identifies a bounded parser failure class.
type FailureCategory string

// FailureSyntax indicates that PostgreSQL rejected the input syntax.
const FailureSyntax FailureCategory = "syntax"

// Error is a privacy-safe failure from the internal parser boundary. It never
// stores or unwraps the backend error because that diagnostic can quote SQL.
type Error struct {
	category FailureCategory
}

// Error returns a constant message that does not disclose parser input.
func (*Error) Error() string {
	return "postgresql parsing failed"
}

// Category returns the bounded parser failure category.
func (e *Error) Category() FailureCategory {
	if e == nil {
		return ""
	}

	return e.category
}
