package pgxprepared

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"

	sqlguard "github.com/almostinf/postgres-sqlguard"
)

type executorCall struct {
	ctx  context.Context
	sql  string
	args []any
}

type recordingExecutor struct {
	mutex sync.Mutex

	calls  []executorCall
	tag    pgconn.CommandTag
	rows   pgx.Rows
	row    pgx.Row
	err    error
	onCall func(context.Context)
}

func (e *recordingExecutor) Exec(
	ctx context.Context,
	sql string,
	arguments ...any,
) (pgconn.CommandTag, error) {
	e.record(ctx, sql, arguments)

	return e.tag, e.err
}

func (e *recordingExecutor) Query(
	ctx context.Context,
	sql string,
	args ...any,
) (pgx.Rows, error) {
	e.record(ctx, sql, args)

	return e.rows, e.err
}

func (e *recordingExecutor) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	e.record(ctx, sql, args)

	return e.row
}

func (e *recordingExecutor) record(ctx context.Context, sql string, args []any) {
	if e.onCall != nil {
		e.onCall(ctx)
	}

	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.calls = append(e.calls, executorCall{
		ctx:  ctx,
		sql:  sql,
		args: append([]any(nil), args...),
	})
}

func (e *recordingExecutor) Calls() []executorCall {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	return append([]executorCall(nil), e.calls...)
}

type rowStub struct {
	err error
}

func (row *rowStub) Scan(...any) error {
	return row.err
}

type queryRewriterStub struct {
	calls atomic.Int64
}

func (r *queryRewriterStub) RewriteQuery(
	context.Context,
	*pgx.Conn,
	string,
	[]any,
) (string, []any, error) {
	r.calls.Add(1)

	return "SELECT 1", nil, nil
}

func TestGuardedDBExecPreservesAllowedCall(t *testing.T) {
	validator := newRealPreparedValidator(t)
	driverError := errors.New("driver unavailable")
	executor := &recordingExecutor{
		tag: pgconn.NewCommandTag("UPDATE 1"),
		err: driverError,
	}
	guarded := mustNewGuardedDBWithExecutor(t, validator, executor)
	ctx := context.WithValue(context.Background(), operationContextKey{}, "request")
	argument := new(int)
	sql := "UPDATE accounts SET active = $1 WHERE id = $2"

	tag, err := guarded.Exec(ctx, sql, argument, 42)

	require.Equal(t, pgconn.NewCommandTag("UPDATE 1"), tag)
	require.ErrorIs(t, err, driverError)
	requireExecutorCall(ctx, t, executor.Calls(), sql, []any{argument, 42})
}

func TestGuardedDBQueryPreservesAllowedCall(t *testing.T) {
	validator := newRealPreparedValidator(t)
	driverError := errors.New("query unavailable")
	rows := pgx.RowsFromResultReader(nil, nil)
	executor := &recordingExecutor{rows: rows, err: driverError}
	guarded := mustNewGuardedDBWithExecutor(t, validator, executor)
	ctx := context.WithValue(context.Background(), operationContextKey{}, "request")
	argument := new(int)
	sql := "SELECT id FROM accounts WHERE active = $1 AND id = $2"

	returnedRows, err := guarded.Query(ctx, sql, argument, 42)

	require.Same(t, rows, returnedRows)
	require.ErrorIs(t, err, driverError)
	requireExecutorCall(ctx, t, executor.Calls(), sql, []any{argument, 42})
}

func TestGuardedDBQueryRowPreservesAllowedCall(t *testing.T) {
	validator := newRealPreparedValidator(t)
	driverError := errors.New("scan unavailable")
	row := &rowStub{err: driverError}
	executor := &recordingExecutor{row: row}
	guarded := mustNewGuardedDBWithExecutor(t, validator, executor)
	ctx := context.WithValue(context.Background(), operationContextKey{}, "request")
	argument := new(int)
	sql := "SELECT id FROM accounts WHERE active = $1 AND id = $2"

	returnedRow := guarded.QueryRow(ctx, sql, argument, 42)

	require.Same(t, row, returnedRow)
	require.ErrorIs(t, returnedRow.Scan(), driverError)
	requireExecutorCall(ctx, t, executor.Calls(), sql, []any{argument, 42})
}

