# postgres-sqlguard Product Requirements

**Status:** Draft  
**Audience:** Maintainers and contributors

## 1. Purpose

`postgres-sqlguard` is a Go library for validating PostgreSQL statements before
they are executed. It helps applications prevent broad or otherwise unsafe
database mutations by applying an explicit, extensible set of rules to SQL.

The library is intended to provide a reliable safety boundary around SQL
execution. It complements database permissions, reviews, and tests; it does
not replace them.

This document defines the product direction and the constraints that future
OpenSpec changes must preserve. Detailed behavior belongs in capability specs,
and implementation choices belong in change designs.

## 2. Intended Users

The primary users are Go developers and maintainers of services that execute
PostgreSQL statements and want enforceable safeguards close to the execution
path.

The library should work for teams that:

- execute SQL directly or through PostgreSQL drivers;
- build SQL dynamically and therefore cannot rely only on source-code checks;
- need organization-specific safety rules;
- want to introduce enforcement gradually through audit mode;
- need useful operational signals without exposing sensitive query data.

## 3. Usage Model

### Direct validation

The core product is a driver-independent SQL validator. An application creates
a validator with an explicit list of rules and passes a context and SQL
statement to it before execution.

Rules are registered explicitly. The library must not silently enable policies
or discover rules through global registration.

Applications may also select and configure built-in rules through a
`rules.yaml` file. The initial YAML configuration supports built-in rules only;
custom rules continue to be registered through Go code.

### Driver integration

The core validator must not depend on pgx or any other database driver.

The project provides a production-ready pgx integration that applies the same
validator to normal pgx and pgxpool execution paths. Direct validation and the
official pgx integration are both supported product use cases.

Other drivers can integrate through the validator contract. The project may
provide examples for those integrations without committing to maintain every
driver adapter.

### Operational integration

The library owns the emission of validation metrics and log events. Consumers
should not need to reproduce library counters or infer validation outcomes from
returned errors.

The core uses replaceable metrics and logging abstractions. The project
provides official Prometheus metrics and `log/slog` logging integrations, while
allowing applications to supply alternative implementations or configuration.

## 4. Product Principles

### PostgreSQL-accurate decisions

Safety decisions must be based on a real PostgreSQL grammar and parsed query
structure. Regular expressions and keyword searches must not determine whether
a statement is safe. Text inside comments or string literals must not be
mistaken for SQL syntax.

### Fail closed enforcement

Enforce mode is the default safety boundary. An unsafe statement or a statement
that cannot be parsed must be rejected before an official driver integration
contacts PostgreSQL.

Any supported mechanism that can rewrite SQL after validation must either
validate the resulting SQL or be explicitly unsupported. The library must not
claim protection for execution paths it cannot validate.

### Explicit and extensible policy

Applications choose their rules explicitly. Adding a custom rule must not
require changes to the parser, validation engine, or driver integrations.

Rule evaluation must use a consistent interpretation of each query.

### Privacy boundary

The library must not include or pass SQL text, SQL arguments, literal values,
tokens, or secrets in metrics, log events, audit findings, or error messages.

Metric labels must remain bounded and must not derive unbounded values from SQL
or application data.

### Honest integration boundaries

The project must document execution paths that cannot be protected, including
operations without SQL text and APIs that expose an underlying unguarded
connection. Limitations must be visible rather than implied away.

## 5. Core Product Requirements

### Validation engine

The validator must:

- parse every statement supplied in a validation request;
- ensure that no statement, including a mutation nested inside a CTE, can
  bypass the registered rule set;
- return a typed violation and allow validation to stop at the first detected
  violation;
- return a distinguishable parser failure when parsing fails;
- preserve cancellation and request-scoped values from the caller's context;
- support concurrent use without data races.

The first built-in policies must reject `UPDATE` and `DELETE` statements that
do not contain a `WHERE` clause. A syntactically present condition, including
`WHERE TRUE`, satisfies these initial rules. Detecting tautological conditions
is a separate possible policy.

### Rule extensibility and configuration

Rules must have stable identifiers and be registered through explicit
construction. The initial built-in rules can also be enabled and configured
through `rules.yaml`.

Configuration errors, unknown built-in rule identifiers, and invalid rule
settings must be reported clearly. Configuration must not silently weaken the
selected policy.

Support for resolving third-party custom rules from YAML is outside the initial
scope and may be proposed later.

### Official pgx integration

