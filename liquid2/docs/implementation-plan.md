# Implementation Plan

## Mission

Build Liquid2 as a SQLite-first personal document repository with a Go API server and Flutter client.

This file is the historical rollout plan for Liquid2. As of the workspace split,
the product lives under `liquid2/`; commands below are product-local unless a
workspace-root command is shown. The implemented codebase is past Wave 9F, while
restore/import remains design-only.

## Checkpoint Rule

Each implementation wave starts only after the relevant plan, architecture, design, and API docs are internally consistent for that wave. Scope changes discovered during implementation must update the docs in the same wave.

## Wave 0: Repository Bootstrap

Goal: create the buildable workspace skeleton.

Tasks:

- initialize git repository
- add Go module
- add Flutter app
- add basic Makefile or task runner
- add lint/test commands
- add lightweight file-size audit command for source files
- add initial CI only after local commands are stable

Done criteria:

- `go test ./...` runs
- Flutter analyzer runs
- README points to current docs
- source file size audit exists and excludes generated files

## Wave 1: Storage Foundation

Goal: implement SQLite schema and typed repository access.

Tasks:

- add SQLite connection lifecycle
- add migrations
- add sqlc config
- implement document/folder/tag/feed/blob/job tables
- implement document note table
- add foreign keys, uniqueness constraints, and indexes for documented invariants
- add migration version bookkeeping
- add job lifecycle status constraints and repository tests
- add transaction helper
- add repository tests

Done criteria:

- migrations apply from empty DB
- document CRUD repository tests pass
- document note repository tests pass
- folder tree tests pass
- tag assignment tests pass
- folder/tag/feed item uniqueness and de-duplication constraints are tested
- job status transitions and startup recovery policy are covered by repository tests
- 1MB blob limit is enforced

## Wave 2: API Contract

Goal: expose the first REST API surface with generated OpenAPI contract.

Tasks:

- integrate OpenAPI generation/validation with the existing `chi` transport boundary
- add health endpoint
- add document list/detail/update endpoints
- add document note/comment endpoints
- add folder and tag endpoints
- add consistent error response shape
- generate OpenAPI spec
- keep `docs/api.md` aligned with the OpenAPI route surface

Wave 2 may use an in-memory application service to exercise the API contract,
but that state is a contract fixture only. Do not introduce DB + memory
dual-write synchronization. Before the API is treated as persistent behavior,
wire application services to the SQLite repositories and make SQLite the single
source of truth.

Done criteria:

- OpenAPI JSON is generated
- API tests cover success and validation failures
- generated Dart client can be produced from the spec
- Wave 2 sections in `docs/api.md` match the generated route inventory

## Wave 3: Flutter Shell

Goal: build the first usable document library UI.

Tasks:

- add generated API client
- add Riverpod providers/repositories
- add app routing
- add document list
- add document detail
- add document note/comment creation
- add folder navigation
- add tag/rating/read controls

Done criteria:

- user can browse documents
- user can create a note/comment on a document
- user can mark read/unread
- user can set or clear rating
- user can filter by folder and tag

Wave 3 uses the generated `client/api` Dart package and a repository/provider
layer in the Flutter app. The app talks to `LIQUID2_API_BASE_URL`, defaulting to
`http://localhost:8080` for desktop/web local development. Android emulator runs
should pass `http://10.0.2.2:8080`, while physical devices need the API host's
LAN URL. Fresh API servers may return an empty document list until ingestion or
repository-backed seed data exists; the UI treats that as a normal empty library
state. For local UI review before ingestion lands, run the API with
`LIQUID2_SEED_DEMO=1`.

## Wave 4: Ingestion

Goal: support URL and file inputs.

Tasks:

- bookmark URL endpoint
- scrape URL endpoint with SSRF guard
- upload endpoint with size/MIME checks
- content extraction hooks
- synchronous ingestion boundary; jobs remain available for later async scrape/extract work
- Flutter ingest screen

Done criteria:

- URL bookmark creates document
- URL scrape creates document content
- upload stores blob and document metadata
- unsafe URLs are rejected

