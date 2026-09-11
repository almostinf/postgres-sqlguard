package pgxexample

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"

	sqlguard "github.com/almostinf/postgres-sqlguard"
	"github.com/almostinf/postgres-sqlguard/rules"
)

type validatorStub struct{}

func (*validatorStub) Validate(context.Context, string) error {
	return nil
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

type recordingValidator struct {
	sequence *[]string
	calls    int
	ctx      context.Context
	sql      string
	err      error
}

func (v *recordingValidator) Validate(ctx context.Context, sql string) error {
	*v.sequence = append(*v.sequence, "validate")
	v.calls++
	v.ctx = ctx
	v.sql = sql

	return v.err
}

type recordingExecutor struct {
	sequence *[]string
	calls    int
	ctx      context.Context
	sql      string
	args     []any
	tag      pgconn.CommandTag
	rows     pgx.Rows
	row      pgx.Row
	err      error
}

func (e *recordingExecutor) Exec(
	ctx context.Context,
	sql string,
	arguments ...any,
) (pgconn.CommandTag, error) {
	e.record(ctx, "exec", sql, arguments)

	return e.tag, e.err
}

func (e *recordingExecutor) Query(
	ctx context.Context,
	sql string,
	args ...any,
) (pgx.Rows, error) {
	e.record(ctx, "query", sql, args)

	return e.rows, e.err
}

func (e *recordingExecutor) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	e.record(ctx, "query_row", sql, args)

	return e.row
}

func (e *recordingExecutor) record(
	ctx context.Context,
	operation string,
	sql string,
	args []any,
) {
	*e.sequence = append(*e.sequence, operation)
	e.calls++
	e.ctx = ctx
	e.sql = sql
	e.args = args
}

type rowStub struct{}

func (*rowStub) Scan(...any) error {
	return nil
}

type queryRewriterStub struct {
	calls int
}

func (rewriter *queryRewriterStub) RewriteQuery(
	context.Context,
	*pgx.Conn,
	string,
	[]any,
) (string, []any, error) {
	rewriter.calls++

	return "SELECT 1", nil, nil
}

func TestNewGuardedDB(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		validator    sqlguard.Validator
		executor     Executor
		requireError bool
	}{
		"accepts_valid_collaborators": {
			validator: &validatorStub{},
			executor:  &executorStub{},
		},
		"rejects_nil_executor": {
			validator:    &validatorStub{},
			requireError: true,
		},
		"rejects_nil_validator": {
			executor:     &executorStub{},
			requireError: true,
		},
		"rejects_typed_nil_executor": {
			validator:    &validatorStub{},
			executor:     (*executorStub)(nil),
			requireError: true,
		},
		"rejects_typed_nil_validator": {
			validator:    (*validatorStub)(nil),
			executor:     &executorStub{},
			requireError: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			guarded, err := NewGuardedDB(test.validator, test.executor)
			if test.requireError {
				require.Error(t, err)
				require.Nil(t, guarded)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, guarded)
		})
	}
}

func TestGuardedDBExecDelegatesAcceptedSQL(t *testing.T) {
	t.Parallel()

	sequence := make([]string, 0, 2)
	validator := &recordingValidator{sequence: &sequence}
	driverError := errors.New("driver unavailable")
	executor := &recordingExecutor{
		sequence: &sequence,
		tag:      pgconn.NewCommandTag("UPDATE 1"),
		err:      driverError,
	}
	guarded, err := NewGuardedDB(validator, executor)
	require.NoError(t, err)

	ctx := context.WithValue(context.Background(), contextKey{}, "request-value")
	argument := new(int)
	sql := "UPDATE accounts SET active = $1 WHERE id = $2"

	tag, err := guarded.Exec(ctx, sql, argument, 42)

	require.Equal(t, pgconn.NewCommandTag("UPDATE 1"), tag)
	require.ErrorIs(t, err, driverError)
	require.Equal(t, []string{"validate", "exec"}, sequence)
	require.Equal(t, 1, validator.calls)
	require.Equal(t, 1, executor.calls)
	require.Same(t, ctx, validator.ctx)
	require.Same(t, ctx, executor.ctx)
	require.Equal(t, sql, validator.sql)
	require.Equal(t, sql, executor.sql)
	require.Len(t, executor.args, 2)
	require.Same(t, argument, executor.args[0])
	require.Equal(t, 42, executor.args[1])
}

