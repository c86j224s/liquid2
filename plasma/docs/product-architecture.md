# Plasma Product Architecture

Plasma is planned as an independent research product inside the workspace. This
document records the boundary decisions that are stable before runtime design
starts.

## Product Identity

Plasma should help a user build grounded research reports by steering an
investigation through conversation. The C1 default product loop is mission,
same agent session, user/controller steering, MCP/source read tools,
conversation results, and report artifacts.

Historical evidence, claims, confidence updates, proposals, and AST-first
reports are legacy ledger machinery. Plasma keeps their tables and read paths
for migration and experiment work, but they are not exposed as the default
product loop and must not become a user-facing old/new mode toggle. Source
candidate review records are allowed in bounded workflow runs only as user
approval prompts; they are not sources and do not create snapshots by
themselves.

The browser UI is one client over Plasma, not the product center. Plasma should
also work as a UI-less research IDE through MCP: a short guidance surface tells
agents how to work, while mission overview, search, random-seek reading,
reference traversal, and report drafting happen through tools over the existing
ledger.

External autonomous-research product and paper scans support this direction:
modern deep-research systems emphasize planning, retrieval, tool use, source
checking, and cited synthesis over a single large prompt. They do not imply that
Plasma should add a strong always-on controller. The 2026-06-26 C0/PAL2/NAV
experiment rejected NAV as a default and left PAL2 inconclusive, so controller
behavior remains telemetry-backed, weak, and conditional until a specific
failure mode is validated.

## Implementation Layer Shape

Start with the [Architecture Map](architecture/README.md) to locate capability
owners and the [Package Boundaries](architecture/package-boundaries.md) to
decide whether an import or package shape is allowed. Those documents are the
normative implementation map; this document records the detailed product and
feature boundaries below.

The Go package is the primary independent implementation unit. Technical
umbrellas such as Web, MCP, CLI, reporting, or SQLite must still be partitioned
when their product features have different contracts or reasons to change.
Issue #66 tracks the current broad `internal/app`, `internal/web`,
`internal/mcp`, and `internal/reporting` boundaries. They are migration debt,
not precedent for new work. SQLite persistence has a stable root facade for
connection and cross-capability transaction ownership, with SQL split into
root-only feature repositories.

## Browser Frontend Composition Boundary

The browser remains a replaceable Plasma client over the mission ledger and HTTP
surface. Its frontend composition keeps classic scripts and uses one browser
namespace, `window.Plasma`, as the only shared global root for new modules. New
shared browser modules live under `internal/web/static/plasma/*.js` and attach
only named submodules to that root.

The foundation owners are explicit:

- `Plasma.dom` owns pure selectors and text formatting only: `$`, escaping,
  short IDs, timestamps, and deterministic byte formatting. It owns no product
  state, modal policy, clipboard behavior, network behavior, or feature markup.
- `Plasma.state` owns the existing flat browser state object without changing
  keys, defaults, or semantics.
- `Plasma.ui` owns genuinely shared visual controls and browser effects. Its
  public API is one object, while the implementation is split physically:
  `ui.js` initializes the namespace and owns count chips, empty-state markup,
  section-empty toggling, generic disabled mechanics, and button text;
  `ui_feedback.js` extends the same object with the error toast and clipboard
  copying; `ui_detail.js` extends it with the detail modal, backdrop close, copy
  behavior, scroll position ratio, and generic `data-detail-json` handling; and
  `ui_tabs.js` owns the shared tab shell. The detail modal exposes a hook
  boundary so feature owners can preserve behavior such as report redpen
  before-leave checks and edited copy content without moving that report-specific
  policy into shared UI. It owns no mission
  lifecycle, pending-state, active-work, executor/model, workflow, report,
  source, conversation, or claim-confidence policy, and no feature-specific body
  markup.
- `Plasma.mission` owns mission load, create, select, reload, archive, restore,
  hard-delete, mission list/lifecycle rendering, local selection storage, mission
  metadata editor wiring, mission-switch transient reset, and mission artifact
  preview URL construction. It owns selection/detail generations, captured owners,
  stale ownership checks, reusable begin/clear transitions, selected-detail
  application, and selected detail reload. It calls explicit composition
  callbacks only for cross-owner effects such as full detail rendering, form
  disabling, error display, and the generic mission transition hooks
  `beforeSelectionChange(currentMissionId, nextMissionId)` and
  `afterSelectionApplied(owner)`. Those hooks are neutral extension points:
  `app.js` composes the existing report before-leave guard and post-selection
  source refresh order, while `Plasma.mission` remains unaware of report redpen
  and Confluence policy. It does not own report redpen policy, source feature
  rendering, workflow policy, or conversation policy.
- `Plasma.transport` owns the browser HTTP helpers, including mission-scoped
  request helpers and their existing response/error behavior.
- `Plasma.polling` owns the two existing recursive `setTimeout` polling loops:
  the 2000 ms selected pending poll and the 3000 ms observed mission activity
  poll. It owns timer and in-flight fields, poll owners, activity cursor parsing
  and storage, mission activity seen watermarks and pruning, non-regressing
  mission activity merge mechanics, stale-owner gates, document-hidden skip
  behavior, refresh scheduling, and the existing bounded selected-detail fallback
  decision. It receives the selected-pending predicate explicitly from `app.js`.
  Bootstrap supplies health-badge success/failure callbacks. `Plasma.polling`
  does not render active-work notices or feature markup.

`app.js` remains served at `/static/app.js` and remains the final browser
composition root. It loads after the foundation scripts, shared UI scripts,
feature owners, `Plasma.reports`, and `Plasma.bootstrap`. It keeps only
cross-owner composition: mission-required validation, whole-detail rendering,
form enablement and blocking predicates, active-work action routing,
`runBulkSequential`, explicit owner configuration, selected-pending derivation,
generic mission transition callback composition, and `Plasma.bootstrap.start`.
The mission transition composition first asks the report redpen controller before
switching between two non-empty different missions, then after detail application
loads Confluence connections, re-checks detail ownership, and loads Confluence
access. Moved implementations must not remain in `app.js` as duplicate
functions, classes, wrappers, or proxy shims.

