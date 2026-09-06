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
)

// Engine is an immutable Validator configured only with the ordered Rules
// supplied explicitly at construction. It has no global registry or implicit
// default rules. Once constructed, an Engine is safe for concurrent use when
// its rules satisfy the Rule concurrency contract.
type Engine struct {
	rules []registeredRule
}

type registeredRule struct {
	ruleID string
	rule   Rule
}

// NewEngine constructs an Engine from rules in registration order. It returns
// no partially configured Engine when any registration is invalid.
func NewEngine(rules ...Rule) (*Engine, error) {
	ruleIDs, err := validateRules(rules)
	if err != nil {
		return nil, err
	}

	registrations := make([]registeredRule, 0, len(rules))
	for index, rule := range rules {
		registrations = append(registrations, registeredRule{ruleID: ruleIDs[index], rule: rule})
	}

	return &Engine{rules: registrations}, nil
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

// Validate parses the complete SQL input once and evaluates registered rules
// in registration order for every statement in deterministic traversal order.
// It returns immediately on the first rejected rule result.
func (e *Engine) Validate(ctx context.Context, sql string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	result, parseErr := parser.Parse(sql)

	if err := ctx.Err(); err != nil {
		return err
	}

	if parseErr != nil {
		return translateParserError(parseErr)
	}

	for _, root := range parser.StatementSequence(result) {
		statement := newStatement(root)

		for _, registration := range e.rules {
			if err := ctx.Err(); err != nil {
				return err
			}

			if registration.rule.Evaluate(ctx, statement).Rejected() {
				return newViolation(registration.ruleID)
			}
		}
	}

	return nil
}

// isNilRule rejects both a nil interface and an interface containing a typed
// nil value before the constructor invokes methods on it.
func isNilRule(rule Rule) bool {
	if rule == nil {
		return true
	}

	value := reflect.ValueOf(rule)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
