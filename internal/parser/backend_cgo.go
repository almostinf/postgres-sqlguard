//go:build cgo

package parser

import pg_query "github.com/pganalyze/pg_query_go/v6"

// parseBackend delegates parsing to the native PostgreSQL parser.
func parseBackend(sql string) (*pg_query.ParseResult, error) {
	return pg_query.Parse(sql)
}
