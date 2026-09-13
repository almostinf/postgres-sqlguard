## MODIFIED Requirements

### Requirement: Consistent parsed representation
Rules SHALL evaluate a structured, parser-neutral statement representation
produced by the engine's supported PostgreSQL parser backend. Within one
validation call, every rule evaluating the same statement MUST receive a
consistent interpretation of that statement. Across CGO and no-CGO builds, the
same successfully parsed input MUST expose semantically equivalent statement
kinds, field presence and names, value kinds and semantic values, and ordered
lists through the public rule contract. Adding a custom rule MUST NOT require
changes to the parser or engine and MUST NOT require knowledge of the active
parser backend.

#### Scenario: Rules share one statement interpretation
- **WHEN** multiple registered rules evaluate the same parsed statement during one validation call
- **THEN** they observe equivalent statement structure derived from the same successfully parsed input

#### Scenario: Custom rule is added without engine modification
- **WHEN** an application implements the public rule contract and registers the rule explicitly
- **THEN** the engine can execute it without changes to engine or parser implementation

#### Scenario: Custom rule behavior is backend-independent
- **WHEN** the same custom rule and SQL input are used in CGO and no-CGO builds
- **THEN** the rule observes semantically equivalent statement data and produces the same result without detecting or depending on the active parser backend
