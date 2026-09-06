package sqlguard_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sqlguard "github.com/almostinf/postgres-sqlguard"
)

type contextKey string

const expectedRelationKey contextKey = "expected_relation"

type cancelAfterParseContext struct {
	context.Context

	mutex  sync.Mutex
	checks int
}

func (c *cancelAfterParseContext) Err() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.checks++
	if c.checks > 1 {
		return context.Canceled
	}

	return nil
}

func TestEngineContext(t *testing.T) {
	preCanceledRule := &ruleSpy{id: "pre_canceled_rule"}
	preCanceledContext, cancelBeforeValidation := context.WithCancel(context.Background())
	cancelBeforeValidation()

	afterParseRule := &ruleSpy{id: "after_parse_rule"}
	afterParseContext := &cancelAfterParseContext{Context: context.Background()}
	afterMalformedParseContext := &cancelAfterParseContext{Context: context.Background()}

	const requestID contextKey = "request_id"

	valueRule := &ruleSpy{id: "context_value_rule"}
	valueContext := context.WithValue(context.Background(), requestID, "request-42")

	duringTraversalContext, cancelDuringTraversal := context.WithCancel(context.Background())
	cancelingRule := &ruleSpy{
		id: "canceling_rule",
		decision: func(context.Context, sqlguard.Statement) sqlguard.RuleResult {
			cancelDuringTraversal()

			return sqlguard.Allow()
		},
	}
	laterContextRule := &ruleSpy{id: "later_context_rule"}

	expiredContext, cancelExpiredContext := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	cancelExpiredContext()

	tests := map[string]struct {
		ctx         context.Context
		engine      *sqlguard.Engine
		input       string
		setupTest   func(t *testing.T)
		checkResult func(t *testing.T)
		checkError  func(t *testing.T, err error)
	}{
		"returns_a_preexisting_cancellation_before_rule_evaluation": {
			ctx:    preCanceledContext,
			engine: mustNewEngine(t, preCanceledRule),
			input:  "SELECT 1",
			checkResult: func(t *testing.T) {
				t.Helper()

				require.Empty(t, preCanceledRule.Calls())
			},
			checkError: requireContextError(context.Canceled),
		},
		"observes_cancellation_immediately_after_parsing": {
			ctx:    afterParseContext,
			engine: mustNewEngine(t, afterParseRule),
			input:  "SELECT 1",
			checkResult: func(t *testing.T) {
				t.Helper()

				require.Empty(t, afterParseRule.Calls())
			},
			checkError: requireContextError(context.Canceled),
		},
		"prioritizes_cancellation_observed_after_a_failed_parse": {
			ctx:        afterMalformedParseContext,
			engine:     mustNewEngine(t),
			input:      "SELECT * FROM",
			checkError: requireContextError(context.Canceled),
		},
		"passes_the_exact_caller_context_to_rules": {
			ctx:    valueContext,
			engine: mustNewEngine(t, valueRule),
			input:  "SELECT 1",
			checkResult: func(t *testing.T) {
				t.Helper()

				calls := valueRule.Calls()
				require.Len(t, calls, 1)
				require.True(t, calls[0].ctx == valueContext)
				require.Equal(t, "request-42", calls[0].ctx.Value(requestID))
			},
			checkError: requireNoValidationError,
		},
		"stops_before_the_next_rule_when_canceled_during_traversal": {
			ctx:    duringTraversalContext,
			engine: mustNewEngine(t, cancelingRule, laterContextRule),
			input:  "SELECT 1",
			checkResult: func(t *testing.T) {
				t.Helper()

				require.Len(t, cancelingRule.Calls(), 1)
				require.Empty(t, laterContextRule.Calls())
			},
			checkError: requireContextError(context.Canceled),
		},
		"preserves_deadline_exceeded_for_errors_is": {
			ctx:        expiredContext,
			engine:     mustNewEngine(t),
			input:      "SELECT 1",
			checkError: requireContextError(context.DeadlineExceeded),
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if testCase.setupTest != nil {
				testCase.setupTest(t)
			}

			err := testCase.engine.Validate(testCase.ctx, testCase.input)

			require.NotNil(t, testCase.checkError, "test case must define checkError")
			testCase.checkError(t, err)

			if testCase.checkResult != nil {
				testCase.checkResult(t)
			}
		})
	}
}
