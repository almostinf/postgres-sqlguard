package pgxexample

import (
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	_ Executor = (*pgx.Conn)(nil)
	_ Executor = (*pgxpool.Pool)(nil)
)
