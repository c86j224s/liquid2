# Liquid Workspace

This repository is a product workspace. Product code is intentionally split at
the repository root so each product can keep its own domain model, storage, and
release surface.

## Products

- `liquid2/` - personal document/source vault with the existing Go API,
  Flutter client, SQLite storage, OpenAPI contract, scripts, and product docs.
- `plasma/` - steerable research workspace with its own Go runtime, SQLite
  storage, Mission Ledger, source snapshots, research records, and connector
  boundary work.

## Boundaries

Liquid2 and Plasma are separate products. Plasma may consume Liquid2 through a
connector or API contract, but it must not import Liquid2 internals, read
Liquid2's SQLite database directly, or store Plasma research state in Liquid2
tables.

Future shared areas such as `infra/`, `shared/`, or root-level docs should be
added only when ownership is explicit and they are not product implementation
dumping grounds.

## Repository Operations

GitHub milestone, issue, PR, and branch workflow guidance lives in
[`docs/github-workflow.md`](docs/github-workflow.md).
Runtime configuration guidance lives in
[`docs/configuration.md`](docs/configuration.md).
Third-party generated and vendored code notices live in
[`THIRD_PARTY_NOTICES.md`](THIRD_PARTY_NOTICES.md).

Public-facing project governance:

- [`SECURITY.md`](SECURITY.md) explains vulnerability reporting and local data
  boundaries.
- [`CONTRIBUTING.md`](CONTRIBUTING.md) explains the current contribution intake
  status.
- [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md) sets the conduct baseline.
- [`LICENSE`](LICENSE) applies the MIT License to this repository.
- Do not make the existing private repository public until Git history, tags,
  remote branches, and GitHub issue/PR/comment metadata have a settled
  publication or sanitization plan.
- Do not make a public copy available until a private security/contact route is
  configured for reports that should not be filed as public issues.

## Liquid2 Development

Run Liquid2 checks from the workspace root with:

```sh
make -C liquid2 check
```

Or work inside the product directory:

```sh
cd liquid2
make check
```

Liquid2 release automation remains root-owned for now through
`.github/workflows/release.yml` and `.releaserc.json`.

## Development Servers

Use the root development server controller when testing Liquid2 and Plasma
together:

```sh
./dev-browser.sh start
./dev-browser.sh status
./dev-browser.sh stop
```

The root script starts Liquid2 before Plasma and stops Plasma before Liquid2.
By default, Liquid2 uses Flutter web port `6001` plus API port `6011`, and
Plasma uses browser port `6002`. Product-specific control is also available:

```sh
./dev-browser.sh liquid2 restart
./dev-browser.sh plasma logs
```

Use the release controller for the local release stack:

```sh
./release-browser.sh start
./release-browser.sh status
./release-browser.sh stop
```

By default, Liquid2 release uses Flutter web port `3001` plus API port `3011`,
and Plasma release uses browser/API port `3002`.

## Plasma Status

Plasma runtime work is under active development on the Plasma product branch.
It remains a separate product: Plasma may read Liquid2 through a connector/API
contract, but it must not import Liquid2 internals or read Liquid2 SQLite
tables directly.
