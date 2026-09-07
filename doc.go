// Package sqlguard validates PostgreSQL statements against explicitly
// registered safety rules without requiring a database connection or driver.
//
// An Engine parses the complete input with the PostgreSQL 17 grammar before it
// invokes any Rule. It visits top-level statements in input order and, for each
// statement, visits the root followed by statement-bearing CTEs depth-first in
// declaration order. Rules run in registration order for every visited
// statement, and validation stops at the first rejection.
//
// A minimal setup is:
//
//	engine, err := sqlguard.NewEngine(sqlguard.EngineOptions{}, applicationRule)
//	if err != nil {
//		return err
//	}
//
//	if err := engine.Validate(ctx, query); err != nil {
//		return err
//	}
//
// Parser failures and rule violations expose only bounded metadata through
// ParseError and Violation. Their messages and unwrap chains never retain the
// submitted SQL or raw parser diagnostics.
//
// Observability is disabled by default. EngineOptions can independently enable
// Metrics and Logger implementations. For every terminal result, Engine calls
// each enabled implementation synchronously with bounded ValidationEvent
// metadata. Sink errors and panics are contained independently and never change
// the validation result or weaken enforce behavior. Events never contain SQL,
// arguments, errors, parser diagnostics, or caller-context values.
//
// Engine is safe for concurrent validation after construction. The same Rule,
// Metrics, or Logger instance may be called concurrently, so implementations
// must be immutable or protect their own state. Context cancellation is checked
// before and after parsing and before every rule call. The synchronous CGO
// parser cannot be interrupted while its C call is in progress; cancellation is
// observed when parsing returns.
package sqlguard
