# Media And Document Source Implementation Design

This document fixes the implementation direction for image, audio, video, and
PDF document sources in Plasma.

## Intended Workflow

The user adds media as original material for a mission. Plasma records where the
media came from, what license and attribution apply, and whether the bytes were
pinned at a point in time. The agent can then discover and cite the media
through MCP/source tools. Reports can include the media in Markdown and
interactive HTML without turning the rendered report back into a source.

The default report rule is:

- Images may be embedded into self-contained interactive HTML when they are
  pinned, small enough, and safe to render.
- Audio and video stay as links or allowlisted provider embeds by default.
- PDFs stay as original document sources. Reports cite the original PDF and use
  extracted text reads as working material, not as new sources.
- Markdown keeps original URLs and attribution visible. It should not use data
  URLs.

## What This Will Do

- Add media as a first-class source shape over the existing source snapshot and
  raw artifact model.
- Store image bytes as pinned raw artifacts when accepted.
- Store source metadata for images, audio, and video in a typed locator payload.
- Store PDF URL sources as pinned raw artifacts with document metadata and
  bounded extracted-text reads.
- Render pinned images as data URLs in self-contained HTML exports when they fit
  policy.
- Render audio and video as links or allowlisted provider embeds, not as
  self-contained blobs.
- Expose media metadata through MCP reads and listings without returning binary
  bytes inline.
- Expose PDF text through MCP/source reads without returning raw PDF bytes
  inline.

## What This Will Not Do

- It will not make generated captions, alt text, report prose, or thumbnails
  into sources.
- It will not loosen the current textual URL source fetcher to accept arbitrary
  binary responses.
- It will not inline audio or video bytes in default HTML exports.
- It will not expose arbitrary absolute server paths for local media.
- It will not add a separate media table in the first implementation slice.
- It will not treat PDF extracted text, OCR output, table extraction, or page
  screenshots as sources. They are read results or derived artifacts tied back to
  the original PDF source.

## Current Code Constraints

The current storage model already has the correct basic split:

- `SourceSnapshot` is the source anchor for a mission.
- `RawArtifact` is immutable stored bytes with media type, size, SHA-256,
  storage URI, filename, producer, created time, and content.
- `SourceSnapshot.ArtifactIDs` links one source snapshot to one or more raw
  artifacts.
- `SourceSnapshot.Locators` is JSON and can carry connector-specific structured
  locators.
- `SourceSnapshot.Access.License` and `RetrievalPolicy` already represent
  source access policy.

The current HTTP URL fetcher is intentionally text-only. Media ingestion must
therefore use a separate media fetcher path that reuses the same network safety
policy, not a relaxed version of the text fetcher.

The current `live_reference` validation is `local_path`-specific. To support
metadata-only audio/video references, the implementation must extend
`live_reference` validation into connector-specific validators while preserving
the current local path rule: local path sources never expose absolute server
paths.

## Source Model

Use the existing tables for the first slice:

- `plasma_source_snapshots`
- `plasma_source_snapshot_artifacts`
- `plasma_raw_artifacts`

Do not add a `media` table until there is a concrete query or indexing need that
cannot be handled by typed locators and source summaries.

Recommended connector types:

- `media_url`: generic HTTP/HTTPS media URL.
- `pdf_url`: HTTP/HTTPS PDF document URL.
- `commons`: Wikimedia Commons API-backed media source.
- `embed`: metadata-only provider embed source.

Recommended retrieval policies:

- `snapshot_only`: pinned bytes exist in `RawArtifact`. This is the default for
  accepted images and PDF URL sources.
- `live_reference`: source points at an external origin and stores no media
  bytes. This is the default for audio and video.

For live media references, `ContentHash` should remain
`{Algorithm:"none", Value:""}` unless bytes are explicitly pinned. If the
implementation records an observation such as `ETag`, `Last-Modified`, or
`Content-Length`, it should be stored as observation metadata, not as a pinned
content hash.

## Media Locator Shape

Store media-specific metadata inside `SourceSnapshot.Locators` as a typed JSON
object or a one-item array. The initial shape should be stable and explicit:

