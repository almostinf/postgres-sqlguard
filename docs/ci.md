# Continuous integration

The repository exposes separate GitHub Actions signals for static checks and
executable tests. Workflows delegate to the same Make targets used locally so
CI decomposition does not define a second quality policy.

## Workflows and checks

The `Lint` workflow reports these independently diagnosable jobs:

- `Module tidiness` runs `make mod-tidy`;
- `Lint (CGO)` runs `make lint-cgo` with a working C compiler;
- `Lint (no CGO)` runs `make lint-no-cgo` with CGO disabled and `CC` set to a
  deliberately missing compiler.

The `Tests` workflow reports:

- `Tests (CGO)` through `make test-cgo`;
- `Tests (no CGO)` through `make test-no-cgo` and the same missing-compiler
  sentinel;
- `Race (CGO)` through `make test-race-cgo`;
- `Coverage (CGO)` on pushes to `main` only.

The coverage job generates `coverage/coverage.out` in atomic mode, normalizes
execution counts to covered/not-covered values, sorts blocks for reproducible
output, and uploads the profile to Codecov with GitHub OIDC. Only that job
receives `id-token: write`; the workflow default and every other job retain
read-only repository access. Pull requests from forks still run all lint and
non-coverage test jobs because they do not depend on an upload token or trusted
OIDC identity.

For the complete local release gate, run:

```sh
make precommit-all
```

## Required-check migration

The former `CI` workflow reported `Precommit (CGO)` and
`Precommit (no CGO)`. Do not remove those names from branch protection before
the replacement workflows exist on the default branch.

After this change reaches `main`:

1. Wait for one successful `Lint` and `Tests` workflow run on the merge commit.
2. Confirm GitHub reports `Module tidiness`, `Lint (CGO)`, `Lint (no CGO)`,
   `Tests (CGO)`, `Tests (no CGO)`, and `Race (CGO)` on that commit.
3. Add those six reported check names to the branch protection rule or ruleset.
4. Remove the obsolete `Precommit (CGO)` and `Precommit (no CGO)` requirements.
5. Open or update a pull request and confirm the six replacement checks are
   requested and can satisfy protection before relying on the new rule.

`Coverage (CGO)` is deliberately absent from pull requests and must not be a
required pull-request check. Its default-branch result and the live Codecov
report provide the publication signal.
