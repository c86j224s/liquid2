# Long-Form Report Finalization

Web long-form reports keep the existing plan, section drafting, Part assembly,
session policy, pre-canonical style editing and semantic validation, and
designed HTML workflow. Manual/post-canonical H5 is a separate deprecated
compatibility path. The default execution strategy is serial. A separate
long-form-only "fast parallel" option may fan out section drafting from the
canonical plan session, then returns to the same Part assembly and finalization
contract.

The active browser default uses the same Part-edit and staged final-edit
handoff in both strategies. For the active long-form narrative-contract default,
the canonical plan event persists `part_edit_enabled: true` through the same
profile contract used by writing guidance. Stored legacy plans that lack the
field are interpreted as false, while explicit compatible non-narrative profiles
persist false; the static projection renders a Part edit phase only when durable
progress contains it and does not synthesize the phase for legacy plans. Explicit
compatible `section_fanout` requests using Part-connective narrative profiles
also persist `part_planning_enabled: true` on the same canonical plan event. No
separate capability event exists; recovery, progress, and static projections
derive Part planning only from that stored plan payload, and omitted fields
remain false for legacy and serial work.

When Part edit is enabled, a bound post-assembly Part editor reads and patches
one immutable source Part artifact, then submits either a new edited Part
artifact or an unchanged no-op completion that reuses the source artifact. New
planned narrative long-form reports store
`final_edit_pipeline: assembly_writer_reader_style_validation_evidence_gate_v3`. The server first
creates a deterministic `final_assembly` artifact from the reviewed Parts with
no agent session, then a final writer works through the dedicated
`plasma.report.long_form.final_write.*` tools. A separate reader editor then
reviews the writer artifact. The existing `post_report_humanize` setting
controls a pre-canonical style edit followed by read-only
`style_semantic_validation`; new browser long-form requests always enable it
with no user toggle, while non-long-form browser requests remain
disabled. The read-only `evidence_gate` judges only report-to-evidence
connections before the server creates exactly one canonical
`report.artifact.created` event from the bound artifact. Manual/post-canonical
H5 remains a separate deprecated compatibility path.

For new browser long-form requests, the Web progress view presents that
execution order as `최종 조립`, `최종 작성`, `독자 편집`, `말투 편집`,
`말투 의미 검증`, and `근거 연결 검증`. Direct API or legacy/replay runs whose
stored `post_report_humanize` is disabled omit both style stages and go from
reader edit directly to `evidence_gate`.

Stored plans with `final_edit_pipeline: reader_style_gate_v1` keep the v1
reader/style/gate path for replay and interrupted work: the server assembles the
reviewed Parts into one immutable reader-source Markdown artifact, then runs
reader edit, optional style edit, and corrective gate without a final writer.
Stored legacy plans that lack `final_edit_pipeline` retain the previous
long-form finalization semantics.

Planned reports and CLI report behavior do not use this command.

## Part Assembly And Part Edit Tools

The browser no longer exposes a report-writing selector. New long-form Web
requests default to the rich section-centered composite profile
`section-brief-cluster-memory-narrative-contract`, while non-long-form requests
default to `narrative-contract`. The reader-facing writing contract is a common
baseline, and older profile values remain accepted only so stored events and
direct API calls are not reinterpreted.

Under the active long-form default, the planner does not script a reader
reaction or require surprise. It derives each Section purpose and order from
source-backed mechanisms, comparisons, causal sequences, groupings, tensions,
or evaluation questions. The Section writer explains the subject rather than
the plan. Each sentence must advance a claim, fact, mechanism, distinction,
consequence, or limit; it does not add abstract restatements, ornamental
contrasts, or redundant paragraph conclusions. Paragraph length and closing
cadence are not regularized, and ordinary connectives are not treated as a
blacklist.

Before writing, the Section writer separates the main explanatory job promised
by the Section title and purpose from supporting work such as catalog metadata,
version or provenance comparison, transmission notes, and source-comparison
tables. Supporting work may remain central only when source criticism,
bibliography, transmission, or holdings history is explicitly the Section's
main subject. If the available evidence supports only supporting work, the
writer returns an evidence gap instead of substituting a source tour for the
planned explanation. A final retry does not lower this evidence threshold.

The Section writer has only two valid outcomes: a Markdown Section draft or the
exact control response `SECTION_EVIDENCE_GAP`. A gap records
`report.section.evidence_gap` with the fixed reason code
`inadequate_section_evidence`, current pending/plan IDs, 1-based Part/Section
coordinates, attempt number, provider/tool session lineage, duration, and
standard `agent_usage`; it does not create a Section artifact or
`report.section.created` event and does not persist free-form diagnosis or
source content. The runner retries only that Section once in the same provider
session and tool-session binding. On attempt 2, the writer performs the final
replacement search or bounded scope reduction inside the existing Section title
and purpose, then returns Markdown or the exact gap token.