func TestGuardedDBOperationsBlockValidationFailures(t *testing.T) {
	unknownError := errors.New("validation unavailable")

	tests := map[string]struct {
		setup  func(t *testing.T) (PreparedValidator, context.Context)
		invoke func(t *testing.T, guarded *GuardedDB, ctx context.Context) error
		check  func(t *testing.T, err error)
	}{
		"exec_blocks_policy_violation": {
			setup: func(t *testing.T) (PreparedValidator, context.Context) {
				t.Helper()

				rule := &mutableRule{}
				rule.reject.Store(true)
				engine, err := sqlguard.NewEngine(sqlguard.EngineOptions{}, rule)
				require.NoError(t, err)

				return engine, context.Background()
			},
			invoke: func(t *testing.T, guarded *GuardedDB, ctx context.Context) error {
				t.Helper()

				tag, err := guarded.Exec(ctx, "SELECT 1")
				require.Equal(t, pgconn.CommandTag{}, tag)

				return err
			},
			check: func(t *testing.T, err error) {
				t.Helper()

				var violation *sqlguard.Violation
				require.ErrorAs(t, err, &violation)
			},
		},
		"exec_blocks_invalid_prepared": {
			setup: func(t *testing.T) (PreparedValidator, context.Context) {
				t.Helper()

				return &preparedValidatorStub{
					prepareFunc: func(context.Context, string) (sqlguard.Prepared, error) {
						return sqlguard.Prepared{}, nil
					},
					validateFunc: func(context.Context, sqlguard.Prepared) error {
						return sqlguard.ErrInvalidPrepared
					},
				}, context.Background()
			},
			invoke: func(t *testing.T, guarded *GuardedDB, ctx context.Context) error {
				t.Helper()

				tag, err := guarded.Exec(ctx, "SELECT 1")
				require.Equal(t, pgconn.CommandTag{}, tag)

				return err
			},
			check: func(t *testing.T, err error) {
				t.Helper()

				require.ErrorIs(t, err, sqlguard.ErrInvalidPrepared)
			},
		},
		"query_blocks_parser_failure": {
			setup: func(t *testing.T) (PreparedValidator, context.Context) {
				t.Helper()

				engine, err := sqlguard.NewEngine(sqlguard.EngineOptions{})
				require.NoError(t, err)

				return engine, context.Background()
			},
			invoke: func(t *testing.T, guarded *GuardedDB, ctx context.Context) error {
				t.Helper()

				rows, err := guarded.Query(ctx, "SELECT * FROM")
				require.Nil(t, rows)

				return err
			},
			check: func(t *testing.T, err error) {
				t.Helper()

				var parseError *sqlguard.ParseError
				require.ErrorAs(t, err, &parseError)
			},
		},
		"query_blocks_unknown_validation_error": {
			setup: func(t *testing.T) (PreparedValidator, context.Context) {
				t.Helper()

				prepared := mustPrepare(t, "SELECT 1")

				return &preparedValidatorStub{
					prepareFunc: func(context.Context, string) (sqlguard.Prepared, error) {
						return prepared, nil
					},
					validateFunc: func(context.Context, sqlguard.Prepared) error {
						return unknownError
					},
				}, context.Background()
			},
			invoke: func(t *testing.T, guarded *GuardedDB, ctx context.Context) error {
				t.Helper()

				rows, err := guarded.Query(ctx, "SELECT 1")
				require.Nil(t, rows)

				return err
			},
			check: func(t *testing.T, err error) {
				t.Helper()

				require.ErrorIs(t, err, unknownError)
			},
		},
		"query_row_defers_cancellation": {
			setup: func(t *testing.T) (PreparedValidator, context.Context) {
				t.Helper()

				engine, err := sqlguard.NewEngine(sqlguard.EngineOptions{})
				require.NoError(t, err)

				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				return engine, ctx
			},
			invoke: func(t *testing.T, guarded *GuardedDB, ctx context.Context) error {
				t.Helper()

				return guarded.QueryRow(ctx, "SELECT 1").Scan()
			},
			check: func(t *testing.T, err error) {
				t.Helper()

				require.ErrorIs(t, err, context.Canceled)
			},
		},
		"query_row_defers_prepared_validation_failure": {
			setup: func(t *testing.T) (PreparedValidator, context.Context) {
				t.Helper()

				prepared := mustPrepare(t, "SELECT 1")

				return &preparedValidatorStub{
					prepareFunc: func(context.Context, string) (sqlguard.Prepared, error) {
						return prepared, nil
					},
					validateFunc: func(context.Context, sqlguard.Prepared) error {
						return unknownError
					},
				}, context.Background()
			},
			invoke: func(t *testing.T, guarded *GuardedDB, ctx context.Context) error {
				t.Helper()

				return guarded.QueryRow(ctx, "SELECT 1").Scan()
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

			validator, ctx := testCase.setup(t)
			executor := &recordingExecutor{}
			guarded := mustNewGuardedDBWithExecutor(t, validator, executor)

			testCase.check(t, testCase.invoke(t, guarded, ctx))
			require.Empty(t, executor.Calls())
		})
	}
}

