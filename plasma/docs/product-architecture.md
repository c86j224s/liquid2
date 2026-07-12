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

The current Go package layout is layered, but `internal/app` is still a broad
service facade rather than a fully split application service layer.

- `cmd/plasma`, `internal/web`, and `internal/mcp` are external entrypoints.
  They parse CLI, HTTP, and MCP requests and map product results back to those
  transports.
- `internal/app` coordinates storage calls, domain package calls, provider
  execution, and compatibility contracts used by Web, CLI, and MCP. It may later
  split into narrower source, report, workflow, connector, and provider
  services.
- Domain and feature packages such as `workflowruns`, `workflowstate`,
  `sourceevents`, `sourcecandidates`, `sourceingest`, `reporting`, and
  `ledgerstate` own product rules, state transitions, and event payload shape.
- `storage/sqlite` persists the ledger, raw artifacts, source snapshots, and
  projections. `connectors/*` and `sources/*` handle replaceable external access
  or source-reading implementations.

New work should keep this direction: transport packages adapt requests, domain
packages define product meaning, app-level services orchestrate use cases, and
storage/connectors remain replaceable implementations.

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
  default reports as Markdown artifacts.
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
  questions, result state, and report artifact state.
- `plasma.research.list`: discovery across sources, evidence, saved knowledge,
  raw artifacts, conversation results, ledger events, and report artifacts by
  default. Legacy claim/report-block object kinds require an explicit legacy
  boundary.
- `plasma.research.read`: direct reading of a specific source, evidence item,
  saved knowledge item, report artifact, raw artifact, or ledger event, with
  range support for long bodies. Agent results are read through ledger events;
  they are not reclassified as sources.
- `plasma.research.grep`: text or pattern search over ledger content, pinned
  source snapshots, and live local path sources through the shared observation
  engine. External connector search remains a separate
  possible original material discovery route.
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
- use the adopted F4 report-writing guidance as the default Markdown report
  style: reuse prior investigation as working memory, silently synthesize facts,
  interpretations, weak signals, conflicts, and reader structure, then write a
  rich report without leaking prompt, run, session, or temporary-path internals
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
  long-text readability over decorative variety.

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

An optional `direction_hint` is request-scoped report pending state, not mission state or evidence. After whitespace trimming, a non-empty value is stored only in the corresponding `report.draft.pending` payload so stale recovery can reproduce that request; omitted legacy payloads decode to empty and later requests do not copy the field. Its fixed advisory treats the hint as a weak editorial axis. Plasma explicitly injects it only into one-take writing, planned planning/writing, and long-form planning/section writing. Normal or resumed conversation, mission reminders, recall, workflows, part/frame assembly, H5 humanization, report patching, and basic or designed HTML export do not receive a new direction block. This allowlist governs application prompt construction; it does not erase provider-session history, so a path that deliberately resumes the same provider session can still retain the earlier report prompt in context.

Agent-backed report generation forks the current
research provider session when possible for every report mode except
`one_take`, keeps report planning and Markdown generation in that report-only
session, and stores returned Markdown as Plasma-owned report artifacts. The
default report path uses the adopted G2 generation-time guidance and leaves the
H5 Korean tone pass disabled unless the user or caller explicitly requests a
humanized Markdown export. When requested, browser and CLI report runners run H5
as a shared post-report Markdown transformation. It does not replace the
original artifact and does not participate in planning, source selection, AST
shaping, content-model generation, or Designed HTML rendering. The H5 pass
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

If the executor cannot fork sessions or the mission has no pre-report research
session, it falls back to the same-session path and records
`report_session_policy_selection`. The default browser path, labeled `보고서`,
creates a planned Markdown report artifact. CLI `reports draft` uses the same
planned default; `--mode one_take` remains an explicit same-session compatibility
path. The slower browser/report API path, labeled `장문 보고서`, creates a
Part/Section plan, drafts sections as separate Markdown artifacts, and assembles
part/final artifacts while preserving the section bodies instead of asking a
final pass to rewrite them. The final assembly applies the C4 experiment's
limited cleanup only at the wrapper boundary: duplicate section headings,
numbered self-headings, frame headings, connective headings, and adjacent
heading repeats are normalized, while fenced code and real section body
subheadings are preserved. Long-form report events record
`assembly_strategy: c4_normalized_section_headings` so later debugging can tell
which assembly rule produced the artifact. CLI
`--mode long_form` is intentionally rejected until the CLI can call the same
section runner rather than simulating it with a single Markdown turn. Both paths
avoid AST repair turns, report versions, and report blocks. A future plan review
step can be inserted before writing, but reports still remain report artifacts
rather than sources or legacy AST report versions. The default guidance is the
F4 experiment carry-forward: prior
conversation, investigation answers, and controller questions are working
memory, not sources; the writer should privately organize facts,
interpretations, weak signals, conflicts, and reader-facing structure before
writing a rich Markdown report.

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

The browser renders agent replies as sanitized Markdown using vendored
`markdown-it` and DOMPurify. That rendering is a display concern only; it does
not make links or agent text into sources.

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