If any coordinate still ends in a gap on attempt 2, the runner permits exactly
one plan-repair round for that report retry lineage. The planner in the original
report-plan session uses read-only research and source tools to review all failed
coordinates together. It may replace only the title, purpose, and `target_refs`
at those same coordinates with supportable explanatory jobs, or return exactly
`SECTION_PLAN_UNREPAIRABLE`. It cannot delete, merge, move, or change successful
Sections, nor can it reassign user requirements. Replacement `target_refs` must
pass the existing mission-scoped reference validation before an outcome is
recorded; a validation failure records no repair. The canonical plan event stays
immutable; `report.plan.section_repair.completed` records an `applied` or
`unrepairable` outcome. An applied repair preserves successful Section artifacts
and gives only replacement coordinates a fresh attempt 1-to-2 budget. An
unrepairable outcome or a second post-repair gap fails explicitly, and the
planner is not called again in the same lineage.

The active long-form default keeps the same visual-aid baseline: source shape
should suggest the aid before the writer falls back to prose, so chronology
tends toward timeline, dependency toward flowchart, actor handoff toward
sequence diagram, lifecycle toward state diagram, ordered values toward
source-backed chart, and scenario or trade-off toward matrix/table. It also
uses the same Part assembly MCP handoff. The Section-reading Part assembler must bounded-read every
immutable Section bound to that Part. Intro, transitions, and closing are not
default output; it adds only the connective text needed to make an actual
Section relationship clear, then returns the `PART_ASSEMBLY_SUBMITTED`
sentinel. This assembler is not the post-assembly Part editor.

The post-assembly Part editor runs only when `part_edit_enabled` is true. It is
bound to exactly one immutable source Part artifact, receives only the
`plasma.report.part_edit.*` tools, and cannot read sources, research, Sections,
other Parts, or reader/style/gate tools. Its submitted outcome records the source
Part event, source artifact, edited artifact, provider session, and profile
metadata. A no-op submission is still a durable completion; it preserves the
source Part as immutable input and binds the outcome to the source artifact
instead of inventing a duplicate artifact. For the active default, the Part
editor reads each adjacent boundary as the previous Section's final substantive
paragraph, connective text, and the next Section's first substantive paragraph.
It edits one side only when a mechanism, conclusion, or reading instruction is
repeated, or when a simple relationship is expressed unnecessarily abstractly;
it does not rewrite ordinary Section-internal voice or rhythm.

The shared reporting start contract writes one canonical
`report.part_edit.started` event before any provider-owned Part-edit draft is
loaded. Web pre-start and direct MCP `plasma.report.part_edit.start` calls both
use the same `StartPartEdit` transaction, so an MCP replay after Web pre-start
returns the stored binding instead of creating a second start. The start payload
stores the complete normalized binding with the same field names as
`report.part.edited`, including the intended edited artifact, filename, tool
session, provider session, previous provider session, requirement-map binding,
model selection, session policy, guidance profile, session chain, report-plan
session, and fork source.

The older `part-assembly-edit-tools` profile remains accepted for experiment
replay and stored-event compatibility, but it is not a separate browser choice.

This handoff does not let the agent rewrite Section bodies or submit complete
Part Markdown. The server still inserts the immutable Section artifacts and
creates the canonical Part artifact. Planned reports use the same writing
contract without Part-edit or staged final-edit stages. The compatibility
one-take Web API uses the shared writing guidance without inventing a plan. CLI
reports, H5 patching,
designed HTML, and cost policy are unchanged.

## Execution Strategies

`serial` is the default long-form strategy. It chains planning, each section,
each Part, and finalization through the existing report session sequence.

`section_fanout` is an explicit browser long-form option. It creates one
canonical plan through the existing `plasma.report.plan.submit` boundary, then
forks the report-plan provider session for independent section workers. Each
section still uses the normal section prompt and bounded source tools. The
browser runner executes at most eight section workers at once. Part assembly
waits for the section artifacts in that Part and preserves their bodies.
For W4-enabled `section_fanout`, the runner creates exactly one durable Part
planning event per Part before Section writing. Section workers and the Part
assembler fork from that Part-owner provider session, and the final Part author
resumes the same Part-owner session through the existing closed
`plasma.report.part_edit.*` tools after mechanical assembly. The finalization
path is determined only by the stored `final_edit_pipeline`. New v3 plans run
deterministic final assembly, then a final writer forked from the report-plan
provider session, an independent reader sibling forked from that same plan
session, optional `style_edit`, read-only `style_semantic_validation`, and
read-only `evidence_gate`. Stored v1/v2 plans are legacy replay/recovery
compatibility paths and keep their historical reader/style/corrective-gate
semantics. Stored legacy profiles continue to use
`plasma.report.long_form.finalize` and do not run staged final-edit stages.

