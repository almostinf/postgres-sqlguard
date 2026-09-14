// Package prometheus provides the official Prometheus metrics integration for
// postgres-sqlguard validation outcomes.
//
// New registers a counter with a caller-supplied registerer. The integration
// never uses the process-global registry implicitly. Its service label is fixed
// at construction, and its remaining labels come only from the bounded
// sqlguard.ValidationEvent dimensions.
package prometheus