func TestGuardedDBQueryDelegatesAcceptedSQL(t *testing.T) {
	t.Parallel()

	sequence := make([]string, 0, 2)
	validator := &recordingValidator{sequence: &sequence}
	returnedRows := pgx.RowsFromResultReader(nil, nil)
	driverError := errors.New("query unavailable")
	executor := &recordingExecutor{
		sequence: &sequence,
		rows:     returnedRows,
		err:      driverError,
	}
	guarded, err := NewGuardedDB(validator, executor)
	require.NoError(t, err)

	ctx := context.WithValue(context.Background(), contextKey{}, "request-value")
	argument := new(int)
	sql := "SELECT id FROM accounts WHERE id = $1"

	rows, err := guarded.Query(ctx, sql, argument)

	require.Same(t, returnedRows, rows)
	require.ErrorIs(t, err, driverError)
	require.Equal(t, []string{"validate", "query"}, sequence)
	require.Equal(t, 1, validator.calls)
	require.Equal(t, 1, executor.calls)
	require.Same(t, ctx, validator.ctx)
	require.Same(t, ctx, executor.ctx)
	require.Equal(t, sql, validator.sql)
	require.Equal(t, sql, executor.sql)
	require.Len(t, executor.args, 1)
	require.Same(t, argument, executor.args[0])
}

func TestGuardedDBQueryRowDelegatesAcceptedSQL(t *testing.T) {
	t.Parallel()

	sequence := make([]string, 0, 2)
	validator := &recordingValidator{sequence: &sequence}
	returnedRow := &rowStub{}
	executor := &recordingExecutor{
		sequence: &sequence,
		row:      returnedRow,
	}
	guarded, err := NewGuardedDB(validator, executor)
	require.NoError(t, err)

	ctx := context.WithValue(context.Background(), contextKey{}, "request-value")
	argument := new(int)
	sql := "SELECT id FROM accounts WHERE id = $1"

	row := guarded.QueryRow(ctx, sql, argument)

	require.Same(t, returnedRow, row)
	require.Equal(t, []string{"validate", "query_row"}, sequence)
	require.Equal(t, 1, validator.calls)
	require.Equal(t, 1, executor.calls)
	require.Same(t, ctx, validator.ctx)
	require.Same(t, ctx, executor.ctx)
	require.Equal(t, sql, validator.sql)
	require.Equal(t, sql, executor.sql)
	require.Len(t, executor.args, 1)
	require.Same(t, argument, executor.args[0])
}

func TestGuardedDBValidationFailurePreventsDelegation(t *testing.T) {
	t.Parallel()

	validationEngine, err := sqlguard.NewEngine(
		sqlguard.EngineOptions{},
		rules.NewDeleteRequiresWhere(),
	)
	require.NoError(t, err)

	violationError := validationEngine.Validate(context.Background(), "DELETE FROM accounts")
	require.Error(t, violationError)

	parseError := validationEngine.Validate(context.Background(), "SELECT '")
	require.Error(t, parseError)

	failures := map[string]struct {
		err        error
		checkError func(t *testing.T, err error)
	}{
		"canceled": {
			err: context.Canceled,
			checkError: func(t *testing.T, err error) {
				t.Helper()

				require.ErrorIs(t, err, context.Canceled)
			},
		},
		"parser_failure": {
			err: parseError,
			checkError: func(t *testing.T, err error) {
				t.Helper()

				var target *sqlguard.ParseError
				require.ErrorAs(t, err, &target)
			},
		},
		"policy_violation": {
			err: violationError,
			checkError: func(t *testing.T, err error) {
				t.Helper()

				var target *sqlguard.Violation
				require.ErrorAs(t, err, &target)
			},
		},
	}

	operations := map[string]func(t *testing.T, guarded *GuardedDB) error{
		"exec": func(t *testing.T, guarded *GuardedDB) error {
			t.Helper()

			tag, err := guarded.Exec(context.Background(), "sensitive sql", "secret argument")
			require.Equal(t, pgconn.CommandTag{}, tag)

			return err
		},
		"query": func(t *testing.T, guarded *GuardedDB) error {
			t.Helper()

			rows, err := guarded.Query(context.Background(), "sensitive sql", "secret argument")
			require.Nil(t, rows)

			return err
		},
		"query_row": func(t *testing.T, guarded *GuardedDB) error {
			t.Helper()

			row := guarded.QueryRow(context.Background(), "sensitive sql", "secret argument")
			require.NotNil(t, row)

			return row.Scan()
		},
	}

	for failureName, failure := range failures {
		for operationName, operation := range operations {
			t.Run(failureName+"_"+operationName, func(t *testing.T) {
				t.Parallel()

				sequence := make([]string, 0, 1)
				validator := &recordingValidator{
					sequence: &sequence,
					err:      failure.err,
				}
				executor := &recordingExecutor{sequence: &sequence}
				guarded, err := NewGuardedDB(validator, executor)
				require.NoError(t, err)

				err = operation(t, guarded)

				failure.checkError(t, err)
				require.Equal(t, []string{"validate"}, sequence)
				require.Equal(t, 1, validator.calls)
				require.Zero(t, executor.calls)
			})
		}
	}
}