```json
{
  "locator_type": "media",
  "media_kind": "image",
  "provider": "wikimedia_commons",
  "provider_asset_id": "File:Example.jpg",
  "canonical_url": "https://commons.wikimedia.org/wiki/File:Example.jpg",
  "source_page_url": "https://commons.wikimedia.org/wiki/File:Example.jpg",
  "direct_media_url": "https://upload.wikimedia.org/...",
  "thumbnail_url": "https://upload.wikimedia.org/...",
  "embed_url": "",
  "title": "Example",
  "creator": "Creator name",
  "credit": "Credit line",
  "license_name": "CC BY-SA 4.0",
  "license_url": "https://creativecommons.org/licenses/by-sa/4.0/",
  "rights_statement": "Attribution required",
  "attribution_text": "Creator name, CC BY-SA 4.0",
  "mime_type": "image/jpeg",
  "byte_size": 123456,
  "sha256": "artifact sha256 when pinned",
  "width": 1024,
  "height": 768,
  "duration_seconds": 0,
  "observed_at": "2026-06-26T00:00:00Z",
  "etag": "",
  "last_modified": ""
}
```

Field rules:

- `media_kind` is one of `image`, `audio`, or `video`.
- `canonical_url` is the human-facing source URL used in Markdown and public
  citation text.
- `direct_media_url` is the byte URL used by the media fetcher when bytes are
  pinned.
- `sha256` is filled only when the binary has been pinned as a raw artifact.
- `license_name`, `license_url`, and `attribution_text` are renderer-owned
  provenance fields. Agent-generated text must not overwrite them.
- Missing or uncertain license data must render as `unknown`, not disappear.

`SourceSnapshot.Access.License` should mirror the coarse license value for
source-level filtering. The locator keeps the renderable attribution details.

## PDF Locator Shape

PDF URL sources use a document locator, not the media locator:

```json
{
  "locator_type": "pdf_document",
  "url": "https://example.com/paper.pdf",
  "fetched_at": "2026-06-27T00:00:00Z",
  "mime_type": "application/pdf",
  "byte_size": 1234567,
  "sha256": "artifact sha256",
  "page_count": 12,
  "text_length": 45678,
  "extraction_support": "pdf_text"
}
```

The original PDF remains the source. Extracted text is a read result that helps
agents navigate the document. If later OCR, page images, table extraction, or
figure inspection are added, those outputs must remain derived observations or
artifacts linked back to the PDF source.

## Fetching Policy

The media fetcher should reuse the secure URL source HTTP client behavior:

- no proxy use
- redirect cap
- response header cap
- request timeout and response-header timeout
- DNS resolution checks
- rejection of loopback, private, link-local, multicast, unspecified, and
  CGNAT addresses

It should use an explicit media allowlist:

- image: `image/png`, `image/jpeg`, `image/gif`, `image/webp`
- audio: `audio/mpeg`, `audio/ogg`, `audio/wav`, `audio/mp4`
- video: `video/mp4`, `video/webm`

SVG is rejected in the first slice. SVG can contain scriptable content and
should require a separate sanitizer and render policy before being accepted.

Initial size policy:

- image hard cap: 10 MiB
- PDF URL hard cap: 100 MiB, pinned by default
- audio hard cap: 30 MiB, but not pinned by default
- video hard cap: 100 MiB, but not pinned by default

The implementation should also track per-mission media byte usage and warn
before media causes the SQLite database to grow unexpectedly. The current server
database already stores raw artifact content in SQLite blobs, so image pinning is
acceptable for the first slice while audio/video pinning should remain opt-in and
limited.

## Storage Policy

Pinned image:

1. Fetch metadata.
2. Fetch bytes through the media fetcher.
3. Create a `RawArtifact` with the image media type, filename, SHA-256, byte
   size, and original bytes.
4. Create a `SourceSnapshot` with `connector_type=media_url` or `commons`,
   `retrieval_policy=snapshot_only`, a media locator, and one artifact ID.
5. Append a normal source snapshot event.

Audio/video default:

1. Fetch or resolve metadata only.
2. Create a `SourceSnapshot` with `connector_type=media_url`, `commons`, or
   `embed`, `retrieval_policy=live_reference`, no artifact IDs, and a media
   locator.
3. Record `ETag`, `Last-Modified`, `Content-Length`, and provider metadata when
   available.
4. Do not store the media bytes unless the user explicitly chooses a future
   pinning option.