`Plasma.conversation` owns conversation-specific browser mechanics: sending and
canceling turns, resetting agent sessions, agent executor/model/reasoning
controls and summaries, active-work notice rendering from projected state,
turn rendering, steering placement, terminal badges, copy controls, and turn
navigation. It uses `Plasma.mission` and `Plasma.transport` for mission capture
and request execution, and it does not own polling timers, in-flight polling
flags, activity cursors, stale polling transitions, refresh scheduling, or
selected-detail fallback. The public surface remains one `Plasma.conversation`
object, while focused classic scripts extend that object by concrete role:
conversation actions, agent state/model/control/session presentation,
active-work notices, turn state, turn rendering, and turn navigation. Cross-
feature blocking decisions are supplied by `app.js` callbacks.

`Plasma.workflow` owns workflow-specific browser mechanics: workflow raw input,
goal draft, start, stop, continue, busy controls, run and step rendering, status
and decision labels, and workflow-list events. The composition root supplies the
explicit raw-input fallback from the conversation composer so the existing
behavior remains `workflowInstruction` first and `turnText` second; this is a
composition dependency, not a new product rule. Workflow does not own polling.
The public surface remains one `Plasma.workflow` object, while focused classic
scripts extend that object for workflow actions, input/goal-draft/busy controls,
and run/status rendering. Cross-feature blocking decisions are supplied by
`app.js` callbacks.

`Plasma.sources` owns source intake, saved-source rendering and locators, source
candidate rendering and source-candidate bulk actions, local-path source
controls, Liquid2 source controls, and mission-scoped Confluence source UI. Its
Confluence files are split by role: actionable error mapping, connection/site
core, common source controls, URL/search flow, one-click flow, OAuth listener,
mission access, browse fetch/rendering, review/approval, update checks, and
result click handling. It uses explicit `Plasma.dom`, `Plasma.state`,
`Plasma.transport`, `Plasma.mission`, and `Plasma.ui` dependencies plus
composition callbacks from `app.js`; it does not own cross-feature form blocking,
mission lifecycle policy, active-work policy, evidence proposal selection, or
report request-time model overrides.

`Plasma.settings` owns global settings UI for persisted model defaults and
Confluence connection management. Model-default settings remain separate from
report request-time model selection. Confluence settings are split between
rendering and actions, and consume `Plasma.sources` Confluence connection
helpers rather than bare `app.js` globals.

`Plasma.proposals` owns evidence-proposal submission and decisions, proposal
queue/detail rendering, candidate source option rendering, selection state,
bulk actions, and extraction status markup. It uses `Plasma.mission`,
`Plasma.transport`, `Plasma.ui`, and explicit `app.js` callbacks for
mission-required validation, errors, and sequential bulk execution.

`Plasma.knowledge` owns `EVIDENCE_TYPE_LABELS`, saved evidence and saved claim
rendering, approved evidence/claims projection, and claim-confidence list,
detail, badge, history, and chip rendering. It uses `Plasma.ui` for the shared
detail modal shell instead of owning modal close/copy behavior.

`Plasma.ledger` owns read-only ledger rendering, event labels, and event times,
and uses `Plasma.ui` for generic detail display.

`Plasma.reports` owns report-specific browser controls, request payload assembly,
draft/cancel/patch actions, report list/state/trace/notice/timing rendering,
pipeline graph and retry presentation, export/view/download actions, conversation
export, report modal content, model selection, direction hints, Markdown/basic
HTML/designed HTML/H5/redpen actions, math/Mermaid/image enhancement, and the
redpen controller. It uses `Plasma.ui` for the modal shell, close behavior,
detail scroll ratio, detail copy, clipboard, and error toast instead of
reimplementing those shared behaviors.

`Plasma.bootstrap` owns DOMContentLoaded setup, feature module configuration,
event binding, local browser chrome initialization, and initial load calls.
`app.js` supplies explicit callbacks for active-work pending derivation,
report/conversation/workflow mutual blocking, mission lifecycle blocking,
active-work action routing, shared `runBulkSequential`, and residual product
composition. Feature modules render projected state and use shared APIs rather
than reimplementing cross-feature behavior.

CSS follows the same composition boundary while keeping the linked stylesheet
entry contract scoped to `/static/app.css`. That entry remains the stable
stylesheet linked by `index.html`, but it is an import-only manifest over
source-order ownership segments under `internal/web/static/plasma/*.css`. The
imported files are contiguous slices of the original cascade: their names
describe the visual responsibility of that source range, not a license to
regroup selectors across earlier or later rules. The late cross-owner responsive
overrides remain the final imported CSS file so their original priority is
preserved. The CSS source reconstructed in import order remains byte-for-byte
stable, while ten new subordinate `/static/plasma/*.css` URLs and browser
resource requests are introduced; partial-load failures and request count
therefore belong to the normal static-asset serving contract rather than a
concatenation fallback. This split is ownership documentation and maintenance
structure only; it is not a redesign, cascade-layer migration, build-step
introduction, minification pass, selector rewrite, or linked stylesheet URL
rename.

Non-goals are unchanged: no visual redesign, no backend/API/MCP/CLI/database
change, no framework, bundler, ES-module, package, or TypeScript migration, no
static URL rename, no DOM ID, data-attribute, or localStorage contract change,
and no product policy or retry/recovery change.

## Storage Boundary

Plasma owns its own database and domain model. The following are not allowed:

- storing Plasma mission state in Liquid2 document tables
- direct SQLite reads from Liquid2
- cross-database foreign keys
- cross-database joins as a product dependency
- direct imports of Liquid2 Go internals

Liquid2 can be integrated only as a source connector or external API provider.

## Mission Ledger

### Explicit mission metadata editing

