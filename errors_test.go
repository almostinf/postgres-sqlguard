package sqlguard

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/almostinf/postgres-sqlguard/internal/parser"
)

type typedErrorTestCase struct {
	err         error
	setupTest   func(t *testing.T)
	checkResult func(t *testing.T, err error)
	checkError  func(t *testing.T, err error)
}

func TestTypedErrors(t *testing.T) {
	tests := map[string]typedErrorTestCase{
		"violation_exposes_rule_identifier": {
			err: newViolation("require_where_clause"),
			checkResult: func(t *testing.T, err error) {
				t.Helper()

				var violation *Violation
				require.ErrorAs(t, err, &violation)
				require.Equal(t, "require_where_clause", violation.RuleID())
				require.Equal(t, "sql validation rejected by rule", violation.Error())
			},
			checkError: requireError,
		},
		"wrapped_violation_remains_inspectable": {
			err: fmt.Errorf("request validation failed: %w", newViolation("deny_drop_table")),
			checkResult: func(t *testing.T, err error) {
				t.Helper()

				var violation *Violation
				require.ErrorAs(t, err, &violation)
				require.Equal(t, "deny_drop_table", violation.RuleID())
			},
			checkError: requireWrappedError,
		},
		"parse_error_exposes_failure_category": {
			err: newParseError(ParseErrorSyntax),
			checkResult: func(t *testing.T, err error) {
				t.Helper()

				var parseError *ParseError
				require.ErrorAs(t, err, &parseError)
				require.Equal(t, ParseErrorSyntax, parseError.Category())
				require.Equal(t, "sql validation parsing failed", parseError.Error())
			},
			checkError: requireError,
		},
		"backend_failure_is_translated_to_parse_error": {
			err: translatedParserFailure("sqlguard_secret_token_6f914c"),
			checkResult: func(t *testing.T, err error) {
				t.Helper()

				var parseError *ParseError
				require.ErrorAs(t, err, &parseError)
				require.Equal(t, ParseErrorSyntax, parseError.Category())

				var violation *Violation
				require.False(t, errors.As(err, &violation))
			},
			checkError: requireError,
		},
		"wrapped_parse_error_remains_inspectable": {
			err: fmt.Errorf(
				"request validation failed: %w",
				translatedParserFailure("sqlguard_wrapped_secret_57c921"),
			),
			checkResult: func(t *testing.T, err error) {
				t.Helper()

				var parseError *ParseError
				require.ErrorAs(t, err, &parseError)
				require.Equal(t, ParseErrorSyntax, parseError.Category())
			},
			checkError: requireWrappedError,
		},
	}

	runTypedErrorTests(t, tests)
}

func TestPrivacySafeErrors(t *testing.T) {
	tests := map[string]struct {
		err         error
		sentinel    string
		setupTest   func(t *testing.T)
		checkResult func(t *testing.T, err error, sentinel string)
		checkError  func(t *testing.T, err error)
	}{
		"violation_omits_validation_input": {
			err:         newViolation("deny_sensitive_query"),
			sentinel:    "sqlguard_secret_literal_f320ac",
			checkResult: requireErrorChainOmitsSentinel,
			checkError:  requireError,
		},
		"parser_failure_omits_parser_input": {
			err:         translatedParserFailure("sqlguard_secret_token_6f914c"),
			sentinel:    "sqlguard_secret_token_6f914c",
			checkResult: requireErrorChainOmitsSentinel,
			checkError:  requireError,
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if testCase.setupTest != nil {
				testCase.setupTest(t)
			}

			require.NotNil(t, testCase.checkError, "test case must define checkError")
			testCase.checkError(t, testCase.err)

			if testCase.checkResult != nil {
				testCase.checkResult(t, testCase.err, testCase.sentinel)
			}
		})
	}
}

func translatedParserFailure(input string) error {
	_, err := parser.Parse(input)

	return translateParserError(err)
}

func requireErrorChainOmitsSentinel(t *testing.T, err error, sentinel string) {
	t.Helper()

	for current := err; current != nil; current = errors.Unwrap(current) {
		require.NotContains(t, current.Error(), sentinel)
		require.NotContains(t, fmt.Sprintf("%#v", current), sentinel)
	}
}

func runTypedErrorTests(t *testing.T, tests map[string]typedErrorTestCase) {
	t.Helper()

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if testCase.setupTest != nil {
				testCase.setupTest(t)
			}

			require.NotNil(t, testCase.checkError, "test case must define checkError")
			testCase.checkError(t, testCase.err)

			if testCase.checkResult != nil {
				testCase.checkResult(t, testCase.err)
			}
		})
	}
}

func requireError(t *testing.T, err error) {
	t.Helper()

	require.Error(t, err)
	require.Nil(t, errors.Unwrap(err))
}

func requireWrappedError(t *testing.T, err error) {
	t.Helper()

	require.Error(t, err)
	require.NotNil(t, errors.Unwrap(err))
}
