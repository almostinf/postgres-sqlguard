package rules_test

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/almostinf/postgres-sqlguard/rules"
)

const concurrentValidationRuns = 32

type concurrentValidationResult struct {
	err        error
	checkError func(t *testing.T, err error)
}

func TestMutationRulesConcurrentUse(t *testing.T) {
	updateRule := rules.NewUpdateRequiresWhere()
	deleteRule := rules.NewDeleteRequiresWhere()
	engine := mustNewEngine(t, updateRule, deleteRule)

	tests := map[string]validationTestCase{
		"allows_safe_delete": {
			input:      "DELETE FROM accounts WHERE id = 42",
			checkError: requireNoValidationError,
		},
		"allows_safe_update": {
			input:      "UPDATE accounts SET active = false WHERE id = 42",
			checkError: requireNoValidationError,
		},
		"rejects_unsafe_delete": {
			input:      "DELETE FROM accounts",
			checkError: requireViolation("delete_requires_where"),
		},
		"rejects_unsafe_update": {
			input:      "UPDATE accounts SET active = false",
			checkError: requireViolation("update_requires_where"),
		},
	}

	resultCount := concurrentValidationRuns * len(tests)
	results := make(chan concurrentValidationResult, resultCount)
	start := make(chan struct{})

	var waitGroup sync.WaitGroup
	waitGroup.Add(resultCount)

	for range concurrentValidationRuns {
		for _, testCase := range tests {
			go func() {
				defer waitGroup.Done()

				<-start

				results <- concurrentValidationResult{
					err:        engine.Validate(context.Background(), testCase.input),
					checkError: testCase.checkError,
				}
			}()
		}
	}

	close(start)
	waitGroup.Wait()
	close(results)

	for result := range results {
		require.NotNil(t, result.checkError, "test case must define checkError")
		result.checkError(t, result.err)
	}
}