Re-adding the same accepted media should dedupe by normalized canonical URL when
possible and by mission plus SHA-256 when bytes are pinned. If SQLite's current
`(mission_id, sha256)` unique index finds an existing artifact, the source
creation path should reuse the existing artifact instead of surfacing a
constraint failure to the user.

PDF URL default:

1. Fetch the PDF through the secure URL source HTTP client policy.
2. Reject non-PDF content and files over the PDF cap.
3. Create a `RawArtifact` with `media_type=application/pdf`, filename, SHA-256,
   byte size, and original bytes.
4. Create a `SourceSnapshot` with `connector_type=pdf_url`,
   `retrieval_policy=snapshot_only`, a PDF document locator, and one artifact ID.
5. Validate PDF structure and basic metadata during fetch. Do not extract the
   full text during ingest; bounded text extraction happens at read time so large
   PDFs do not force unnecessary memory or token use.

## MCP Policy

MCP tools should expose media as source metadata, not binary payloads.

`plasma.research.read` on a media-bearing source should return:

- source snapshot metadata
- media locator metadata
- artifact metadata when pinned
- canonical URL
- attribution and license
- dimensions or duration when known
- render policy hints

It should not return base64 image bytes, audio bytes, or video bytes inline. If
an agent needs to inspect the image itself, that is a later multimodal tool
surface, not the default text MCP read.

`plasma.sources.read` and `plasma.research.read` on a PDF source should return
bounded extracted chunks:

- source snapshot metadata
- raw artifact metadata
- bounded extracted text
- offset, next_offset, content_length, and truncation metadata
- extraction metadata such as `type=pdf_text` and `page_count`

They should not return raw PDF bytes inline. If a PDF is scanned or image-heavy,
the correct follow-up is a future OCR/page-inspection tool, not prompt-stuffing
the PDF bytes.

The intended inspect workflow is metadata-first:

1. The agent lists or reads media source metadata through the normal research
   tools.
2. The agent decides whether the current user question requires visual or
   scan-derived observations.
3. If metadata is enough, the agent answers without inspecting bytes and says
   that the media itself was not visually inspected.
4. If the task requires what the image or scan actually shows, the agent calls a
   future explicit inspect tool with source snapshot IDs or artifact IDs.
5. The inspect result is stored as a result/observation linked to the original
   source. It is not a new source and does not overwrite provider metadata.

Good inspect triggers include map/portrait/diagram description, screenshot or
image-only PDF analysis, document handwriting, seals, layout, visible damage,
and checking whether a provider page or metadata description matches the actual
image. Non-triggers include license checks, attribution checks, file inventory,
and cases where a provider transcription is sufficient for the user's question.

Before adding that multimodal surface to the product, run the media inspection
experiment recorded in
`experiments/media-inspect-2026-06-26/README.md`. The experiment compares the
metadata-only default against explicit image inspection and document OCR/vision
inspection. A successful inspection tool must improve visual or scan-based
research without weakening the source/result boundary: the original media stays
the source, while OCR, captions, and visual observations remain tool-produced
results with provenance back to the source.

`plasma.research.list` and `outline` should mark media sources clearly enough
that an agent can discover them:

- `media_kind`
- title
- provider
- pinned/live reference state
- license
- byte size
- dimensions or duration

## Report Rendering Policy

Markdown:

- Use the canonical URL for image syntax when an inline image is useful:
  `![alt](canonical_url)`.
- Add a compact attribution line under the media block.
- Render audio/video as title, link, provider, duration when known, and
  attribution.
- Never use data URLs in Markdown.

Interactive HTML:

- In self-contained mode, inline pinned images as
  `data:<mime>;base64,<bytes>` when the image is under the cap and the media
  type is allowlisted.
- Include `alt`, caption, attribution, source URL, license, and source snapshot
  reference in the figure metadata.
- If an image cannot be inlined, render a linked fallback and explain why in a
  non-noisy metadata line.
- Render audio/video as links by default.
- Allow iframe embeds only for a hard provider allowlist such as Wikimedia,
  YouTube, and Vimeo. Use sandboxing, `referrerpolicy`, and canonical embed URL
  generation. Reject generic third-party iframe HTML.
