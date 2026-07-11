# API

This document is the human-readable v1 API contract. Product-local paths are
relative to the `liquid2/` directory unless stated otherwise. The generated
OpenAPI spec is the machine contract.

## Principles

- API style is REST over HTTP.
- Request and response bodies are JSON unless the endpoint explicitly uses multipart upload.
- IDs are application-generated text IDs.
- Timestamps are Unix milliseconds.
- Document and note delete operations are soft-delete. Folder, feed, backup, and export delete behavior must be documented per endpoint.
- Long-running work returns a `job` or `document + job` pair instead of blocking the request.
- Domain rules live in the domain/application layer. API schema validation may duplicate simple checks for defense-in-depth.

## Versioning

Client-facing routes are mounted under:

```text
/api/v1
```

`/healthz`, `/openapi.json`, `/openapi-3.0.json`, and `/docs` remain unversioned operational endpoints.

Examples below omit the `/api/v1` prefix for readability.

## Implementation Waves

This file is the v1 draft contract. Wave 2 implements the first generated OpenAPI route inventory.

Wave ownership:

```text
Wave 2: health, documents, document notes, folders, tags, common errors, OpenAPI
Wave 4: bookmark, scrape, upload
Wave 5: persistence boundary and SQLite repository wiring
Wave 6: feeds, jobs
Wave 7: search
Wave 8: scrape-translate, document translation
Wave 9: backup, export
Follow-up: document re-scrape
```

Generated route inventory through Wave 9F:

```text
GET    /healthz
GET    /api/v1/documents
POST   /api/v1/documents/bookmark
POST   /api/v1/documents/scrape
POST   /api/v1/documents/scrape-translate
POST   /api/v1/documents/upload
GET    /api/v1/documents/{id}
PATCH  /api/v1/documents/{id}
DELETE /api/v1/documents/{id}
POST   /api/v1/documents/{id}/move-to-trash
POST   /api/v1/documents/{id}/rescrape
POST   /api/v1/documents/{id}/mark-read
POST   /api/v1/documents/{id}/mark-unread
PUT    /api/v1/documents/{id}/rating
POST   /api/v1/documents/{id}/translate
GET    /api/v1/documents/{id}/notes
POST   /api/v1/documents/{id}/notes
PATCH  /api/v1/documents/{id}/notes/{noteId}
DELETE /api/v1/documents/{id}/notes/{noteId}
PUT    /api/v1/documents/{id}/tags
GET    /api/v1/folders
POST   /api/v1/folders
PATCH  /api/v1/folders/{id}
DELETE /api/v1/folders/{id}
GET    /api/v1/tags
POST   /api/v1/tags
GET    /api/v1/feeds
POST   /api/v1/feeds
PATCH  /api/v1/feeds/{id}
DELETE /api/v1/feeds/{id}
POST   /api/v1/feeds/{id}/refresh
GET    /api/v1/settings
PATCH  /api/v1/settings
GET    /api/v1/jobs
GET    /api/v1/jobs/{id}
POST   /api/v1/backup
POST   /api/v1/export
GET    /api/v1/exports/{id}
```

Generated machine contract:

```text
api/openapi/openapi-3.0.json
```

From the workspace root, the same file is
`liquid2/api/openapi/openapi-3.0.json`.

## Authentication

Initial development is local-only and has no user account model.

If the API is exposed beyond localhost, authentication must be added before exposure. The preferred future shape is a bearer token or local device pairing token; do not add multi-user auth until the product scope requires it.

## HTTP Status Codes

```text
200 OK                  successful read or synchronous write
201 Created             resource created
202 Accepted            long-running job accepted
204 No Content          successful write with no body
400 Bad Request         malformed input
404 Not Found           missing resource
409 Conflict            uniqueness or state conflict
413 Payload Too Large   upload exceeds limit
415 Unsupported Media   unsupported upload MIME type
422 Unprocessable       schema or parameter validation failure
503 Service Unavailable configured feature is unavailable
500 Internal Error      unexpected server failure
```

## Common Types

