## Context

See `proposal.md` for motivation and
`specs/builtin-mutation-rules/spec.md` for observable behavior. The root
`sqlguard` package already exposes an immutable Engine, an explicit `Rule`
contract, and a parser-neutral read-only `Statement` tree. Engine traversal
already covers every top-level statement and statement-bearing CTE, so the new
policies can remain ordinary rules instead of introducing policy logic into
the Engine or parser.

The module does not yet have a public policy package. Introducing `rules` now
keeps the root package focused on validation contracts and establishes a home
for a growing set of opt-in built-ins. The dependency direction remains one
way: `rules` imports the root `sqlguard` package, while the root package never
imports `rules`, so no import cycle or implicit registration is introduced.

The parser-neutral tree preserves PostgreSQL protobuf field names. Both
`UpdateStmt` and `DeleteStmt` represent a syntactically present predicate in
the `where_clause` child field; absence of that child represents no `WHERE`
clause. Rules receive no source SQL, which prevents comments and literals from
being mistaken for syntax through textual matching.

## Goals / Non-Goals

**Goals:**

- Add two immutable, zero-configuration built-ins behind public constructors
  in a dedicated `rules` package.
- Make the WHERE decision from the existing parser-neutral statement view.
- Keep Engine, parser traversal, errors, and registration behavior unchanged.
- Organize tests around the public API and every scenario in the new capability
  spec.

**Non-Goals:**

- Add semantic predicate analysis, including detection of `WHERE TRUE` or other
  tautologies.
- Add YAML configuration, default rule registration, audit behavior, or driver
  integration.
- Add mutation-specific methods to the public `Statement` facade.
- Change PostgreSQL parsing or statement traversal order.

## Decisions

### Expose constructors from a dedicated public `rules` package

Add `rules/doc.go` and `rules/mutation.go`. The package exposes
`NewUpdateRequiresWhere()` and `NewDeleteRequiresWhere()`, each returning the
root package's `sqlguard.Rule` interface. Concrete implementations remain
unexported zero-sized value types with value-receiver methods:

```go
import (
	sqlguard "github.com/almostinf/postgres-sqlguard"
	"github.com/almostinf/postgres-sqlguard/rules"
)

engine, err := sqlguard.NewEngine(
	rules.NewUpdateRequiresWhere(),
	rules.NewDeleteRequiresWhere(),
)
```

The constructors cannot fail because the rules have no configuration or
invalid state, so they return one value and no error. Returning
`sqlguard.Rule` keeps concrete types and zero-value construction out of the
public contract. The private types contain no fields and cannot acquire
per-call mutable state, making one returned instance safe for concurrent
Engine calls. Their `ID` methods return the stable identifiers required by the
spec.

Exported concrete types in the root package were rejected because they would
grow the core namespace and permanently expose representation and zero-value
semantics. Exported concrete types in `rules` were rejected because callers
only need the `Rule` contract. Package-level singleton variables were rejected
because public mutable variables would weaken API and concurrency guarantees.
Keeping built-ins in the root package was rejected because the product roadmap
anticipates additional policies and configuration, while Engine must remain
independent of all built-in implementations.

### Inspect statement kind and `where_clause` through the public facade

Each rule first compares `Statement.Kind()` with its target parser-neutral kind
(`UpdateStmt` or `DeleteStmt`). Non-target statements return `Allow()`.
For a target statement, the rule calls `Statement.Root().Child("where_clause")`:
a present child returns `Allow()`, and an absent child returns `Reject()`.

This deliberately tests syntactic presence only. The child may represent any
valid PostgreSQL expression, so `WHERE TRUE` is accepted without a special
case. Comments and string literals cannot create this child and therefore
cannot satisfy the policy.

Searching SQL text was rejected because rules do not receive source SQL and
text matching would violate the PostgreSQL-accurate safety boundary. Adding a
public `HasWhereClause` convenience method was rejected because the generic
facade already expresses the required inspection and the new method would
expand the public statement contract for only two rules. Moving the check into
`internal/parser` was rejected because policy belongs in rules and future rules
must remain addable without parser changes.

### Share only the private mechanics common to both rules

