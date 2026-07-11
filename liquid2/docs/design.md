# Design

## Product Model

Liquid2는 문서 원본, 메타데이터, 읽기 상태, 분류, 검색 index를 하나의 데이터 모델로 관리한다. 파일명과 폴더 경로는 문서의 identity가 아니다.

현재 workspace에서 이 문서의 경로 예시는 별도 언급이 없으면
`liquid2/` 제품 디렉터리 기준이다. 오래된 package-shape 예시는 설계
근거와 목표 형태를 보존하기 위한 것이며, 현재 구현은 주로
`internal/app`, `internal/ingest`, `internal/feeds`, `internal/jobs`,
`internal/storage/sqlite`, `internal/translation`, `internal/transport/http`,
`internal/exporter`에 나뉘어 있다.

## Module Design Rules

Liquid2의 구현 단위는 작고 교체 가능해야 한다.

규칙:

- 한 file은 하나의 역할만 가진다.
- source file은 200 lines 안팎을 넘기지 않는 것을 목표로 한다.
- 200 lines를 넘으면 split이 필요한지 먼저 검토한다.
- 한 package 또는 Flutter feature는 하나의 domain capability를 온전히 관리한다.
- domain rule의 source of truth는 domain/application layer다.
- SQL, handler, widget은 각자의 관심사를 온전히 소유하고 이름과 호출 경로로 domain source까지 추적 가능해야 한다.
- 외부 구현은 port/interface 뒤에 둔다.
- interface는 provider가 아니라 consumer가 필요한 모양으로 정의한다.

Go package shape:

```text
internal/<capability>/
  model.go
  service.go
  ports.go
  errors.go
  service_test.go
```

Flutter feature shape:

```text
features/library/
  library_page.dart
  document_list.dart
  library_controller.dart
  library_state.dart
```

Flutter에서도 widget, state, API repository를 한 파일에 섞지 않는다. UI widget은 rendering과 interaction binding만 담당하고, state transition은 controller/provider가 담당한다.

관심사 분리 기준:

```text
business rule       -> domain/application
persistence query   -> SQL query file
query-to-domain map -> repository adapter
protocol shape      -> HTTP transport/OpenAPI
screen state        -> Flutter controller/provider
visual structure    -> Flutter widget
```

허용되는 중복과 금지되는 중복을 구분한다. 예를 들어 rating range는 domain이 최종 판단하고, API schema나 DB constraint가 같은 범위를 한 번 더 막는 것은 defense-in-depth로 허용한다. 하지만 서로 다른 범위나 서로 다른 상태 전이 규칙을 각 layer가 독자적으로 갖는 것은 금지한다.

## Core Concepts

### Document

사용자가 관리하는 가장 중요한 단위다.

문서는 다음 입력에서 생성될 수 있다.

- bookmark URL
- scraped URL
- translated scrape result
- uploaded file
- RSS item
- manual note document, later phase

문서는 여러 content variant를 가질 수 있다. 예를 들면 원문 HTML, 추출 markdown, 한국어 번역본이 같은 document 아래에 붙을 수 있다.

### Document Note

Document note는 스크랩, 업로드, RSS 등으로 모인 기존 document에 붙는 사용자 comment다. MVP에 포함한다.

Note는 독립 document가 아니다. folder, tag, rating, read status를 따로 갖지 않고 parent document에 종속된다. Note도 soft-delete를 사용한다.

### Folder

폴더는 문서의 primary category다. 문서는 하나의 폴더에만 속한다.

폴더는 tree 구조를 가진다.

```text
folders
  id
  parent_id
  name
  system_role
  sort_order
  created_at
  updated_at
```

`Inbox`, `Feeds`, and `Trash` are system folders. System folders are app-owned
roots and cannot be renamed, moved, or deleted by folder management. `Feeds`
owns RSS subscription folders: creating a feed creates a child folder under
`Feeds`, and imported feed documents use that child folder by default. `Trash`
means discarded but not soft-deleted; default document lists hide it unless the
user selects Trash or explicitly includes trash documents. New documents and
feeds cannot be assigned directly to Trash; documents enter Trash through the
dedicated move-to-trash transition. RSS documents moved to Trash keep their
feed item linkage so refresh de-duplication still works.