Wave 4 may still run against the in-memory application state. Treat that as an
MVP execution detail, not as the persistence boundary.

## Wave 5: Persistence Boundary

Goal: make SQLite the application source of truth through a replaceable
repository boundary before adding RSS/jobs.

### Wave 5A: Repository Port And Memory Adapter

Tasks:

- define repository ports around app use cases, not generic CRUD
- keep domain rules in `app.Service`
- move current `serviceState` maps behind a memory repository adapter
- preserve actor serialization only where it is still part of the memory adapter
- keep demo seed and app tests passing against the memory adapter

Done criteria:

- app service depends on repository interfaces rather than direct state maps
- memory adapter has parity with current document/folder/tag/note/blob behavior
- generated API and Flutter behavior do not change

### Wave 5B: SQLite Repository Adapter

Tasks:

- implement SQLite repository adapter with transaction boundaries
- wire `cmd/api` to open, migrate, and close SQLite
- add configuration for DB path and in-memory test DBs
- map SQLite constraint errors into app-level errors
- move demo seed to repository-backed writes

Done criteria:

- API writes survive process restart
- SQLite adapter tests cover document, note, folder, tag, blob, and ingestion writes
- no dual-write path exists between memory and SQLite

### Wave 5C: Persistence Integration Verification

Tasks:

- add end-to-end tests for persisted bookmark, scrape, upload, notes, folders, and tags
- add a local smoke path for starting API with a real SQLite file
- update operator docs for DB path, migration, and reset behavior

Done criteria:

- `make check` covers memory and SQLite repository paths where practical
- manual smoke can create data, restart API, and read the same data back
- repository boundary docs explain how future PostgreSQL or test adapters plug in

## Wave 6: RSS And Background Work

Goal: add RSS registration and scheduled polling.

Execution rules:

- split Wave 6 into reviewable PRs instead of one large branch
- keep job/feed domain rules in the app layer; SQL, HTTP, worker, and Flutter code own only their concerns
- keep job persistence behind consumer-owned ports so SQLite is replaceable later
- add explicit `Close`/shutdown paths for runners, schedulers, repositories, and API wiring
- recover child worker panic at the worker shell and return a typed failure to the parent runner
- represent multi-stage work as pipeline stages with explicit input/output, idempotency, and retry boundaries
- wrap/log internal errors with detail while returning stable, sanitized API errors

### Wave 6A: Feed And Job Application Boundary

Tasks:

- add app models and repository methods for feeds, feed items, and jobs
- implement memory and SQLite adapters for feed CRUD, feed item de-duplication, and job list/detail/status transitions
- expose feed CRUD and job read endpoints through HTTP/OpenAPI
- regenerate and commit the Dart API client if the OpenAPI surface changes

Done criteria:

- feed CRUD persists through SQLite and has memory adapter parity
- duplicate feed URLs and duplicate feed items are mapped to app-level conflicts or idempotent outcomes intentionally
- job list/detail endpoints expose status without starting worker execution
- backend, OpenAPI, generated client, and focused Flutter repository tests pass

### Wave 6B: Supervised Job Runner Shell

Tasks:

- add a parent-supervised runner for in-process jobs
- convert child panic into a typed failure event returned to the parent runner
- add runner shutdown and startup recovery behavior
- define pipeline stage interfaces and retry/idempotency contracts
- add structured logs for job lifecycle success, retry, and failure

Done criteria:

- child panic does not crash the API process
- parent runner records retry, restart, or failure decisions explicitly
- running jobs are recovered by an explicit startup policy
- runner tests cover panic, cancellation, shutdown, retry, and failure visibility

### Wave 6C: RSS Refresh Pipeline

Tasks:

- add RSS fetch/parse ports and a first adapter
- implement feed refresh as a `poll_feed` pipeline
- create unread documents for new feed items
- preserve item de-duplication by GUID, canonical URL, URL, or content hash according to available data
- add the feed refresh endpoint and connect it to job creation/execution

Done criteria:

- manual feed refresh imports new items once
- repeated refreshes do not create duplicate documents
- failed feed checks are visible through job status and logs
- pipeline stages remain small modules rather than one large worker function

