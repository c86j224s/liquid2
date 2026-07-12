# Plasma C1 Default Loop

C1 is the default Plasma product loop.

The loop is:

1. Create or open a mission.
2. Continue the same agent provider session for that mission.
3. Let the user, or a controller acting like a user, steer the next turn.
4. Record which controller steering strategy was selected and why.
5. Let the agent read original materials through MCP/source read tools.
6. Store the agent answer as a conversation result.
7. Store reports as Plasma-owned artifacts, not as sources.
8. Let the user view or download the report artifact without converting it into
   a legacy AST report.

Report generation has two product modes over the same artifact model. `보고서`
creates a planned Markdown artifact. When the executor can fork provider
sessions and the mission already has a research session, the report runs in a
forked report-only session so later research resumes the pre-report session. If
that is not available, it falls back to the same session and records why.
`장문 보고서` is slower and writes through a Part/Section path: it creates a
visible generation plan, drafts each section separately, then mechanically
preserves those section bodies while assembling part and final Markdown
artifacts. A future plan review step may be added before writing, but the final
output should still be stored as a report artifact rather than a source or
legacy AST report version.

Sources remain original materials: pasted text, fetched URLs, files, PDFs,
images, audio/video metadata references, Liquid2 documents, or other connector
material. Agent answers, controller outputs, rendered media captions, and
reports are results or artifacts. They are not reclassified as sources.

Source reads have two modes. `snapshot_only` sources are pinned Plasma artifacts.
`live_reference` local path sources are mutable original materials under
configured allowlisted roots. Accepted source snapshots keep the server-side
`root_id` and `relative_path` locator, but default agent reads address the source
by `snapshot_id` and optional `subpath`; they never receive arbitrary absolute
paths or root-wide `root_id` browsing. A live read, grep, or tree view creates a
`source.observed` event with observation metadata; it does not create legacy
evidence/claim records and does not copy the file into a snapshot.
PDF URL sources are pinned `snapshot_only` artifacts, but read tools return
bounded extracted text and extraction metadata rather than raw PDF bytes. A
local path `.pdf` remains a live reference and records a `source.observed` event
when read.

Bounded workflow runs stay inside this loop. A workflow step records a
controller-like `workflow_steering` user turn, resumes the same provider session,
stores the agent response as a result, and records a small workflow step summary
in the mission ledger. Workflow events describe progress and stop conditions;
they do not create a separate mission mode or make workflow summaries into
sources. Normal turns and workflow steps may record explicit source candidates
when the agent identifies new original material worth user review and gives a
reason for accepting it. A source candidate is not a source and does not create a
source snapshot until the user approves it.

URL source candidates are staged when possible. Plasma records
`source.candidate.staging_started`, then later records either
`source.candidate.staged` with a raw artifact or
`source.candidate.staging_failed` with the fetch failure. This staging artifact
is still an unapproved candidate: agents may read it through the dedicated
candidate-read MCP tool to avoid duplicate or low-value proposals, but default
source lists, normal raw artifact reads, and report generation exclude it until
the user accepts the candidate and Plasma creates a source snapshot.

Controller strategy selection is an observable steering event. It may add short
guidance to the agent prompt, but the strategy selection event itself must not
create sources, evidence, claims, confidence updates, source candidates,
proposal bundles, or report artifacts. The following agent result may still
surface explicit source candidates for user review when it finds useful original
material. The first browser implementation allows automatic selection or
explicit V2/V3 selection for comparison.

The 2026-06-26 C0/PAL2/NAV experiment did not validate a stronger always-on
controller as a product default. NAV was worse than the C0-like baseline, and
PAL2 remained inconclusive. The durable product rule is therefore conservative:
keep normal turns close to the same-session C0 flow, and use controller behavior
only as weak, conditional steering when a run is stuck, repetitive, too narrow,
or drifting. Controller output remains a steering result, never a source or a
stored knowledge object.

The default loop does not create evidence, claims, confidence updates, proposal
bundles, or AST report objects. Normal conversation turns and bounded workflow
runs may create source candidate review records only when the agent explicitly
identifies new original material and gives a reason for review. Existing
historical records and legacy code are kept for read-only inspection, migration
checks, and explicit developer experiments. They are not exposed as a product
mode toggle. If evidence or claim records are reintroduced later, they should
help agents and users find, compare, and explain source-backed signals; they
must not become a gate that blocks source reading, investigation, or report
generation.

The default MCP path is read-first: `plasma.research.outline`,
`plasma.research.list`, `plasma.research.grep`, `plasma.research.read`,
`plasma.research.references`, and source read/search tools. For accepted live
local path directory sources, `plasma.sources.read`, `plasma.sources.tree`, and
`plasma.sources.grep` may use `subpath` inside the accepted source boundary.
These tools must not be replaced by large mission prompt packs, source body
stuffing, report-only corpora, or root-wide local filesystem browsing.

Soft-removed sources are excluded from default source lists, reads, reporting,
and workflow use. They remain in audit history and can be shown with explicit
`include_removed` controls or restored. Removal is not physical purge/redaction.
If a source is removed during a bounded workflow run, the next step records
`workflow.source.skipped` and proceeds without silently using the removed source.

Reports that depend on live local path material cite the observation, not only
the source ID. The useful public reference is the human-readable locator plus
metadata such as `observation_event_id`, `observed_at`, sha256, and git state
when available.

Media-aware reports follow the same boundary. Pinned image bytes may be embedded
into self-contained interactive HTML exports, but the HTML remains a report
artifact. Audio and video stay linked or rendered through allowlisted provider
embeds by default. Markdown keeps original media URLs and attribution visible
rather than hiding provenance behind internal artifact IDs.

Designed HTML exports are an additional report artifact view over existing
report material, not a new source type and not a legacy AST report. The product
slice follows the DH23 experiment with the visual-grammar update:
agent-authored JSON content model, deterministic mobile-safe renderer,
source/caveat preservation, and self-contained HTML where possible. The product
renderer promotes the strongest visual unit into the first viewport as a
connected relationship map so the artifact opens with the report's core
relation rather than generic cards. Later visual units dispatch to timeline,
evidence-chain, dependency-path, trade-off matrix, loop, or relationship-map
renderers based on the content model's information shape. The
JSON content model is an internal report-rendering artifact; the user-facing
outputs remain Markdown, basic HTML, and designed HTML. This is still not a
final visual system: the first viewport remains a compact relationship map, and
the renderer must stay replaceable as more report-specific grammars mature.
Distinct swimlane, cost ladder, and richer decision-route renderers remain
follow-up work outside the current productization scope.

Report draft and designed HTML generation use the shared report runner boundary
rather than browser-owned request goroutines. Pending designed HTML work is
durable ledger state and should be resumed or retried through that runner; it is
not failed merely because an in-memory browser worker disappeared.

Workflow requests made from inside an active agent/MCP turn are deferred.
Plasma records the request in the mission ledger and waits until the current
provider turn has a terminal event before resuming the same provider session for
the requested work. Report drafts are provider-backed work too, but this slice
starts them only when no normal turn, workflow run, or report draft is active for
the mission.

Each report draft resolves its model and reasoning effort once before pending append: explicit request, latest same-executor mission session, then configured provider default. An explicit model with omitted effort uses that model's default. Invalid pairs create no pending event or provider work. Recovery uses the frozen pending values and preserves a separate compatibility branch for source-less legacy pending events; report prompts, modes, fork behavior, H5, patch, and designed HTML remain unchanged.