### Tag

태그는 교차 분류다. 문서는 여러 태그를 가질 수 있다.
문서에서 태그 연결이 제거될 때, 제거된 태그가 더 이상 어떤 문서에도
연결되어 있지 않으면 태그 자체도 삭제된다.

### View

조건별 모아보기는 저장된 query preset이다. 예:

- unread
- rating >= 4
- folder subtree
- specific tag set
- feed source
- recently added
- has translation
- uploaded PDFs

초기에는 view definition을 DB에 저장하지 않고 고정 query로 시작할 수 있다. 사용자가 custom view를 만들 필요가 생기면 `saved_views`를 추가한다.

## Proposed Schema

초기 schema 초안이다. migration 작성 시 SQLite 문법으로 구체화한다.

```text
documents
  id TEXT PRIMARY KEY
  title TEXT NOT NULL
  kind TEXT NOT NULL
  folder_id TEXT NULL
  canonical_url TEXT NULL
  source_url TEXT NULL
  language TEXT NULL
  status TEXT NOT NULL
  rating INTEGER NULL
  created_at INTEGER NOT NULL
  updated_at INTEGER NOT NULL
  read_at INTEGER NULL
  deleted_at INTEGER NULL

document_contents
  id TEXT PRIMARY KEY
  document_id TEXT NOT NULL
  role TEXT NOT NULL
  format TEXT NOT NULL
  language TEXT NULL
  content TEXT NOT NULL
  source_content_id TEXT NULL
  created_at INTEGER NOT NULL

document_notes
  id TEXT PRIMARY KEY
  document_id TEXT NOT NULL
  body TEXT NOT NULL
  format TEXT NOT NULL
  created_at INTEGER NOT NULL
  updated_at INTEGER NOT NULL
  deleted_at INTEGER NULL

folders
  id TEXT PRIMARY KEY
  parent_id TEXT NULL
  name TEXT NOT NULL
  sort_order INTEGER NOT NULL
  created_at INTEGER NOT NULL
  updated_at INTEGER NOT NULL

tags
  id TEXT PRIMARY KEY
  name TEXT NOT NULL
  slug TEXT NOT NULL UNIQUE
  created_at INTEGER NOT NULL

document_tags
  document_id TEXT NOT NULL
  tag_id TEXT NOT NULL
  PRIMARY KEY (document_id, tag_id)

blobs
  id TEXT PRIMARY KEY
  document_id TEXT NOT NULL
  filename TEXT NOT NULL
  mime_type TEXT NOT NULL
  size INTEGER NOT NULL
  sha256 TEXT NOT NULL
  data BLOB NOT NULL
  created_at INTEGER NOT NULL

feeds
  id TEXT PRIMARY KEY
  url TEXT NOT NULL UNIQUE
  title TEXT NULL
  folder_id TEXT NULL -- feed document folder, auto-created under system Feeds on create
  enabled INTEGER NOT NULL
  last_checked_at INTEGER NULL
  created_at INTEGER NOT NULL
  updated_at INTEGER NOT NULL

feed_items
  id TEXT PRIMARY KEY
  feed_id TEXT NOT NULL
  document_id TEXT NOT NULL
  guid TEXT NULL
  url TEXT NOT NULL
  canonical_url TEXT NULL
  content_hash TEXT NULL
  published_at INTEGER NULL
  created_at INTEGER NOT NULL

jobs
  id TEXT PRIMARY KEY
  kind TEXT NOT NULL
  status TEXT NOT NULL
  payload_json TEXT NOT NULL
  error TEXT NULL
  attempts INTEGER NOT NULL
  created_at INTEGER NOT NULL
  updated_at INTEGER NOT NULL
  started_at INTEGER NULL
  finished_at INTEGER NULL
```

Wave 1 migration must add foreign keys, uniqueness constraints, and indexes where the schema text only names columns. Required constraints include sibling folder name uniqueness, tag slug uniqueness, document/tag pair uniqueness, feed URL uniqueness, feed item de-duplication by feed GUID or URL, and indexes for common document filters.