Use unexported constants for target kinds, the `where_clause` field, and stable
rule identifiers. A small unexported helper may implement the shared
target-kind and child-presence decision, while each private rule type retains
its own `ID` and `Evaluate` methods and is created by its matching public
constructor.

This avoids duplicating safety-critical branching without introducing a public
generic mutation-rule configuration API. A single configurable rule was
rejected because the requested public API names and violation identifiers are
distinct, and exposing configuration would add invalid-state handling with no
current benefit.

### Rely on Engine traversal rather than recursing inside rules

The rules evaluate only the `Statement` value provided for the current
statement root. They do not walk descendants looking for CTE mutations. The
Engine already owns deterministic complete traversal and presents each
statement-bearing CTE as a separate `Statement`; duplicating traversal in a
rule could evaluate nested mutations twice or diverge from Engine ordering.

Multiple statements and nested CTE coverage will therefore be tested through
`Engine.Validate`, with both rules registered. No changes are planned for
`engine.go`, `internal/parser`, or the public Statement API.

Rule-owned traversal was rejected because complete statement discovery is an
Engine responsibility established by `validation-engine`, not a policy-specific
behavior.

### Verify behavior with public black-box table-driven tests

Add `rules/mutation_test.go` in external package `rules_test`. Tests import both
the root `sqlguard` package and the public `rules` package, exercising the same
dependency boundary as consumers. Use the repository ADR's
`map[string]struct{...}` table style, `snake_case` case names, parallel
independent subtests, setup/check helpers where useful, and `testify/require`
assertions.

Test groups will cover:

- each constructor's returned rule, stable ID, and implementation of the
  public `sqlguard.Rule` contract;
- target mutations with no WHERE, normal predicates, and `WHERE TRUE`;
- non-target statements;
- misleading `WHERE`, `UPDATE`, and `DELETE` text in comments and literals;
- unsafe later statements in multi-statement input;
- unsafe and safe mutations in nested statement-bearing CTEs;
- both built-ins registered alongside a test-only custom rule, proving that
  Engine registration and execution require no built-in-specific path.

The extensibility proof remains black-box: the test defines a new rule using
only exported contracts and registers it with the built-ins. `engine.go` and
parser files remaining unchanged are an implementation boundary checked during
review, not a runtime assertion.

Direct construction of synthetic `Statement` values was rejected because the
facade intentionally has no public mutable builder. Parsing through
`Engine.Validate` exercises the same public path consumers use and proves that
comments, literals, batches, and CTE traversal interact correctly with the
rules.

### Document explicit registration in the public usage guide

Update `README.md` with a concise example importing the public `rules` package
and passing the results of both constructors to `NewEngine`. The documentation
will emphasize that they are opt-in and that a syntactically present predicate,
including `WHERE TRUE`, satisfies these baseline rules.

Silently adding the rules to `NewEngine` was rejected because it would violate
explicit registration and change existing consumers' behavior.

## Risks / Trade-offs

- [The rules depend on parser-neutral node and field names] → Keep names in
  private constants and cover them through public end-to-end fixtures; any
  parser adapter change must preserve the Statement facade contract or update
  the affected capability deliberately.
- [Syntactic WHERE presence can allow an intentionally broad predicate] →
  Document the limitation and leave tautology detection to a separate rule and
  capability.
- [A second public package adds an import for consumers] → Keep its purpose
  narrow and let it depend only on the root contracts; the separation prevents
  built-in growth from expanding the core namespace or coupling Engine to
  policies.
- [No-argument constructors may look unnecessary for stateless rules] → Use
  them to hide concrete representations and keep construction style consistent
  with future built-ins that may require configuration.
- [Engine-level fixtures overlap existing traversal tests] → Keep parser
  traversal mechanics tested internally and use the new fixtures only to prove
  observable built-in policy outcomes.

## Migration Plan

1. Add the public `rules` package, its package documentation, private stateless
   implementations, constructors, and compile-time contract assertions.
2. Add public black-box scenario coverage in `rules_test`, including
   custom-rule coexistence.
3. Update README usage and run the strict OpenSpec and repository precommit
   gates.

The change is additive and requires no consumer migration. Rollback removes
the new `rules` package and its documentation before a release; Engine and
parser behavior remain unaffected.
