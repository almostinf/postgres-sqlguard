package sqlguard

import "github.com/almostinf/postgres-sqlguard/internal/parser"

// Prepared is an opaque immutable representation of completely parsed SQL.
// A successfully prepared value may be copied and validated concurrently by
// any Engine. The zero value is invalid and is rejected by ValidatePrepared.
type Prepared struct {
	result *parser.Result
}
