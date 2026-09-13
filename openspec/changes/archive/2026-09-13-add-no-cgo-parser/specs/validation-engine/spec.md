## ADDED Requirements

### Requirement: Supported CGO and no-CGO builds
The library SHALL provide supported builds with both `CGO_ENABLED=1` and
`CGO_ENABLED=0` without changing its public API. The active parser backend MUST
be selected by the Go build context, MUST NOT require runtime parser
configuration, and a no-CGO build MUST NOT require a C compiler or load native
parser code.

#### Scenario: No-CGO consumer build succeeds
- **WHEN** a consumer builds and tests the library on a supported platform with `CGO_ENABLED=0` and no C compiler available
- **THEN** the build succeeds and the public validation API is usable

#### Scenario: CGO-enabled consumer behavior remains available
- **WHEN** a consumer builds the library on a supported platform with `CGO_ENABLED=1`
- **THEN** the existing public validation API remains source compatible and uses a supported PostgreSQL parser backend

#### Scenario: Parser selection requires no caller configuration
- **WHEN** the same consumer source is built once with CGO enabled and once with CGO disabled
- **THEN** each build selects its supported parser backend without a public option, environment lookup at runtime, or application code change

### Requirement: Cross-backend validation conformance
For the same SQL input, the CGO and no-CGO builds SHALL preserve the same
documented PostgreSQL grammar classification and public validation semantics.
When parsing succeeds, both builds MUST expose semantically equivalent
parser-neutral statement kinds, fields, values, and list ordering to rules and
MUST use the same top-level and nested-CTE traversal order. When parsing fails,
both builds MUST fail closed with the same public error category and MUST NOT
evaluate rules. Direct validation, preparation, prepared validation, rule
selection, first-violation behavior, context precedence, and terminal
observability outcomes MUST remain equivalent across the supported builds.

#### Scenario: PostgreSQL grammar corpus has equivalent outcomes
- **WHEN** the same corpus of valid PostgreSQL-specific, multi-statement, nested-CTE, comment, literal, empty, and malformed inputs is validated by CGO and no-CGO builds
- **THEN** both builds agree on parse success or failure and return equivalent public validation outcomes for every case

#### Scenario: Rules observe equivalent parsed structure
- **WHEN** a custom conformance rule records all statement kinds, field names, value kinds, semantic values, and ordered lists exposed for the same successfully parsed input in each build mode
- **THEN** the two observations are semantically equivalent and contain no parser-backend-specific types

#### Scenario: Built-in rules agree across build modes
- **WHEN** the same input and ordered built-in rule set are evaluated by CGO and no-CGO builds
- **THEN** both builds allow the input or return the same first rule violation after visiting statements in the same order

#### Scenario: Malformed input fails closed in both build modes
- **WHEN** either supported build validates input outside the documented PostgreSQL grammar
- **THEN** both builds return the same typed parser-failure category before invoking a rule and expose no SQL or raw parser diagnostic

#### Scenario: Prepared validation remains equivalent
- **WHEN** the same valid SQL is prepared and repeatedly validated with equivalent engines in each build mode
- **THEN** both builds parse once during preparation and produce equivalent rule, context, error, and observability behavior during prepared validation

### Requirement: Parser backend comparison evidence
The repository SHALL provide a reproducible comparison of the current parser
and viable no-CGO alternatives. The comparison MUST identify supported
PostgreSQL grammar versions and known syntax differences, parser-neutral AST
mapping and rule-semantic compatibility, validation latency and allocations,
binary-size impact, dependency and licensing impact, build requirements, and
deployment or maintenance constraints. Performance and size results MUST state
the toolchain, platform, build settings, inputs, and commands used so that
maintainers can repeat them.

#### Scenario: Maintainer reviews parser alternatives
- **WHEN** a maintainer evaluates the no-CGO backend decision
- **THEN** the documented comparison covers grammar, AST and rule semantics, performance, footprint, dependencies, licensing, and operational tradeoffs for each credible candidate

#### Scenario: Backend measurements are reproducible
- **WHEN** a maintainer follows the documented benchmark and binary-size commands on the stated platform and toolchain
- **THEN** the commands produce separate absolute measurements for the CGO and no-CGO builds using equivalent validation inputs and build settings

#### Scenario: Incompatible candidate is not selected
- **WHEN** a no-CGO candidate cannot preserve the required grammar classification, parser-neutral structure, fail-closed behavior, or rule semantics
- **THEN** the comparison records the incompatibility and the candidate is not presented as a supported backend