Current mission metadata is edited through the single `UpdateMissionMetadata` application service. Web `PATCH /api/missions/{id}`, CLI `missions update`, and the mission-bound idempotent MCP tool `plasma.mission.update` are adapters over that service. A successful user edit appends one sparse `mission.metadata.updated` event containing only the supplied `title`, `objective`, and whole `scope` fields. Supplied values win independently by ledger sequence; omitted fields remain unchanged. An empty supplied objective clears it, an empty supplied scope clears both lists, and a blank supplied title is invalid.

The MCP mutation is available to an explicit user-controlled MCP client, but is excluded from the default tool allowlist of Plasma-spawned research agents. This keeps the event user-owned instead of allowing an agent to impersonate a user edit.

The ledger remains authoritative and `plasma_missions` remains a rebuildable projection cache. Explicit editing does not rewrite earlier events and is distinct from conversational `mission.steered`; its producer ownership and conflict semantics are unchanged. Existing ledgers without metadata events remain compatible.

### Mission archive and restore

Mission archive is a mission-level soft delete. Web and CLI adapters append
user-owned `mission.archived` and `mission.restored` events through the
application service; projection replay maps them to `active` or `archived`
lifecycle state. Default mission lists include only active missions, while an
explicit include-archived request shows both states. Archive and restore do not
delete ledger events, source snapshots, saved knowledge, reports, raw artifacts,
or provider-session records.

Plasma has one durable Mission Ledger. User-driven turns, bounded workflow runs,
MCP tool calls, and report requests are event producers over the same ledger:

- User-driven turns record user direction, constraints, questions, corrections,
  and approval decisions.
- Bounded workflow runs record requested, started, per-step, stop-requested,
  paused, completed, stopped, failed, and interrupted events. Each workflow step
  reuses the normal conversation path with a `workflow_steering` user turn and
  an agent result; it does not own a separate mission state.
- MCP tool calls record bounded trace events for mission-bound research and
  workflow control operations.
- Report requests record pending, artifact-created, or failed events and save
  default reports as Markdown artifacts. A logical report-run projection in the
  same SQLite database uses the root `report.draft.pending` event ID as the run
  identity and stores only run state, revision, final artifact link, membership
  identities, ownership roles, and a compact usage aggregate. Ledger payloads
  and artifact bodies remain single-source in the mission ledger and raw
  artifact tables.
- Report attempts use `report.draft.pending` event IDs as durable identities.
  The read model replays pending, plan, section, section evidence-gap, part,
  artifact, and terminal events into discrete pipeline states. A failed retry appends a new pending
  attempt with origin/parent lineage and a durable request ID; it never reopens
  or mutates the failed attempt. `resume_failed` may read only validated
  completed artifacts from its own ancestor chain, while `restart` starts with
  no ancestor output. Section and part failures have additive safe failure
  events; prompts, source bodies, provider responses, credentials, and provider
  session identifiers are not transport error content.
- Long-running report work goes through a shared report runner boundary used by
  browser, CLI, and export surfaces. The runner owns pending/failure events,
  mode defaults, and in-flight ownership; surfaces supply an executor and request
  work through that boundary instead of owning report policy. A pending report
  draft or designed HTML export may be resumed after restart by reattaching a
  runner to the same pending event; long-form drafts reuse existing plan,
  section, and part artifacts before continuing. The current in-flight ownership
  registry is process-local and assumes one report runner process per database;
  multi-process deployment must add a ledger-backed report-run lease before
  parallel server instances share the same Plasma database.
- Completed canonical Markdown report artifacts have an explicit preview/delete
  flow. Deletion is blocked for active work, ambiguous legacy lineage, open
  pending attempts, any malformed out-of-run ledger payload in the same SQLite
  DB, or unclear external references. A successful delete removes member report events and run-owned
  unshared intermediate/final/derivative artifacts, preserves inputs and shared
  artifacts, and leaves one purged report-run tombstone with mission identity,
  run identity, purge metadata, revision, lifecycle state, and compact token
  usage only. Mission hard delete removes report-run rows and tombstones inside
  the mission-wide transaction.
- Source lifecycle and observation events are ledger-backed. `source.removed`
  and `source.restored` project active/removed state without deleting source
  rows or raw artifacts. `source.observed` records bounded read/tree/grep
  metadata for mutable live sources.

No producer owns a separate source of truth. Workflow status is a projection
from ledger events, not a durable mode flag or a separate workflow table in the
first implementation slice.

The ledger is also the shared substrate for replaceable clients and adapters.
The browser UI, agent provider, search backend, and report renderer should be
replaceable components over the same ledger and MCP contract rather than owners
of separate state.

## Agent Provider Boundary

Agent providers are replaceable adapters over the same mission ledger and MCP
surface. The first provider-backed action in a mission currently locks that
mission to one provider type, such as Codex or Claude. Later requests for the
same mission must use the locked provider and must fail before invoking another
provider. This keeps provider session identity, resume behavior, and report
forking understandable while preserving the existing `agent_executor` event
payload for future mixed-provider work.

The provider lock is derived from ledger events rather than a separate schema
field. Source-only events, source candidates, and non-provider administrative
events do not lock a mission. Browser, CLI, workflow, and report surfaces must
all route through the same provider lookup and lock validation so a provider
switch cannot happen through a secondary entry point.

## Source Modes And Local Path Connector

Connector and source are separate axes. A connector is an adapter for reaching an
external origin, such as Liquid2, Confluence, or eventually a settings-managed
local filesystem root. A source is mission research material accepted or staged
inside Plasma, such as a URL, PDF, uploaded file, Liquid2 document, Confluence
page, or local path file/directory. A connector may discover or fetch source
material, but it is not itself the source.

Source registration normally creates or reuses raw artifacts, creates a mission
source snapshot when the user accepts the material, and records the action in
the mission ledger. Candidate staging may create a raw artifact before approval,
but that staged artifact remains candidate-only until the user promotes it to a
source snapshot.

Plasma source snapshots share one model across Web, CLI, MCP, and agent tools.
The persisted retrieval policies are:

- `snapshot_only`: canonical pinned source policy. The snapshot references one
  or more raw artifacts stored by Plasma and is the default for pasted text,
  browser/CLI file uploads, fetched URL content, and Liquid2 snapshots. File
  uploads use the `file_upload` connector type for provenance, while their
  locator `locator_type` describes the content shape (`full_document`,
  `pdf_document`, or `media`). The locator records original/sanitized filename,
  MIME type, byte size, SHA-256, upload time, and content kind. Duplicate
  uploads within a mission reuse the existing raw artifact by content SHA while
  creating a new source snapshot/event.
- `live_reference`: mutable source policy for `local_path` in the first
  implementation. The source stores no raw artifact body and uses
  `ContentHash{Algorithm:"none", Value:""}` rather than pretending that an empty
  artifact list has a content hash.

The `local_path` connector stores only a locator shaped like `root_id`,
`relative_path`, and `path_kind`. Configured root absolute paths remain
server-side configuration and must not appear in source snapshots, Web JSON,
MCP responses, CLI output, prompts, or reports. All local path access goes
through the local path engine, which canonicalizes configured roots, rejects
absolute paths and traversal, rejects symlinks and special files, applies deny
patterns and caps, and returns public DTOs with only root IDs and relative paths.
Agent reads are source-scoped: after a user accepts a live local path file or
directory as a source snapshot, the default MCP surface addresses it by
`snapshot_id` plus optional `subpath`. The default agent surface may read, tree,
or grep inside that accepted source boundary, but it does not expose root-wide
`root_id` browsing or arbitrary `root_id` plus `relative_path` reads.

Live local path reads, greps, and directory trees append `source.observed`
events with operation metadata: observed time, root alias, relative path,
optional subpath, file kind, size, mtime, sha256 when bytes were read, read
range, truncation/cap state, producer/session provenance, and best-effort git
metadata. These events are observation records, not new sources and not legacy
evidence records.

Source removal is soft by default. Removed sources are hidden from default
lists, reads, research/reporting, and workflow use, but remain visible with an
explicit audit option such as `include_removed`. Re-adding the exact same
removed local path source requires explicit restore and reactivates the existing
source identity instead of creating duplicate active rows. Physical purge or
redaction is an admin follow-up boundary, not normal Web/MCP/CLI behavior.

Media and document sources follow the same source snapshot boundary. The media
direction is documented in `media-source-implementation-design.md`: images may
be pinned as raw artifacts and embedded into self-contained interactive HTML
exports, while audio/video default to metadata/live-reference links or
allowlisted provider embeds. PDF URL sources are document snapshots: Plasma
pins the original PDF bytes, stores metadata such as page count, extraction
support, and `text_length_known=false` at ingest, and returns bounded extracted
text through source read tools
instead of raw PDF bytes. Generated captions, report renderings, thumbnails,
PDF extraction text, and alt text are results or artifacts, not sources.

## MCP Research IDE Surface

The MCP-first surface should remain narrow and retrieval-oriented:

- `plasma.research.outline`: whole-mission overview of goals, scope, open
  questions, result state, and report artifact state, with the latest ledger
  sequence for later change checks.
- `plasma.research.changes`: bounded summaries of meaningful mission changes
  after a caller-owned ledger sequence. Execution telemetry and report events
  stay out of this feed; an invalid future cursor requires a fresh outline.
- `plasma.research.list`: discovery across sources, evidence, saved knowledge,
  raw artifacts, conversation results, ledger events, and report artifacts by
  default. Legacy claim/report-block object kinds require an explicit legacy
  boundary.
- `plasma.research.read`: direct reading of a specific source, evidence item,
  saved knowledge item, report artifact, raw artifact, or ledger event, with
  range support for long bodies. Agent results are read through ledger events;
  they are not reclassified as sources.
- `plasma.research.grep`: case-insensitive literal substring search over ledger
  content, pinned source snapshots, and live local path sources. The entire
  query must be contiguous, separate concepts should be split into separate
  short searches, and all non-overlapping matches found within each retrieved
  candidate are returned through the existing cursor and limit pagination.
  External connector search remains a separate possible original material
  discovery route.
- `plasma.research.references`: graph traversal among sources, evidence, saved
  knowledge, results, and report artifacts by default. Legacy claim/report-block
  references remain behind explicit legacy access.

A guide, prompt, or helper tool may explain this workflow, but it must stay
thin. It must not duplicate source/evidence/saved-knowledge/report data into a
large prompt, a report-only corpus, or a prebuilt report pack. Search results and
snippets are candidates; report statements must be grounded by explicit source
reads or, when the optional evidence layer is active, saved evidence that points
back to original sources. When a statement depends on live local path material,
the report should cite the human locator and observation metadata rather than
only the source ID.

Mission-bound MCP calls are observable product events. Plasma records them as
`mcp.tool.called` ledger events with tool name, timing, success state, bounded
argument summary, and bounded result summary. This gives the browser and UI-less
clients a way to debug whether an agent actually used outline/list/grep/read/
references, without copying source bodies into the prompt or creating a
separate report-only corpus.

## Product Surfaces

The implementation slices should share the same ledger and MCP contract:

- create a research mission
- mount or snapshot sources, including Liquid2 through a connector boundary and
  media sources through explicit media connectors
- accept steering directives from conversation or an MCP client
- record controller steering strategy selection as an observable ledger event
  without treating any controller strategy as a validated default product
  controller
- start, inspect, and stop bounded workflow runs
- keep agent answers and controller outputs as results, not sources
- draft reports through thin guidance plus MCP/source reads, not large mission
  recall JSON injection
- expose both a planned report mode and a slower Part/Section long-form report
  mode over the same Markdown artifact model
- use the adopted F4 and visual-planning guidance together with a small writing
  contract that records the central question, reader takeaway, reading path,
  must-keep details, and compressible or supporting material. Writers digest the
  sources and explain the subject directly to a reader who may not read them,
  without leaking prompt, run, session, or temporary-path internals
