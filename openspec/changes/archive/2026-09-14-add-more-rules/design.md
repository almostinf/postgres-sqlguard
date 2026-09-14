## Context

See `proposal.md` for motivation and
`specs/builtin-mutation-rules/spec.md` for observable behavior. The existing
`rules` package contains two immutable zero-configuration implementations that
inspect the parser-neutral `sqlguard.Statement` facade. The Engine already owns
complete deterministic traversal, rule ordering, typed violations, context
checks, and observability, so this change does not require Engine or parser
changes.

The public facade exposes the structure needed by the new policies. An
`InsertStmt` stores explicit target columns in the repeated `cols` field;
`TruncateStmt` is a dedicated kind; a `DropStmt` distinguishes tables with
`remove_type=OBJECT_TABLE`; and PostgreSQL represents table alterations as
either `AlterTableStmt` or `AlterTableMoveAllStmt`, distinguished from related
object families by `objtype=OBJECT_TABLE`. Field names and symbolic enum values
are part of the existing backend-equivalence contract rather than generated
parser types exposed to `rules`.

## Goals / Non-Goals

**Goals:**

- Keep every new built-in immutable, free of rule-owned mutable state, and safe
  for concurrent Engine use.
- Make decisions only from parser-neutral kinds, fields, and enum values.
- Fail closed when a matching generic statement kind lacks the structural
  discriminator required to prove it is outside a deny policy.
- Keep the dependency direction `rules` -> root `sqlguard`, with no dependency
  from Engine or parser packages back to built-ins.
- Verify the public behavior in both CGO and no-CGO builds through the existing
  repository gates.

**Non-Goals:**

- Introduce shared mutable registries, built-in bundles, rule configuration, or
  YAML loading.
- Inspect table names, column names, expression values, SQL text, or caller
  context.
- Add parser-specific convenience APIs or expose generated PostgreSQL types.
- Reimplement Engine traversal inside a rule.

## Decisions

### Add focused private rule types behind public constructors

Add `rules/insert.go` for `NewInsertRequiresColumns` and `rules/ddl.go` for
`NewDenyTruncate`, `NewDenyDropTable`, and `NewDenyAlterTable`. Each constructor
returns `sqlguard.Rule`; each concrete implementation remains an unexported,
zero-sized value type with value-receiver `ID` and `Evaluate` methods and a
compile-time interface assertion.

Focused constructors keep registration explicit and give each violation one
stable operational identifier. A single configurable deny-list rule was
rejected because it would introduce configuration validation, dynamic
identifiers, and a larger public contract before YAML configuration exists. A
preassembled "safe defaults" bundle was rejected because no policy is safe for
every consumer and the Engine must not imply defaults.

### Detect explicit INSERT columns by list presence

For `InsertStmt`, `NewInsertRequiresColumns` allows only when
`len(statement.Root().Children("cols")) > 0`; a missing or empty list rejects.
Other statement kinds allow immediately. This covers `VALUES`, `INSERT SELECT`,
and `DEFAULT VALUES` without inspecting their data source.

Counting parser-neutral column nodes was chosen over walking for `ResTarget`
nodes because the latter would also encounter expressions outside the target
list. Text matching was rejected because rules do not receive SQL text and
would misclassify comments, literals, and quoted identifiers.

### Implement each DDL policy from structural discriminators

`NewDenyTruncate` rejects `TruncateStmt` directly.

`NewDenyDropTable` first matches `DropStmt`, then reads `remove_type`. It
rejects `OBJECT_TABLE`, allows a present non-table object type, and rejects a
missing or unreadable discriminator. This preserves `DROP VIEW` and other
non-table operations while failing closed if the parser-neutral contract drifts
for a generic drop node.

`NewDenyAlterTable` handles both table-alter representations:

- `AlterTableStmt` with `objtype=OBJECT_TABLE` for ordinary table subcommands;
- `AlterTableMoveAllStmt` with `objtype=OBJECT_TABLE` for
  `ALTER TABLE ALL IN TABLESPACE`.

