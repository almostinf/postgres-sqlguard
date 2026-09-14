## Context

See [proposal.md](proposal.md) for the release motivation. The root module is
already the stable driver-independent API, while built-in rules currently live
at `/rules` and official sinks at `/observability/{prometheus,slog}`. The
project has no `v1` tag, so this is the last practical point to make an
intentional import-path break without maintaining forwarding packages.

The README currently carries detailed API contracts, examples, backend
measurements, and contributor instructions. Much of that information is useful
but obscures the first-use path. Existing engine and parser benchmarks already
cover representative complexity and both parser backends, and
`docs/parser-backends.md` records an initial reproducible measurement set.

CI currently reports two compound precommit jobs. This makes lint, ordinary
tests, race tests, and backend-specific failures indistinguishable in badges
and branch protection. The Make targets remain the canonical local interface
and must not lose coverage when CI is decomposed.

The repository is Apache-2.0 licensed and carries a reviewed
`THIRD_PARTY_NOTICES.md`. Apache-2.0 and MIT are permissive, but Apache-2.0 also
provides an explicit patent grant and patent-litigation termination terms. Its
redistribution and NOTICE rules are more detailed, while MIT is shorter and
requires retention of its copyright and permission notice. Primary references
are the [Apache-2.0 text](https://www.apache.org/licenses/LICENSE-2.0) and the
[OSI MIT text](https://opensource.org/license/mit).

The requested release art also crosses third-party identity boundaries. The Go
gopher is a Renée French work licensed under CC BY 4.0 and requires attribution
when adapted. PostgreSQL's current trademark policy says the official Slonik
mark must not be modified and generally must not be combined with other marks
without permission. Release art therefore cannot casually edit the official
elephant logo.

## Goals / Non-Goals

**Goals:**

- Establish one unambiguous public package layout before `v1.0.0`.
- Make the README useful in order: understand the value, install, run a safe
  example, choose a parser backend, then follow links for depth.
- Publish attractive benchmark summaries whose raw evidence, environment, and
  transformations can be reproduced and audited.
- Expose lint, test, no-CGO, race, and coverage health without weakening the
  local precommit contract.
- Resolve licensing and artwork provenance before the release tag.
- Make the published tag immutable and traceable to a verified main-branch
  commit containing the archived OpenSpec change.

**Non-Goals:**

- Redesigning root engine, rule, statement, error, prepared-validation, or
  observability semantics.
- Adding compatibility shims for pre-v1 package paths.
- Turning performance measurements into universal latency promises or CI
  regression thresholds.
- Building a documentation website, release automation platform, or official
  database-driver adapter.
- Claiming that SQLGuard prevents every SQL injection or replaces PostgreSQL
  authorization, transactions, constraints, or application review.
- Republishing, modifying, or implying endorsement by the official PostgreSQL
  Slonik trademark.

## Decisions

### 1. Move extension packages physically and remove the old paths

Move `rules/` to `pkg/rules/` and `observability/` to
`pkg/observability/`, preserving the package names `rules`, `prometheus`, and
`slog`. Update all repository imports, examples, GoDoc links, documentation,
and tests in the same change. Do not leave deprecated forwarding packages,
type aliases, or empty directories at the old locations.

Encode the required and forbidden package paths in the capability specs and
verify the final module enumeration in both build modes during release
verification. Existing behavioral and concurrency tests move with the packages
and continue to test identical rule and sink behavior. Package enumeration,
rather than a source search, supplies evidence for the path requirements.

Alternatives considered:

- Compatibility wrappers reduce migration friction but create two supported
  paths on the first stable release and contradict the agreed clean break.
- Keeping extension packages at the module root is idiomatic in many Go
  projects, but does not implement the selected public layout.
- Moving all public APIs below `pkg/` would create a larger root API break with
  no release-readiness benefit; the engine remains at the module root.

### 2. Use a concise README backed by durable topic and package documentation

The README will follow a benefit-first structure inspired by the scannability
of the referenced goose and golem projects without copying their prose:

1. release illustration, project name, one-sentence promise, and badges;
2. a short list of differentiators: real PostgreSQL grammar, fail-closed full
   traversal, opt-in composable policies, driver independence, privacy-safe
   typed failures/observability, and automatic CGO/no-CGO selection;
3. installation and one compact quick-start example using `pkg/rules`;
4. a small built-in policy table and an explicit statement of what SQLGuard
   does not do;
5. the CGO versus no-CGO decision summary and benchmark graphics;
6. links to deeper usage, parser-backend, observability, integration,
   contribution, license, and third-party-notice documentation.

Long-form material moves rather than disappears. Parser details and complete
evidence remain in `docs/parser-backends.md`; reusable API guidance belongs in
package GoDoc where it is discoverable on pkg.go.dev; cross-package guides live
under `docs/`. README examples must compile in tests or be sourced from tested
examples so the marketing surface cannot silently drift from the API.

Alternatives considered:

- Keeping a comprehensive single README preserves one search surface but makes
  onboarding slow and mixes user, maintainer, and evidence audiences.
- Launching a documentation site offers richer navigation but adds hosting,
  versioning, and publishing scope that is unnecessary for the first release.

### 3. Generate benchmark charts from checked-in, machine-readable evidence

Use the existing benchmark cases as the source of truth. Collect CGO and
no-CGO results with the same Go version, host, workload, flags, warm-up, run
count, and benchmark names. Record at least five runs and report medians. Store
raw command output plus a small machine-readable release dataset and metadata
under a dated `docs/benchmarks/` directory.

A repository-local generator will read that dataset and emit deterministic SVG
assets under `docs/assets/benchmarks/`. Prefer a small Go standard-library tool
or script so chart regeneration does not introduce a production dependency or
require a Python plotting environment. The generated assets are committed so
GitHub renders the README without executing tooling.

The README will show four compact comparisons:

- direct validation latency (`ns/op`) for simple, medium, multi-statement, and
  nested-CTE inputs;
- heap bytes allocated per operation (`B/op`) for the same complexity levels;
- cold-process maximum resident memory for the fixed size-probe workload;
- stripped linked executable size for the same size-probe program.

Use grouped bars and explicit units, keep CGO/no-CGO colors consistent, include
the measurement date and host near the charts, and link to methodology and raw
results. Use a logarithmic scale only when necessary and label it prominently.
CPU means benchmark execution time, not sampled host utilization. Allocation
count and parallel/cold-start details remain in the evidence document when they
would overload the landing page. No chart compares against an invented
"without SQLGuard" baseline or implies database round-trip performance.

Alternatives considered:

- Hand-edited charts are visually flexible but cannot prove correspondence to
  recorded measurements.
- Generating charts in every CI run creates noisy artifacts and machine-driven
  drift. CI will instead verify that regeneration produces no diff.
- A hosted benchmark dashboard adds an external service and historical model
  beyond the stable-release need.

### 4. Split CI by signal and use one workflow badge per signal

Replace the compound precommit workflow with separately badgeable lint and test
workflows:

- `lint.yml` runs module-tidiness verification and lint for CGO and no-CGO
  configurations as independently named jobs.
- `test.yml` runs ordinary tests for both configurations, a CGO race job, and
  the coverage-producing CGO test path.

The workflows call Make targets instead of reconstructing tool commands,
preserve the unavailable-compiler sentinel for no-CGO, use least-privilege
permissions, cancellation, timeouts, and immutable action revisions where
practical. `make precommit`, `make precommit-all`, and their component targets
remain authoritative locally; decomposition changes reporting, not the gate.

The README displays distinct `Lint` and `Tests` workflow badges for `main`.
Because GitHub badges describe workflows rather than individual jobs, separate
workflow files are required for honest badges. Branch-protection required-check
names must be migrated only after the new workflows have completed successfully
on the default branch.

Alternatives considered:

- One workflow with multiple jobs improves Actions diagnostics but cannot
  provide stable per-signal workflow badges.
- Four backend-by-signal workflow files maximize badge granularity but duplicate
  setup and clutter the README.

### 5. Publish CGO test coverage through Codecov using OIDC

The CGO test job produces an atomic Go coverage profile for all packages and
uploads it to Codecov. Use Codecov's OIDC mode with job-scoped
`id-token: write`, keep all other permissions read-only, pin the uploader action
to a reviewed immutable revision, and fail the coverage job if generation or
upload fails. Fork pull requests must remain testable even when trusted upload
credentials are unavailable; coverage publication is authoritative from the
default branch.

Add a Codecov badge scoped to `github.com/almostinf/postgres-sqlguard` and link
it to the coverage report. Do not display a hard-coded percentage. no-CGO tests
remain a required compatibility job, but their substantially overlapping source
profile is not merged into the headline number.

Alternatives considered:

- A locally generated static badge becomes stale and is easy to misrepresent.
- A repository secret works but complicates fork behavior and secret rotation;
  OIDC provides short-lived identity for default-branch uploads.
- Uploading both backends adds duplicate coverage data without improving the
  primary source-coverage signal.

### 6. Retain Apache-2.0 for `v1.0.0`

Keep the current Apache-2.0 project license. Its explicit patent license and
contribution terms are valuable for a reusable infrastructure library; changing
to MIT would mainly shorten compliance text while discarding that explicit
patent framework. Dual licensing would create ambiguity without a demonstrated
consumer need.

Retain the canonical full text in `LICENSE`, identify it as `Apache-2.0` in the
README badge/link, and audit `THIRD_PARTY_NOTICES.md` against the final
dependency and embedded-artifact closure. Do not add blanket license headers to
every existing Go file: the top-level license file applies to the distributed
work, and mass headers would add churn without changing the selected terms.
Only add a project `NOTICE` or file-specific attribution where the final audit
identifies content that requires it. This is an engineering compliance decision,
not legal advice; uncertainty about ownership or relicensing authority must be
escalated before release.

Alternatives considered:

- MIT is shorter and widely recognized but offers no clear adoption advantage
  here that outweighs Apache-2.0's explicit patent terms.
- Apache-2.0-or-MIT dual licensing increases choice but complicates provenance
  and requires confidence that all project copyright holders authorized both.

### 7. Create original release art with explicit provenance

Generate a compact, readable-at-small-size illustration of a sword-bearing Go
gopher guarding an original blue elephant/database character. The elephant
must not reproduce, trace, distort, or combine the official Slonik mark; the
composition must not imply PostgreSQL project sponsorship or endorsement. If
the exact official Slonik is later required, publication is blocked until the
trademark owner grants permission for that composition.

Commit the optimized raster output under `docs/assets/`, plus a nearby
provenance file recording the generation prompt, date, tool, edits, and
attributions. Credit the Go gopher design to Renée French under CC BY 4.0 and
include an appropriate no-affiliation/trademark note for PostgreSQL. Keep the
mascot compact and decorative and provide concise alt text in the README.

Alternatives considered:

- Modifying the official PostgreSQL elephant is closer to the initial visual
  request but conflicts with the published trademark constraints without
  permission.
- A generic shield/database icon avoids all mascot issues but loses the desired
  Go/PostgreSQL personality.

### 8. Treat `v1.0.0` as an irreversible release gate

Prepare release notes that state the supported Go/PostgreSQL grammar matrix,
new stable import paths, backend trade-offs, and known non-goals. The release
candidate commit must contain the completed implementation, generated assets,
license audit, documentation, and archived OpenSpec capability updates.

After merge to `main`, require the new lint and test workflows to pass, run
strict OpenSpec validation and `make precommit-all` on the exact commit, then
create an annotated `v1.0.0` tag at that commit and push only that tag. Do not
move or replace a published `v1.0.0`; release corrections as a higher semantic
version. A GitHub Release and automated module publishing are outside scope
unless requested separately.

Alternatives considered:

- Tagging the implementation before archiving OpenSpec omits the final source
  of truth from the released commit.
- A lightweight tag carries less release context and is easier to create
  accidentally; an annotated tag makes the release intent explicit.

## Risks / Trade-offs

- **[Pre-v1 consumers break after the package move]** → Document exact old/new
  import mappings, update every repository example, and verify package
  enumeration against both presence and absence contracts before release.
- **[README becomes promotional at the expense of safety accuracy]** → Keep
  claims traceable to specs/tests, retain a visible limitations block, and link
  to detailed contracts rather than deleting them.
- **[Benchmark charts overstate portable performance]** → Publish raw results,
  environment metadata, reproducible commands, medians, explicit units, and a
  statement that results are evidence from one machine rather than guarantees.
- **[Generated charts drift from recorded data]** → Make generation
  deterministic and verify a clean regeneration diff.
- **[Split CI accidentally drops a precommit check]** → Map every existing Make
  gate to a named job and run `make precommit-all` as the final local release
  gate.
- **[Coverage upload introduces external availability or identity failures]** →
  Isolate upload permissions to the coverage job, pin the action, and keep local
  test success independently reproducible.
- **[Apache redistribution obligations or third-party notices are incomplete]**
  → Review the final source and embedded dependency closure and block tagging on
  unresolved attribution.
- **[Mascot art infringes or implies endorsement]** → Avoid the official Slonik,
  record provenance, credit the Go gopher, add a no-affiliation note, and seek
  permission before any exact-logo composition.
- **[A bad public tag cannot be safely rolled back]** → Tag only a green,
  archived main commit; never retarget a published version and use a patch
  release for corrections.

## Migration Plan

1. Move the extension packages and update all imports, tests, examples, GoDoc,
   docs, and capability contracts in one commit-sized unit.
2. Restructure documentation and add tested quick-start snippets.
3. Capture the release benchmark dataset, generate committed charts, and verify
   deterministic regeneration.
4. Replace compound CI with lint/test workflows, add OIDC coverage publication,
   and update README badges after the new default-branch checks exist.
5. Complete the Apache-2.0 and third-party-notice audit; generate and review the
   attributed release art.
6. Run OpenSpec verification, strict validation, `make precommit-all`, and all
   additional release checks. Resolve every warning before archive.
7. Archive the change so the main capability specs contain the new package
   paths, merge the archive commit to `main`, and confirm all required workflows
   on that exact commit.
8. Create and push the annotated `v1.0.0` tag. If any problem is found before
   pushing, delete only the local tag and fix the candidate. If found after
   publishing, leave `v1.0.0` immutable and issue a patch release.

Rollback before the tag is a normal revert of the package, docs, assets, or CI
changes. There is no compatibility rollback after `v1.0.0`; restoring old
import paths later would be a separately specified API addition.