- treat that contract as a common baseline under the existing visual-planning,
  section-centered, and richer section-centered choices, not as a fourth user
  option
- save default reports as Markdown artifacts
- export self-contained interactive HTML from report artifacts by embedding
  pinned images when policy allows, while keeping audio/video linked or embedded
  through allowlisted providers
- expose designed HTML exports through a replaceable deterministic renderer
  adapter. The current product slice follows the 2026-06-28 DH23-style
  content-model path and the 2026-07-05 visual-grammar update: the selected
  agent creates a JSON content model from the Markdown report artifact, Plasma
  stores that model as an internal rendering artifact, and the renderer promotes
  the strongest visual unit into a compact first-viewport connected relationship
  map before dispatching later visual units to deterministic timeline,
  evidence-chain, dependency-path, trade-off matrix, loop, or relationship-map
  renderers. The output remains a self-contained HTML report artifact.
  This is not final reference-grade parity: the renderer still depends on a
  compact content model and must preserve source notes, caveats, URLs, and
  long-text readability over decorative variety. The content-model generation
  prompt instructs the agent to preserve every source `\(...\)` and `\[...\]`
  expression exactly, including delimiters, in a relevant visible body or table
  field, rather than rewriting, translating, inventing, or preserving formulas
  only inside SVG text. This instruction is not deterministically validated by
  the current coverage: QA preserved and rendered 23 of 41 expressions.

## Implemented Browser Workspace Slice

The current browser workspace is a local testing surface over the Plasma-owned
runtime. It can create missions, record user turns, run a Codex-backed agent
turn when explicitly configured, snapshot pasted text, snapshot HTTP/HTTPS
textual URL sources, attach allowlisted local path files/directories as live
references, attach Liquid2 documents through the read-only connector,
start and stop bounded workflow runs through the shared ledger projection,
run non-one-take report requests in a forked report-only provider session when the
executor and mission state allow it, and save generated Markdown reports as raw
artifacts. Browser evidence/proposal/confidence and AST report features are
legacy history/experiment surfaces rather than the default product loop; see
`legacy-ledger-loop.md`.

The default MCP research surface may search connectors, propose source
candidates, read staged unapproved source candidates in bounded chunks, read
accepted sources in bounded chunks, and inspect accepted live local path
directories through source-scoped tree/grep operations. Staged candidate reads
are conversation/research aids only: they must identify the material as an
unapproved candidate and are excluded from normal raw artifact lists and default
report inputs. The default surface does not expose source mutation tools that
promote a candidate or local path into an accepted source, and it does not
expose root-wide local path browsing to agents. Local path root browsing,
attach, source remove, and source restore remain available through explicit
user/operator surfaces such as browser/CLI source commands or an
operator-enabled MCP server.

Duplicate URL source posts reuse an existing source snapshot when the normalized
URL already belongs to the same mission.

URL source fetching is intentionally bounded. The generic URL fetcher only
accepts HTTP/HTTPS textual responses, disables proxy use, applies a 60 second
overall timeout with a 45 second response-header timeout, 5 redirect cap, 64 KiB
response-header cap, and 20 MiB body cap, and rejects resolved loopback, private,
link-local, multicast, unspecified, and `100.64.0.0/10` CGNAT
addresses. Redirected requests are checked through the same DNS and address
policy before connecting. PDF URL sources use a separate `pdf_url` path that
reuses the same network safety policy, pins PDFs up to 100 MiB, validates that
the content is a PDF, and exposes bounded extracted text chunks through read
tools instead of returning raw PDF bytes inline.

Agent turns resume the provider session id from the latest agent response when
one exists. Plasma sends only a short mission reminder and the latest user turn
to that provider session; it does not paste prior turn history or source body
excerpts back into the prompt. This keeps agent-produced answers as results
while sources remain original material such as Liquid2 documents, URLs, files,
PDFs, or external repositories. Source inspection should happen through
available tools/connectors, not by copying every source body into every agent
turn. Report generation should follow the same rule: the report writer receives
thin guidance and performs MCP reads over the ledger instead of receiving a
large injected recall payload.

An optional `direction_hint` is request-scoped report pending state, not mission
state or evidence. After whitespace trimming, a non-empty value is stored only
in the corresponding `report.draft.pending` payload so stale recovery can
reproduce that request; omitted legacy payloads decode to empty and later
requests do not copy the field. Its fixed advisory treats the hint as a weak
editorial axis.

One-take writing and planned planning/writing receive the request direction
directly. For long-form reports, the planner receives the user's original
wording before the plan is frozen and interprets it lightly through the report
structure and `ReportWritingContract`. The direction may adjust emphasis,
interpretation, ordering, and presentation, but it must not reduce the
mission-relevant coverage or depth required by the report objective and
sources. Long-form Section writing, Part planning, Part assembly and editing,
final writing, and reader editing receive both the original wording and the
writing contract. This keeps the user's text available as the authority while
the planner's interpretation gives downstream stages a shared editorial axis.

Normal or resumed conversation, mission reminders, recall, workflows,
deterministic assembly, H5 or pre-canonical style editing, semantic validation,
the evidence gate, report patching, and basic or designed HTML export do not
receive a new direction block. The raw wording remains durable only in the
request's pending event; the long-form plan stores the interpreted writing
contract rather than copying the raw hint into a second product-state field.
This allowlist governs application prompt construction; it does not erase
provider-session history, so a path that deliberately resumes the same provider
session can still retain the earlier report prompt in context.