Active documents always belong to a folder. Document creation requests that omit
`folderId` are assigned to the default root `Inbox` folder.
`Inbox`, `Feeds`, and `Trash` are system folders. `Feeds` is the app-owned root
for RSS subscription folders. `Trash` is for discarding documents without
soft-deleting them.

### `DocumentSummary`

```json
{
  "id": "doc_01h...",
  "title": "SQLite as an Application File Format",
  "kind": "scraped_article",
  "folderId": "folder_01h...",
  "canonicalUrl": "https://example.com/article",
  "sourceUrl": "https://example.com/article?ref=rss",
  "language": "en",
  "status": "unread",
  "rating": 4,
  "createdAt": 1760000000000,
  "updatedAt": 1760000000000,
  "publishedAt": null,
  "readAt": null,
  "deletedAt": null,
  "tags": ["sqlite", "storage"]
}
```

`publishedAt` is set when the source provides an original publication timestamp,
such as RSS feed items. It is `null` for ordinary documents.

### `DocumentDetail`

```json
{
  "document": {
    "id": "doc_01h...",
    "title": "SQLite as an Application File Format",
    "kind": "scraped_article",
    "folderId": "folder_01h...",
    "canonicalUrl": "https://example.com/article",
    "sourceUrl": "https://example.com/article",
    "language": "en",
    "status": "unread",
    "rating": 4,
    "createdAt": 1760000000000,
    "updatedAt": 1760000000000,
    "publishedAt": null,
    "readAt": null,
    "deletedAt": null
  },
  "contents": [
    {
      "id": "content_01h...",
      "role": "extracted",
      "format": "markdown",
      "language": "en",
      "content": "# Title\n\nBody..."
    }
  ],
  "tags": [
    {"id": "tag_01h...", "name": "SQLite", "slug": "sqlite"}
  ],
  "blobs": []
}
```

### `Folder`

```json
{
  "id": "folder_01h...",
  "parentId": null,
  "name": "Inbox",
  "systemRole": "inbox",
  "sortOrder": 0,
  "createdAt": 1760000000000,
  "updatedAt": 1760000000000,
  "children": []
}
```

`systemRole` is omitted for user-managed folders. Current system roles are
`inbox`, `feeds`, and `trash`.

### `ErrorResponse`

Errors use Huma's RFC Problem Details-compatible shape.

```json
{
  "title": "Unprocessable Entity",
  "status": 422,
  "detail": "validation failed",
  "errors": [
    {"location": "query.ratingMin", "message": "expected number <= 5"}
  ]
}
```

Common error codes:

```text
validation_failed
not_found
conflict
unsafe_url
payload_too_large
unsupported_media_type
job_failed
service_unavailable
internal_error
```

## Pagination And Filters

List endpoints use cursor pagination.
The cursor is opaque to clients; clients must pass it back unchanged and must
not infer ordering data from it. Current SQLite-backed document lists apply
SQL-side `limit + 1` on the first page, while full keyset pagination is a future
repository-adapter optimization.

Request query:

```text
limit=50
cursor=opaque_cursor
```

Response shape:

```json
{
  "items": [],
  "nextCursor": null,
  "totalCount": 0
}
```

Common document filters:

```text
q=search text
status=unread|read
folderId=folder_01h...
includeFolderDescendants=true
tag=sqlite
ratingMin=4
kind=scraped_article
sort=relevance|recent|created_desc|rating_desc
includeDeleted=false
includeTrash=false
```

## Health

### `GET /healthz`

Checks process health.

Response:

```json
{"ok": true}
```

## Documents

### `GET /documents`

Lists documents with filters and pagination.

Query:

```text
q=search text
status=unread
folderId=folder_01h...
includeFolderDescendants=true
tag=sqlite
ratingMin=4
sort=relevance
includeDeleted=false
includeTrash=false
limit=50
cursor=opaque_cursor
```

Rules:

- default document lists and searches exclude documents in `Trash`
- setting `folderId` returns documents in that folder, including `Trash`
- setting `includeTrash=true` includes trash documents in unscoped lists/searches
- create, update, upload, bookmark, scrape, and feed assignment requests cannot
  directly target `Trash`; use `POST /documents/{id}/move-to-trash`

