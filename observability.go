package sqlguard

// ValidationMode identifies a bounded validation execution mode for
// observability implementations.
type ValidationMode string

const (
	// ValidationModeEnforce identifies validation that rejects unsafe or
	// unparseable input.
	ValidationModeEnforce ValidationMode = "enforce"
)

// ValidationOutcome identifies a bounded terminal validation result for
// observability implementations.
type ValidationOutcome string

const (
	// ValidationOutcomeAllowed indicates that parsing and all registered rule
	// evaluations succeeded.
	ValidationOutcomeAllowed ValidationOutcome = "allowed"

	// ValidationOutcomePolicyViolation indicates that a registered rule
	// rejected a statement.
	ValidationOutcomePolicyViolation ValidationOutcome = "policy_violation"

	// ValidationOutcomeParserFailure indicates that the PostgreSQL parser
	// rejected the complete input.
	ValidationOutcomeParserFailure ValidationOutcome = "parser_failure"

	// ValidationOutcomeCanceled indicates that validation stopped because the
	// caller context was canceled or its deadline expired.
	ValidationOutcomeCanceled ValidationOutcome = "canceled"
)

// ValidationEvent is immutable bounded metadata for one terminal validation
// result. It never contains SQL, arguments, errors, parser diagnostics, or
// caller-context values.
type ValidationEvent struct {
	mode    ValidationMode
	outcome ValidationOutcome
	ruleID  string
}

// newValidationEvent constructs an event and preserves a rule identifier only
// for a policy violation.
func newValidationEvent(mode ValidationMode, outcome ValidationOutcome, ruleID string) ValidationEvent {
	if outcome != ValidationOutcomePolicyViolation {
		ruleID = ""
	}

	return ValidationEvent{mode: mode, outcome: outcome, ruleID: ruleID}
}

// Mode returns the bounded validation execution mode.
func (e ValidationEvent) Mode() ValidationMode {
	return e.mode
}

// Outcome returns the bounded terminal validation result.
func (e ValidationEvent) Outcome() ValidationOutcome {
	return e.outcome
}

// RuleID returns the stable rejecting rule identifier for a policy violation
// and an empty string for every other outcome.
func (e ValidationEvent) RuleID() string {
	return e.ruleID
}

// Metrics records terminal validation events. An Engine calls an enabled
// Metrics implementation synchronously once per validation and contains any
// returned error or panic. An Engine may call the same implementation
// concurrently, so implementations must either be immutable or synchronize
// their own state.
type Metrics interface {
	// RecordValidation records one terminal validation event.
	RecordValidation(event ValidationEvent) error
}

// Logger logs terminal validation events. An Engine calls an enabled Logger
// implementation synchronously once per validation and contains any returned
// error or panic. An Engine may call the same implementation concurrently, so
// implementations must either be immutable or synchronize their own state.
type Logger interface {
	// LogValidation logs one terminal validation event.
	LogValidation(event ValidationEvent) error
}

// EngineOptions configures independently optional observability
// implementations for an Engine. Its zero value disables both metrics and
// logging. A nil interface disables its sink; an interface containing a typed
// nil is invalid and causes NewEngine to fail.
type EngineOptions struct {
	// Metrics records terminal validation events. A nil value disables metrics.
	Metrics Metrics

	// Logger logs terminal validation events. A nil value disables logging.
	Logger Logger
}
