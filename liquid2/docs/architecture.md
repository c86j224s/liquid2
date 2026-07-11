# Architecture

## 목표

Liquid2는 개인 문서 저장소다. 사용자는 URL, 업로드 파일, RSS 항목을 하나의 라이브러리로 모으고 폴더, 태그, 검색, 읽음 상태, 평점으로 관리한다.

초기 구조는 modular monolith다. 처음부터 여러 프로세스로 쪼개지 않는다. 대신 문서, 수집, 피드, 검색, 번역, 파일 저장 경계를 명확히 둬서 나중에 MSA로 분리할 수 있게 한다.

이 문서의 경로는 별도 언급이 없으면 workspace의 `liquid2/` 제품
디렉터리를 기준으로 한다. workspace root에서 볼 때 Liquid2 코드는
`liquid2/cmd`, `liquid2/internal`, `liquid2/client`, `liquid2/api` 아래에
있다.

## 아키텍처 원칙

- SQLite가 source of truth다.
- 파일 시스템은 import/export/backup 산출물로만 사용한다.
- 문서 identity는 제목이나 경로가 아니라 `document.id`다.
- 폴더는 주 분류, 태그는 보조/교차 분류다.
- API contract는 OpenAPI로 고정하고 Flutter 클라이언트는 생성된 client를 사용한다.
- backend business logic은 HTTP handler, DB adapter, worker adapter에 직접 묶지 않는다.
- 초기에는 REST를 사용한다. 내부 서비스 통신이 실제로 필요해질 때 ConnectRPC 또는 Kratos를 검토한다.
- source file은 200 lines 안팎을 soft limit로 둔다. 넘기면 역할 분리, private helper file 분리, 또는 abstraction 누락을 검토한다.
- module은 하나의 일을 한다. 하나의 module이 맡은 domain ownership은 온전히 관리하되, 외부 구현은 port/interface 뒤로 숨긴다.

## Module Sizing And Ownership

Go에서 module boundary는 package 단위로 잡는다. 하나의 package는 하나의 domain capability를 소유한다. 예를 들면 `documents` package는 document identity, lifecycle, rating/read state, content variant rule을 소유하고, `feeds` package는 RSS source와 feed item import rule을 소유한다.

크기 기준:

- source file target: 80-180 lines
- source file soft limit: 200 lines
- package target: 3-8 focused files
- package가 10 files를 넘으면 subpackage 또는 boundary 재검토
- generated files, migrations, OpenAPI output, sqlc output은 size rule에서 제외

파일 분리 기준:

```text
model.go        domain types and value validation
service.go      use-case orchestration
ports.go        interfaces required by this package
errors.go       domain errors
*_test.go       behavior tests
```

Adapter는 domain package 안에 섞지 않는다. SQLite, HTTP, worker, external client 구현은 별도 adapter package에 둔다.
Wave 5 repository transition has one explicit exception: the SQLite app
repository adapter may live under `internal/app/sqlite_*` while it maps the
app-owned unexported `documentRecord` aggregate. SQLC imports stay confined to
those adapter/mapping files, and application service methods must not depend on
SQLite types. If the aggregate boundary becomes exported later, move the adapter
back behind a separate storage/app adapter package.

```text
internal/app
internal/storage/sqlite
internal/transport/http
```

Dependency direction:

```text
transport/http -> app services -> domain packages -> ports
storage/sqlite -> domain ports
workers        -> app services
```

Domain package는 SQLite, Huma, chi, Flutter, generated client를 알면 안 된다.

Wave 9 data-protection boundaries:

- document history rules are application rules; SQL persists versions but does
  not decide when to create them.
- SQLite backup belongs to the SQLite storage/operator boundary, not
  `app.Repository`.
- markdown export is app-level and portable; export code reads app snapshots and
  must not import SQLite or sqlc packages.
- HTTP handlers expose artifact IDs and API routes, never local filesystem
  paths.

## Orthogonal Concern Ownership

Domain rule이 가장 중요한 product policy를 소유하지만, 모든 코드가 domain package에 들어가야 한다는 뜻은 아니다. 각 관심사는 자기 영역을 온전히 관리해야 한다.

