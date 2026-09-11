// Package pgxexample demonstrates how to place a sqlguard validator in front
// of a narrow pgx execution boundary.
//
// This package is an illustrative example, not an official production adapter.
// Its API has no compatibility guarantee. GuardedDB protects only Exec, Query,
// and QueryRow calls that use positional arguments. QueryRow reports validation
// failures through Scan, following pgx's deferred-error convention.
//
// Batch execution, prepared statement workflows, transactions (including
// nested transactions), CopyFrom, and calls made through a retained raw pgx
// connection or pool are outside this example's validation boundary. Arguments
// implementing pgx.QueryRewriter are rejected before validation or delegation;
// consequently pgx NamedArgs, StrictNamedArgs, StructArgs, and StrictStructArgs
// are unsupported.
package pgxexample
