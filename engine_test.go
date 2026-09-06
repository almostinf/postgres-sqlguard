package sqlguard_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	sqlguard "github.com/almostinf/postgres-sqlguard"
)

type ruleStub struct {
	id       string
	evaluate func(context.Context, sqlguard.Statement) sqlguard.RuleResult
}

type ruleCall struct {
	ctx       context.Context
	kind      sqlguard.Kind
	structure []sqlguard.Kind
}

type ruleSpy struct {
	id       string
	decision func(context.Context, sqlguard.Statement) sqlguard.RuleResult

	mutex sync.Mutex
	calls []ruleCall
}

func (r *ruleSpy) ID() string {
	return r.id
}

func (r *ruleSpy) Evaluate(ctx context.Context, statement sqlguard.Statement) sqlguard.RuleResult {
	structure := make([]sqlguard.Kind, 0)

	statement.Walk(func(node sqlguard.Node) bool {
		structure = append(structure, node.Kind())

		return true
	})

	r.mutex.Lock()
	r.calls = append(r.calls, ruleCall{ctx: ctx, kind: statement.Kind(), structure: structure})
	r.mutex.Unlock()

	if r.decision == nil {
		return sqlguard.Allow()
	}

	return r.decision(ctx, statement)
}

func (r *ruleSpy) Calls() []ruleCall {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return append([]ruleCall(nil), r.calls...)
}

func (r *ruleStub) ID() string {
	return r.id
}

func (r *ruleStub) Evaluate(ctx context.Context, statement sqlguard.Statement) sqlguard.RuleResult {
	if r.evaluate == nil {
		return sqlguard.Allow()
	}

	return r.evaluate(ctx, statement)
}

type newEngineTestCase struct {
	rules       []sqlguard.Rule
	setupTest   func(t *testing.T)
	checkResult func(t *testing.T, engine *sqlguard.Engine)
	checkError  func(t *testing.T, err error)
}

func TestNewEngine(t *testing.T) {
	var typedNilRule *ruleStub

	tests := map[string]newEngineTestCase{
		"accepts_an_empty_rule_collection": {
			checkResult: requireEngine,
			checkError:  requireNoEngineError,
		},
		"accepts_valid_rule_identifiers": {
			rules: []sqlguard.Rule{
				&ruleStub{id: "require_where-clause.v1"},
			},
			checkResult: requireEngine,
			checkError:  requireNoEngineError,
		},
		"rejects_a_nil_rule": {
			rules:       []sqlguard.Rule{nil},
			checkResult: requireNilEngine,
			checkError:  requireEngineError("sqlguard: rule must not be nil"),
		},
		"rejects_a_typed_nil_rule": {
			rules:       []sqlguard.Rule{typedNilRule},
			checkResult: requireNilEngine,
			checkError:  requireEngineError("sqlguard: rule must not be nil"),
		},
		"rejects_an_empty_rule_identifier": {
			rules:       []sqlguard.Rule{&ruleStub{}},
			checkResult: requireNilEngine,
			checkError:  requireEngineError("sqlguard: rule identifier is invalid"),
		},
		"rejects_an_identifier_with_unsafe_characters": {
			rules:       []sqlguard.Rule{&ruleStub{id: "secret\ncredential=value"}},
			checkResult: requireNilEngine,
			checkError: func(t *testing.T, err error) {
				t.Helper()

				require.EqualError(t, err, "sqlguard: rule identifier is invalid")
				require.NotContains(t, err.Error(), "credential=value")
			},
		},
		"rejects_an_identifier_longer_than_64_characters": {
			rules:       []sqlguard.Rule{&ruleStub{id: strings.Repeat("a", 65)}},
			checkResult: requireNilEngine,
			checkError:  requireEngineError("sqlguard: rule identifier is invalid"),
		},
		"rejects_a_duplicate_rule_identifier": {
			rules: []sqlguard.Rule{
				&ruleStub{id: "deny_drop"},
				&ruleStub{id: "deny_drop"},
			},
			checkResult: requireNilEngine,
			checkError:  requireEngineError("sqlguard: duplicate rule identifier: deny_drop"),
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if testCase.setupTest != nil {
				testCase.setupTest(t)
			}

			engine, err := sqlguard.NewEngine(testCase.rules...)

			require.NotNil(t, testCase.checkError, "test case must define checkError")
			testCase.checkError(t, err)

			if testCase.checkResult != nil {
				testCase.checkResult(t, engine)
			}
		})
	}
}