관심사별 ownership:

- domain package: business invariants, state transition, value validation, use-case rule
- SQL query file: persistence shape, joins, filtering, ordering, index-friendly query form
- repository adapter: domain port와 SQL query 사이의 mapping
- HTTP transport: route, request decoding, protocol validation, response mapping
- Flutter widget: rendering, layout, user interaction binding
- Flutter controller/provider: screen state transition and API orchestration
- repository search read: index-friendly document ID query execution

Traceability rule:

```text
SQL query name
  -> repository method
  -> domain port/interface
  -> application service
  -> transport handler or worker
```

반대로 사용자 흐름에서는 다음 방향으로 추적 가능해야 한다.

```text
route/widget action
  -> application service/controller
  -> domain method
  -> port
  -> adapter/query
```

SQL query file을 뒤져서 persistence behavior를 찾을 수 있어야 한다. 다만 SQL query가 read status transition, rating range policy, folder membership rule 같은 domain policy의 원본이 되면 안 된다. 필요한 DB constraint와 index는 domain invariant의 persistence mirror로 허용한다.

## Replaceability Rule

갈아끼울 가능성이 있는 구현은 interface 뒤에 둔다. 단, 의미 없는 interface는 만들지 않는다. interface는 consumer side에 둔다.

초기부터 port로 분리할 대상:

- `DocumentRepository`
- `FolderRepository`
- `TagRepository`
- `BlobStore`
- `FeedRepository`
- `JobQueue`
- `Scraper`
- `Translator`

Search starts as a repository read method because SQLite FTS rows and document
rows must be read in one transaction. A standalone `SearchIndex` port is a
future extraction point for an external search service or eventually consistent
index.

초기에는 SQLite 구현만 있어도 된다. PostgreSQL, external object storage, alternate search engine, different translation provider는 같은 port를 구현하는 adapter로 추가한다.

## 상위 구성

```text
Flutter app
  -> generated API client
  -> Go API server
       -> HTTP transport
       -> application services
       -> repositories
       -> SQLite
       -> background workers
```

## Backend Stack

초기 선택:

```text
Go
chi on net/http
Huma for OpenAPI generation/validation
sqlc for typed SQL access
SQLite
log/slog for structured logging
```

`chi`는 `net/http` 호환 라우터로 유지보수성과 Go 생태계 호환성이 좋다. Backend transport boundary는 계속 `http.Handler`로 노출해 router 구현이 domain/application layer로 새지 않게 한다.

Huma는 handler registration에서 OpenAPI 3.x spec과 docs를 생성할 수 있다. Flutter 쪽은 OpenAPI Generator의 `dart-dio` client 생성을 기본 후보로 둔다.

## Logging

Backend logging uses Go `log/slog`.

Runtime configuration:

- `LIQUID2_LOG_LEVEL`: `trace`, `debug`, `info`, `warn`, `error`
- `LIQUID2_LOG_FORMAT`: `json` or `text`
- `LIQUID2_LOG_SOURCE`: `1` enables source location
- `LIQUID2_DB_PATH`: SQLite database path. Empty keeps the in-memory repository.
- `LIQUID2_SEED_DEMO`: `1` populates demo documents through the active repository for local UI review
- `LIQUID2_CORS_ORIGINS`: comma-separated browser origins allowed to call the API. Empty disables CORS; `*` is available for local-only development.
- `LIQUID2_JOBS_ENABLED`: `1` enables the SQLite-backed in-process job runner.
- `LIQUID2_TRANSLATION_PROVIDER`: empty disables translation workers; `codex`
  enables the local Codex CLI provider; `passthrough` is only for deterministic
  development/test runs.
- `LIQUID2_CODEX_COMMAND`: Codex executable used by the `codex` translation provider.
  Defaults to `codex`.
- `LIQUID2_CODEX_MODEL`: optional model passed to `codex exec`.
- `LIQUID2_CODEX_TIMEOUT_SECONDS`: optional provider timeout in seconds. Defaults
  to `300`.

Level policy:

- core/bootstrap/storage failures log at `error`.
- application/business success paths log at `debug` or `trace`.
- application/business failures log at `warn` for expected validation or state conflicts, and `error` for system or dependency failures.
- HTTP requests log at `debug` by default, `warn` for 4xx, and `error` for 5xx.

Common fields:

- `component`
- `operation`
- `request_id` when available
- `duration_ms` for timed work
- domain identifiers such as `document_id`, `job_id`, `job_kind`, `job_status`

Do not log document body, uploaded blob data, auth tokens, cookies, or full URL query strings.
Translation providers also must not log prompt text, provider raw responses, or credentials.

## Frontend Stack

초기 선택:

```text
Flutter
Riverpod
go_router
generated OpenAPI client
repository + view model 구조
```

Flutter 앱은 UI layer와 data layer를 분리한다. View는 business logic을 갖지 않고, view model/provider가 repository를 통해 API client를 호출한다.

## Backend Modules

```text
cmd/api
cmd/backup
cmd/openapi
internal/app
internal/exporter
internal/ingest
internal/feeds
internal/jobs
internal/logging
internal/storage/sqlite
internal/translation
internal/transport/http
api/openapi
client
```

### `documents`

문서 identity, 제목, 상태, 평점, 원문/번역 content metadata를 소유한다.

### `folders`

폴더 트리와 문서의 primary location을 소유한다. 문서는 하나의 폴더에만 속한다.
`Inbox`, `Feeds`, `Trash` 같은 system folder role, system folder 수정/삭제 금지,
Feeds 하위 피드별 폴더 자동 생성, Trash 기본 목록 제외 규칙은 이 도메인의 정책이다.

### `tags`

문서와 태그의 many-to-many 관계를 소유한다.

### `ingest`

URL bookmark, scrape, scrape+translate, upload ingestion use case를 소유한다.

### `feeds`

RSS feed 등록, check scheduling, item de-duplication을 소유한다.
Global RSS auto-refresh enablement and interval are application settings read
by the scheduler at runtime; process env only controls whether the job runtime
is available.

### `search`

문서 검색은 현재 `RepositoryReader.ListDocumentIDs(DocumentFilters)`의
adapter 구현으로 제공한다. SQLite adapter는 FTS5 index와 query를 내부에서
소유하고, PostgreSQL adapter는 같은 port를 `tsvector` 등으로 구현한다.
SQLite document list/search queries limit the first page at the SQL boundary
with `limit + 1`, then preserve the current opaque document-ID cursor contract
for later pages. This is an incremental refresh-path optimization, not full
keyset pagination.

Future full keyset pagination should replace ID-only cursors with sort-aware
cursor payloads for each supported ordering (`recent`, `created_desc`,
`rating_desc`, and relevance search). That migration must keep the API cursor
opaque while moving page continuation predicates into each repository adapter.

외부 검색 엔진, embedding 검색, 또는 비동기 reindex처럼 repository
transaction과 다른 일관성 모델이 필요해지면 이 영역을 별도 search port로
추출한다.

### `blobs`

업로드 파일과 scrape 원본 보존 payload를 관리한다. 초기 제한은 파일당 1MB다.

### `jobs`

RSS polling, scraping, translation 같은 background work의 lifecycle을 관리한다.

`jobs` owns runner orchestration, retry policy, and idempotency expectations. The
SQLite `jobs` table is durable persistence for that lifecycle, not the job
runner abstraction itself. When workers are introduced, define the consumer-side
`JobQueue` port under the worker/jobs package and implement it with a SQLite
adapter first. A later PostgreSQL or external queue adapter must preserve the
same claim, transition, recovery, and idempotency semantics.

Job runner가 직접 goroutine을 띄우는 경우, child goroutine entrypoint는 panic을 recover해 typed failure event로 parent runner에 돌려줘야 한다. Child는 자체적으로 재시작/무시/실패 확정 전략을 결정하지 않는다. Parent runner가 job을 재시도할지, 새 worker를 띄울지, 실패로 기록할지 결정한다. Recovery boundary는 domain code가 아니라 worker/adapter shell이 소유한다.

