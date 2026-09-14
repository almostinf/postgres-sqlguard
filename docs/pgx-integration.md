# Integrating with pgx

SQLGuard is driver-independent. The repository's compiling pgx packages show
how an application can place a narrow validation boundary in front of selected
driver calls; they are examples, not stable production adapters.

## Direct validation wrapper

The [`example/pgx`](../example/pgx) package wraps a connection or pool:

```go
pool, err := pgxpool.New(ctx, databaseURL)
if err != nil {
	return err
}

guarded, err := pgxexample.NewGuardedDB(engine, pool)
if err != nil {
	return err
}

_, err = guarded.Exec(
	ctx,
	"UPDATE accounts SET active = $1 WHERE id = $2",
	false,
	accountID,
)
```

It protects only `Exec`, `Query`, and `QueryRow` calls made through `GuardedDB`
with positional arguments. A rejected `QueryRow` exposes its error from `Scan`,
consistent with pgx behavior.

The boundary does not cover batch execution, pgx or server prepared-statement
workflows, transactions including nested transactions, `CopyFrom`, or direct
use of the retained connection or pool. Arguments implementing
`pgx.QueryRewriter` are rejected before validation or delegation, so
`pgx.NamedArgs`, `pgx.StrictNamedArgs`, `pgx.StructArgs`, and
`pgx.StrictStructArgs` are unsupported.

## Prepared-validation cache wrapper

The [`example/pgx-prepared`](../example/pgx-prepared) package wraps the same
`Exec`, `Query`, and `QueryRow` subset with a bounded concurrent exact-key LRU
cache. A hit skips parsing but still calls `ValidatePrepared`; a miss calls
`Prepare` and then `ValidatePrepared`. Every positional-argument invocation
therefore receives a fresh policy evaluation.

The cache retains exact SQL keys and parsed values. Capacity and maximum key
length are retention controls, not a heap quota. Eviction or garbage collection
does not guarantee memory zeroization. Prefer stable parameterized SQL, keep
capacity bounded, and do not embed secrets in SQL text.

The prepared wrapper has the same unsupported pgx paths as the direct wrapper.
It does not cache validation decisions and does not make PostgreSQL server
prepared statements part of SQLGuard's API.

## Adopting the pattern

An application remains responsible for ensuring that every database operation
which requires validation crosses its guarded boundary. Keep the raw driver
handle private, enumerate supported methods explicitly, reject rewriting paths
that change SQL after validation, and test that errors are returned before the
driver receives rejected input.

Read the package GoDoc and tests for the exact example behavior. If an
application needs transactions, batches, copy operations, or query rewriting,
design and specify those boundaries explicitly rather than assuming the example
protects them.