Response:

```json
{
  "items": [
    {
      "id": "doc_01h...",
      "title": "Example",
      "kind": "bookmark",
      "folderId": "folder_research",
      "folderPath": [
        {"id": "folder_inbox", "name": "Inbox"},
        {"id": "folder_research", "name": "Research"}
      ],
      "canonicalUrl": "https://example.com",
      "sourceUrl": "https://example.com",
      "language": null,
      "status": "unread",
      "rating": null,
      "createdAt": 1760000000000,
      "updatedAt": 1760000000000,
      "publishedAt": null,
      "readAt": null,
      "deletedAt": null,
      "tags": []
    }
  ],
  "nextCursor": null,
  "totalCount": 1
}
```

### `GET /documents/{id}`

Returns document metadata, content variants, tags, and blob metadata.
The response also includes `folderPath`, an ordered breadcrumb list from the
root folder to the document folder.

### `PATCH /documents/{id}`

Updates editable document metadata that does not have a dedicated state-transition endpoint.

Request:

```json
{
  "title": "New title",
  "folderId": "folder_01h..."
}
```

Rules:

- `title` mutation may create a history entry.
- empty `folderId` moves the document to the default root `Inbox` folder.
- read/unread transitions must use the dedicated mark-read/mark-unread endpoints.
- rating changes must use the dedicated rating endpoint.

### `DELETE /documents/{id}`

Soft-deletes a document.

Response:

```json
{"deleted": true, "deletedAt": 1760000000000}
```

### `POST /documents/{id}/move-to-trash`

Moves a document to the system `Trash` folder without setting `deletedAt`.
This is the only supported way to place an active document in `Trash`.

Response: `DocumentDetail`

### `POST /documents/{id}/mark-read`

Marks a document as read and sets `readAt`.

### `POST /documents/{id}/mark-unread`

Marks a document as unread and clears `readAt`.

### `PUT /documents/{id}/rating`

Request:

```json
{"rating": 4}
```

`rating` may be `null` to clear the rating.

### `POST /documents/{id}/translate`

Enqueues translation for an existing document content variant.

Request:

```json
{
  "sourceContentId": "content_01h...",
  "targetLanguage": "ko"
}
```

Response:

```json
{
  "job": {"id": "job_01h...", "kind": "translate_document", "status": "queued"}
}
```

Status: `202 Accepted`

Errors:

- `400 Bad Request`: invalid target language or source content semantics
- `404 Not Found`: document or source content does not exist
- `409 Conflict`: translation already exists or is already queued/running
- `422 Unprocessable Entity`: request body fails schema validation
- `503 Service Unavailable`: translation worker/provider is not configured

Provider execution failures are recorded on the translation job. Poll
`GET /jobs/{id}` for final `completed` or `failed` status.

### `GET /documents/{id}/blobs/{blobId}`

Downloads an uploaded or preserved blob.

Response body is the raw file bytes. The server must set a safe `Content-Type` and `Content-Disposition`.

Rules:

- blob must belong to the document path parameter
- local filesystem paths are never exposed
- content sniffing must not override the stored allowlisted MIME type

## Document Notes

Document notes are user comments attached to an existing document. They are not independent documents and do not have folder, tag, rating, or read status.

### `GET /documents/{id}/notes`

Lists non-deleted notes for a document.

Response:

```json
{
  "items": [
    {
      "id": "note_01h...",
      "documentId": "doc_01h...",
      "body": "Important point to revisit.",
      "format": "text",
      "createdAt": 1760000000000,
      "updatedAt": 1760000000000,
      "deletedAt": null
    }
  ]
}
```

### `POST /documents/{id}/notes`

Creates a note/comment on an existing document.

Request:

```json
{
  "body": "Important point to revisit.",
  "format": "text"
}
```

Rules:

- `format` may be `text` or `markdown`.
- note creation is synchronous and does not create a background job.
- target document must not be soft-deleted.

### `PATCH /documents/{id}/notes/{noteId}`

Updates note body or format.

Request:

```json
{
  "body": "Updated note.",
  "format": "markdown"
}
```

### `DELETE /documents/{id}/notes/{noteId}`

Soft-deletes a note.

Response:

```json
{"deleted": true, "deletedAt": 1760000000000}
```

## Ingestion

All URL-taking ingestion endpoints must validate the target before any network fetch. Validation must reject private, loopback, link-local, metadata, and malformed targets, and must re-check every redirect hop.
When `folderId` is omitted, created documents use the default root `Inbox`.

### `POST /documents/bookmark`

Creates a document from a URL without scraping full content.

Request:

```json
{
  "url": "https://example.com/article",
  "title": "Optional title",
  "folderId": "folder_01h...",
  "tagIds": ["tag_01h..."]
}
```

Response: `DocumentDetail`

### `POST /documents/scrape`

Fetches a URL, extracts readable content, stores a document, and indexes it.

Request:

```json
{
  "url": "https://example.com/article",
  "folderId": "folder_01h...",
  "tagIds": ["tag_01h..."]
}
```

Response: `DocumentDetail`

Stored extracted content may be `format: "markdown"` or `format: "text"`.
HTML responses prefer readable Markdown extraction. If readable extraction or
conversion fails, the server tries semantic article/main content selection and
Markdown conversion before falling back to stripped text. Clients must render
based on the returned content `format`, not on the source MIME type.

The Wave 4 endpoint is synchronous. If scraping moves to jobs later, introduce
a versioned or explicitly async response rather than changing this response
shape silently.

### `POST /documents/{id}/rescrape`

Fetches the source URL of an existing scraped document or RSS item, extracts
fresh readable content, replaces the existing `extracted` content variant, and
records document history before mutation.

Response: `DocumentDetail`

Rules:

- allowed document kinds are `scraped_article` and `rss_item`
- the fetch target is `sourceUrl` when present, otherwise `canonicalUrl`
- title, folder, tags, read state, rating, notes, blobs, and translations are
  preserved
- the existing extracted content ID is preserved when one exists, so translation
  source links remain valid
- a document version is recorded before the extracted content is replaced

Status:

- `200 OK` when content is refreshed
- `400 Bad Request` for non-rescrapable documents, missing source URL, unsafe
  URL, or scrape fetch failure
- `404 Not Found` for missing or soft-deleted documents
- `413 Request Entity Too Large` when fetched content is too large
- `415 Unsupported Media Type` when fetched content is not supported
- `422 Unprocessable Entity` for request schema violations

### `POST /documents/scrape-translate`

Creates a scraped document and enqueues translation.

Request:

```json
{
  "url": "https://example.com/article",
  "targetLanguage": "ko",
  "folderId": "folder_01h...",
  "tagIds": ["tag_01h..."]
}
```

Response:

```json
{
  "document": {
    "document": {"id": "doc_01h...", "title": "Example"},
    "contents": [{"id": "content_01h...", "role": "extracted", "format": "markdown"}],
    "tags": [],
    "blobs": []
  },
  "job": {"id": "job_01h...", "kind": "translate_document", "status": "queued"}
}
```

Status:

- `201 Created` when the scraped document is created and translation is queued
- `400 Bad Request` for an unsafe URL, scrape fetch failure, or invalid target language
- `404 Not Found` for missing folder or tag references
- `409 Conflict` if translation is already queued or running for the same source/target
- `413 Request Entity Too Large` when fetched content is too large
- `415 Unsupported Media Type` when fetched content is not supported
- `422 Unprocessable Entity` for request schema violations
- `503 Service Unavailable` when translation enqueueing is not configured

If translation enqueueing fails after the document has been created, the API
returns the enqueue failure and keeps the scraped document. Callers can retry
translation through `POST /documents/{id}/translate`.

### `POST /documents/upload`

Uploads a file and creates a document.

Request content type:

```text
multipart/form-data
```

Fields:

```text
file: required, max 1MB
title: optional
folderId: optional
tagIds: optional repeated field
```

Rules:

- max file size is 1MB
- omitted `folderId` assigns the uploaded document to the default root `Inbox`
- supported MIME types are `text/plain`, `text/markdown`, `text/html`, and `application/pdf`
- text and markdown uploads preserve their text/markdown format
- HTML uploads prefer readable Markdown extraction and fall back to stripped text
- PDF uploads store blob metadata and bytes, but PDF text extraction is deferred
- unsupported MIME types return `unsupported_media_type`
- oversized files return `payload_too_large`

Response: `DocumentDetail`

## Folders

### `GET /folders`

Returns the folder tree. System folders include `systemRole`.

### `POST /folders`

Request:

```json
{
  "parentId": null,
  "name": "Databases",
  "sortOrder": 10
}
```

### `PATCH /folders/{id}`

Updates folder name, parent, or ordering.

Rules:

- system folders cannot be edited
- moving a folder under its own descendant is invalid
- moving a folder under `Trash` is invalid
- folder names should be unique among siblings

### `DELETE /folders/{id}`

Deletes an empty folder or moves its documents according to the selected action.
Documents are never left folderless; root-folder documents move to the default
root `Inbox` when `move_to_parent` cannot use a parent. The legacy
`move_to_uncategorized` action also moves documents to the default `Inbox`.
System folders cannot be deleted.

Query:

```text
documentAction=move_to_parent
```

Allowed `documentAction` values:

```text
move_to_parent
move_to_uncategorized
reject_if_not_empty
```

## Tags

### `GET /tags`

Lists tags.

### `POST /tags`

Request:

```json
{"name": "SQLite"}
```

### `PUT /documents/{id}/tags`

Replaces all tags on a document.
Any removed tag that is no longer assigned to any document is deleted.

Request:

```json
{"tagIds": ["tag_01h...", "tag_02h..."]}
```

## Feeds

Wave 6A implements feed CRUD. Wave 6C adds manual refresh through the background
job queue.
Creating a feed automatically creates a user-managed child folder under the
system `Feeds` root and assigns imported RSS documents to that folder. The
create request must not provide `folderId`; `folderId` remains mutable through
`PATCH /feeds/{id}` for explicit feed relocation.

### `GET /feeds`

Lists RSS feeds.

### `POST /feeds`

Request:

```json
{
  "url": "https://example.com/feed.xml",
  "title": "Example Feed",
  "enabled": true
}
```

### `PATCH /feeds/{id}`

Updates feed metadata.

### `DELETE /feeds/{id}`

Deletes the feed subscription record. Imported documents are not deleted. Use `PATCH /feeds/{id}` with `enabled: false` to pause a feed without deleting the subscription.
Moving an imported RSS document to `Trash` preserves its `feed_items` record, so
later refreshes still de-duplicate the discarded item.

### `POST /feeds/{id}/refresh`

Enqueues an immediate refresh and returns the queued `poll_feed` job.

Status: `202 Accepted`

Response:

```json
{
  "job": {"id": "job_01h...", "kind": "poll_feed", "status": "queued"}
}
```

Errors:

- `404 Not Found`: feed does not exist
- `409 Conflict`: feed is disabled or a refresh for the same feed is already queued/running

## Settings

Settings are persisted application-level user preferences.

### `GET /settings`

Returns current app settings.

Response:

```json
{
  "feedSchedulerEnabled": false,
  "feedPollIntervalSeconds": 7200,
  "feedNextPollAt": null,
  "updatedAt": 1760000000000
}
```

### `PATCH /settings`

Updates app settings. Fields are partial. `feedPollIntervalSeconds` must be
between 60 and 86400 seconds. `feedNextPollAt` is read-only scheduler status
and is updated when the in-process scheduler schedules its next RSS poll.

Request:

```json
{
  "feedSchedulerEnabled": true,
  "feedPollIntervalSeconds": 900
}
```
- `503 Service Unavailable`: feed refresh worker is not configured

## Jobs

Background jobs are reserved for work that may outlive the request/response cycle, needs retry and failure visibility, or runs without a direct user request.

Jobs are not required for document note/comment creation, read/unread changes, rating changes, folder edits, or tag edits.

Required job-backed work:

- `poll_feed`: RSS polling must run periodically.
- `translate_document`: translation can be slow and provider-dependent.