Background job이 여러 단계로 길어지면 단일 worker 함수에 누적하지 않고 pipeline으로 승격한다. Pipeline stage는 각각 하나의 일을 소유하고, input/output contract, idempotency rule, retry boundary를 명시한다. Parent runner는 stage 결과를 받아 다음 stage 진행, 재시도, 중단, 실패 기록을 결정한다.

### `storage/sqlite`

schema, migration, sqlc generated query, transaction helper를 소유한다.

### `transport/http`

HTTP route, request decoding, protocol validation, response mapping을 소유한다. Business rule은 domain/application owner로 위임하고, handler는 호출 경로를 명확히 남긴다.

## Data Flow

### Document Note

```text
POST /documents/{id}/notes
  -> validate target document
  -> validate note body
  -> create document_note
  -> update note search index if notes are indexed
```

### URL Bookmark

```text
POST /documents/bookmark
  -> validate URL
  -> normalize canonical URL
  -> create document with source_url
  -> optional metadata fetch
  -> return document
```

### URL Scrape

```text
POST /documents/scrape
  -> validate URL and SSRF guard
  -> fetch HTML
  -> extract title/body
  -> persist document + content + raw metadata
  -> update search index
```

### Scrape And Translate

```text
POST /documents/scrape-translate
  -> scrape document
  -> create translation job
  -> persist translated content as content variant
```

### Upload

```text
POST /documents/upload
  -> enforce max size and mime allowlist
  -> store blob
  -> extract text when supported
  -> create document/content
  -> update search index
```

### RSS Poll

```text
scheduled feed check
  -> fetch RSS
  -> de-duplicate by guid/canonical_url/content_hash
  -> create unread feed documents under the feed's Feeds child folder
  -> optionally enqueue scrape job
```

## Storage Boundary

SQLite is the only initial database. PostgreSQL compatibility is handled by conservative schema choices:

- app-generated text IDs
- integer unix milliseconds for timestamps
- text enums
- join tables instead of arrays
- limited JSON use
- SQLite FTS5 behind repository search reads

In-memory application state is allowed only as a contract-test fixture before
repository wiring. Production behavior must not synchronize between SQLite and a
separate memory store. Once persistence is enabled, SQLite-backed repositories
are the single source of truth and in-memory maps must be removed or confined to
tests.

Application services depend on the app-owned `Repository` port. The SQLite
adapter is the production implementation, while the memory adapter is retained
for focused tests and local contract fixtures. A future PostgreSQL adapter must
implement the same consumer-side port and preserve transaction callback
semantics, ID generation behavior, soft-delete visibility, note/tag/folder
relationship rules, blob size constraints, and document search/list semantics.
SQLite FTS5 and PostgreSQL text search will diverge inside their adapters, but
the app boundary remains `RepositoryReader.ListDocumentIDs(DocumentFilters)`
until an external or eventually consistent search backend is actually adopted.

## Future MSA Boundary

Do not split services initially. If needed later:

```text
api-service
document-service
ingest-worker
feed-worker
translation-worker
search-service
```

Potential internal communication options:

- ConnectRPC when typed service contracts are needed.
- Kratos when full microservice governance, HTTP/gRPC dual transport, registry, middleware, and code generation are worth the added framework weight.

## Security Boundaries

- URL ingestion must reject private, loopback, link-local, metadata, and malformed targets.
- Redirect chains must be validated at every hop.
- Uploads must enforce size, MIME, and extension policy.
- HTML content must be sanitized before display.
- Translation/scraping prompts must not overwrite original content.
- Delete should be soft-delete first.
- Backup/export must be explicit product features, not manual database copying only.

## References

- Go routing enhancements: https://go.dev/blog/routing-enhancements
- chi: https://github.com/go-chi/chi
- Huma OpenAPI generation: https://huma.rocks/features/openapi-generation/
- OpenAPI Generator dart-dio: https://openapi-generator.tech/docs/generators/dart-dio/
- sqlc: https://sqlc.dev/
- Flutter app architecture: https://docs.flutter.dev/app-architecture/guide
- Riverpod: https://docs-v2.riverpod.dev/docs/introduction/why_riverpod
- ConnectRPC: https://connectrpc.com/
- Kratos: https://github.com/go-kratos/kratos
