package pgxprepared

import (
	"context"
	"errors"
	"reflect"
	"strings"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	sqlguard "github.com/almostinf/postgres-sqlguard"
)

var (
	errNilValidator       = errors.New("sqlguard pgx-prepared example: validator must not be nil")
	errNilExecutor        = errors.New("sqlguard pgx-prepared example: executor must not be nil")
	errInvalidCapacity    = errors.New("sqlguard pgx-prepared example: cache capacity must be positive")
	errInvalidMaxSQLBytes = errors.New("sqlguard pgx-prepared example: maximum SQL bytes must be positive")
)

// ErrQueryRewriterUnsupported reports that an argument could replace SQL
// after it has passed validation.
var ErrQueryRewriterUnsupported = errors.New(
	"sqlguard pgx-prepared example: query rewriting is unsupported",
)

// PreparedValidator is the prepared-validation behavior required by GuardedDB.
type PreparedValidator interface {
	Prepare(context.Context, string) (sqlguard.Prepared, error)
	ValidatePrepared(context.Context, sqlguard.Prepared) error
}

// Executor is the subset of pgx connection and pool operations guarded by the
// example.
type Executor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// CacheOptions configures bounded prepared-value retention for one GuardedDB.
type CacheOptions struct {
	Capacity    int
	MaxSQLBytes int
}

// GuardedDB demonstrates a prepared-validation cache around a narrow subset of
// pgx. It is an illustrative example rather than a production-ready adapter.
type GuardedDB struct {
	validator   PreparedValidator
	executor    Executor
	cache       *lru.Cache[string, sqlguard.Prepared]
	maxSQLBytes int
}

// NewGuardedDB validates its collaborators and cache limits and constructs an
// illustrative pgx wrapper with a private fixed-capacity LRU cache.
func NewGuardedDB(
	validator PreparedValidator,
	executor Executor,
	options CacheOptions,
) (*GuardedDB, error) {
	if isNil(validator) {
		return nil, errNilValidator
	}

	if isNil(executor) {
		return nil, errNilExecutor
	}

	if options.Capacity <= 0 {
		return nil, errInvalidCapacity
	}

	if options.MaxSQLBytes <= 0 {
		return nil, errInvalidMaxSQLBytes
	}

	cache, err := lru.New[string, sqlguard.Prepared](options.Capacity)
	if err != nil {
		return nil, errInvalidCapacity
	}

	return &GuardedDB{
		validator:   validator,
		executor:    executor,
		cache:       cache,
		maxSQLBytes: options.MaxSQLBytes,
	}, nil
}

// Exec validates SQL through the prepared cache before delegating to the
// wrapped executor.
func (db *GuardedDB) Exec(
	ctx context.Context,
	sql string,
	arguments ...any,
) (pgconn.CommandTag, error) {
	if containsQueryRewriter(arguments) {
		return pgconn.CommandTag{}, ErrQueryRewriterUnsupported
	}

	if err := db.validate(ctx, sql); err != nil {
		return pgconn.CommandTag{}, err
	}

	return db.executor.Exec(ctx, sql, arguments...)
}

// Query validates SQL through the prepared cache before delegating to the
// wrapped executor.
func (db *GuardedDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if containsQueryRewriter(args) {
		return nil, ErrQueryRewriterUnsupported
	}

	if err := db.validate(ctx, sql); err != nil {
		return nil, err
	}

	return db.executor.Query(ctx, sql, args...)
}

// QueryRow validates SQL through the prepared cache before delegating to the
// wrapped executor. Guard failures are returned by the resulting row's Scan.
func (db *GuardedDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if containsQueryRewriter(args) {
		return failedRow{err: ErrQueryRewriterUnsupported}
	}

	if err := db.validate(ctx, sql); err != nil {
		return failedRow{err: err}
	}

	return db.executor.QueryRow(ctx, sql, args...)
}

func (db *GuardedDB) validate(ctx context.Context, sql string) error {
	if prepared, ok := db.cache.Get(sql); ok {
		return db.validator.ValidatePrepared(ctx, prepared)
	}

	prepared, err := db.validator.Prepare(ctx, sql)
	if err != nil {
		return err
	}

	err = db.validator.ValidatePrepared(ctx, prepared)
	if len(sql) <= db.maxSQLBytes && isCacheableValidationOutcome(err) {
		db.cache.Add(strings.Clone(sql), prepared)
	}

	return err
}

func isCacheableValidationOutcome(err error) bool {
	if err == nil {
		return true
	}

	var violation *sqlguard.Violation

	return errors.As(err, &violation)
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