For either matching generic kind, a missing or unreadable `objtype` rejects;
a present non-table object type allows. Other kinds allow immediately. Matching
only `AlterTableStmt` was rejected because it would miss the tablespace-wide
form. Rejecting every `AlterTableStmt` was rejected because PostgreSQL reuses
that node family for related objects such as indexes, views, sequences, and
foreign tables.

Keep symbolic kind, field, enum, and rule-ID strings in private constants near
the rules that use them. Do not import `internal/parser` or `pg_query_go` from
`rules`.

### Keep evaluation local and stateless

Each `Evaluate` examines only the statement root passed by the Engine and
returns `sqlguard.Allow()` or `sqlguard.Reject()`. It does not retain the
Statement, traverse CTEs, allocate errors, or derive diagnostic data. Engine
traversal will present later top-level statements and statement-bearing CTEs
to the registered rules in the established order.

Parser-neutral facade helpers may materialize defensive snapshots while reading
statement structure, so the rules make no general allocation-free evaluation
guarantee.

This preserves concurrency and privacy mechanically: the rule values have no
mutable state, and neither SQL nor AST-derived identifiers are stored or added
to errors, logs, or metrics. Recursive rule-owned traversal was rejected
because it could duplicate evaluation and diverge from Engine ordering.

### Test through the public API and both parser backends

Add external-package, table-driven tests organized by policy lifecycle:

- constructor contract and stable IDs for all four new rules;
- accepted and rejected `INSERT` variants, including `DEFAULT VALUES` and
  `INSERT SELECT`;
- ordinary, conditional, multi-object, and option-bearing destructive DDL;
- non-target statement kinds and adjacent object families such as `DROP VIEW`
  and `ALTER ROLE`;
- misleading keywords in comments and literals;
- later-statement and nested-CTE behavior where PostgreSQL permits it;
- composition with existing built-ins and first-violation ordering;
- concurrent reuse of the new stateless rule instances.

Tests will follow ADR 0001 and exercise `Engine.Validate` rather than construct
synthetic Statement values. `make precommit-all` will run the same black-box
rule suite with CGO and no-CGO backends. New parser-specific golden fixtures are
not required because this change consumes the already-specified public
parser-neutral contract; if implementation exposes a backend mismatch, fix and
cover that mismatch at the parser boundary before completing the rule task.

### Extend documentation without changing defaults

Update the README built-in-rules section with constructor names, stable policy
meanings, and an explicit registration example. State that the DDL rules deny
operation families rather than particular objects and that the INSERT rule
checks syntactic target-column presence only.

No constructor will be added to an existing example silently, since doing so
would change that example's policy selection and obscure the opt-in contract.

## Risks / Trade-offs

- [Symbolic AST fields may change with a future parser schema] -> Keep them
  private, exercise behavior under both backends, and fail closed on missing
  discriminators for generic deny nodes.
- [The INSERT rule does not prove that listed columns are correct or complete]
  -> Document that it enforces syntactic explicitness only and leave semantic
  validation to PostgreSQL.
- [Focused DDL rules do not block every destructive PostgreSQL operation] ->
  Keep their names and docs precise; add other operation families through
  separately specified rules rather than broadening identifiers silently.
- [An ALTER TABLE syntax can map to more than one AST kind] -> Cover both known
  PostgreSQL representations and include the tablespace-wide form in public
  tests.
- [Fail-closed discriminator handling can reject after an incompatible parser
  change] -> Prefer a visible safe failure over allowing a protected operation;
  backend conformance and precommit-all should detect drift before release.

## Migration Plan

1. Add the four opt-in implementations and public constructors without
   changing existing constructors or Engine behavior.
2. Add public black-box coverage and README documentation.
3. Run strict OpenSpec validation and the CGO/no-CGO repository gates before
   verification.

The change is additive and requires no consumer migration. Rollback removes
the new constructors, implementations, tests, and documentation before a
release; consumers that do not register them are unaffected.