The official pgx integration must validate SQL before delegating supported
operations. It must cover normal command, query, row, batch, prepared
statement, and transaction workflows, including nested transactions.

When pgx reports an error through a deferred interface, such as row scanning
or batch results, validation failures must remain discoverable through the same
interface. Rejected operations must not reach the underlying connection.

Operations that do not provide SQL text, such as `CopyFrom`, are delegated
without AST validation and documented as a limitation. APIs that expose the
raw pgx connection are also documented as escape hatches from the guard.

### Errors

Callers must be able to distinguish policy violations, parser failures, and
underlying driver failures without parsing error messages. Wrapped errors must
work with Go's standard error inspection mechanisms.

Error messages must be safe to log and must not contain SQL text, SQL
arguments, literal values, tokens, or secrets.

### Observability

The library must record validation outcomes through configurable metrics and
logging integrations.

At a product level, observability must make it possible to distinguish:

- successful validations;
- policy violations by stable rule identifier;
- parser failures;
- enforce and audit outcomes.

The engine owns updating these signals. Consumers can replace or disable the
configured implementations without changing validation behavior.

Observability failures must not weaken enforce-mode safety. The exact failure
policy for custom observability implementations must be defined in the
relevant capability spec.

## 6. Delivery Stages

### Stage 1: Enforce MVP

The first usable release includes:

- the driver-independent validation engine;
- explicit programmatic rule registration;
- built-in `UPDATE` and `DELETE` policies;
- configuration of built-in rules through `rules.yaml`;
- fail-closed parser and policy handling;
- typed, privacy-safe errors;
- replaceable metrics and logging abstractions, plus official Prometheus and
  `log/slog` integrations;
- the production-ready pgx integration;
- tests and benchmarks for the supported safety boundary.

The MVP may require CGO. This constraint must be documented clearly for
consumers.

### Stage 2: Audit mode

Audit mode is the required next product stage after the enforce MVP. It allows
execution to continue while reporting a privacy-safe finding for policy
violations or parser failures, according to the audit specification.

Audit mode must reuse the same parser and rule behavior as enforce mode. It
must not introduce SQL logging or an inline query bypass mechanism.

### Stage 3: No-CGO support

A later stage evaluates and, if viable, provides a supported parser backend for
`CGO_ENABLED=0` environments. It may be proposed only after audit mode is
delivered. The backend must preserve the same product-level validation
guarantees and rule semantics.

No-CGO support must be introduced through a separate OpenSpec change.

### Conditional performance work

Performance optimization begins with benchmarks. A validation cache may be
introduced only when measurements show a useful benefit. It must be optional,
bounded, concurrency-safe, and preserve validation correctness and privacy.
Cache design belongs in the corresponding change design.

## 7. Product Boundaries and Non-Goals

The following boundaries apply to the product or its initial delivery stages:

- using regular expressions or string searches as a safety decision mechanism;
- accepting inline comments that disable safety-critical rules;
- detecting tautological `WHERE` clauses as part of the initial built-in rules;
- maintaining official adapters for every PostgreSQL or database driver;
- validating operations that do not expose SQL text;
- concealing escape hatches that expose an underlying driver connection;
- including SQL text, arguments, literals, tokens, or secrets in observability
  output or errors;
- loading third-party custom rule implementations from `rules.yaml`.

## 8. Quality Expectations

The library targets Go 1.26 and later.

Public APIs must be documented. Normal failures must be inspectable without
parsing error messages, and malformed input must not cause a panic.

Safety-critical behavior must be tested, including proof that rejected SQL
does not reach protected executors. The library must remain safe for concurrent
use.

Changes are ready to archive only after strict OpenSpec verification and the
project precommit gate pass. Detailed test matrices and repository commands
belong in the contributor documentation.

Representative benchmarks are required before performance features are
accepted.

## 9. Decision Rule for Future Changes

Maintainers use this document as a product-level filter for future OpenSpec
proposals.

A proposed change should:

- strengthen or preserve the stated safety and privacy guarantees;
- keep the core validator independent from database drivers;
- maintain explicit rule registration and extension boundaries;
- state which delivery stage it advances;
- document any newly exposed bypass or unsupported execution path;
- avoid adding performance complexity without measurement;
- update this document when it intentionally changes product direction.

Detailed behavior must be captured in OpenSpec capability requirements and
scenarios. Technical choices, dependencies, APIs, and trade-offs must be
resolved in the corresponding design rather than in this product document.