Wave 1 schema contract:

- enable SQLite foreign key enforcement.
- add `CHECK` constraints for document status, document kind, content role, note format, job kind/status, and rating bounds.
- add unique indexes for sibling folder names, tag slugs, document/tag pairs, feed URLs, feed item GUIDs when present, and feed item canonical URL or content hash when GUID is absent.
- add indexes for common document filters: status, folder, kind, rating, created time, soft-delete, and read time.
- add migration bookkeeping so a database can report its schema version.

Wave 9 history table:

```text
document_versions
  id TEXT PRIMARY KEY
  document_id TEXT NOT NULL
  sequence INTEGER NOT NULL
  mutation_kind TEXT NOT NULL
  title TEXT NOT NULL
  content_snapshot_json TEXT NULL
  metadata_snapshot_json TEXT NULL
  created_at INTEGER NOT NULL
```

`document_versions` stores old document state snapshots. Application code
decides when to create a version; SQL triggers must not infer history from
aggregate persistence writes.

## Document Status

Initial values:

```text
unread
read
```

Deletion is represented by `deleted_at`, not by a document status. Default document lists exclude soft-deleted rows. Moving a document to `Trash` is folder movement, not deletion. Hard delete can be implemented later as a cleanup command.

## Document Kind

Initial values:

```text
bookmark
scraped_article
uploaded_file
rss_item
```

Translation should usually be a content variant, not a separate document. A separate translated document is only needed if the translated output needs independent status, tags, folder, or rating.

Later document kind:

```text
manual_note
```

`manual_note` means an independent user-authored document. It is not part of the MVP. MVP note/comment support is handled by `document_notes`.

## Content Role

Initial values:

```text
original
extracted
translation
summary
```

Search should prefer `extracted` and `translation`, then fall back to `original` when text is usable.

## Extraction And Rendering

`internal/ingest` owns content extraction behind a package-local `Extractor`
interface. `DefaultExtractor` is the current implementation:

- `text/plain` is stored as `format = "text"`.
- `text/markdown` is stored as `format = "markdown"`.
- `text/html` and `application/xhtml+xml` first try readable article extraction
  with Readeck readability and HTML-to-Markdown conversion.
- successful readable HTML extraction is stored as `format = "markdown"`.
- if readability, rendering, conversion, or empty-output checks fail, HTML tries
  semantic `article`/`main` candidate selection with HTML-to-Markdown
  conversion.
- if semantic Markdown extraction also fails, HTML falls back to stripped plain
  text with `format = "text"`.

The HTTP fetcher passes the final response URL after redirects into the
extractor so relative article links can be resolved during HTML-to-Markdown
conversion. Upload extraction uses the same package-level extractor with no
page URL context.

Extractor replacement is intentionally local to `internal/ingest`; do not add a
domain or per-site registry until there is a concrete site-specific rule worth
owning.

Flutter detail rendering follows the stored content format. Markdown content is
rendered with the Markdown widget path, selectable text, and soft line breaks so
scraped prose keeps source newlines readable. Non-Markdown content uses the
detail-body formatter, so legacy text and HTML rows remain readable without raw
tag clutter or collapsed paragraphs. Raw HTML display is not part of the MVP UI.

## Search

Initial implementation:

- SQLite FTS5
- index title, URL, extracted text, translation text
- tag/folder filters remain normal SQL filters

Search query shape:

```text
text query + structured filters
```

Examples:

- query: `sqlite blob`
- status: `unread`
- tag: `database`
- folder subtree: `papers/programming`
- rating min: `4`

## API Shape

REST endpoints are the public API. OpenAPI is the machine contract, and [API](api.md) is the human-readable contract draft.

Initial route groups:

```text
GET    /healthz

GET    /documents
POST   /documents/bookmark
POST   /documents/scrape
POST   /documents/scrape-translate
POST   /documents/upload
GET    /documents/{id}
PATCH  /documents/{id}
DELETE /documents/{id}
POST   /documents/{id}/move-to-trash
POST   /documents/{id}/rescrape
POST   /documents/{id}/mark-read
POST   /documents/{id}/mark-unread
PUT    /documents/{id}/rating
POST   /documents/{id}/translate
GET    /documents/{id}/blobs/{blobId}
GET    /documents/{id}/notes
POST   /documents/{id}/notes
PATCH  /documents/{id}/notes/{noteId}
DELETE /documents/{id}/notes/{noteId}

GET    /folders
POST   /folders
PATCH  /folders/{id}
DELETE /folders/{id}

GET    /tags
POST   /tags
PUT    /documents/{id}/tags

GET    /feeds
POST   /feeds
PATCH  /feeds/{id}
DELETE /feeds/{id}
POST   /feeds/{id}/refresh

GET    /settings
PATCH  /settings

GET    /jobs
GET    /jobs/{id}

POST   /backup
POST   /export
GET    /exports/{id}
```

## Background Jobs

Initial jobs run in the API process. This keeps deployment simple.

Background jobs are not required for simple synchronous actions such as creating a document note/comment, marking read/unread, rating, folder changes, or tag changes.

Jobs are needed when work may outlive an HTTP request, needs retry/failure visibility, or should run without a direct user request. RSS polling is the clearest required case because it runs periodically. Translation can be slow and provider-dependent. Scraping and upload text extraction may be synchronous at first, but the job model lets them become asynchronous without changing the user-facing workflow.

Wave 6 execution rules:

- Job queue persistence must sit behind an app-owned port. SQLite can be the first adapter, but job claiming, retry, and de-duplication rules belong to the application/job boundary.
- Worker, scheduler, repository, and server lifecycle code must have explicit shutdown paths. Startup or runtime failures should return through stack unwinding so deferred cleanup runs; nested setup code should not call `os.Exit`.
- Internal failures should be wrapped or logged with enough detail for debugging. HTTP/API responses expose stable failure classes and safe messages, not raw provider errors, URLs with tokens, or panic payloads.
- Wave 6 implementation must be split into reviewable PRs. Each PR should own one backend or Flutter slice, include focused tests, and update the generated API/client only when its public contract changes.

Worker goroutines must be launched through a runner that turns child panic into a typed failure event and returns it to the parent runner. The child boundary should recover, attach enough context for debugging, and stop. The parent runner decides whether to retry, restart a worker, or mark the job failed. Domain/application code should return errors normally; panic recovery belongs to the worker shell.

If a background job grows beyond a short action, model it as a pipeline instead of a large worker function. Pipeline stages must be modules with explicit input/output contracts, idempotency expectations, and retry boundaries. The parent runner owns orchestration and decides whether a stage result advances, retries, restarts, or fails the job.

RSS scheduling lives with the feed boundary, not the generic job runner. The
in-process scheduler reads enabled feeds from the app service and enqueues
`poll_feed` jobs through the queue port. Global enablement and the polling
interval are persisted app settings controlled through the API/UI, not process
environment flags. The scheduler treats an active duplicate refresh job as a
skipped feed rather than an error. Shutdown stops the scheduler before the job
runner so no new scheduled jobs are added while the runner is draining.

Job types:

- `scrape_url`
- `translate_document`
- `poll_feed`
- `extract_upload_text`

Later, the same job table can be consumed by a separate worker process.

Job status values:

```text
queued
running
completed
failed
```

Initial legal transitions:

```text
queued -> running
running -> completed
running -> failed
running -> queued
failed -> queued
```

The final two transitions are retry paths owned by the parent runner. `attempts` increments when a queued job is claimed for running. Jobs that are `running` during startup recovery must be moved to `failed` or `queued` by an explicit recovery policy before new work is claimed.

## Flutter App Structure

Initial package layout:

```text
lib/
  main.dart
  app/
  api/
  data/
    documents/
    folders/
    tags/
    feeds/
  features/
    library/
    document_detail/
    ingest/
    feeds/
    search/
    settings/
```

UI principles:

- first screen is the document library, not a landing page
- left navigation or adaptive navigation for folders/views
- list-detail layout on desktop/tablet
- bottom navigation or compact drawer on mobile
- folder and tag controls must be first-class, not hidden metadata
- document read state and rating must be fast inline actions

