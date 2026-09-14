## ADDED Requirements

### Requirement: Official observability package locations
The official Prometheus integration SHALL be publicly importable from
`github.com/almostinf/postgres-sqlguard/pkg/observability/prometheus`, and the
official `log/slog` integration SHALL be publicly importable from
`github.com/almostinf/postgres-sqlguard/pkg/observability/slog`. The module MUST
NOT retain the former public packages below
`github.com/almostinf/postgres-sqlguard/observability`. Moving the packages MUST
NOT change their integration APIs, emitted dimensions, privacy guarantees, or
failure-isolation behavior.

#### Scenario: Prometheus integration uses the stable package path
- **WHEN** a consumer imports `github.com/almostinf/postgres-sqlguard/pkg/observability/prometheus`
- **THEN** the official Prometheus integration is available with its existing public API and behavior

#### Scenario: Slog integration uses the stable package path
- **WHEN** a consumer imports `github.com/almostinf/postgres-sqlguard/pkg/observability/slog`
- **THEN** the official `log/slog` integration is available with its existing public API and behavior

#### Scenario: Former observability packages are removed
- **WHEN** the module packages for the stable release are enumerated
- **THEN** no package remains below `github.com/almostinf/postgres-sqlguard/observability`

#### Scenario: Package move preserves observability contracts
- **WHEN** either official integration is used from its new package path
- **THEN** its validation outcomes, bounded dimensions, privacy boundary, and failure isolation match the existing observability requirements
