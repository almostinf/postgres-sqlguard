## ADDED Requirements

### Requirement: Invalid prepared value failure
Prepared validation SHALL reject a zero or otherwise invalid prepared value
before rule evaluation. The returned error MUST be discoverable with
`errors.Is` against the public `ErrInvalidPrepared` sentinel and MUST be
distinct from policy violations, parser failures, and caller-context errors.

#### Scenario: Zero prepared value is rejected
- **WHEN** a caller passes the zero value of `Prepared` to `ValidatePrepared` with an active context
- **THEN** validation fails closed with an error discoverable as `ErrInvalidPrepared` and no rule is invoked

#### Scenario: Wrapped invalid-prepared error remains inspectable
- **WHEN** an invalid-prepared error is wrapped with application context using standard Go error wrapping
- **THEN** `errors.Is` still discovers `ErrInvalidPrepared`

## MODIFIED Requirements

### Requirement: Privacy-safe validation errors
Violation, parser-failure, and invalid-prepared values; their error strings;
and their unwrap chains MUST NOT contain or expose submitted SQL text, SQL
literals, comments, tokens, credentials, secrets, raw parser diagnostics, or
parsed structure. Error metadata SHALL be limited to bounded structural
information needed for programmatic handling, including stable rule
identifiers and failure categories.

#### Scenario: Violation omits sensitive input
- **WHEN** rejected SQL contains a unique literal, token, or credential-like value
- **THEN** neither the violation value nor any error reachable through its unwrap chain exposes that value or the submitted SQL

#### Scenario: Parser failure omits sensitive input
- **WHEN** malformed SQL contains a unique literal, token, or credential-like value
- **THEN** neither the parser-failure value nor any error reachable through its unwrap chain exposes that value or the submitted SQL

#### Scenario: Invalid prepared failure exposes no parsed data
- **WHEN** prepared validation rejects an invalid prepared value
- **THEN** neither the returned error nor any error reachable through its unwrap chain exposes SQL, literal values, or parsed structure
