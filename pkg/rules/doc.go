// Package rules provides opt-in PostgreSQL-aware validation rules for use with
// sqlguard.
//
// The package includes policies that require WHERE clauses on UPDATE and
// DELETE, require explicit INSERT target columns, and deny TRUNCATE, DROP TABLE,
// or ALTER TABLE operations. No policy is enabled automatically. Each returned
// rule implements sqlguard.Rule and can be registered alongside application
// rules in the same deterministic order.
//
// Rules inspect parsed statement structure. Keywords in comments, identifiers,
// and literals do not create matching operations or satisfy required clauses.
// WHERE TRUE is syntactically a WHERE clause; the package does not perform
// tautology analysis.
package rules
