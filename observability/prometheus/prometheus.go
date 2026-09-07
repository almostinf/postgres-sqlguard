package prometheus

import (
	"errors"
	"fmt"
	"reflect"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"

	sqlguard "github.com/almostinf/postgres-sqlguard"
)

const defaultMetricName = "sqlguard_validations_total"

var (
	errNilRegisterer     = errors.New("sqlguard prometheus: registerer must not be nil")
	errEmptyService      = errors.New("sqlguard prometheus: service must not be empty")
	errInvalidMetricName = errors.New("sqlguard prometheus: metric name is invalid")
)

// Config contains immutable construction-time settings for Metrics.
type Config struct {
	Registerer prometheusclient.Registerer
	Service    string
	MetricName string
}

// Metrics records bounded postgres-sqlguard validation outcomes in a
// Prometheus counter registered with the configured registry.
type Metrics struct {
	service     string
	validations *prometheusclient.CounterVec
}

// New validates config, registers the validation counter with the supplied
// registerer, and returns a Prometheus metrics implementation.
func New(config Config) (*Metrics, error) {
	if isNilRegisterer(config.Registerer) {
		return nil, errNilRegisterer
	}

	if config.Service == "" {
		return nil, errEmptyService
	}

	metricName := config.MetricName
	if metricName == "" {
		metricName = defaultMetricName
	}

	if !model.IsValidMetricName(model.LabelValue(metricName)) {
		return nil, errInvalidMetricName
	}

	validations := prometheusclient.NewCounterVec(
		prometheusclient.CounterOpts{
			Name: metricName,
			Help: "Total number of postgres-sqlguard validation outcomes.",
		},
		[]string{"service", "mode", "outcome", "rule_id"},
	)

	if err := config.Registerer.Register(validations); err != nil {
		return nil, fmt.Errorf("sqlguard prometheus: register metric: %w", err)
	}

	return &Metrics{
		service:     config.Service,
		validations: validations,
	}, nil
}

// RecordValidation records one bounded validation event.
func (m *Metrics) RecordValidation(event sqlguard.ValidationEvent) error {
	m.validations.WithLabelValues(
		m.service,
		string(event.Mode()),
		string(event.Outcome()),
		event.RuleID(),
	).Inc()

	return nil
}

func isNilRegisterer(registerer prometheusclient.Registerer) bool {
	if registerer == nil {
		return true
	}

	value := reflect.ValueOf(registerer)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
