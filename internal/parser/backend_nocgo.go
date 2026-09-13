//go:build !cgo

package parser

import (
	pg_query "github.com/pganalyze/pg_query_go/v6"
	wasm_query "github.com/wasilibs/go-pgquery"
)

// parseBackend delegates parsing to the WebAssembly PostgreSQL parser.
func parseBackend(sql string) (*pg_query.ParseResult, error) {
	return wasm_query.Parse(sql)
}
