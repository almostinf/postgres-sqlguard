# Stable release runbook

This runbook publishes the first stable `v1.0.0` tag from one reviewed,
verified, archived, and green commit on `main`. A GitHub Release and automated
module publishing are outside this runbook.

## Preconditions

Before archiving, complete every task in
`stabilize-enforce-mvp`, resolve all verification findings, and obtain review
approval. Run the pre-archive gates from the candidate branch:

```sh
openspec validate stabilize-enforce-mvp --strict
make precommit-all
make release-bench-check
```

Run `$openspec-verify-change stabilize-enforce-mvp` in Codex and proceed only
when it reports no critical issues and every warning is resolved or explicitly
reviewed and accepted. Do not create a candidate tag during implementation or
verification.

## Publish `v1.0.0`

Follow these steps in order. Stop at the first failure; do not skip or reorder
a gate.

1. Archive the verified change and synchronize its delta specs into the main
   capability specs:

   ```sh
   openspec archive stabilize-enforce-mvp --yes
   openspec validate --all --strict
   ```

   Review and commit the archive result on the release branch. The archive
   commit must include the implementation, documentation, generated assets,
   licensing material, and synchronized capability specs.

2. Merge the reviewed release branch into `main` through the repository's
   normal protected-branch process. Update the local branch without creating a
   merge commit, record the exact release commit, and require a clean worktree.
   Keep the same shell session open for all remaining commands so the read-only
   `release_commit` value cannot be replaced accidentally:

   ```sh
   set -eu
   git switch main
   git pull --ff-only origin main
   readonly release_commit=$(git rev-parse HEAD)
   test -n "$release_commit"
   test -z "$(git status --porcelain=v1)"
   ```

3. On that exact commit, wait for successful `Lint` and `Tests` GitHub Actions
   workflow runs. Confirm these six required checks are present and green:

   - `Module tidiness`
   - `Lint (CGO)`
   - `Lint (no CGO)`
   - `Tests (CGO)`
   - `Tests (no CGO)`
   - `Race (CGO)`

   Also confirm the default-branch `Coverage (CGO)` job succeeded and the
   Codecov upload is visible. The workflow run commit SHA must equal
   `$release_commit`; a green run from another commit is not evidence for this
   release.

4. Without changing files or commits, rerun the release gates locally on that
   exact commit:

   ```sh
   openspec validate --all --strict
   make precommit-all
   make release-bench-check
   test "$(git rev-parse HEAD)" = "$release_commit"
   test -z "$(git status --porcelain=v1)"
   ```

5. Check that `v1.0.0` is absent both locally and from `origin`. Capturing the
   remote result makes a network or authentication error fail the command while
   accepting a successful empty result:

   ```sh
   test -z "$(git tag --list v1.0.0)"
   remote_tag=$(git ls-remote --tags origin refs/tags/v1.0.0)
   test -z "$remote_tag"
   ```

   If the remote result is non-empty, stop: `v1.0.0` is already published and
   must not be moved or deleted.

6. Only when the local and remote checks show no tag, create an annotated tag
   at the recorded commit and verify where it points:

   ```sh
   git tag -a v1.0.0 "$release_commit" -m "postgres-sqlguard v1.0.0"
   test "$(git rev-list -n 1 v1.0.0)" = "$release_commit"
   git show --no-patch v1.0.0
   ```

7. Push only the annotated tag. Do not use `git push --tags` and do not combine
   the tag push with a branch push:

   ```sh
   git push origin refs/tags/v1.0.0
   git ls-remote --exit-code --tags origin refs/tags/v1.0.0
   ```

The final remote object must resolve to the same release commit whose archive,
workflows, and local gates were reviewed.

## Immutable-tag recovery

If a problem is found after creating the tag but before pushing it, first prove
that the tag is absent from `origin`. Only then discard the local candidate,
fix the release branch, and repeat the entire archive, workflow, and release
gate sequence for the new commit:

```sh
set -eu
remote_tag=$(git ls-remote --tags origin refs/tags/v1.0.0)
test -z "$remote_tag"
git tag -d v1.0.0
```

The first two commands must complete successfully and prove that no remote tag
exists. If the lookup fails or returns a match, do not run the delete command.

Once `v1.0.0` has been pushed, it is immutable. Never force-update, move, or
delete the published local or remote tag. Correct a published release through
a new reviewed change and a higher semantic version such as `v1.0.1`, using
the same verification and publication gates.
