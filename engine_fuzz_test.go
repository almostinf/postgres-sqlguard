package sqlguard_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	sqlguard "github.com/almostinf/postgres-sqlguard"
)

const fuzzInputSentinel = "sqlguard_fuzz_secret_7f3c9182"

func FuzzEngineValidate(f *testing.F) {
	for _, seed := range []string{
		"SELECT 1",
		"SELECT ARRAY[1, 2]::int[] @> ARRAY[1]",
		"SELECT 'DELETE FROM secrets'; -- UPDATE accounts\n",
		"SELECT 1; DELETE FROM accounts",
		"WITH changed AS (UPDATE accounts SET active = false RETURNING *) SELECT * FROM changed",
		"",
		";;;",
		"SELECT * FROM",
		"SELECT 1; SELECT * FROM",
		"unterminated 'literal",
	} {
		f.Add(seed)
	}

	engine, err := sqlguard.NewEngine(sqlguard.EngineOptions{}, &ruleStub{
		id: "fuzz_reject",
		evaluate: func(context.Context, sqlguard.Statement) sqlguard.RuleResult {
			return sqlguard.Reject()
		},
	})
	require.NoError(f, err)

	f.Fuzz(func(t *testing.T, sql string) {
		input := sql + "\n/* " + fuzzInputSentinel + " */"

		validationErr := engine.Validate(context.Background(), input)
		if validationErr == nil {
			return
		}

		for current := validationErr; current != nil; current = errors.Unwrap(current) {
			if strings.Contains(current.Error(), fuzzInputSentinel) {
				t.Fatalf("error message contains submitted sentinel: %q", current.Error())
			}

			if value := fmt.Sprintf("%#v", current); strings.Contains(value, fuzzInputSentinel) {
				t.Fatalf("error value contains submitted sentinel: %q", value)
			}
		}
	})
}
