# Long-Form Report Finalization

Web long-form reports keep the existing plan, section drafting, Part assembly,
session policy, H5, and designed HTML workflow. The default execution strategy
is serial. A separate long-form-only "fast parallel" option may fan out section
drafting from the canonical plan session, then returns to the same Part
assembly and finalization contract.

The active Web writing choices use the same Part-edit and staged final-edit
handoff in both strategies. For active narrative-contract profiles, the
canonical plan event persists `part_edit_enabled: true` through the same
profile contract used by writing guidance. Stored legacy plans that lack the
field are interpreted as false, while new non-narrative profiles persist false;
the static projection renders a Part edit phase only when durable progress
contains it and does not synthesize the phase for legacy plans. Explicit
`section_fanout` requests using the Part-connective narrative profiles also
persist `part_planning_enabled: true` on the same canonical plan event. No
separate capability event exists; recovery, progress, and static projections
derive Part planning only from that stored plan payload, and omitted fields
remain false for legacy and serial work.

When Part edit is enabled, a bound post-assembly Part editor reads and patches
one immutable source Part artifact, then submits either a new edited Part
artifact or an unchanged no-op completion that reuses the source artifact. New
planned narrative long-form reports store
`final_edit_pipeline: assembly_writer_reader_style_gate_v2`. The server first
creates a deterministic `final_assembly` artifact from the reviewed Parts with
no agent session, then a final writer works through the dedicated
`plasma.report.long_form.final_write.*` tools. A separate reader editor then
reviews the writer artifact, the existing `post_report_humanize` setting may
run an optional pre-canonical style edit, and the corrective provenance gate
checks source and requirement boundaries before creating exactly one canonical
`report.artifact.created` event.

The Web progress view presents that execution order as `최종 조립`, `최종 작성`,
`독자 편집`, `말투 편집`, and `근거·요구 교정`. `말투 편집` appears only when
`post_report_humanize` is enabled; disabled runs omit that node without changing
the order of the remaining stages.

Stored plans with `final_edit_pipeline: reader_style_gate_v1` keep the v1
reader/style/gate path for replay and interrupted work: the server assembles the
reviewed Parts into one immutable reader-source Markdown artifact, then runs
reader edit, optional style edit, and corrective gate without a final writer.
Stored legacy plans that lack `final_edit_pipeline` retain the previous
long-form finalization semantics.

Planned reports and CLI report behavior do not use this command.

## Part Assembly And Part Edit Tools

The browser keeps three visible writing choices: visual planning,
section-centered writing, and section-centered writing with richer cluster
memory. The reader-facing writing contract is a common baseline under all
three, not a fourth choice. Internally, new requests use distinct composite
profile values so stored legacy profile values are not reinterpreted.

All three choices keep the same visual-aid default: source shape should suggest
the aid before the writer falls back to prose, so chronology tends toward
timeline, dependency toward flowchart, actor handoff toward sequence diagram,
lifecycle toward state diagram, ordered values toward source-backed chart, and
scenario or trade-off toward matrix/table. They also use the same Part assembly
MCP handoff. The Section-reading Part assembler must bounded-read every
immutable Section bound to that Part before writing intro, transitions, and
closing, then returns the `PART_ASSEMBLY_SUBMITTED` sentinel. This assembler is
not the post-assembly Part editor.

The post-assembly Part editor runs only when `part_edit_enabled` is true. It is
bound to exactly one immutable source Part artifact, receives only the
`plasma.report.part_edit.*` tools, and cannot read sources, research, Sections,
other Parts, or reader/style/gate tools. Its submitted outcome records the source
Part event, source artifact, edited artifact, provider session, and profile
metadata. A no-op submission is still a durable completion; it preserves the
source Part as immutable input and binds the outcome to the source artifact
instead of inventing a duplicate artifact.

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
path is determined only by the stored `final_edit_pipeline`. New v2 plans run
deterministic final assembly, then a final writer forked from the report-plan
provider session, an independent reader sibling forked from that same plan
session, an optional style editor forked from the reader provider session, and a
corrective gate sibling forked from the report-plan session. Stored v1 plans
continue to run reader/style/gate without writer or final-assembly progress.
Stored legacy profiles continue to use `plasma.report.long_form.finalize` and
do not run staged final-edit stages.

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

The corrective gate reuses the existing final-edit tool names, but only in a
gate-stage session with both the complete gate binding and the matching final
binding:

- `plasma.report.long_form.final_edit.start`
- `plasma.report.long_form.final_edit.read`
- `plasma.report.long_form.final_edit.patch`
- `plasma.report.long_form.final_edit.submit`

The gate may use approved read tools to verify unclear support. It is not a
mandatory compression or censorship pass: it corrects only source/evidence
boundary violations, owner-bound requirement violations, and unsupported claims
that need one of the approved repair actions. Gate findings persist server-
computed statement hashes, classifications, repair actions, and approved
evidence IDs only; raw statement text is transient tool input and is not stored.

The agent cannot select artifact IDs, filenames, title, report mode, Part
order, section order, provider provenance, model settings, or binding identity.
It cannot choose stage artifact IDs or canonical event IDs. Reader and style
stages cannot read sources or research and cannot mutate Section, source Part,
edited Part, reader-source, or prior-stage artifacts. The gate is the only stage
that may create the canonical finalization event. Legacy
`plasma.report.long_form.finalize` remains bound to its closed opening/closing
input for stored-profile compatibility only, and is used when the stored plan
lacks the active pipeline field.

For staged pipelines, writer, reader, and style submissions are durable
intermediate events only. The corrective gate submission is followed by exactly
one canonical event in the same gate/finalization boundary, or recovery resumes
from the stored gate submission and canonicalizes once without re-running
completed providers. A no-op gate canonicalizes the prior durable artifact
rather than creating an alias artifact. A changed gate creates the planned final
artifact. An identical binding and content SHA replays the canonical result. A
different identity, provenance value, Part order, idempotency key, pipeline
marker, stage lineage, approved-evidence state, or content causes a conflict,
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
