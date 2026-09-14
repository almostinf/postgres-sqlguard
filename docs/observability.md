# Observability guide

Observability is optional and disabled by default. `EngineOptions` accepts
independently replaceable `Metrics` and `Logger` implementations:

```go
engine, err := sqlguard.NewEngine(sqlguard.EngineOptions{
	Metrics: applicationMetrics,
	Logger:  applicationLogger,
}, applicationRule)
```

A nil interface disables that sink. An interface containing a typed nil is
invalid and causes engine construction to fail. Configured implementations must
be safe for concurrent calls.

For each terminal validation result, the engine synchronously notifies every
enabled sink exactly once. Sink errors and panics are isolated independently:
they are not retried, cannot cause recursive events, do not prevent the other
sink from running, and never replace a successful result or weaken an
enforce-mode rejection.

## Stable event model

Custom sinks receive an immutable `sqlguard.ValidationEvent` with only these
bounded dimensions:

| Dimension | Values |
| --- | --- |
| `mode` | `enforce` |
| `outcome` | `allowed`, `policy_violation`, `parser_failure`, `canceled`, or `invalid_prepared` |
| `rule_id` | The rejecting construction-time rule identifier for a policy violation; empty otherwise |

Events never contain SQL text, SQL arguments, literals, comments, tokens,
credentials, secrets, parser diagnostics, returned errors, parsed structure, or
caller-context values. Audit mode is not implemented or implied.

## Prometheus

Import the official adapter from:

```go
import sqlguardprometheus "github.com/almostinf/postgres-sqlguard/pkg/observability/prometheus"
```

The adapter requires a caller-owned registry and non-empty, immutable service
identity:

```go
registry := prometheus.NewRegistry()
metrics, err := sqlguardprometheus.New(sqlguardprometheus.Config{
	Registerer: registry,
	Service:    "payments_api",
	// MetricName: "payments_sql_validations_total",
})
if err != nil {
	return err
}

engine, err := sqlguard.NewEngine(
	sqlguard.EngineOptions{Metrics: metrics},
	applicationRule,
)
```

The default counter is `sqlguard_validations_total`. An optional `MetricName`
can replace it with another valid Prometheus metric name. The counter has
exactly the labels `service`, `mode`, `outcome`, and `rule_id`. Collectors are
registered only with the supplied registerer, never implicitly with the
process-global registry. Invalid configuration and registration conflicts are
reported by the constructor.

## log/slog

Import the official adapter from:

```go
import sqlguardslog "github.com/almostinf/postgres-sqlguard/pkg/observability/slog"
```

Wrap an existing non-nil logger:

```go
validationLogger, err := sqlguardslog.New(
	slog.New(slog.NewJSONHandler(os.Stdout, nil)),
)
if err != nil {
	return err
}

engine, err := sqlguard.NewEngine(
	sqlguard.EngineOptions{Logger: validationLogger},
	applicationRule,
)
```

Every record uses the stable message `sqlguard validation` and exactly the
attributes `mode`, `outcome`, and `rule_id`. Levels are INFO for `allowed`,
ERROR for `policy_violation`, `parser_failure`, and `invalid_prepared`, and
DEBUG for `canceled`. The adapter invokes the configured handler with a clean
background context so request-scoped values cannot cross the privacy boundary.

## Write a custom sink

Implement only the applicable root interface:

```go
type applicationMetrics struct{}

func (applicationMetrics) RecordValidation(event sqlguard.ValidationEvent) error {
	// Update a bounded application counter from event accessors.
	return nil
}

type applicationLogger struct{}

func (applicationLogger) LogValidation(event sqlguard.ValidationEvent) error {
	// Emit a bounded application record from event accessors.
	return nil
}
```

Do not derive unbounded labels or log fields from rule internals or external
request state. The engine limits what it supplies, while each custom sink owns
the concurrency safety of its own state.
