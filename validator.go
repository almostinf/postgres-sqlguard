package sqlguard

import "context"

// Validator checks SQL against a configured policy without requiring a
// database connection or a database-driver dependency.
type Validator interface {
	// Validate checks the complete SQL input and returns the first failure.
	Validate(ctx context.Context, sql string) error
}
