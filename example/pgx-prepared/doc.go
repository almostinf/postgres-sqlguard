// Package pgxprepared demonstrates a bounded cache of SQLGuard prepared
// values in front of pgx-style Exec, Query, and QueryRow operations. Cache
// hits reuse parsing but every call repeats prepared validation before the
// wrapped executor is invoked.
//
// The cache retains exact SQL strings as keys. Its SQLGuard Prepared values
// retain parsed identifiers, literals, and byte values. Applications should
// prefer stable parameterized SQL with positional arguments instead of
// embedding sensitive values. Cache capacity and MaxSQLBytes bound entry count
// and eligible key length; they are retention controls, not a heap quota,
// because parsed-tree size is not measured. Eviction and Go garbage collection
// do not guarantee prompt memory zeroization.
//
// SQLGuard Prepared values are client-side parsed representations, not pgx or
// PostgreSQL server prepared statements. Executing a manually registered
// statement by passing its name as SQL is unsupported because pgx may execute
// text different from the string validated by this wrapper. Batch operations,
// transactions (including nested transactions), CopyFrom, and calls through an
// unwrapped executor are also outside the guarded boundary. Arguments that
// implement pgx.QueryRewriter are rejected.
//
// This package is an illustrative example, not an official production adapter.
// Its API has no compatibility guarantee.
package pgxprepared
