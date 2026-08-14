# Liquid2

Liquid2 is a personal source vault for collecting, reading, organizing, and
finding reference material again.

It is built as a local-first document library: a Go API server, a Flutter
client, and SQLite storage. Filesystem paths are used for import, export, and
backup, but the product state lives in the application database.

## What You Can Use It For

- Save URLs and scraped web pages.
- Read and organize documents with folders, tags, ratings, and read state.
- Subscribe to RSS feeds and collect items into the library.
- Store notes and comments alongside documents.
- Create translated content variants without overwriting the original.
- Search and filter the library by common reading workflows.
- Export and back up personal data.

Liquid2 is the source vault. Plasma, the sibling product in this repository,
may read selected Liquid2 material through an API/connector boundary, but Plasma
does not store its research state in Liquid2.

## Product Shape

- Backend: Go
- Frontend: Flutter
- Storage: SQLite first
- API: REST with committed OpenAPI artifacts
- Database access: raw SQL plus sqlc
- Runtime shape: modular monolith with explicit boundaries for ingestion, RSS,
  translation, jobs, export, and future service extraction

## Quick Start For Development

From the workspace root, run both product development services when Plasma also
needs Liquid2:

```sh
./dev-browser.sh start
./dev-browser.sh status
./dev-browser.sh stop
```

The canonical Liquid2 development flow uses the browser script:

```sh
./dev-browser.sh liquid2 start
./dev-browser.sh liquid2 status
./dev-browser.sh liquid2 logs
./dev-browser.sh liquid2 stop
```

It uses Flutter web port `6001`, API port `6011`, and a local SQLite database
under `~/research-artifacts/liquid2/liquid2/runtime/dev-6011/`. Open
`http://127.0.0.1:6001/` for the Flutter web client. Runtime settings can be
moved from environment variables into TOML files; see the workspace
[configuration guide](../docs/configuration.md).

Run the local release surface:

```sh
./release-browser.sh liquid2 start
./release-browser.sh liquid2 status
./release-browser.sh liquid2 logs
./release-browser.sh liquid2 stop
```

Liquid2 release defaults to Flutter web port `3001` and API port `3011`. The
default database, export directory, and backup directory live under
`~/Library/Application Support/Liquid2` on macOS and
`${XDG_DATA_HOME:-$HOME/.local/share}/liquid2` on WSL2.

## Common Commands

Run all checks:

```sh
make -C liquid2 check
```

Work from the product directory:

```sh
cd liquid2
make check
make backend-test
make frontend-analyze
make frontend-test
make openapi
make openapi-client
make persistence-smoke
```

For a standalone API flow, use port `8080` explicitly:

```sh
cd liquid2
LIQUID2_ADDR=127.0.0.1:8080 LIQUID2_DB_PATH=./liquid2.db LIQUID2_JOBS_ENABLED=1 go run ./cmd/api
```

This standalone flow is separate from the canonical `./dev-browser.sh liquid2`
development flow, which uses API port `6011` and Flutter web port `6001`.

Run the Flutter client against a local API:

```sh
cd liquid2/client
flutter run -d macos --dart-define=LIQUID2_API_BASE_URL=http://localhost:8080
```

## Core Concepts

- Document: the item a user saves and organizes.
- Content variant: original scraped text, uploaded content, translation, or
  other stored body associated with a document.
- Feed: an RSS source that can create or update documents through jobs.
- Job: durable background work such as RSS refresh or translation.
- Export: portable output assembled through the app layer, not by scraping
  SQLite internals.

## Documentation

- [Architecture](docs/architecture.md)
- [Design](docs/design.md)
- [API](docs/api.md)
- [Implementation Plan](docs/implementation-plan.md)

## Release Notes

Public releases use Conventional Commit titles. Squash merge titles should
follow this shape:

```text
feat: add document folder move action
fix: handle duplicate translation jobs
ci: update macOS release workflow
```

A public release is a source-only snapshot selected from an internal source
commit. After the public snapshot is reviewed and pushed, an annotated `vX.Y.Z`
tag is created and a GitHub pre-release publishes the release notes. The GitHub
Release is the tag-and-notes surface; binary application assets are not attached
or part of the installation contract.
