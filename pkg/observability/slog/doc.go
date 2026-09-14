// Package slog provides the official log/slog integration for
// postgres-sqlguard validation outcomes.
//
// Logger emits the stable message "sqlguard validation" with only mode,
// outcome, and rule_id attributes. It invokes the configured handler with a
// clean background context so SQL data and caller-context values do not cross
// the integration boundary.
package slog
