package pgxprepared

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"

	sqlguard "github.com/almostinf/postgres-sqlguard"
)

var (
	_ PreparedValidator = (*sqlguard.Engine)(nil)
	_ Executor          = (*pgx.Conn)(nil)
)

type preparedValidatorStub struct {
	mutex sync.Mutex

	prepareFunc       func(context.Context, string) (sqlguard.Prepared, error)
	validateFunc      func(context.Context, sqlguard.Prepared) error
	prepareCallsBySQL map[string]int
	validateCalls     int
}

func (v *preparedValidatorStub) Prepare(ctx context.Context, sql string) (sqlguard.Prepared, error) {
	v.mutex.Lock()
	if v.prepareCallsBySQL == nil {
		v.prepareCallsBySQL = make(map[string]int)
	}

	v.prepareCallsBySQL[sql]++
	prepareFunc := v.prepareFunc
	v.mutex.Unlock()

	return prepareFunc(ctx, sql)
}

func (v *preparedValidatorStub) ValidatePrepared(ctx context.Context, prepared sqlguard.Prepared) error {
	v.mutex.Lock()
	v.validateCalls++
	validateFunc := v.validateFunc
	v.mutex.Unlock()

	return validateFunc(ctx, prepared)
}

func (v *preparedValidatorStub) Calls(sql string) (int, int) {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	return v.prepareCallsBySQL[sql], v.validateCalls
}

type executorStub struct{}

func (*executorStub) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (*executorStub) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, nil
}

func (*executorStub) QueryRow(context.Context, string, ...any) pgx.Row {
	return nil
}

type mutableRule struct {
	reject atomic.Bool
}

func (*mutableRule) ID() string {
	return "mutable_rule"
}

func (r *mutableRule) Evaluate(context.Context, sqlguard.Statement) sqlguard.RuleResult {
	if r.reject.Load() {
		return sqlguard.Reject()
	}

	return sqlguard.Allow()
}

func TestNewGuardedDB(t *testing.T) {
	validValidator := newRealPreparedValidator(t)

	tests := map[string]struct {
		validator PreparedValidator
		executor  Executor
		options   CacheOptions
		wantError string
	}{
		"accepts_valid_configuration": {
			validator: validValidator,
			executor:  &executorStub{},
			options: CacheOptions{
				Capacity:    2,
				MaxSQLBytes: 1024,
			},
		},
		"rejects_nil_executor": {
			validator: validValidator,
			options:   CacheOptions{Capacity: 1, MaxSQLBytes: 1},
			wantError: "sqlguard pgx-prepared example: executor must not be nil",
		},
		"rejects_nil_validator": {
			executor:  &executorStub{},
			options:   CacheOptions{Capacity: 1, MaxSQLBytes: 1},
			wantError: "sqlguard pgx-prepared example: validator must not be nil",
		},
		"rejects_non_positive_capacity": {
			validator: validValidator,
			executor:  &executorStub{},
			options:   CacheOptions{MaxSQLBytes: 1},
			wantError: "sqlguard pgx-prepared example: cache capacity must be positive",
		},
		"rejects_non_positive_sql_limit": {
			validator: validValidator,
			executor:  &executorStub{},
			options:   CacheOptions{Capacity: 1},
			wantError: "sqlguard pgx-prepared example: maximum SQL bytes must be positive",
		},
		"rejects_typed_nil_executor": {
			validator: validValidator,
			executor:  (*executorStub)(nil),
			options:   CacheOptions{Capacity: 1, MaxSQLBytes: 1},
			wantError: "sqlguard pgx-prepared example: executor must not be nil",
		},
		"rejects_typed_nil_validator": {
			validator: (*preparedValidatorStub)(nil),
			executor:  &executorStub{},
			options:   CacheOptions{Capacity: 1, MaxSQLBytes: 1},
			wantError: "sqlguard pgx-prepared example: validator must not be nil",
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			guarded, err := NewGuardedDB(testCase.validator, testCase.executor, testCase.options)
			if testCase.wantError != "" {
				require.EqualError(t, err, testCase.wantError)
				require.Nil(t, guarded)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, guarded)
			require.NotNil(t, guarded.cache)
		})
	}
}