func TestGuardedDBRejectsQueryRewritersBeforeGuardedWork(t *testing.T) {
	type queryArguments struct {
		AccountID int
	}

	rewriters := map[string]func() any{
		"custom": func() any {
			return &queryRewriterStub{}
		},
		"named_args": func() any {
			return pgx.NamedArgs{"account_id": 42}
		},
		"strict_named_args": func() any {
			return pgx.StrictNamedArgs{"account_id": 42}
		},
		"struct_args": func() any {
			return pgx.StructArgs(queryArguments{AccountID: 42})
		},
		"strict_struct_args": func() any {
			return pgx.StrictStructArgs(queryArguments{AccountID: 42})
		},
	}
	operations := map[string]func(*GuardedDB, any) error{
		"exec": func(guarded *GuardedDB, rewriter any) error {
			_, err := guarded.Exec(context.Background(), "SELECT 1", 42, rewriter)

			return err
		},
		"query": func(guarded *GuardedDB, rewriter any) error {
			_, err := guarded.Query(context.Background(), "SELECT 1", 42, rewriter)

			return err
		},
		"query_row": func(guarded *GuardedDB, rewriter any) error {
			return guarded.QueryRow(context.Background(), "SELECT 1", 42, rewriter).Scan()
		},
	}

	for rewriterName, newRewriter := range rewriters {
		for operationName, operation := range operations {
			t.Run(rewriterName+"_"+operationName, func(t *testing.T) {
				t.Parallel()

				validator := newRealPreparedValidator(t)
				executor := &recordingExecutor{}
				guarded := mustNewGuardedDBWithExecutor(t, validator, executor)
				require.NoError(t, guarded.validate(context.Background(), "SELECT 1"))
				require.NoError(t, guarded.validate(context.Background(), "SELECT 2"))

				keysBefore := guarded.cache.Keys()
				preparesBefore, validationsBefore := validator.Calls("SELECT 1")

				rewriter := newRewriter()
				err := operation(guarded, rewriter)

				require.ErrorIs(t, err, ErrQueryRewriterUnsupported)

				preparesAfter, validationsAfter := validator.Calls("SELECT 1")

				require.Equal(t, preparesBefore, preparesAfter)
				require.Equal(t, validationsBefore, validationsAfter)
				require.Equal(t, keysBefore, guarded.cache.Keys())
				require.Empty(t, executor.Calls())

				if custom, ok := rewriter.(*queryRewriterStub); ok {
					require.Zero(t, custom.calls.Load())
				}
			})
		}
	}
}

func TestGuardedDBConcurrentOperationsValidateBeforeDelegating(t *testing.T) {
	const callCount = 32

	operations := map[string]func(*GuardedDB, context.Context, int) error{
		"exec": func(guarded *GuardedDB, ctx context.Context, index int) error {
			_, err := guarded.Exec(ctx, "SELECT 1", index)

			return err
		},
		"query": func(guarded *GuardedDB, ctx context.Context, index int) error {
			_, err := guarded.Query(ctx, "SELECT 1", index)

			return err
		},
		"query_row": func(guarded *GuardedDB, ctx context.Context, index int) error {
			guarded.QueryRow(ctx, "SELECT 1", index)

			return nil
		},
	}

	for name, operation := range operations {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			prepared := mustPrepare(t, "SELECT 1")
			arrived := make(chan struct{}, callCount)
			release := make(chan struct{})

			var completed sync.Map

			validator := &preparedValidatorStub{
				prepareFunc: func(context.Context, string) (sqlguard.Prepared, error) {
					return prepared, nil
				},
				validateFunc: func(ctx context.Context, _ sqlguard.Prepared) error {
					arrived <- struct{}{}

					<-release
					completed.Store(ctx.Value(operationContextKey{}), struct{}{})

					return nil
				},
			}

			var delegatedBeforeValidation atomic.Bool

			executor := &recordingExecutor{
				row: &rowStub{},
				onCall: func(ctx context.Context) {
					if _, ok := completed.Load(ctx.Value(operationContextKey{})); !ok {
						delegatedBeforeValidation.Store(true)
					}
				},
			}
			guarded := mustNewGuardedDBWithExecutor(t, validator, executor)
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
					ctx := context.WithValue(context.Background(), operationContextKey{}, index)

					return operation(guarded, ctx, index)
				})
			}()

			waitForConcurrentArrivals(t, arrived, errorsDone, callCount, releaseAll)
			require.Empty(t, executor.Calls())
			releaseAll()
			requireNoErrors(t, waitForConcurrentErrors(t, errorsDone))

			prepares, validations := validator.Calls("SELECT 1")
			require.Equal(t, callCount, prepares)
			require.Equal(t, callCount, validations)
			require.Len(t, executor.Calls(), callCount)
			require.False(t, delegatedBeforeValidation.Load())
		})
	}
}

type operationContextKey struct{}

func mustNewGuardedDBWithExecutor(
	t *testing.T,
	validator PreparedValidator,
	executor Executor,
) *GuardedDB {
	t.Helper()

	guarded, err := NewGuardedDB(validator, executor, CacheOptions{
		Capacity:    2,
		MaxSQLBytes: 1024,
	})
	require.NoError(t, err)

	return guarded
}

func requireExecutorCall(
	ctx context.Context,
	t *testing.T,
	calls []executorCall,
	sql string,
	args []any,
) {
	t.Helper()

	require.Len(t, calls, 1)
	require.Same(t, ctx, calls[0].ctx)
	require.Equal(t, sql, calls[0].sql)
	require.Equal(t, args, calls[0].args)
}