func TestEngineRuleRegistration(t *testing.T) {
	activeRule := &ruleSpy{id: "active_rule"}
	unregisteredRule := &ruleSpy{id: "unregistered_rule"}

	firstEngineRule := &ruleSpy{id: "first_engine_rule"}
	secondEngineRule := &ruleSpy{id: "second_engine_rule"}

	copiedRule := &ruleSpy{id: "copied_rule"}
	replacementRule := &ruleSpy{id: "replacement_rule"}
	rules := []sqlguard.Rule{copiedRule}
	copiedEngine := mustNewEngine(t, rules...)
	rules[0] = replacementRule

	tests := map[string]struct {
		engines     []*sqlguard.Engine
		input       string
		setupTest   func(t *testing.T)
		checkResult func(t *testing.T)
		checkError  func(t *testing.T, errors []error)
	}{
		"invokes_only_explicitly_supplied_rules": {
			engines: []*sqlguard.Engine{mustNewEngine(t, activeRule)},
			input:   "SELECT 1",
			checkResult: func(t *testing.T) {
				t.Helper()

				require.Len(t, activeRule.Calls(), 1)
				require.Empty(t, unregisteredRule.Calls())
			},
			checkError: requireNoValidationErrors,
		},
		"keeps_separate_engines_independent": {
			engines: []*sqlguard.Engine{
				mustNewEngine(t, firstEngineRule),
				mustNewEngine(t, secondEngineRule),
			},
			input: "SELECT 1",
			checkResult: func(t *testing.T) {
				t.Helper()

				require.Len(t, firstEngineRule.Calls(), 1)
				require.Len(t, secondEngineRule.Calls(), 1)
			},
			checkError: requireNoValidationErrors,
		},
		"copies_the_supplied_rule_collection": {
			engines: []*sqlguard.Engine{copiedEngine},
			input:   "SELECT 1",
			checkResult: func(t *testing.T) {
				t.Helper()

				require.Len(t, copiedRule.Calls(), 1)
				require.Empty(t, replacementRule.Calls())
			},
			checkError: requireNoValidationErrors,
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if testCase.setupTest != nil {
				testCase.setupTest(t)
			}

			errors := make([]error, 0, len(testCase.engines))
			for _, engine := range testCase.engines {
				errors = append(errors, engine.Validate(context.Background(), testCase.input))
			}

			require.NotNil(t, testCase.checkError, "test case must define checkError")
			testCase.checkError(t, errors)

			if testCase.checkResult != nil {
				testCase.checkResult(t)
			}
		})
	}
}

func mustNewEngine(t *testing.T, rules ...sqlguard.Rule) *sqlguard.Engine {
	t.Helper()

	engine, err := sqlguard.NewEngine(rules...)
	require.NoError(t, err)

	return engine
}

func requireNoValidationError(t *testing.T, err error) {
	t.Helper()

	require.NoError(t, err)
}

func requireNoValidationErrors(t *testing.T, errors []error) {
	t.Helper()

	for _, err := range errors {
		require.NoError(t, err)
	}
}

func requireParseFailure(t *testing.T, err error) {
	t.Helper()

	var parseError *sqlguard.ParseError
	require.ErrorAs(t, err, &parseError)
	require.Equal(t, sqlguard.ParseErrorSyntax, parseError.Category())
}

func requireContextError(target error) func(*testing.T, error) {
	return func(t *testing.T, err error) {
		t.Helper()

		require.ErrorIs(t, err, target)
	}
}

func requireViolation(ruleID string) func(*testing.T, error) {
	return func(t *testing.T, err error) {
		t.Helper()

		var violation *sqlguard.Violation
		require.ErrorAs(t, err, &violation)
		require.Equal(t, ruleID, violation.RuleID())
	}
}

func requireErrorChainOmits(t *testing.T, err error, sentinel string) {
	t.Helper()

	for current := err; current != nil; current = errors.Unwrap(current) {
		require.NotContains(t, current.Error(), sentinel)
		require.NotContains(t, fmt.Sprintf("%#v", current), sentinel)
	}
}

func requireEngine(t *testing.T, engine *sqlguard.Engine) {
	t.Helper()

	require.NotNil(t, engine)
}

func requireNilEngine(t *testing.T, engine *sqlguard.Engine) {
	t.Helper()

	require.Nil(t, engine)
}

func requireNoEngineError(t *testing.T, err error) {
	t.Helper()

	require.NoError(t, err)
}

func requireEngineError(message string) func(*testing.T, error) {
	return func(t *testing.T, err error) {
		t.Helper()

		require.EqualError(t, err, message)
	}
}