func TestGuardedDBCacheHitAndExactMiss(t *testing.T) {
	validator := newRealPreparedValidator(t)
	guarded := mustNewGuardedDB(t, validator, 2, 1024)

	require.NoError(t, guarded.validate(context.Background(), "SELECT 1"))
	require.NoError(t, guarded.validate(context.Background(), "SELECT 1"))
	require.NoError(t, guarded.validate(context.Background(), "SELECT  1"))

	firstPrepares, validations := validator.Calls("SELECT 1")
	secondPrepares, _ := validator.Calls("SELECT  1")

	require.Equal(t, 1, firstPrepares)
	require.Equal(t, 1, secondPrepares)
	require.Equal(t, 3, validations)
	require.Equal(t, 2, guarded.cache.Len())
}

func TestGuardedDBCachesAreIndependent(t *testing.T) {
	validator := newRealPreparedValidator(t)
	first := mustNewGuardedDB(t, validator, 1, 1024)
	second := mustNewGuardedDB(t, validator, 1, 1024)

	require.NoError(t, first.validate(context.Background(), "SELECT 1"))
	require.NoError(t, first.validate(context.Background(), "SELECT 1"))
	require.NoError(t, second.validate(context.Background(), "SELECT 1"))

	prepares, validations := validator.Calls("SELECT 1")
	require.Equal(t, 2, prepares)
	require.Equal(t, 3, validations)
	require.Equal(t, 1, first.cache.Len())
	require.Equal(t, 1, second.cache.Len())
}

func TestGuardedDBCacheUpdatesRecencyAndEvicts(t *testing.T) {
	validator := newRealPreparedValidator(t)
	guarded := mustNewGuardedDB(t, validator, 2, 1024)

	for _, sql := range []string{"SELECT 1", "SELECT 2", "SELECT 1", "SELECT 3", "SELECT 2"} {
		require.NoError(t, guarded.validate(context.Background(), sql))
	}

	firstPrepares, _ := validator.Calls("SELECT 1")
	secondPrepares, _ := validator.Calls("SELECT 2")
	thirdPrepares, validations := validator.Calls("SELECT 3")

	require.Equal(t, 1, firstPrepares)
	require.Equal(t, 2, secondPrepares)
	require.Equal(t, 1, thirdPrepares)
	require.Equal(t, 5, validations)
	require.Equal(t, 2, guarded.cache.Len())
}

func TestGuardedDBCacheBypassesOversizedSQL(t *testing.T) {
	validator := newRealPreparedValidator(t)
	guarded := mustNewGuardedDB(t, validator, 1, len("SELECT 1")-1)

	require.NoError(t, guarded.validate(context.Background(), "SELECT 1"))
	require.NoError(t, guarded.validate(context.Background(), "SELECT 1"))

	prepares, validations := validator.Calls("SELECT 1")
	require.Equal(t, 2, prepares)
	require.Equal(t, 2, validations)
	require.Zero(t, guarded.cache.Len())
}

func TestGuardedDBCacheOwnsInsertedKey(t *testing.T) {
	validator := newRealPreparedValidator(t)
	guarded := mustNewGuardedDB(t, validator, 1, 1024)
	backing := strings.Repeat("x", 4096) + "SELECT 1"
	sql := backing[len(backing)-len("SELECT 1"):]
	originalData := unsafe.StringData(sql)

	require.NoError(t, guarded.validate(context.Background(), sql))

	keys := guarded.cache.Keys()
	require.Equal(t, []string{sql}, keys)
	require.False(t, originalData == unsafe.StringData(keys[0]))
	runtime.KeepAlive(backing)
}

func TestGuardedDBCacheExcludesFailedPreparation(t *testing.T) {
	validator := newRealPreparedValidator(t)
	guarded := mustNewGuardedDB(t, validator, 1, 1024)

	for range 2 {
		err := guarded.validate(context.Background(), "SELECT * FROM")

		var parseError *sqlguard.ParseError
		require.ErrorAs(t, err, &parseError)
	}

	prepares, validations := validator.Calls("SELECT * FROM")
	require.Equal(t, 2, prepares)
	require.Zero(t, validations)
	require.Zero(t, guarded.cache.Len())
}

