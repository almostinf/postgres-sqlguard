package sqlguard

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/almostinf/postgres-sqlguard/internal/parser"
)

type statementTestCase struct {
	input       string
	setupTest   func(t *testing.T)
	checkResult func(t *testing.T, statement Statement)
	checkError  func(t *testing.T, err error)
}

func TestStatement(t *testing.T) {
	tests := map[string]statementTestCase{
		"exposes_statement_and_child_kinds": {
			input: "SELECT 1",
			checkResult: func(t *testing.T, statement Statement) {
				t.Helper()

				require.Equal(t, Kind("SelectStmt"), statement.Kind())

				targets := statement.Root().Children("target_list")
				require.Len(t, targets, 1)
				require.Equal(t, Kind("ResTarget"), targets[0].Kind())
			},
			checkError: requireNoStatementError,
		},
		"provides_typed_named_field_access": {
			input: "UPDATE public.accounts SET active = false",
			checkResult: func(t *testing.T, statement Statement) {
				t.Helper()

				relation, ok := statement.Root().Child("relation")
				require.True(t, ok)

				schema, ok := relation.String("schemaname")
				require.True(t, ok)
				require.Equal(t, "public", schema)

				name, ok := relation.String("relname")
				require.True(t, ok)
				require.Equal(t, "accounts", name)
			},
			checkError: requireNoStatementError,
		},
		"walks_nodes_in_structural_pre_order": {
			input: "SELECT 1",
			checkResult: func(t *testing.T, statement Statement) {
				t.Helper()

				kinds := make([]Kind, 0)

				statement.Walk(func(node Node) bool {
					kinds = append(kinds, node.Kind())

					return true
				})

				require.NotEmpty(t, kinds)
				require.Equal(t, Kind("SelectStmt"), kinds[0])
				require.Contains(t, kinds, Kind("ResTarget"))
			},
			checkError: requireNoStatementError,
		},
		"stops_walking_when_visitor_returns_false": {
			input: "SELECT 1",
			checkResult: func(t *testing.T, statement Statement) {
				t.Helper()

				visits := 0

				statement.Walk(func(Node) bool {
					visits++

					return false
				})

				require.Equal(t, 1, visits)
			},
			checkError: requireNoStatementError,
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if testCase.setupTest != nil {
				testCase.setupTest(t)
			}

			result, err := parser.Parse(testCase.input)
			testCase.checkError(t, err)

			statements := parser.StatementSequence(result)
			require.NotEmpty(t, statements)

			testCase.checkResult(t, newStatement(statements[0]))
		})
	}
}

func requireNoStatementError(t *testing.T, err error) {
	t.Helper()

	require.NoError(t, err)
}
