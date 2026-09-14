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
	insertRule := rules.NewInsertRequiresColumns()
	truncateRule := rules.NewDenyTruncate()
	dropTableRule := rules.NewDenyDropTable()
	alterTableRule := rules.NewDenyAlterTable()
	engine := mustNewEngine(
		t,
		updateRule,
		deleteRule,
		insertRule,
		truncateRule,
		dropTableRule,
		alterTableRule,
	)

	tests := map[string]validationTestCase{
		"allows_alter_role": {
			input:      "ALTER ROLE app_user SET statement_timeout = '5s'",
			checkError: requireNoValidationError,
		},
		"allows_safe_delete": {
			input:      "DELETE FROM accounts WHERE id = 42",
			checkError: requireNoValidationError,
		},
		"allows_safe_insert": {
			input:      "INSERT INTO accounts (id, active) VALUES (42, false)",
			checkError: requireNoValidationError,
		},
		"allows_safe_update": {
			input:      "UPDATE accounts SET active = false WHERE id = 42",
			checkError: requireNoValidationError,
		},
		"rejects_alter_table": {
			input:      "ALTER TABLE accounts DROP COLUMN legacy_id",
			checkError: requireViolation("deny_alter_table"),
		},
		"rejects_drop_table": {
			input:      "DROP TABLE accounts",
			checkError: requireViolation("deny_drop_table"),
		},
		"rejects_unsafe_delete": {
			input:      "DELETE FROM accounts",
			checkError: requireViolation("delete_requires_where"),
		},
		"rejects_unsafe_insert": {
			input:      "INSERT INTO accounts VALUES (42, false)",
			checkError: requireViolation("insert_requires_columns"),
		},
		"rejects_unsafe_update": {
			input:      "UPDATE accounts SET active = false",
			checkError: requireViolation("update_requires_where"),
		},
		"rejects_truncate": {
			input:      "TRUNCATE TABLE accounts",
			checkError: requireViolation("deny_truncate"),
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