## Import, Export, Backup

SQLite is the source of truth, but user escape hatches are mandatory.

### Export

Initial export is a portable directory or archive layout:

```text
export/
  documents/
    {document-id}.md
  blobs/
  manifest.json
```

The manifest is versioned JSON. Manifest paths are relative to the export root
and must never be local server filesystem paths.

Manifest v1 shape:

```json
{
  "manifestVersion": 1,
  "exportId": "export_01h...",
  "createdAt": 1760000000000,
  "source": {
    "app": "liquid2",
    "appVersion": null,
    "schemaVersion": 5
  },
  "counts": {
    "documents": 1,
    "blobs": 1
  },
  "documents": [
    {
      "id": "doc_01h...",
      "markdownPath": "documents/doc_01h.md",
      "title": "SQLite as an Application File Format",
      "kind": "scraped_article",
      "folderId": "folder_01h...",
      "sourceUrl": "https://example.com/article",
      "canonicalUrl": "https://example.com/article",
      "language": "en",
      "contents": [
        {
          "id": "content_01h...",
          "role": "extracted",
          "format": "markdown",
          "language": "en"
        }
      ],
      "tags": [
        {"id": "tag_01h...", "name": "SQLite", "slug": "sqlite"}
      ],
      "blobs": [
        {
          "id": "blob_01h...",
          "path": "blobs/blob_01h-report.pdf",
          "filename": "report.pdf",
          "mimeType": "application/pdf",
          "sizeBytes": 12345,
          "sha256": "hex..."
        }
      ]
    }
  ]
}
```

Markdown rendering rules:

- each non-deleted document gets one markdown file.
- markdown content is written as markdown.
- text and `pdf_text` content are written as plain markdown-safe text.
- HTML content is preserved in a clearly labelled section; HTML-to-markdown
  conversion can be improved later without changing manifest v1.
- blobs are written with deterministic sanitized names and recorded in the
  manifest by relative path and checksum.

Export implementation must read through app-owned service/repository behavior.
It must not import SQLite or sqlc packages.

### Backup

- produce timestamped SQLite backup
- use SQLite-native backup behavior, not raw copying of a live database file
- include artifact ID, creation time, SQLite source type, schema version,
  `sizeBytes`, and SHA-256 checksum
- keep local destination paths inside operator/configuration code only

Backup artifact metadata shape:

```json
{
  "id": "backup_01h...",
  "createdAt": 1760000000000,
  "sourceType": "sqlite",
  "schemaVersion": 5,
  "sizeBytes": 1048576,
  "sha256": "hex...",
  "downloadUrl": null
}
```

`downloadUrl` is optional. If present, it must be an API route, not a local
filesystem path.

### Restore And Import Design

Restore/import is not implemented in Wave 9. The design constraints are:

- SQLite backup restore is an operator action, not an HTTP endpoint.
- Full restore must run with the API server stopped or target a new database.
- Restore validates checksum and schema compatibility before replacing data.
- Export import starts with a dry-run report before any write behavior ships.
- Import preserves source document IDs in the manifest for conflict reporting.
- Default ID conflict policy is reject; future duplicate import may map source
  IDs to new IDs, but overwrite requires explicit rollback/history design.
- Blob import validates size and SHA-256 before committing metadata.
- Manifest versions are accepted only when the importer explicitly supports
  that version.
- Failed restore/import must leave the current database unchanged.

## History

Do not build full version control initially.

Rule:

- Before a real existing-document title or content mutation, insert the old
  state into `document_versions`.
- Metadata-only changes such as read/unread, rating, folder movement, tag
  changes, notes, feeds, jobs, and soft delete do not create document history.
- Document creation does not create a history entry because no previous state
  exists.
- Content append/mutation paths, including translation append, should create
  history entries before persisting the new content.

## Non-goals For Initial Build

- collaboration
- multi-user auth
- public sharing
- distributed sync
- full-text ranking beyond basic local search
- object storage
- immediate PostgreSQL implementation
- immediate MSA deployment
