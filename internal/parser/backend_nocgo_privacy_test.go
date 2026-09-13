//go:build !cgo

package parser

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWASMBackendFailureIsSanitizedAtParserBoundary(t *testing.T) {
	t.Parallel()

	const sentinel = "sqlguard_wasm_secret_2c91ae"

	input := "SELECT '" + sentinel + "' FROM"
	_, backendErr := parseBackend(input)
	require.Error(t, backendErr)

	result, err := Parse(input)

	require.Nil(t, result)

	var parseError *Error
	require.ErrorAs(t, err, &parseError)
	require.Equal(t, FailureSyntax, parseError.Category())
	require.Equal(t, "postgresql parsing failed", err.Error())
	require.False(t, errors.Is(err, backendErr))
	require.Nil(t, errors.Unwrap(err))
	require.NotContains(t, fmt.Sprintf("%#v", err), sentinel)
}