### Wave 6D: Scheduler And Flutter Feed Management

Execution is split for review size:

- Wave 6D-a: backend in-process feed scheduler, persisted scheduler settings,
  shutdown behavior, and operator docs.
- Wave 6D-b: Flutter feed management and feed/job status UI.

Tasks:

- add in-process scheduled polling for enabled feeds
- add persisted app settings for polling interval and scheduler enablement
- add Flutter feed management screen
- surface feed refresh status and failed checks in the UI

Done criteria:

- enabled feeds are polled periodically when RSS auto refresh is enabled
- API shutdown stops scheduler and in-flight workers cleanly
- user can create, edit, disable, delete, and manually refresh feeds from Flutter
- UI can show the latest feed/job status without blocking document-library workflows

## Wave 7: Search And Views

Goal: make the library findable.

Execution rules:

- split Wave 7 into reviewable PRs: app/repository boundary, SQLite FTS5,
  API/generated client, then Flutter UI
- implement search through the app-owned repository read boundary
  `RepositoryReader.ListDocumentIDs(DocumentFilters)`, not a standalone
  `SearchIndex` port
- keep SQLite FTS5 and query details inside the SQLite adapter with dedicated,
  grep-friendly SQL query files
- treat views as fixed filter presets; do not add saved-view persistence or a
  `/views` resource in Wave 7

Tasks:

- add SQLite FTS5 index
- index document title and content variants
- add structured filters
- add unread/rated/recent/folder/tag views
- add Flutter search UI

Done criteria:

- text search works
- filters compose with search query
- unread and rating views work

## Wave 8: Translation

Goal: support translate-after-scrape and document translation as jobs.

Execution is split for review size:

- Wave 8A: content variant model, source-content linkage, and app service
  append behavior.
- Wave 8B: translation provider boundary and `translate_document` worker
  pipeline.
- Wave 8C: HTTP/OpenAPI/generated Dart client contract.
- Wave 8D: Flutter translation action and job status UI.

Tasks:

- define translation provider interface
- add translation job
- store translated content variant
- expose job status
- add Flutter action and status display

Done criteria:

- translation never overwrites original content
- translated content is searchable
- failed translation keeps original document intact

## Wave 9: Export, Backup, And History

Goal: protect user data.

Tasks:

- land docs and artifact-format decisions first
- add document history schema and app-owned write-on-mutation rules
- add SQLite backup operator command
- add SQLite backup API
- add app-level markdown export with manifest
- add export API
- add restore/import design before implementation

Done criteria:

- backup file can be created
- backup API exposes artifact metadata without local filesystem paths
- export includes documents, blobs, and manifest
- title/content mutation creates history entry
- metadata-only changes do not create document history entries
- restore/import behavior is documented but not implemented in Wave 9

## Verification Commands

Workspace root:

```bash
make -C liquid2 check
```

Liquid2 product root:

```bash
make check
make backend-test
make frontend-analyze
make frontend-test
```

## Approval Gates

- Gate 1: approve architecture/design docs
- Gate 2: approve OpenAPI integration spike result
- Gate 3: approve first schema migration before UI work depends on it
- Gate 4: approve ingestion SSRF/upload policy before enabling scraping/upload
- Gate 5: review package/file layout after each wave; files over 200 lines require an explicit reason or split plan

## Risks

- URL scraping introduces SSRF and unsafe HTML display risks.
- Generated API client can become noisy if OpenAPI shape is unstable.
- SQLite schema decisions become costly after real data accumulates.
- Translation providers can create long-running job failure modes.
- Search adapter will differ between SQLite and PostgreSQL.
- In-process RSS polling keeps deployment simple, but later separation into a
  worker process or command-driven scheduler must preserve the same job queue
  and runner contracts.
- Small-file discipline can create needless fragmentation if applied mechanically; ownership and replaceability outrank raw line count.

## Deferred Decisions

- per-domain extraction rules, if concrete examples justify them
- whether custom saved views are needed in v1
- restore/import implementation scope and operator workflow