Potentially job-backed work:

- `scrape_url`: can start synchronous, but may move to jobs for slow pages, retries, or rate limiting.
- `extract_upload_text`: can start synchronous for small files, but may move to jobs for PDF/text extraction failures and retry visibility.

### `GET /jobs`

Lists background jobs.

Query:

```text
status=queued|running|completed|failed
kind=scrape_url|translate_document|poll_feed|extract_upload_text
limit=50
```

### `GET /jobs/{id}`

Returns job status and failure details.

Response:

```json
{
  "id": "job_01h...",
  "kind": "translate_document",
  "status": "running",
  "attempts": 1,
  "error": null,
  "createdAt": 1760000000000,
  "updatedAt": 1760000000000,
  "startedAt": 1760000000000,
  "finishedAt": null
}
```

## Backup And Export

These endpoints are Wave 9 API space. Responses expose artifact IDs and API
download references only; they never expose local server filesystem paths.

```text
POST /api/v1/backup
POST /api/v1/export
GET  /api/v1/exports/{id}
```

### `BackupArtifact`

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

Rules:

- `sourceType` is `sqlite` for the initial implementation.
- `downloadUrl` is optional and, when present, must be an API route.
- local filesystem paths must not appear in the response.

### `ExportArtifact`

```json
{
  "id": "export_01h...",
  "createdAt": 1760000000000,
  "manifestVersion": 1,
  "documentCount": 12,
  "blobCount": 3,
  "sizeBytes": 2097152,
  "sha256": "hex...",
  "downloadUrl": null
}
```

Rules:

- `manifestVersion` is `1` for the initial markdown export format.
- `downloadUrl` is optional and, when present, must be an API route.
- local filesystem paths must not appear in the response.

### `POST /api/v1/backup`

Creates a SQLite backup artifact using server-side configuration.

Request:

```json
{}
```

The request body must not accept destination paths.

Response:

```json
{
  "backup": {
    "id": "backup_01h...",
    "createdAt": 1760000000000,
    "sourceType": "sqlite",
    "schemaVersion": 5,
    "sizeBytes": 1048576,
    "sha256": "hex...",
    "downloadUrl": null
  }
}
```

Errors:

- `503 Service Unavailable`: backup storage is not configured, or the server is
  not using SQLite persistence.

### `POST /api/v1/export`

Creates a markdown export artifact using the manifest v1 layout from
[Design](design.md).

Request:

```json
{
  "documentIds": null,
  "includeBlobs": true
}
```

Rules:

- `documentIds=null` exports all non-deleted documents.
- `documentIds=[]` is invalid.
- `includeBlobs=false` is reserved for a later export mode and is invalid in
  the initial implementation.
- request fields must not accept destination paths.
- export content and manifest paths are relative to the export artifact root.

Response:

```json
{
  "export": {
    "id": "export_01h...",
    "createdAt": 1760000000000,
    "manifestVersion": 1,
    "documentCount": 12,
    "blobCount": 3,
    "sizeBytes": 2097152,
    "sha256": "hex...",
    "downloadUrl": null
  }
}
```

Errors:

- `400 Bad Request`: an API-safe option is recognized but unsupported.
- `404 Not Found`: any requested document ID does not exist.
- `422 Unprocessable Entity`: request body fails schema validation.
- `503 Service Unavailable`: export artifact storage is not configured.

### `GET /api/v1/exports/{id}`

Returns export artifact metadata. If a download endpoint is added later, it
must be represented by an API route or content negotiation, never by returning
a server filesystem path.

Response:

```json
{
  "export": {
    "id": "export_01h...",
    "createdAt": 1760000000000,
    "manifestVersion": 1,
    "documentCount": 12,
    "blobCount": 3,
    "sizeBytes": 2097152,
    "sha256": "hex...",
    "downloadUrl": null
  }
}
```

Errors:

- `404 Not Found`: export artifact does not exist.
- `503 Service Unavailable`: export artifact storage is not configured.

## Open Questions

- Raw HTML sanitization policy for any future raw-HTML display surface.
