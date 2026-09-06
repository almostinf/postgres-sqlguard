package parser_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/almostinf/postgres-sqlguard/internal/parser"
)

func TestStatementSequence(t *testing.T) {
	tests := map[string]parseTestCase{
		"visits_root_before_a_data_modifying_cte": {
			input: `
				WITH changed AS (
					UPDATE accounts SET active = false RETURNING *
				)
				SELECT * FROM changed
			`,
			checkResult: requireSequenceKinds("SelectStmt", "UpdateStmt"),
			checkError:  requireNoError,
		},
		"visits_all_statements_in_deterministic_order": {
			input: `
				SELECT 1;
				WITH inserted AS (
					INSERT INTO audit_log (message) VALUES ('created') RETURNING *
				), changed AS (
					WITH removed AS (
						DELETE FROM sessions WHERE expired RETURNING *
					)
					UPDATE accounts SET active = false
					FROM removed
					WHERE accounts.id = removed.account_id
					RETURNING accounts.*
				)
				SELECT * FROM inserted, changed;
				DELETE FROM obsolete_rows;
			`,
			checkResult: requireSequenceKinds(
				"SelectStmt",
				"SelectStmt",
				"InsertStmt",
				"UpdateStmt",
				"DeleteStmt",
				"DeleteStmt",
			),
			checkError: requireNoError,
		},
		"returns_a_sequence_snapshot": {
			input: "SELECT 1; DELETE FROM obsolete_rows",
			checkResult: func(t *testing.T, result *parser.Result) {
				t.Helper()

				sequence := parser.StatementSequence(result)
				require.Len(t, sequence, 2)

				sequence[0] = nil

				require.Equal(t, parser.Kind("SelectStmt"), parser.StatementSequence(result)[0].Kind())
			},
			checkError: requireNoError,
		},
	}

	runParseTests(t, tests)
}

func requireSequenceKinds(want ...parser.Kind) func(*testing.T, *parser.Result) {
	return func(t *testing.T, result *parser.Result) {
		t.Helper()

		sequence := parser.StatementSequence(result)
		require.Len(t, sequence, len(want))

		for index, kind := range want {
			require.Equal(t, kind, sequence[index].Kind())
		}
	}
}