func TestGuardedDBRejectsQueryRewriterBeforeValidation(t *testing.T) {
	t.Parallel()

	operations := map[string]func(t *testing.T, guarded *GuardedDB, args []any) error{
		"exec": func(t *testing.T, guarded *GuardedDB, args []any) error {
			t.Helper()

			tag, err := guarded.Exec(context.Background(), "sensitive sql", args...)
			require.Equal(t, pgconn.CommandTag{}, tag)

			return err
		},
		"query": func(t *testing.T, guarded *GuardedDB, args []any) error {
			t.Helper()

			rows, err := guarded.Query(context.Background(), "sensitive sql", args...)
			require.Nil(t, rows)

			return err
		},
		"query_row": func(t *testing.T, guarded *GuardedDB, args []any) error {
			t.Helper()

			row := guarded.QueryRow(context.Background(), "sensitive sql", args...)
			require.NotNil(t, row)

			return row.Scan()
		},
	}

	argumentPositions := map[string]func(rewriter pgx.QueryRewriter) []any{
		"first": func(rewriter pgx.QueryRewriter) []any {
			return []any{rewriter, "secret argument"}
		},
		"later": func(rewriter pgx.QueryRewriter) []any {
			return []any{"secret argument", rewriter}
		},
	}

	for operationName, operation := range operations {
		for positionName, arguments := range argumentPositions {
			t.Run(operationName+"_"+positionName, func(t *testing.T) {
				t.Parallel()

				sequence := make([]string, 0, 2)
				validator := &recordingValidator{sequence: &sequence}
				executor := &recordingExecutor{sequence: &sequence}
				guarded, err := NewGuardedDB(validator, executor)
				require.NoError(t, err)

				rewriter := &queryRewriterStub{}
				err = operation(t, guarded, arguments(rewriter))

				require.ErrorIs(t, err, ErrQueryRewriterUnsupported)
				require.Empty(t, sequence)
				require.Zero(t, rewriter.calls)
				require.Zero(t, validator.calls)
				require.Zero(t, executor.calls)
			})
		}
	}
}

func TestGuardedDBUsesEngineValidationBoundary(t *testing.T) {
	t.Parallel()

	engine, err := sqlguard.NewEngine(
		sqlguard.EngineOptions{},
		rules.NewDeleteRequiresWhere(),
	)
	require.NoError(t, err)

	sequence := make([]string, 0, 1)
	executor := &recordingExecutor{sequence: &sequence}
	guarded, err := NewGuardedDB(engine, executor)
	require.NoError(t, err)

	_, err = guarded.Exec(context.Background(), "DELETE FROM accounts")

	var violation *sqlguard.Violation
	require.ErrorAs(t, err, &violation)
	require.Zero(t, executor.calls)

	_, err = guarded.Exec(
		context.Background(),
		"DELETE FROM accounts WHERE id = $1",
		42,
	)

	require.NoError(t, err)
	require.Equal(t, 1, executor.calls)
	require.Equal(t, "DELETE FROM accounts WHERE id = $1", executor.sql)
	require.Equal(t, []any{42}, executor.args)
}

type contextKey struct{}
