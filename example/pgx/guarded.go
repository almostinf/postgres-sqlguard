package pgxexample

import (
	"context"
	"errors"
	"reflect"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	sqlguard "github.com/almostinf/postgres-sqlguard"
)

var (
	errNilValidator = errors.New("sqlguard pgx example: validator must not be nil")
	errNilExecutor  = errors.New("sqlguard pgx example: executor must not be nil")
)

// ErrQueryRewriterUnsupported reports that an argument could replace SQL
// after it has passed validation.
var ErrQueryRewriterUnsupported = errors.New("sqlguard pgx example: query rewriting is unsupported")

// Executor is the subset of pgx connection and pool operations guarded by the
// example.
type Executor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// GuardedDB demonstrates a validation boundary around a narrow subset of pgx.
// It is an illustrative example rather than a production-ready adapter.
type GuardedDB struct {
	validator sqlguard.Validator
	executor  Executor
}

// NewGuardedDB constructs an illustrative pgx wrapper from non-nil
// collaborators.
func NewGuardedDB(validator sqlguard.Validator, executor Executor) (*GuardedDB, error) {
	if isNil(validator) {
		return nil, errNilValidator
	}

	if isNil(executor) {
		return nil, errNilExecutor
	}

	return &GuardedDB{
		validator: validator,
		executor:  executor,
	}, nil
}

// Exec validates SQL before delegating to the wrapped executor.
func (db *GuardedDB) Exec(
	ctx context.Context,
	sql string,
	arguments ...any,
) (pgconn.CommandTag, error) {
	if containsQueryRewriter(arguments) {
		return pgconn.CommandTag{}, ErrQueryRewriterUnsupported
	}

	if err := db.validator.Validate(ctx, sql); err != nil {
		return pgconn.CommandTag{}, err
	}

	return db.executor.Exec(ctx, sql, arguments...)
}

// Query validates SQL before delegating to the wrapped executor.
func (db *GuardedDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if containsQueryRewriter(args) {
		return nil, ErrQueryRewriterUnsupported
	}

	if err := db.validator.Validate(ctx, sql); err != nil {
		return nil, err
	}

	return db.executor.Query(ctx, sql, args...)
}

// QueryRow validates SQL before delegating to the wrapped executor.
func (db *GuardedDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if containsQueryRewriter(args) {
		return failedRow{err: ErrQueryRewriterUnsupported}
	}

	if err := db.validator.Validate(ctx, sql); err != nil {
		return failedRow{err: err}
	}

	return db.executor.QueryRow(ctx, sql, args...)
}

type failedRow struct {
	err error
}

func (row failedRow) Scan(...any) error {
	return row.err
}

func containsQueryRewriter(arguments []any) bool {
	for _, argument := range arguments {
		if _, ok := argument.(pgx.QueryRewriter); ok {
			return true
		}
	}

	return false
}

func isNil(value any) bool {
	if value == nil {
		return true
	}

	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}