Agent-backed report generation forks the current
research provider session when possible for every report mode except
`one_take`, keeps report planning and Markdown generation in that report-only
session, and stores returned Markdown as Plasma-owned report artifacts. The
default report path uses the adopted G2 generation-time guidance. Manual
post-canonical H5 export is deprecated in the browser and remains only for
historical artifacts and direct API compatibility. It does not replace the
original artifact and does not participate in planning, source selection, AST
shaping, content-model generation, or Designed HTML rendering. For new
long-form reports, the browser always sends `post_report_humanize=enabled` to
run the pre-canonical style edit plus read-only semantic validation before the
read-only evidence gate; there is no user toggle, and non-long-form browser
requests keep it disabled.
Manual or legacy humanized Markdown export keeps the post-report H5 meaning. The H5 pass
resumes the report session and exposes only the bounded
`plasma.report.patch.*` MCP tools, so it reads the saved Markdown artifact in
slices and applies targeted patch operations instead of pasting the whole report
into a prompt or returning a full rewritten report body. A passing H5 result is
stored as a separate
`humanized_markdown` report artifact export that points back to the original
Markdown artifact and records `humanize_transport: mcp_patch`; agent failures,
context cancellation, missing MCP finalization, or fidelity guard failures leave
only the original Markdown available. If a patch artifact was finalized before a
fidelity guard failure, Plasma records `report.patch.rejected` and hides that
artifact from default research raw-artifact reads/lists so later agent work does
not consume a rejected intermediate result. If the pass reports `NO_H5_CHANGES`, the
runtime records a no-change skip instead of creating a duplicate artifact. MCP
report-composition tools do not spawn a nested provider turn; they preserve the
Markdown artifact and record H5-ready metadata so an executor-owning surface can
apply the same pass later without pretending a humanized artifact already exists.

Report patching is provider-backed work over an existing Markdown report
artifact. It must not paste the whole report into a prompt or mutate the base
artifact in place. The patch run resumes the report session that created the
base artifact, or forks that report session when the executor supports it, and
temporarily exposes `plasma.report.patch.*` MCP tools. Those tools let the agent
start a bounded patch draft, read slices of the base Markdown, apply exact
replace/insert/append operations, and finalize a new Markdown report artifact.
Normal conversation turns do not receive those patch tools. The patch artifact
records the base artifact id, pending request id, operation summary, provider
session lineage, and report-session policy selection so later UI/CLI/MCP
surfaces can show the version chain without reclassifying the prior report as a
source.

Browser redpen editing is a separate user-owned path, not provider-backed
report patching. The Markdown preview lets the user replace one supported
rendered block in place while the surrounding report remains visible; complex
containers such as code fences and tables remain read-only. Saving never
mutates the selected base or humanized Markdown report artifact. Instead it
advances one logical workcopy, stores each changed body as a raw Markdown
artifact, and appends `report.redpen.saved` with artifact IDs, revision, hash,
media type, filename, and whether the artifact was created or only referenced.
Repeated saves update that workcopy, identical
content is a no-op, stale browser tabs receive a conflict, and the browser view
and download actions resolve the latest saved revision.

If the executor cannot fork sessions or the mission has no pre-report research
session, it falls back to the same-session path and records
`report_session_policy_selection`. The default browser path, labeled `보고서`,
creates a planned Markdown report artifact. CLI `reports draft` uses the same
planned default; `--mode one_take` remains an explicit same-session compatibility
path. The slower browser/report API path, labeled `장문 보고서`, creates a
Part/Section plan and immutable Section Markdown artifacts. The Section-reading
Part assembler is not a Part editor: it can bounded-read only its runner-bound
Sections and writes intro, transition, and closing material without mutating
them. Active narrative-contract profiles persist `part_edit_enabled: true` on
the plan event through the same profile contract used by writing guidance;
legacy plans that lack the field are interpreted as false, new non-narrative
plans persist false, and the browser projection does not synthesize a Part edit
phase for legacy plans. For explicit `section_fanout` requests using the
Part-connective narrative profiles, the same plan event is also the only source
of truth for `part_planning_enabled: true`; there is no separate capability
event or projection path. When enabled, the runner creates one validated
`report.part_plan.created` event per Part, forks Section writers and the Part
assembler from that Part-owner session, then resumes the same session as the
final Part author through the existing closed Part-edit tools. Part-plan replay
validates the stored event's envelope and provenance and returns the stored
canonical brief; it does not compare a retry request's newly generated brief to
that canonical brief.

The long-form report now runs through `internal/reportworkflow` as a typed
product graph through canonical finalization. The root runner owns serial versus `section_fanout` selection,
optional node activation from the canonical plan event, session forks, fan-out
scheduling with a maximum concurrency of 8, and content-free node observations.
The stage packages own their own prompt bytes, MCP allowlists, provider calls,
validation, typed recovery APIs, and durable replay. Root recovery only iterates
lineage, dispatches events to those stage recovery APIs, detects duplicate typed
outputs, and aggregates recovered results by plan index. Requirements recovery
may skip the legacy mapping stage only from an explicit matching started event or
from a typed downstream signal after stage recovery has validated Section/Part
artifact content, plan index bounds, and lineage; raw created-event envelopes are
not enough. The Section draft stage returns a typed Markdown draft or typed
evidence gap; an evidence gap is non-artifact stage data, retries only the same
Section once in the same provider and tool session, and becomes a stable
typed terminal gap on the second attempt unless a later created Section completed
that coordinate. The root permits one bounded repair in the original report-plan
session per retry lineage, using read-only research and source tools for all
failed coordinates together. Only title, purpose, and `target_refs` at those
coordinates may change. The canonical plan stays immutable while
`report.plan.section_repair.completed` records an `applied` or `unrepairable`
outcome and provider lineage. Applied repairs preserve successful Section
artifacts and give only replacement coordinates a fresh two-attempt budget;
unrepairable or post-repair terminal gaps are stable conflicts with no second
repair round. Each budget is scoped to the current pending event, plan event, and
1-based Part/Section coordinate. Web keeps request normalization, executor/service wiring, and
response serialization. The legacy finalizer contract is now the
`legacyfinalize` stage: serial uses `finalize sectional long-form markdown
report`, fanout uses `finalize section-fanout long-form markdown report`, fanout
always forks from the report-plan session, and serial forks only when Part edit
is enabled. V1/V2/V3 final edit stages live under `internal/reportworkflow`
stage packages and adopt the canonical result through `finalstore.AdoptGate`.