func TestGuardedDBCacheExcludesCanceledInvalidAndUnknownValidation(t *testing.T) {
	unknownError := errors.New("validation unavailable")

	tests := map[string]struct {
		setup func(t *testing.T) (context.Context, *preparedValidatorStub)
		check func(t *testing.T, err error)
	}{
		"cancellation_after_preparation": {
			setup: func(t *testing.T) (context.Context, *preparedValidatorStub) {
				t.Helper()

				prepared := mustPrepare(t, "SELECT 1")
				ctx, cancel := context.WithCancel(context.Background())

				return ctx, &preparedValidatorStub{
					prepareFunc: func(context.Context, string) (sqlguard.Prepared, error) {
						cancel()

						return prepared, nil
					},
					validateFunc: func(ctx context.Context, _ sqlguard.Prepared) error {
						return ctx.Err()
					},
				}
			},
			check: func(t *testing.T, err error) {
				t.Helper()

				require.ErrorIs(t, err, context.Canceled)
			},
		},
		"invalid_prepared": {
			setup: func(t *testing.T) (context.Context, *preparedValidatorStub) {
				t.Helper()

				return context.Background(), &preparedValidatorStub{
					prepareFunc: func(context.Context, string) (sqlguard.Prepared, error) {
						return sqlguard.Prepared{}, nil
					},
					validateFunc: func(context.Context, sqlguard.Prepared) error {
						return sqlguard.ErrInvalidPrepared
					},
				}
			},
			check: func(t *testing.T, err error) {
				t.Helper()

				require.ErrorIs(t, err, sqlguard.ErrInvalidPrepared)
			},
		},
		"unknown_error": {
			setup: func(t *testing.T) (context.Context, *preparedValidatorStub) {
				t.Helper()

				return context.Background(), &preparedValidatorStub{
					prepareFunc: func(context.Context, string) (sqlguard.Prepared, error) {
						return mustPrepare(t, "SELECT 1"), nil
					},
					validateFunc: func(context.Context, sqlguard.Prepared) error {
						return unknownError
					},
				}
			},
			check: func(t *testing.T, err error) {
				t.Helper()

				require.ErrorIs(t, err, unknownError)
			},
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx, validator := testCase.setup(t)
			guarded := mustNewGuardedDB(t, validator, 1, 1024)

			testCase.check(t, guarded.validate(ctx, "SELECT 1"))
			require.Zero(t, guarded.cache.Len())
		})
	}
}

func TestGuardedDBCacheReevaluatesPolicyViolation(t *testing.T) {
	rule := &mutableRule{}
	rule.reject.Store(true)
	engine, err := sqlguard.NewEngine(sqlguard.EngineOptions{}, rule)
	require.NoError(t, err)

	validator := &preparedValidatorStub{
		prepareFunc:  engine.Prepare,
		validateFunc: engine.ValidatePrepared,
	}
	guarded := mustNewGuardedDB(t, validator, 1, 1024)

	err = guarded.validate(context.Background(), "SELECT 1")

	var violation *sqlguard.Violation
	require.ErrorAs(t, err, &violation)
	require.Equal(t, 1, guarded.cache.Len())

	rule.reject.Store(false)
	require.NoError(t, guarded.validate(context.Background(), "SELECT 1"))

	prepares, validations := validator.Calls("SELECT 1")
	require.Equal(t, 1, prepares)
	require.Equal(t, 2, validations)
}

func TestGuardedDBCacheConcurrentExactHits(t *testing.T) {
	const callCount = 64

	validator := newRealPreparedValidator(t)
	guarded := mustNewGuardedDB(t, validator, 2, 1024)
	require.NoError(t, guarded.validate(context.Background(), "SELECT 1"))

	errors := runConcurrentValidations(callCount, func(_ int) error {
		return guarded.validate(context.Background(), "SELECT 1")
	})

	requireNoErrors(t, errors)

	prepares, validations := validator.Calls("SELECT 1")
	require.Equal(t, 1, prepares)
	require.Equal(t, callCount+1, validations)
	require.Equal(t, 1, guarded.cache.Len())
}

