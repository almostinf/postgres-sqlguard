package sqlguard

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/almostinf/postgres-sqlguard/internal/parser"
)

var (
	errNilRule       = errors.New("sqlguard: rule must not be nil")
	errInvalidRuleID = errors.New("sqlguard: rule identifier is invalid")
	errNilMetrics    = errors.New("sqlguard: metrics must not be nil")
	errNilLogger     = errors.New("sqlguard: logger must not be nil")
)

// Engine is an immutable Validator configured with independently optional
// observability implementations and the ordered Rules supplied explicitly at
// construction. It has no global registry or implicit default rules. Once
// constructed, an Engine is safe for concurrent use when its rules and
// observability implementations satisfy their concurrency contracts.
type Engine struct {
	rules   []registeredRule
	metrics Metrics
	logger  Logger
}

type registeredRule struct {
	ruleID string
	rule   Rule
}

// NewEngine constructs an Engine from options and rules in registration order.
// Zero-valued options disable observability. A nil Metrics or Logger interface
// disables that sink, while an interface containing a typed nil is invalid. It
// returns no partially configured Engine when any option or registration is
// invalid.
func NewEngine(options EngineOptions, rules ...Rule) (*Engine, error) {
	if isTypedNil(options.Metrics) {
		return nil, errNilMetrics
	}

	if isTypedNil(options.Logger) {
		return nil, errNilLogger
	}

	ruleIDs, err := validateRules(rules)
	if err != nil {
		return nil, err
	}

	registrations := make([]registeredRule, 0, len(rules))
	for index, rule := range rules {
		registrations = append(registrations, registeredRule{ruleID: ruleIDs[index], rule: rule})
	}

	return &Engine{
		rules:   registrations,
		metrics: options.Metrics,
		logger:  options.Logger,
	}, nil
}

// validateRules verifies the complete ordered rule collection before Engine
// construction and returns the stable identifiers read during validation.
func validateRules(rules []Rule) ([]string, error) {
	ruleIDs := make([]string, 0, len(rules))
	seenRuleIDs := make(map[string]struct{}, len(rules))

	for _, rule := range rules {
		if isNilRule(rule) {
			return nil, errNilRule
		}

		ruleID := rule.ID()
		if !validRuleID(ruleID) {
			return nil, errInvalidRuleID
		}

		if _, exists := seenRuleIDs[ruleID]; exists {
			return nil, fmt.Errorf("sqlguard: duplicate rule identifier: %s", ruleID)
		}

		seenRuleIDs[ruleID] = struct{}{}
		ruleIDs = append(ruleIDs, ruleID)
	}

	return ruleIDs, nil
}

// Validate prepares the complete SQL input and validates the resulting parsed
// representation. It preserves the same errors, traversal, context behavior,
// and single terminal outcome as calling Prepare followed by ValidatePrepared.
func (e *Engine) Validate(ctx context.Context, sql string) error {
	prepared, err := e.Prepare(ctx, sql)
	if err != nil {
		return err
	}

	return e.ValidatePrepared(ctx, prepared)
}

// Prepare synchronously parses the complete SQL input once without evaluating
// rules. A successful preparation emits no terminal validation outcome. Parse
// and context failures are emitted once through the Engine's observability
// implementations before the error is returned.
func (e *Engine) Prepare(ctx context.Context, sql string) (Prepared, error) {
	if err := ctx.Err(); err != nil {
		return Prepared{}, e.finishValidation(ValidationOutcomeCanceled, "", err)
	}

	result, parseErr := parser.Parse(sql)

	if err := ctx.Err(); err != nil {
		return Prepared{}, e.finishValidation(ValidationOutcomeCanceled, "", err)
	}

	if parseErr != nil {
		return Prepared{}, e.finishValidation(
			ValidationOutcomeParserFailure,
			"",
			translateParserError(parseErr),
		)
	}

	return Prepared{result: result}, nil
}

// ValidatePrepared evaluates every registered rule against a successfully
// prepared value in deterministic statement and registration order. It checks
// the caller context before prepared-value validity and before each rule,
// returns the first failure, and emits exactly one terminal outcome.
func (e *Engine) ValidatePrepared(ctx context.Context, prepared Prepared) error {
	if err := ctx.Err(); err != nil {
		return e.finishValidation(ValidationOutcomeCanceled, "", err)
	}

	if prepared.result == nil {
		return e.finishValidation(ValidationOutcomeInvalidPrepared, "", ErrInvalidPrepared)
	}

	for _, root := range parser.StatementSequence(prepared.result) {
		statement := newStatement(root)

		for _, registration := range e.rules {
			if err := ctx.Err(); err != nil {
				return e.finishValidation(ValidationOutcomeCanceled, "", err)
			}

			if registration.rule.Evaluate(ctx, statement).Rejected() {
				return e.finishValidation(
					ValidationOutcomePolicyViolation,
					registration.ruleID,
					newViolation(registration.ruleID),
				)
			}
		}
	}

	return e.finishValidation(ValidationOutcomeAllowed, "", nil)
}

// finishValidation synchronously emits the already classified terminal event
// before returning the original validation result.
func (e *Engine) finishValidation(outcome ValidationOutcome, ruleID string, result error) error {
	event := newValidationEvent(ValidationModeEnforce, outcome, ruleID)

	if e.metrics != nil {
		isolateObservabilityFailure(func() error {
			return e.metrics.RecordValidation(event)
		})
	}

	if e.logger != nil {
		isolateObservabilityFailure(func() error {
			return e.logger.LogValidation(event)
		})
	}

	return result
}

// isolateObservabilityFailure contains one runtime sink call. Runtime errors
// and panics are deliberately discarded without retrying or emitting another
// event, leaving validation results unchanged.
func isolateObservabilityFailure(call func() error) {
	defer func() {
		_ = recover()
	}()

	_ = call()
}

// isNilRule rejects both a nil interface and an interface containing a typed
// nil value before the constructor invokes methods on it.
func isNilRule(rule Rule) bool {
	if rule == nil {
		return true
	}

	return isTypedNil(rule)
}

// isTypedNil reports whether a non-nil interface contains a nil value.
func isTypedNil(value any) bool {
	if value == nil {
		return false
	}

	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}