New planned narrative long-form plans carry
`final_edit_pipeline: assembly_writer_reader_style_validation_evidence_gate_v3`. After reviewed
Part outputs exist, the runner creates a deterministic final assembly in product
code with no agent session, then forks a final writer from the report-plan
provider session. The writer receives only
`plasma.report.long_form.final_write.*` tools and may work on the whole-report
opening, conclusion, Part transitions, and global connective logic without
adding research, external facts, or whole Part/Section reorders. The reader
editor is an independent sibling fork from the same report-plan session and
consumes the writer artifact without inheriting the writer session. If
normalized `post_report_humanize` is enabled, a pre-canonical style stage forks
from the reader provider session with style-edit tools only and may make small
tone/fluency patches without changing claims, citations, structure, or
requirement coverage. Its submitted ledger event stores ordered
`style_operation_diagnoses` records with `style_operation_diagnoses_version=1`;
each record carries the 1-based operation ordinal, validated category token,
concrete summary reason, exact match text, exact replacement text, and effective
occurrence used for the accepted patch. It still omits broader document
excerpts and prompt/provider output. A separate read-only
`style_semantic_validation` stage
then receives only reader/style comparison and verdict-submit tools. The server
applies `accepted_equivalent` or `rejected_revert_to_reader` verdicts
deterministically from durable paragraph lineage. The read-only `evidence_gate`
is another sibling fork from the report-plan session, receives approved read
tools plus evidence read/submit tools only, and judges report-to-evidence
connections without prose repair authority. Its read packet pairs deterministic
report-owned Markdown blocks with server-computed `statement_sha256` values.
The reporting layer reloads the bound source artifact, verifies lineage/SHA,
rejects submitted hashes outside that exact content, stores connection judgments
without raw passages, and canonicalizes the byte-identical artifact with
`operation_count=0`.

This v3 path is the adopted default finalization path for new planned narrative
long-form reports. For new browser long-form requests, the Web progress view
shows the actual sequence as `최종 조립`, `최종 작성`, `독자 편집`, `말투 편집`,
`말투 의미 검증`, and `근거 연결 검증`. Direct API or legacy/replay runs whose
stored `post_report_humanize` is disabled omit both style nodes and flow from
reader edit directly to `evidence_gate`.

Stored plans with `final_edit_pipeline: reader_style_gate_v1` keep their
existing replay semantics: reviewed Parts are assembled into an immutable
reader-source Markdown artifact, then reader edit, optional style edit, and the
same corrective gate run without deterministic final assembly or final writer
stages.
Stored `assembly_writer_reader_style_gate_v2` plans also keep their historical
corrective-gate repair semantics for decode, replay, and interrupted recovery.

Recovery derives Part planning only from the stored plan payload, rejects
missing, duplicate, malformed, wrong-Part, wrong-plan, wrong-session, or stale
Part plans, locks executor consistency on `report.part_plan.created`, and uses
`part-plan-N` failure IDs with `report.part_plan.failed` terminal companions.
`resume_failed` may reuse validated ancestor plan, section, Part, Part-plan,
and Part-edit outcomes; `restart` starts a new lineage and does not reuse
ancestor Part output. Open Part-edit start recovery is scoped to the exact
current pending: the W3 Part editor and W4 final Part author adopt a stored full
binding only when exactly one valid `report.part_edit.started` event exists for
that current pending, and `FinalizePartEdit` requires one matching start before
accepting an outcome. Direct MCP `plasma.report.part_edit.start` calls and Web
pre-starts share the same `StartPartEdit` transaction, so replay creates no
duplicate start and no MCP-specific policy event. Writer, reader, style, and
gate restart recovery reuse stored starts or submissions and do not re-run completed
providers. Each provider-backed stage gets one technical retry; completed intermediate
artifacts remain durable, and a second failure before successful submission
blocks canonical completion with the existing report failure events.

Stored long-form plans that lack `final_edit_pipeline` keep the legacy
`legacyfinalize` path and do not run staged final-edit stages. Prior stored profile
values also keep their C4 `sectional_preserve_markdown` and
`c4_normalized_section_headings` semantics for replay and interrupted-work
compatibility. CLI
`--mode long_form` is intentionally rejected until the CLI can call the same
section runner rather than simulating it with a single Markdown turn. Both paths
avoid AST repair turns, report versions, and report blocks. A future plan review
step can be inserted before writing, but reports still remain report artifacts
rather than sources or legacy AST report versions. The default guidance carries
forward F4 and visual planning, then adds the reader-facing writing contract: prior
conversation, investigation answers, and controller questions are working
memory, not sources; the writer should privately organize facts,
interpretations, weak signals, conflicts, and reader-facing structure before
writing a rich Markdown report.

Web planned and long-form planning use the durable MCP submission lifecycle
defined in [`report-plan-submission.md`](report-plan-submission.md). The agent's
final response is only an exact completion sentinel; the runner validates it and
the actual returned provider-session lineage before atomically promoting the current tool
session's submission. Submission `session_id` and producer name that MCP tool
session, not a provider session; only canonical provenance records the returned,
validated provider session. CLI planned Markdown planning and CLI long-form rejection
remain separate, unchanged contracts.

Workflow runs follow the same session rule. A run starts from
`workflow.run.requested`, resumes the latest provider session one bounded step
at a time, records the user-visible agent response as a result, strips the small
workflow control marker before saving the result, and writes terminal status back
to the mission ledger. If a workflow start is requested from inside an active
agent/MCP turn, Plasma records the request and defers provider execution until
the enclosing turn has a terminal event. MCP workflow starts must be bound to
the current user event and current agent executor; a request for a different
executor is rejected before it creates a queued run. If an in-process runner
disappears, the projection reports the run as interrupted so the user can stop it
or start a new bounded run without manual database edits.

