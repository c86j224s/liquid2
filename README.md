# Liquid Workspace

[![CI](https://github.com/c86j224s/liquid2/actions/workflows/ci.yml/badge.svg)](https://github.com/c86j224s/liquid2/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
![Status](https://img.shields.io/badge/status-pre--1.0-orange)

Liquid Workspace is a local-first workspace for saving personal source material,
reading it again, and turning selected evidence into research reports.

The repository contains two companion products:

- **Liquid2** is a personal document and source vault. It stores saved pages,
  files, feeds, notes, tags, folders, translations, exports, and backups.
- **Plasma** is a steerable research workspace. It organizes missions, accepted
  sources, agent-assisted investigation, source snapshots, and report artifacts.

Both products are built for local use first. Product state lives in local SQLite
databases, runtime configuration stays outside the repository, and committed
docs summarize experiments without committing raw run artifacts.

This repository is under active development. Interfaces, workflows,
configuration, and local data shapes may change without a migration path until a
stable release line exists.

## What Is Here

| Area | Purpose |
| --- | --- |
| `liquid2/` | Go API, Flutter client, SQLite storage, OpenAPI contract, and Liquid2 product docs. |
| `plasma/` | Go browser/API server, research mission state, source connectors, report generation, and Plasma product docs. |
| `docs/` | Workspace-level configuration and GitHub operating rules. |
| `.github/` | Manual GitHub workflows, issue template, and PR template. |
| `dev-browser.sh` | Starts and manages the local development stack. |
| `release-browser.sh` | Starts and manages the local release-style stack. |

## Quick Start

Read [SETUP.md](SETUP.md) for the full agent/operator setup path. The short
version for a prepared macOS environment is:

```sh
git clone https://github.com/c86j224s/liquid2.git
cd liquid2
./dev-browser.sh start
./dev-browser.sh status
```

Development ports use the `6000` range:

- Liquid2 web: `http://127.0.0.1:6001`
- Liquid2 API: `http://127.0.0.1:6011`
- Plasma: `http://127.0.0.1:6002`

Local release-style servers use the `3000` range:

```sh
./release-browser.sh start
./release-browser.sh status
```

## Common Checks

```sh
make -C liquid2 check
make -C plasma check
```

The GitHub workflows in this repository are intentionally manual for now. Local
build and verification remain the primary path while the project is pre-1.0.

## Product Docs

- [Liquid2 README](liquid2/README.md)
- [Liquid2 Architecture](liquid2/docs/architecture.md)
- [Liquid2 Design](liquid2/docs/design.md)
- [Plasma README](plasma/README.md)
- [Plasma Product Flow](plasma/docs/product-flow.md)
- [Plasma Architecture](plasma/docs/product-architecture.md)
- [Configuration](docs/configuration.md)

## Project Status

This is an early public snapshot. The code is useful for local development and
inspection, but the project is still pre-1.0 and not yet stable:

- external contributions are not being accepted yet;
- macOS/local development is the primary supported environment;
- APIs, UI flows, configuration, and storage details may change frequently;
- release automation is manual;
- public issue and security triage are still intentionally lightweight.

## Governance

- [LICENSE](LICENSE): MIT License.
- [SECURITY.md](SECURITY.md): vulnerability reporting and local data boundary.
- [CONTRIBUTING.md](CONTRIBUTING.md): current contribution intake policy.
- [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md): conduct expectations.
- [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md): generated and vendored code notices.