The strategy is stored on `report.draft.pending` as `execution_strategy` so
restart and stale recovery use the same path. Omitted or `serial` values keep
the default serial behavior. `section_fanout` is invalid for planned,
one-take, CLI, H5, patch, or designed HTML requests.

## Public Tool Contract

The active Part editor tools are exposed only in a dedicated Part-edit session
with a complete hidden runner binding and explicit tool enablement:

- `plasma.report.part_edit.start` creates the bounded Part-edit draft for one
  source Part.
- `plasma.report.part_edit.read` returns bounded UTF-8 slices from that source
  Part.
- `plasma.report.part_edit.patch` applies bounded exact edits to the Part-edit
  draft only.
- `plasma.report.part_edit.submit` commits the edited outcome or unchanged
  completion through the canonical Part-edit transaction.

For v2, deterministic final assembly is a server-owned product step. It creates
`report.final_assembly.created` with producer `reporting_final_assembly` and
schema `plasma.final_assembly.v1`; it has no agent session and no MCP tools.
The final writer tools are exposed only in a writer-stage session bound to that
assembly:

- `plasma.report.long_form.final_write.start`
- `plasma.report.long_form.final_write.read`
- `plasma.report.long_form.final_write.patch`
- `plasma.report.long_form.final_write.submit`

The writer may add or adjust the whole-report opening, conclusion, Part
transitions, and global connective logic, and may merge or move cross-Part
duplicate paragraphs when no unique fact, number, condition, citation,
uncertainty, or owner requirement is lost. It cannot perform research, add
external facts, reorder complete Parts or Sections, or change the fixed Part
order.

The active reader edit tools are exposed only in a reader-stage session with a
complete hidden runner binding and explicit tool enablement:

- `plasma.report.long_form.reader_edit.start` creates the bounded reader draft
  from the immutable reader-source Markdown assembled from reviewed Parts.
- `plasma.report.long_form.reader_edit.read` returns bounded UTF-8 slices.
- `plasma.report.long_form.reader_edit.patch` applies bounded exact replace,
  insert-after, or append operations.
- `plasma.report.long_form.reader_edit.submit` commits the edited outcome or
  unchanged completion as a durable stage submission, without canonicalizing.

The optional pre-canonical style tools use the same stage-scoped contract when
the stored plan's normalized `post_report_humanize` is enabled:

- `plasma.report.long_form.style_edit.start`
- `plasma.report.long_form.style_edit.read`
- `plasma.report.long_form.style_edit.patch`
- `plasma.report.long_form.style_edit.submit`

In that enabled path, the style editor follows the experiment-61 natural-voice
contract. It may only submit exact `replace` patches with `replace_all=false`,
one non-empty Markdown block per operation, and exactly one of these diagnosis
categories in each patch summary:
`opaque_or_strained_mapping`, `unnatural_collocation`, `vague_reference`,
`nominalized_or_bureaucratic`, `compressed_abstraction`,
`report_process_meta`, or `formulaic_transition`. The style stage remains
paragraph-preserving and keeps the existing deterministic final-style Markdown
guard and source fallback.

`report.final_edit.style.submitted` stores those diagnoses as
`style_operation_diagnoses`: an ordered array of records with exactly
`operation_ordinal`, `category`, `reason`, `match_text`, `replacement`, and
`occurrence`. New changed style submissions also set
`style_operation_diagnoses_version=1` and require one full record for each
operation in 1..N order. The `reason` is the concrete issue text after the
semicolon in the validated patch summary; `match_text`, `replacement`, and
`occurrence` are copied from the accepted patch input, with non-positive
occurrence normalized to 1. No-op or structural-fallback submissions store
`operation_count=0`, `style_operation_diagnoses_version=1`, and
`style_operation_diagnoses: []`. The ledger does not store broader excerpts,
prompt/provider output, replacement byte counts, paragraph ordinals, or
paragraph hashes for these style operations. Historical style submitted events
that lack the field remain replayable as legacy unknown, and versionless
two-field records remain replayable as legacy diagnosis records.

The v3 style semantic validation stage receives only read-only comparison and
verdict-submit tools:

- `plasma.report.long_form.style_semantic_validation.read`
- `plasma.report.long_form.style_semantic_validation.submit`

Its verdicts are only `accepted_equivalent` and
`rejected_revert_to_reader`. The agent cannot submit prose, patches, final
paragraph ordinals, manuscript Markdown, or `repaired_by_gate`; the server
builds the resolved Markdown from durable reader/style paragraph lineage and
fails closed if paragraph count, ordering, delimiters, or protected Markdown
invariants cannot be proved.

The v3 evidence gate receives approved read tools plus only the read-only
evidence-gate tools:

- `plasma.report.long_form.evidence_gate.read`
- `plasma.report.long_form.evidence_gate.submit`

It cannot submit repair actions, patches, replacement prose, manuscript
Markdown, semantic acceptance, or operation counts. Evidence findings contain
only `statement_sha256`, `classification`, and approved `evidence_ids`. The
read surface provides deterministic report-owned Markdown block passages paired
with server-computed `statement_sha256` values, so the provider never calculates
hashes. The reporting layer reloads the bound `SourceArtifactID`, verifies
lineage/SHA, rejects hashes outside that exact source content, stores connection
judgments without raw passages or unapproved refs, and canonicalizes byte-
identical source content with `operation_count=0`. Evidence judgments do not
block canonicalization and do not trigger automatic repair.

Legacy v1/v2 corrective gate events remain decodable and replayable with their
historical semantics. The legacy corrective gate uses the existing final-edit
tool names only in a gate-stage session with both the complete gate binding and
the matching final binding:

- `plasma.report.long_form.final_edit.start`
- `plasma.report.long_form.final_edit.read`
- `plasma.report.long_form.style_review.read` (`post_report_humanize=enabled`
  gate sessions only)
- `plasma.report.long_form.final_edit.patch`
- `plasma.report.long_form.final_edit.submit`

The gate may use approved read tools to verify unclear support. It is not a
mandatory compression or censorship pass: it corrects only source/evidence
boundary violations, owner-bound requirement violations, and unsupported claims
that need one of the approved repair actions. Gate findings persist server-
computed statement hashes, classifications, repair actions, and approved
evidence IDs only; raw statement text is transient tool input and is not stored.
When an enabled style stage changed paragraph text, the gate must also read the
complete bounded style-review packet from byte offset 0 through returned
`next_offset` values until `truncated=false`. That packet is derived by the
reporting layer from durable reader/style lineage and contains only changed
reader/style paragraph pairs, not precomputed final text or final hashes. The
gate submit payload then attests each changed source paragraph with
`paragraph_ordinal`, `final_paragraph_ordinal`, and one verdict:
`accepted_equivalent`, `reverted_to_reader`, or `repaired_by_gate`.

The server derives reader, style, and final paragraph hashes from durable
artifacts and the submitted final manuscript. It requires exactly one valid
mapping for every style-changed source paragraph and unique final paragraph
ordinals, sorts records by source paragraph ordinal, and stores only the
ordinals, verdict, hashes, record count, and digest. Raw comparison text is not
persisted. Corrective-gate paragraph or footnote insertion remains legal because
final paragraph ordinals are mapped independently; a changed source paragraph
must still map to one non-empty final block. Existing events without semantic
acceptance fields remain replayable, and no-op style, absent-style, and disabled
paths persist no semantic zero-count field.

The agent cannot select artifact IDs, filenames, title, report mode, Part
order, section order, provider provenance, model settings, or binding identity.
It cannot choose stage artifact IDs or canonical event IDs. Reader and style
stages cannot read sources or research and cannot mutate Section, source Part,
edited Part, reader-source, or prior-stage artifacts. The gate is the only stage
that may create the canonical finalization event. Legacy
`plasma.report.long_form.finalize` remains bound to its closed opening/closing
input for stored-profile compatibility only, and is used when the stored plan
lacks the active pipeline field.

For staged v3 pipelines, writer, reader, style, style semantic validation, and
evidence gate submissions are durable intermediate events only. The evidence
gate submission is followed by exactly one canonical event in the same
gate/finalization boundary, or recovery resumes from the stored gate submission
and canonicalizes once without re-running completed providers. The evidence gate
always canonicalizes the prior durable artifact rather than creating an alias or
repair artifact. An identical binding and content SHA replays the canonical
result. A different identity, provenance value, Part order, idempotency key,
pipeline marker, stage lineage, approved-evidence state, or content causes a conflict,
including after restart or a concurrent call. The conditional transaction also
decides against the current ledger state, so a terminal event for the pending
report cannot race with creation of the final canonical artifact and event.

## Completion And Retry

