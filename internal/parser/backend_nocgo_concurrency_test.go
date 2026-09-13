//go:build !cgo

package parser

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWASMBackendConcurrentFirstUse(t *testing.T) {
	const callCount = 64

	results := make([]*Result, callCount)
	errorsByCall := make([]error, callCount)

	var waitGroup sync.WaitGroup
	waitGroup.Add(callCount)

	for index := range callCount {
		go func() {
			defer waitGroup.Done()

			if index%2 == 0 {
				results[index], errorsByCall[index] = Parse(fmt.Sprintf("SELECT %d", index))

				return
			}

			results[index], errorsByCall[index] = Parse(fmt.Sprintf(
				"SELECT 'sqlguard_concurrent_secret_%03d' FROM", index,
			))
		}()
	}

	waitGroup.Wait()

	for index := range callCount {
		if index%2 == 0 {
			require.NoError(t, errorsByCall[index])
			require.Len(t, results[index].Statements(), 1)
			require.Equal(t, Kind("SelectStmt"), results[index].Statements()[0].Kind())

			continue
		}

		require.Nil(t, results[index])
		requireSyntaxFailure(t, errorsByCall[index])
		require.NotContains(t, errorsByCall[index].Error(), "sqlguard_concurrent_secret_")
		require.NotContains(t, fmt.Sprintf("%#v", errorsByCall[index]), "sqlguard_concurrent_secret_")
	}
}

func TestWASMBackendMalformedInputsDoNotPanic(t *testing.T) {
	tests := map[string]string{
		"nul_byte":                    "SELECT \x00",
		"deep_unbalanced_parentheses": strings.Repeat("(", 256) + "SELECT 1",
		"unterminated_dollar_quote":   "SELECT $body$unfinished",
		"unterminated_nested_comment": "/* outer /* inner */",
	}

	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := Parse(input)

			require.Nil(t, result)
			requireSyntaxFailure(t, err)
		})
	}
}