func TestGuardedDBCacheConcurrentIdenticalMissesKeepDecisionsIndependent(t *testing.T) {
	const callCount = 32

	prepared := mustPrepare(t, "SELECT 1")
	arrived := make(chan struct{}, callCount)
	release := make(chan struct{})
	validator := &preparedValidatorStub{
		prepareFunc: func(context.Context, string) (sqlguard.Prepared, error) {
			arrived <- struct{}{}

			<-release

			return prepared, nil
		},
		validateFunc: func(ctx context.Context, _ sqlguard.Prepared) error {
			if index, ok := ctx.Value(contextKey{}).(int); ok && index%2 != 0 {
				return errPerCallDecision
			}

			return nil
		},
	}
	guarded := mustNewGuardedDB(t, validator, 4, 1024)

	errorsDone := make(chan []error, 1)

	var releaseOnce sync.Once

	releaseAll := func() {
		releaseOnce.Do(func() {
			close(release)
		})
	}
	defer releaseAll()

	go func() {
		errorsDone <- runConcurrentValidations(callCount, func(index int) error {
			ctx := context.WithValue(context.Background(), contextKey{}, index)

			return guarded.validate(ctx, "SELECT 1")
		})
	}()

	waitForConcurrentArrivals(t, arrived, errorsDone, callCount, releaseAll)
	releaseAll()

	errors := waitForConcurrentErrors(t, errorsDone)
	for index, validationError := range errors {
		if index%2 == 0 {
			require.NoError(t, validationError)
		} else {
			require.ErrorIs(t, validationError, errPerCallDecision)
		}
	}

	prepares, validations := validator.Calls("SELECT 1")
	require.Equal(t, callCount, prepares)
	require.Equal(t, callCount, validations)
	require.Equal(t, 1, guarded.cache.Len())
}

func TestGuardedDBCacheConcurrentInsertionsStayWithinCapacity(t *testing.T) {
	const (
		callCount = 64
		capacity  = 4
	)

	validator := newRealPreparedValidator(t)
	guarded := mustNewGuardedDB(t, validator, capacity, 1024)

	inputs := make([]string, callCount)
	for index := range inputs {
		inputs[index] = fmt.Sprintf("SELECT %d", index)
	}

	errors := runConcurrentValidations(callCount, func(index int) error {
		return guarded.validate(context.Background(), inputs[index])
	})

	requireNoErrors(t, errors)
	require.LessOrEqual(t, guarded.cache.Len(), capacity)
}

var errPerCallDecision = errors.New("per-call validation decision")

type contextKey struct{}

func newRealPreparedValidator(t *testing.T) *preparedValidatorStub {
	t.Helper()

	engine, err := sqlguard.NewEngine(sqlguard.EngineOptions{})
	require.NoError(t, err)

	return &preparedValidatorStub{
		prepareFunc:  engine.Prepare,
		validateFunc: engine.ValidatePrepared,
	}
}

func mustPrepare(t *testing.T, sql string) sqlguard.Prepared {
	t.Helper()

	engine, err := sqlguard.NewEngine(sqlguard.EngineOptions{})
	require.NoError(t, err)
	prepared, err := engine.Prepare(context.Background(), sql)
	require.NoError(t, err)

	return prepared
}

func mustNewGuardedDB(
	t *testing.T,
	validator PreparedValidator,
	capacity int,
	maxSQLBytes int,
) *GuardedDB {
	t.Helper()

	guarded, err := NewGuardedDB(validator, &executorStub{}, CacheOptions{
		Capacity:    capacity,
		MaxSQLBytes: maxSQLBytes,
	})
	require.NoError(t, err)

	return guarded
}

func runConcurrentValidations(callCount int, validate func(int) error) []error {
	errors := make([]error, callCount)

	var waitGroup sync.WaitGroup
	waitGroup.Add(callCount)

	for index := range callCount {
		go func() {
			defer waitGroup.Done()

			errors[index] = validate(index)
		}()
	}

	waitGroup.Wait()

	return errors
}

func requireNoErrors(t *testing.T, errors []error) {
	t.Helper()

	for _, err := range errors {
		require.NoError(t, err)
	}
}

func waitForConcurrentArrivals(
	t *testing.T,
	arrived <-chan struct{},
	errorsDone <-chan []error,
	want int,
	release func(),
) {
	t.Helper()

	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

	for range want {
		select {
		case <-arrived:
		case errors := <-errorsDone:
			release()
			t.Fatalf("concurrent calls completed before reaching validation barrier: %v", errors)
		case <-timer.C:
			release()
			t.Fatal("timed out waiting for concurrent validation barrier")
		}
	}
}

func waitForConcurrentErrors(t *testing.T, errorsDone <-chan []error) []error {
	t.Helper()

	select {
	case errors := <-errorsDone:
		return errors
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for concurrent calls")

		return nil
	}
}