`FINAL_EDIT_STAGE_SUBMITTED` for writer/reader/style and `REPORT_FINALIZED` for
the gate are the normal acknowledgement strings, but they are not the authority
for completion. After each provider invocation, the runner reloads durable
state. A matching writer/reader/style submission is adopted regardless of
returned text; the gate is complete when its durable submission and canonical
report artifact event exist. If the gate submission exists without the
canonical event, recovery completes canonicalization from that stored submission
without re-running the provider. Only when the required durable state is absent
may a stage receive one technical retry. Both invocations reuse the same durable
binding, tool session, idempotency key, artifact identity, and provider-session
chain; plan, section, Part, and completed final-edit stages are not repeated.

Section evidence-gap attempts are scoped to the current pending event, plan
event, and 1-based Part/Section coordinate. If recovery finds attempt 1 and no
created Section, it resumes attempt 2 in the same provider session and
tool-session binding. If recovery finds attempt 2 and no created Section, it
reconstructs the Section failure without a provider call. If a created Section
exists after a gap, recovery treats the Section as complete. A new explicit
report retry pending receives a fresh two-attempt Section budget only while no
plan-repair outcome exists for those coordinates. When the completion event
exists, `resume_failed` reconstructs the effective plan from the immutable
canonical plan plus its same-coordinate amendment and excludes only pre-repair
gaps from the replacement budget. It also recovers an `unrepairable` outcome, so
a restart or retry cannot invoke a second plan-repair round.

For `resume_failed`, the runner reuses only validated plan, section, Part, and
Part-edit outcomes from the failed attempt's ancestor chain. If the failed
attempt reached Part assembly but not Part edit completion, the resumed Part
editor binds to the accepted ancestor Part. If a prior Part edit completed with
no changes, the downstream finalization chain still consumes the durable edited
outcome lineage even though the artifact ID is the immutable source Part. The
failed attempt is not reopened or altered. A `restart` begins a new lineage and
does not reuse ancestor Part output.

When `part_planning_enabled` is stored on the plan event, recovery must continue
the Part-owner path even if the process crashed before any Part plan was
written. Existing Part plans are accepted only when they match the current
pending report, report plan session, Part index, owner session, fork source, and
stored envelope/provenance constraints. A retry request may supply a different
brief, but replay validates the stored canonical brief and returns it instead
of comparing the retry's new brief. Missing, duplicate, malformed, wrong-Part,
wrong-plan, wrong-session, and stale Part plans are rejected. `resume_failed`
may reuse accepted ancestor Part plans, while `restart` must not. Part-plan
terminal companions use `part-plan-N` failure IDs and `report.part_plan.failed`.

Open Part-edit start recovery is exact-current-pending only. The W3 separate
Part editor and W4 final Part author both check for exactly one valid current
`report.part_edit.started` event before creating any provider fork, tool
session, artifact ID, or filename, and adopt that stored binding unchanged when
it exists. `FinalizePartEdit` refuses every edited outcome unless exactly one
matching valid start exists. A new `resume_failed` pending or `restart` must not
reuse an ancestor partial start; completed accepted outcomes keep the existing
accepted-lineage replay behavior.

The first response may supply a retry hint only when it is one legacy object
with exactly `front_matter` and `closing` string fields and exactly one root
trailing comma. The scanner removes that comma only. It rejects valid JSON,
fences, surrounding prose, extra values, unknown or duplicate fields, nested
trailing commas, and truncated input. Recovered text is non-durable guidance
for the second provider invocation and is never used by Web code to create the
artifact or event.

If the command commits but the exact sentinel is missing, the runner adopts the
durable submission or canonical event without retrying solely because of the
acknowledgement text. If no matching durable state exists after the first
invocation, the second invocation is the one allowed technical retry.
If a provider fails twice before its stage submission exists, the runner records
the existing `report.final.failed` companion and terminal `report.draft.failed`
for that pending attempt. Any completed intermediate artifact remains preserved,
and canonical completion is blocked after the second failure.

## Provenance And Observation

The public tool `producer` follows the existing MCP tool-session convention.
The final artifact and canonical event producer instead use the server-bound
report provider session. The canonical payload preserves the existing report
metadata and records the final tool session separately. Provider usage that is
known only after the tool call is not fabricated into the canonical event or a
conversation ledger event. The redacted operational log records only whether a
returned session exists and matches the bound session, together with token
aggregates and duration; it does not record the returned session ID or provider
usage details in canonical state.

The generic `mcp.tool.called` payload is unchanged. Tool name, success, and
created event IDs can be joined with canonical report provenance without
recording opening, closing, prompts, or full report text in trace summaries.
