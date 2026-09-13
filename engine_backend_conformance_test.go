package sqlguard_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	sqlguard "github.com/almostinf/postgres-sqlguard"
)

func TestEngineAcceptsPostgreSQL17Syntax(t *testing.T) {
	t.Parallel()

	input, err := os.ReadFile(filepath.Join(
		"internal", "parser", "testdata", "conformance", "valid", "postgresql_17_copy_on_error.sql",
	))
	require.NoError(t, err)

	rule := &ruleSpy{id: "record_copy"}
	engine := mustNewEngine(t, rule)

	require.NoError(t, engine.Validate(context.Background(), string(input)))
	require.Equal(t, []sqlguard.Kind{"CopyStmt"}, ruleCallKinds(rule.Calls()))
}

func ruleCallKinds(calls []ruleCall) []sqlguard.Kind {
	kinds := make([]sqlguard.Kind, 0, len(calls))

	for _, call := range calls {
		kinds = append(kinds, call.kind)
	}

	return kinds
}
