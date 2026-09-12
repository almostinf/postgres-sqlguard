package slog

import (
	"context"
	"errors"
	stdslog "log/slog"
	"time"

	sqlguard "github.com/almostinf/postgres-sqlguard"
)

var errNilLogger = errors.New("sqlguard slog: logger must not be nil")

// Logger emits bounded postgres-sqlguard validation records through a
// configured log/slog handler.
type Logger struct {
	handler stdslog.Handler
}

// New validates logger and returns a logging implementation backed by its
// handler.
func New(logger *stdslog.Logger) (*Logger, error) {
	if logger == nil {
		return nil, errNilLogger
	}

	return &Logger{handler: logger.Handler()}, nil
}

// LogValidation logs one bounded validation event.
func (l *Logger) LogValidation(event sqlguard.ValidationEvent) error {
	level := validationLevel(event.Outcome())

	ctx := context.Background()
	if !l.handler.Enabled(ctx, level) {
		return nil
	}

	record := stdslog.NewRecord(time.Now(), level, "sqlguard validation", 0)
	record.AddAttrs(
		stdslog.String("mode", string(event.Mode())),
		stdslog.String("outcome", string(event.Outcome())),
		stdslog.String("rule_id", event.RuleID()),
	)

	return l.handler.Handle(ctx, record)
}

func validationLevel(outcome sqlguard.ValidationOutcome) stdslog.Level {
	switch outcome {
	case sqlguard.ValidationOutcomePolicyViolation,
		sqlguard.ValidationOutcomeParserFailure,
		sqlguard.ValidationOutcomeInvalidPrepared:
		return stdslog.LevelError
	case sqlguard.ValidationOutcomeCanceled:
		return stdslog.LevelDebug
	default:
		return stdslog.LevelInfo
	}
}
