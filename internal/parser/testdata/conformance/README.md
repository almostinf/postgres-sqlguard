# PostgreSQL 17 parser conformance corpus

This directory contains synthetic, non-secret SQL inputs for comparing the
supported parser backends. Each `.sql` file is one complete parser input:

- `valid/` inputs must parse successfully with the pinned PostgreSQL 17
  grammar;
- `invalid/` inputs must be rejected as a complete input.

Tests enumerate recursive `.sql` paths in lexicographic order.
Classification comes only from the first directory component, so adding a case
does not require updating a duplicate manifest. Files are data, not executable
fixtures. Expected parser-neutral AST snapshots for valid inputs live in
`golden/` and use an explicit, deterministic JSON representation that preserves
node kinds, field order, value kinds and values, and list order. Regenerate them
from the reviewed CGO baseline with:

```bash
CGO_ENABLED=1 UPDATE_CONFORMANCE_GOLDENS=1 go test ./internal/parser \
  -run '^TestParserConformanceCorpus$'
```

Always review golden diffs and confirm the ordinary test passes with both
`CGO_ENABLED=1` and `CGO_ENABLED=0` before accepting an update.

All identifiers and values are invented for this repository. Do not add
production SQL, customer data, credentials, access tokens, or other secrets.

## Coverage map

| Area | Cases |
| --- | --- |
| PostgreSQL 17 syntax | `valid/postgresql_17_copy_on_error.sql`, `valid/postgresql_17_merge_not_matched_by_source.sql` |
| Comments and keywords in non-syntax text | `valid/comments.sql` |
| Quoted identifiers, strings, escape strings, and dollar quoting | `valid/quoted_forms.sql`, `valid/ddl_function_dollar_body.sql` |
| Parameters, arrays, casts, operators, and representative literals | `valid/parameters_expressions_literals.sql` |
| DML | `valid/dml_select.sql`, `valid/dml_insert.sql`, `valid/dml_update.sql`, `valid/dml_delete.sql`, and both `valid/postgresql_17_*.sql` cases |
| DDL | `valid/ddl_table.sql`, `valid/ddl_alter_index.sql`, `valid/ddl_function_dollar_body.sql` |
| Empty complete input | `valid/empty_input.sql`; whitespace-, comment-, and semicolon-only forms remain covered by the focused parser unit test |
| Multiple top-level statements | `valid/multi_statement_batch.sql` |
| Statement-bearing data-modifying CTEs | `valid/data_modifying_ctes.sql` |
| Recursive nesting and declaration order | `valid/nested_data_modifying_ctes.sql` |
| Built-in rule fields | `valid/dml_update.sql` and `valid/dml_delete.sql` expose `UpdateStmt`/`DeleteStmt` with `where_clause`; `valid/data_modifying_ctes.sql` also includes each kind without `where_clause` |
| Traversal fields | Both CTE cases exercise `with_clause`, ordered `ctes`, and `ctequery`; the nested case exercises those fields recursively |
| Complete-input and batch rejection | `invalid/malformed_batch_middle.sql` proves a valid prefix and suffix cannot hide an invalid member; focused parser tests cover standalone truncation |
| Malformed quoted/comment/parameter forms | `invalid/unterminated_quoted_identifier.sql`, `invalid/unterminated_string.sql`, `invalid/unterminated_dollar_quote.sql`, `invalid/unterminated_comment.sql`, `invalid/malformed_parameter.sql` |