- Never inline audio/video binaries in the default self-contained export.

Report artifacts:

- Markdown remains the primary report artifact in the current C1 product path.
- Self-contained HTML is an export/rendering artifact derived from the Markdown
  report plus source/artifact metadata.
- Designed HTML is also derived from the Markdown report artifact. It may store
  an internal JSON content model before rendering, but that model is not a
  source and is not the primary user-facing report artifact.
- Designed HTML content models may place pinned image sources into relevant
  sections by stable local image references such as `image_1`. The model must
  not carry image bytes; the renderer resolves references back to source
  snapshots/raw artifacts and leaves unused images in the media panel.
- The HTML export may contain copied image bytes, but that does not make the
  HTML a source. It remains a report artifact.

## User Interface Workflow

The browser should eventually expose media without crowding the default source
flow:

1. User opens the source panel and chooses URL/media add.
2. Plasma previews the resolved media metadata, license, byte size, and whether
   it will be pinned or kept as a live reference.
3. User accepts the media source.
4. The source appears in the source list with media type, provider, license, and
   pinned/live status.
5. The user can remove or restore the source like existing sources.
6. Report download offers Markdown and interactive HTML. The HTML option should
   say when images are self-contained and when media is linked.

## Implementation Waves

1. Done: add typed media locator DTOs and validation helpers in the app layer.
2. Done: add a media fetcher that reuses the secure URL fetch boundary but has
   an explicit media MIME allowlist and size caps.
3. Done: add `media_url` source attach support for pinned PNG/JPEG/GIF images
   and live-reference audio/video metadata.
4. Later: add a `commons` connector for metadata-first Wikimedia Commons
   collection.
5. Done: extend source snapshot validation so `live_reference` is
   connector-specific. `local_path` keeps its current root/relative-path rule,
   while media URL connectors can hold external canonical URLs without raw
   artifacts for audio/video.
6. Partial: extend Web source attach/read surfaces to show media metadata and
   removal/restore status. CLI source attach/read remains a later surface.
7. Done: extend MCP source read/list/outline to expose media metadata without
   binary content.
8. Partial: add report rendering support for self-contained image HTML and
   media-aware Markdown/HTML attribution. Self-contained and Designed HTML can
   inline accepted image sources within size caps; richer Markdown media
   authoring and audio/video embeds remain later work.
9. Partial: add tests for pinned image storage, metadata-only audio/video, and
   MCP no-binary responses. MIME rejection, SVG rejection, size cap, duplicate
   artifact reuse, and self-contained image rendering need follow-up tests as
   those surfaces are expanded.

## Current Product Slice

The first product slice implements metadata-first media sources without a
vision engine:

- HTTP/HTTPS PNG, JPEG, and GIF image URLs are accepted as `snapshot_only`
  media sources backed by raw artifacts.
- HTTP/HTTPS audio and video URLs are accepted as `live_reference` media
  sources with metadata only; Plasma does not pin audio/video bytes.
- Browser and MCP source reads return media metadata and artifact metadata only.
  They do not return image/audio/video bytes.
- `plasma.research.inspect_image` is a reserved MCP name only. It is not
  registered in `tools/list` until a real vision engine exists.
- Audio/video inspect is not defined. Transcript, keyframe, OCR, or waveform
  extraction are separate future source/result pipelines.

## Acceptance Criteria

- A user can add an HTTP/HTTPS image source and Plasma stores it as a source
  snapshot backed by a raw image artifact.
- The same image added twice to the same mission does not produce a user-facing
  unique constraint error.
- A user can add an audio or video URL as a metadata/live-reference source
  without pinning the binary by default.
- Markdown reports preserve original media URLs and attribution.
- Self-contained HTML reports inline accepted images as data URLs and do not
  inline audio/video bytes.
- MCP read/list surfaces describe media sources without returning binary bytes.
- Removed media sources are excluded from default report generation and remain
  inspectable through explicit audit controls.

## Open Decisions Before Coding

- Exact per-mission media byte warning and hard thresholds.
- Whether to generate and store thumbnail artifacts for pinned images in the
  first slice.
- The initial provider embed allowlist and canonicalization rules.
- Whether audio/video pinning should exist at all before raw artifact storage
  moves from SQLite blobs to a filesystem-backed artifact store.
