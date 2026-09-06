package parser_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/almostinf/postgres-sqlguard/internal/parser"
)

type parseTestCase struct {
	input       string
	setupTest   func(t *testing.T)
	checkResult func(t *testing.T, result *parser.Result)
	checkError  func(t *testing.T, err error)
}

func TestParse(t *testing.T) {
	tests := map[string]parseTestCase{
		"returns_immutable_parser_neutral_tree": {
			input: "SELECT 1",
			checkResult: func(t *testing.T, result *parser.Result) {
				t.Helper()

				statements := result.Statements()
				require.Len(t, statements, 1)
				require.Equal(t, parser.Kind("SelectStmt"), statements[0].Kind())

				targets, ok := statements[0].Field("target_list")
				require.True(t, ok)

				values, ok := targets.List()
				require.True(t, ok)
				require.Len(t, values, 1)

				target, ok := values[0].Node()
				require.True(t, ok)
				require.Equal(t, parser.Kind("ResTarget"), target.Kind())

				statements[0] = nil

				require.Equal(t, parser.Kind("SelectStmt"), result.Statements()[0].Kind())

				values[0] = parser.Value{}

				targets, ok = result.Statements()[0].Field("target_list")
				require.True(t, ok)

				values, ok = targets.List()
				require.True(t, ok)
				require.Len(t, values, 1)

				_, ok = values[0].Node()
				require.True(t, ok)
			},
			checkError: requireNoError,
		},
		"accepts_postgresql_specific_syntax": {
			input:       "SELECT ARRAY[1, 2]::int[] @> ARRAY[1]",
			checkResult: requireStatementKinds("SelectStmt"),
			checkError:  requireNoError,
		},
		"ignores_keywords_inside_literals_and_comments": {
			input:       "SELECT 'DELETE FROM secrets'; -- UPDATE accounts\n",
			checkResult: requireStatementKinds("SelectStmt"),
			checkError:  requireNoError,
		},
		"accepts_an_empty_string": {
			input:       "",
			checkResult: requireStatementKinds(),
			checkError:  requireNoError,
		},
		"accepts_whitespace_only_input": {
			input:       " \t\n",
			checkResult: requireStatementKinds(),
			checkError:  requireNoError,
		},
		"accepts_comment_only_input": {
			input:       "-- comment only\n/* still only a comment */",
			checkResult: requireStatementKinds(),
			checkError:  requireNoError,
		},
		"accepts_semicolon_only_input": {
			input:       ";;;",
			checkResult: requireStatementKinds(),
			checkError:  requireNoError,
		},
		"rejects_truncated_input": {
			input:       "SELECT * FROM",
			checkResult: requireNilResult,
			checkError:  requireSyntaxError,
		},
		"rejects_a_malformed_statement_in_a_batch": {
			input:       "SELECT 1; SELECT * FROM",
			checkResult: requireNilResult,
			checkError:  requireSyntaxError,
		},
		"sanitizes_backend_parser_diagnostics": {
			input:       "sqlguard_secret_token_6f914c",
			checkResult: requireNilResult,
			checkError: func(t *testing.T, err error) {
				t.Helper()

				requireSyntaxError(t, err)
				require.Equal(t, "postgresql parsing failed", err.Error())
				require.Nil(t, errors.Unwrap(err))
				require.NotContains(t, fmt.Sprintf("%#v", err), "sqlguard_secret_token_6f914c")
			},
		},
	}

	runParseTests(t, tests)
}

func runParseTests(t *testing.T, tests map[string]parseTestCase) {
	t.Helper()

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if testCase.setupTest != nil {
				testCase.setupTest(t)
			}

			result, err := parser.Parse(testCase.input)

			require.NotNil(t, testCase.checkError, "test case must define checkError")
			testCase.checkError(t, err)

			if testCase.checkResult != nil {
				testCase.checkResult(t, result)
			}
		})
	}
}

func requireNoError(t *testing.T, err error) {
	t.Helper()

	require.NoError(t, err)
}

func requireSyntaxError(t *testing.T, err error) {
	t.Helper()

	var parseError *parser.Error
	require.ErrorAs(t, err, &parseError)
	require.Equal(t, parser.FailureSyntax, parseError.Category())
}

func requireNilResult(t *testing.T, result *parser.Result) {
	t.Helper()

	require.Nil(t, result)
}

func requireStatementKinds(want ...parser.Kind) func(*testing.T, *parser.Result) {
	return func(t *testing.T, result *parser.Result) {
		t.Helper()

		statements := result.Statements()
		require.Len(t, statements, len(want))

		for index, kind := range want {
			require.Equal(t, kind, statements[index].Kind())
		}
	}
}
