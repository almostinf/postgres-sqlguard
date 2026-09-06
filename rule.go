package sqlguard

import "context"

const maxRuleIDLength = 64

// Rule evaluates one parsed statement. The same Rule instance may be invoked
// concurrently by separate validation calls, so implementations must either be
// immutable or synchronize their own state.
type Rule interface {
	// ID returns a stable identifier used for programmatic error handling.
	ID() string

	// Evaluate inspects one immutable parsed statement and returns a bounded
	// decision without constructing or returning an error.
	Evaluate(ctx context.Context, statement Statement) RuleResult
}

// RuleResult is the bounded outcome of Rule evaluation. Its zero value rejects
// validation so accidentally omitted decisions fail closed.
type RuleResult struct {
	allowed bool
}

// Allow returns a result that permits validation to continue.
func Allow() RuleResult {
	return RuleResult{allowed: true}
}

// Reject returns a result that stops validation with a policy violation.
func Reject() RuleResult {
	return RuleResult{}
}

// Rejected reports whether validation must stop for this result.
func (r RuleResult) Rejected() bool {
	return !r.allowed
}

// validRuleID limits identifiers to short printable ASCII operational
// metadata. The first character must be alphanumeric; subsequent characters
// may additionally use dot, dash, and underscore separators.
func validRuleID(ruleID string) bool {
	if len(ruleID) == 0 || len(ruleID) > maxRuleIDLength {
		return false
	}

	for index := range len(ruleID) {
		character := ruleID[index]
		if isASCIIAlphaNumeric(character) {
			continue
		}

		if index > 0 && (character == '.' || character == '-' || character == '_') {
			continue
		}

		return false
	}

	return true
}

func isASCIIAlphaNumeric(character byte) bool {
	return character >= 'a' && character <= 'z' ||
		character >= 'A' && character <= 'Z' ||
		character >= '0' && character <= '9'
}
