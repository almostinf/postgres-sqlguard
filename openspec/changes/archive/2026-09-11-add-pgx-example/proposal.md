## Why

Consumers need a concrete, reviewable example of placing SQLGuard in front of
common pgx execution methods. The current product requirements instead promise
a production-ready official pgx integration whose breadth and compatibility
commitments exceed the intended scope.

## What Changes

- Add a compiling `example/pgx` package that demonstrates a narrow pgx wrapper
  for `Exec`, `Query`, and `QueryRow`.
- Require the example wrapper to validate before delegation, preserve the
  caller's context, SQL, and arguments for allowed calls, and never invoke the
  underlying executor for rejected or unparsable SQL.
- Preserve validation and pgx errors through their natural APIs, including the
  deferred `QueryRow().Scan()` error path.
- Reject arguments implementing pgx `QueryRewriter` before validation or
  delegation so pgx cannot replace already validated SQL; this includes pgx
  named- and struct-argument helpers built on that mechanism.
- Document unsupported and bypassable pgx paths, including batch, prepared
  statements, transactions, `CopyFrom`, and access to an unwrapped connection.
- Revise the product requirements to describe pgx as a documented integration
  example without a stable or production-ready adapter commitment.

## Capabilities

### New Capabilities

- `pgx-example`: Defines the supported behavior, safety boundary, and
  documented limitations of the pgx wrapper example.

### Modified Capabilities

None.

## Impact

- Adds example and contract-test code under `example/pgx`.
- Adds pgx as an example-scoped dependency without coupling the core
  `sqlguard` package to a database driver.
- Updates `README.md` with example usage and explicit integration limitations.
- Updates `docs/product-requirements.md` to replace the official pgx adapter
  commitment with an explicitly limited example.
- Does not change the core validation API or its fail-closed and privacy
  guarantees.
