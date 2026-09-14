# Licensing and attribution

`postgres-sqlguard` will retain the Apache License 2.0 for `v1.0.0`. This is an
engineering release decision, not legal advice. Organizations with specific
distribution, patent, trademark, or compliance requirements should obtain
their own legal review.

## Why Apache-2.0 remains the project license

Apache-2.0 is a permissive license: consumers may use, modify, and distribute
the library in source or object form, including as part of proprietary
software, provided they comply with its redistribution conditions. In
particular, recipients must receive the license, modified files must carry
prominent change notices, applicable notices must be retained, and any NOTICE
material shipped by the work must be reproduced as described by section 4.

The deciding advantage over MIT for this infrastructure library is the
explicit contributor patent grant in Apache-2.0 section 3, together with its
patent-litigation termination provision. MIT is shorter and also permissive,
but its standard text does not contain an equivalent express patent grant.
Changing or dual-licensing before `v1.0.0` would therefore remove or complicate
a useful protection without a demonstrated compatibility or adoption benefit.

The canonical project license remains [`LICENSE`](../LICENSE). It contains the
complete [standard Apache-2.0 text](https://www.apache.org/licenses/LICENSE-2.0.txt)
published by the Apache Software Foundation. The project does not add blanket
headers to Go files: the repository-level license identifies the terms for
project-authored work, while file-specific notices are reserved for files that
actually require them. The alternative reviewed for this decision was the
[standard MIT text](https://opensource.org/license/mit).

## Dependency and embedded-artifact audit

The 2026-09-14 audit enumerated non-standard-library modules reachable from all
non-test repository packages in both build modes:

```sh
CGO_ENABLED=1 CC=cc go list -deps ./...
CGO_ENABLED=0 CC=/definitely/missing/postgres-sqlguard-cc go list -deps ./...
```

That scope includes the root library, both parser backends, official
observability integrations, and the compiling pgx examples. It excludes
test-only packages, the pinned linter toolchain, GitHub Actions, and other
repository tooling because they are not linked into these packages. Exact
versions and applicable license/NOTICE material are recorded in
[`THIRD_PARTY_NOTICES.md`](../THIRD_PARTY_NOTICES.md).

The no-CGO path incorporates the `go-pgquery` WebAssembly parser artifact; its
PostgreSQL-derived and individually attributed embedded components remain
listed in full. The audit also added the previously omitted Prometheus runtime
closure, the pgx example closure, and `hashicorp/golang-lru/v2`. The latter is
MPL-2.0 and includes a BSD-licensed Go list implementation, so executable-form
distributors must preserve its notices and satisfy MPL-2.0 source-availability
requirements. The repository provides the exact MPL-2.0 text and a link to the
pinned source in its third-party notices. Mozilla also publishes the
[authoritative MPL-2.0 text](https://www.mozilla.org/en-US/MPL/2.0/).

## NOTICE and source-header conclusion

No project-level `NOTICE` file is added for `v1.0.0`. The project itself did not
previously ship one, and Apache-2.0 does not require inventing a NOTICE file.
For Apache-licensed dependencies that do ship NOTICE text, the applicable
attributions are reproduced in `THIRD_PARTY_NOTICES.md`; Apache-2.0 section
4(d) permits those notices to be carried in distributed documentation.

No project source file contains copied third-party implementation requiring a
new file-specific header. Dependency code remains in separately licensed Go
modules, and the checked-in benchmark SVGs are generated from project-owned
data by project-owned tooling. The generated release artwork uses an adapted
Go-gopher-inspired character; its required credit, modification history, and
PostgreSQL trademark/no-affiliation statement are recorded in the [artwork
provenance](assets/sqlguard-mascot.md) and summarized in
`THIRD_PARTY_NOTICES.md`. This attribution does not require a project-level
`NOTICE` file or blanket source headers.

Anyone redistributing a compiled application remains responsible for shipping
the licenses, notices, and source-availability information required by the
specific dependency closure in that application; importing fewer optional
packages may produce a smaller closure than this repository-wide audit.
