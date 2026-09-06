package sqlguard_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	sqlguard "github.com/almostinf/postgres-sqlguard"
)

type isolationRule struct {
	mutex      sync.Mutex
	seen       map[string]int
	mismatches []string
}

func (r *isolationRule) ID() string {
	return "statement_isolation"
}

func (r *isolationRule) Evaluate(ctx context.Context, statement sqlguard.Statement) sqlguard.RuleResult {
	expected, _ := ctx.Value(expectedRelationKey).(string)
	fromClause := statement.Root().Children("from_clause")

	actual := ""
	if len(fromClause) == 1 {
		actual, _ = fromClause[0].String("relname")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.seen[expected]++
	if actual != expected {
		r.mismatches = append(r.mismatches, fmt.Sprintf("expected %q, got %q", expected, actual))
	}

	return sqlguard.Allow()
}

func (r *isolationRule) Snapshot() (map[string]int, []string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	seen := make(map[string]int, len(r.seen))
	for relation, count := range r.seen {
		seen[relation] = count
	}

	return seen, append([]string(nil), r.mismatches...)
}

func TestEngineConcurrentUse(t *testing.T) {
	tests := map[string]struct {
		callCount   int
		setupTest   func(t *testing.T) (*sqlguard.Engine, *isolationRule)
		checkResult func(t *testing.T, rule *isolationRule, callCount int)
		checkError  func(t *testing.T, errors []error)
	}{
		"keeps_context_and_statements_isolated_between_calls": {
			callCount: 64,
			setupTest: func(t *testing.T) (*sqlguard.Engine, *isolationRule) {
				t.Helper()

				rule := &isolationRule{seen: make(map[string]int)}

				return mustNewEngine(t, rule), rule
			},
			checkResult: func(t *testing.T, rule *isolationRule, callCount int) {
				t.Helper()

				seen, mismatches := rule.Snapshot()
				require.Empty(t, mismatches)
				require.Len(t, seen, callCount)

				for _, count := range seen {
					require.Equal(t, 1, count)
				}
			},
			checkError: requireNoValidationErrors,
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			require.NotNil(t, testCase.setupTest, "test case must define setupTest")
			engine, rule := testCase.setupTest(t)

			errors := make([]error, testCase.callCount)

			var waitGroup sync.WaitGroup
			waitGroup.Add(testCase.callCount)

			for index := range testCase.callCount {
				go func() {
					defer waitGroup.Done()

					relation := fmt.Sprintf("account_%03d", index)
					ctx := context.WithValue(context.Background(), expectedRelationKey, relation)
					errors[index] = engine.Validate(ctx, "SELECT * FROM "+relation)
				}()
			}

			waitGroup.Wait()

			require.NotNil(t, testCase.checkError, "test case must define checkError")
			testCase.checkError(t, errors)

			if testCase.checkResult != nil {
				testCase.checkResult(t, rule, testCase.callCount)
			}
		})
	}
}