New runs default to 20 steps and no total workflow duration limit. After defaulting,
`max_steps` must be within 1..20 and `max_duration_ms` within 0..86400000; zero
duration retains no run-wide elapsed-time budget. A positive `max_duration_ms`
retains the legacy run-wide elapsed-time budget. Each workflow
step gives the agent execution chain one fixed 25-minute deadline shared by the
initial call, automatic compaction, and retry. Source refreshes, ledger appends,
workflow terminal writes, normal conversation, workflow goal drafting, and all
report generation paths use separate contexts that do not inherit this workflow
step deadline.

Before opening the next durable workflow step, the runner proactively compacts a
resumed Codex research session when the latest trustworthy provider observation
reaches 55% of the model context window. The compaction event records the
triggering response event and numeric observation so runner restarts do not repeat
the same attempt. Providers without trustworthy current-occupancy telemetry are
not estimated and retain the existing error-triggered compaction path.

If an active source is soft-removed during a workflow, the next step refreshes
source state and appends `workflow.source.skipped` for that source and removal
event before continuing. The runner does not silently use removed sources by
default.

CLI and MCP are control surfaces over the same semantics. CLI can create/list/
show missions, send turns, start/status/stop workflows, and draft Markdown
reports against the same SQLite ledger. In the first slice, CLI commands that
need provider execution require `--wait` because there is no separate CLI
background worker. MCP workflow tools are mission-bound and only append or read
workflow events; `plasma.workflow.start` does not invoke the provider inside the
MCP call and must be tied to the current user turn and bound executor so the host
can drain it after that turn has a terminal response.

Plasma-spawned Codex and Claude research agents receive `plasma.workflow.status`
and `plasma.workflow.stop`, but not `plasma.workflow.start`, in their default MCP
tool allowlist. Browser and CLI workflow start remain available, and an explicit
user-controlled MCP invocation can enable start with `-enabled-tool plasma.workflow.start`.

Report drafting is also provider-backed work. It can run after a conversation or
workflow reaches a terminal state, but it must not overlap a normal turn or
workflow run for the same mission because the report may fork or resume provider
session state and writes durable report artifacts.

The first slice enforces that no-overlap rule at the shared service boundary.
Normal turn starts, report draft starts, agent session resets, and workflow run
requests re-check active mission work inside the same conditional ledger append
that records the new pending/request event. The SQLite store runs those
conditional appends in one transaction with immediate transaction locking, so
separate Web and CLI processes share the same final guard instead of relying only
on process-local locks.

The Web mission detail adds a mission-scoped `active_work` read model for the
selected mission only. It projects every open agent turn, report pending event,
and non-terminal workflow run from durable state, with stable reason codes,
affected controls, and exact cancel or view actions. The browser clears request-local busy
and notice state on a mission switch, then renders this projection; report
in-flight handles remain process-local cancellation and ownership helpers, never
product state.

The additive mission-list activity summary is also ledger-derived. It carries
the latest full-ledger sequence, current active work, and the latest classified
terminal activity without creating a global notification ledger. SQLite loads
the list-specific event subset in bulk; the browser keeps a per-mission seen
sequence only in local storage, removes entries for missions no longer listed,
and refreshes only already observed active mission summaries periodically. A
successful mission detail selection advances that local watermark;
the server ledger and product state do not change.

The browser renders agent replies as sanitized Markdown using vendored
`markdown-it` and DOMPurify. That rendering is a display concern only; it does
not make links or agent text into sources.

Report math follows the same presentation boundary. Stored Markdown and
designed-report content-model text remain the source of truth. The frontend-owned
runtime uses vendored `markdown-it-texmath` 1.0.1 with bracket delimiters; only `\(...\)` and `\[...\]` are math syntax.
The browser sanitizes Markdown first, runs the
locally vendored KaTeX 0.17.0, and
sanitizes the resulting HTML plus MathML a second time. KaTeX always uses
`throwOnError: true`, `trust: false`, `output: htmlAndMathml`, `maxSize: 20`, and
`maxExpand: 1000`. The second sanitizer keeps the HTML, MathML, and SVG profiles
needed by KaTeX while `trust: false` continues to reject unsafe commands.
`maxSize` clamps user-specified dimensions to 20em and still renders the formula.
Unmatched delimiters, unavailable KaTeX, parse errors, and `maxExpand` violations
remain visible as escaped delimiter-inclusive text without aborting the document.

Basic self-contained HTML embeds inert raw Markdown and visibly retains it when
JavaScript fails; its frontend runtime renders Markdown only after load. Designed
self-contained HTML uses the same runtime only for visible text nodes, excluding
code, preformatted text, links and URLs, script/style, SVG, and attributes.
Per-formula errors retain raw source. Both exports embed local JavaScript, CSS,
and WOFF2 fonts, so they need no CDN, package manager, file URL, or network
access. Dollar delimiters are ordinary text. This rendering does
not mutate source Markdown, content-model JSON, existing artifacts or events, or
database state, and it does not create sources, evidence, results, or saved
knowledge.

## Deferred Decisions

The next design wave should decide:

- Plasma runtime stack
- database engine and migration tooling
- API shape and service boundaries
- Liquid2 connector contract
- report canvas and renderer adapter model
- designed HTML artifact productization from the DH23 experiment, with visual
  grammar dispatch for non-hero visual units
- auth integration strategy with neutral subject identity fields
- unbound MCP mission create/open tools
- cross-process durable queue/lease tables for background execution
- MCP report control tools beyond the read-first research surface

## Report model selection boundary

Web and CLI adapters collect the raw request, latest same-executor mission-session metadata, and configured provider defaults. The reporting package owns precedence and capability validation. A successful start writes the effective model, effort, and `agent_selection_source` to `report.draft.pending`; new-event recovery only deserializes that frozen selection, while source-less legacy pending events retain the legacy resume path. Ledger payloads provide durable state, so this requires no database migration. This does not add an MCP report tool or model-tier allowlist and does not change prompts, report modes, session forks, H5, patch, designed HTML, or experiments.
