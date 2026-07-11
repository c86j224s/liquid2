# Report Composition Follow-up Experiment

Status: first implementation pilot completed on 2026-06-25.

This experiment tests whether the current productized "visible plan before
drafting" report path actually improves final reports, or whether it makes the
writer overfit to the plan and lose detail. It also tests whether report
thinness comes from the writing surface itself: returning the whole report as a
chat response may encourage answer-like brevity, while writing through an MCP
artifact surface may frame the task as durable document composition.

The experiment crosses two axes:

- composition strategy: single-pass `F4`, visible-plan-then-draft, or
  `book-writer`-inspired sectional composition;
- writing surface: final chat response or MCP artifact writing.

That creates six primary variants, `R0` through `R5`. A later follow-up adds
`R6` and `R7` to isolate the sectional integration collapse observed in `R5`.

The experiment is a report-generation experiment only. It does not change the
investigation phase and does not mutate the original Plasma development
database or existing G0 run directories.

## Implementation Slice

The first implementation adds only an experiment-gated MCP writing surface:

- CLI flag: `plasma mcp -experimental-report-composition`
- tools:
  - `plasma.experiment.report.create`
  - `plasma.experiment.report.append`
  - `plasma.experiment.report.read`
  - `plasma.experiment.report.finalize`
- event type: `experiment.report.artifact.created`
- runner:
  `plasma/scripts/experiments/g0-controller-mcp-followup/report_composition_experiment.py`

The default browser server and default Codex MCP enabled-tool list do not expose
these tools. They are for controlled report-composition experiments only. The
generated Markdown is a report artifact/result, not a source, and it does not
revive legacy evidence/claim/confidence/AST machinery.

Run a small pilot:

```bash
python3 plasma/scripts/experiments/g0-controller-mcp-followup/report_composition_experiment.py run \
  --limit 1 \
  --variants R0,R1,R2,R3,R4,R5 \
  --jobs 2 \
  --timeout-sec 1200
```

When rerunning MCP variants with `--force`, the runner also rebuilds
the temporary experiment binary so implementation fixes are not masked by a
stale build. `--force-binary` remains available when only the binary needs to be
refreshed.

The runner copies source snapshots and sanitized transcripts into a local
temporary workspace, creates per-run output there, and for MCP variants creates
a per-run `experiment.db`. It does not mutate the original G0 run directories
or the Plasma development database.

## Corrected Pilot Result: CQ1 AUTO Seed 0001

The pilot used source run `CQ1-AUTO-seed-0001-attempt-1` in transcript mode.
All six variants passed hard-fail checks. The table below is the corrected
pilot result after two implementation-review fixes:

- the runner now extracts Codex session ids from `thread.started.thread_id`, so
  multi-turn variants can verify same-session resume;
- MCP artifact events now record `producer_tool_name`, not `source_tool_name`,
  to avoid confusing generated report artifacts with source material.

| Variant | Strategy | Surface | Words | Lines | Duration sec | MCP artifacts |
| --- | --- | --- | ---: | ---: | ---: | ---: |
| `R0` | single-pass F4 | response | 1407 | 108 | 91.09 | 0 |
| `R1` | single-pass F4 | MCP artifact | 1874 | 117 | 149.59 | 1 |
| `R2` | visible plan -> draft | response | 1297 | 83 | 194.22 | 0 |
| `R3` | visible plan -> draft | MCP artifact | 2403 | 131 | 214.24 | 1 |
| `R4` | sectional -> integration | response | 1477 | 55 | 271.78 | 0 |
| `R5` | sectional -> integration | MCP artifact | 799 | 40 | 382.31 | 5 |

Immediate read:

- MCP artifact writing did not fail and did not leak into default product paths.
- `R1` was longer than `R0`, and `R3` was materially longer than `R2`,
  suggesting the artifact surface may help avoid answer-like compression for
  single-pass and visible-plan drafting.
- `R3` was the strongest corrected pilot output by size and direct inspection:
  visible plan plus MCP artifact writing produced a fuller report than the
  response-surface counterpart.
- `R5` wrote intermediate section artifacts plus a final integration artifact,
  but the final integrated report collapsed into a concise summary. Sectional
  composition therefore remains unproven; it needs a better integration prompt
  or a separate follow-up experiment before product adoption.
- `R4` and `R5` were more expensive without a reliable final-report gain in
  this pilot.
- This is not a statistical conclusion. It is a successful implementation and
  smoke-quality pilot that justifies running broader judged batches focused on
  `R0`/`R1`/`R2`/`R3` and a redesigned sectional integration path.

## R5 Collapse Follow-up

The corrected pilot made `R5` surprising: it created four section artifacts plus
a final artifact, but the final report was only 799 words. The likely failure
mode is that the integration turn treated section drafts as raw material for a
new short report instead of preserving and editing those drafts.

The follow-up adds two variants that keep the same sectional workflow but change
only the integration instruction:

| Variant | Composition Strategy | Writing Surface | Integration contract |
| --- | --- | --- | --- |
| `R6` | sectional composition -> preservation integration | final response | preserve and edit section draft material |
| `R7` | sectional composition -> preservation integration | MCP artifact | preserve and edit section draft material |

The added measurement is `final_to_section_ratio`, defined as final report words
divided by the accumulated section draft words. If `R5` is low but `R7` is
consistently higher on the same source runs, the collapse is probably an
integration-contract problem, not a sectional-composition problem. If both are
low, the issue is more likely structural: the section drafts are not being
carried into the final artifact reliably.

Run the targeted follow-up on AUTO controller runs:

```bash
python3 plasma/scripts/experiments/g0-controller-mcp-followup/report_composition_experiment.py run \
  --source-runs CQ1-AUTO-seed-0001-attempt-1,CQ2-AUTO-seed-0002-attempt-1,CQ3-AUTO-seed-0003-attempt-1,CQ4-AUTO-seed-0004-attempt-1,CQ5-AUTO-seed-0005-attempt-1,CQ6-AUTO-seed-0006-attempt-1,CQ7-AUTO-seed-0007-attempt-1,CQ8-AUTO-seed-0008-attempt-1 \
  --variants R5,R7 \
  --jobs 4 \
  --timeout-sec 1500 \
  --force
```

This is still not a full statistical proof, but it gives eight paired samples
across different mission types while keeping the comparison narrowly focused on
the integration contract.

### Follow-up Result: AUTO Runs CQ1-CQ8

The follow-up ran `R5` and `R7` on all eight AUTO 02-controller-quality source
runs. `R3` was also rerun on the same eight runs as the current product-candidate
baseline.

All compared runs passed hard-fail checks after narrowing the leak detector so
source-file citations and source-discussed terms such as `V2`, `V3`, and
`hard-fail` are not mistaken for internal experiment leakage. The leak detector
still rejects internal report-composition paths and judge setup language.

| Mission | R3 words | R5 words | R5 final/sections | R7 words | R7 final/sections | R7-R5 | R7-R3 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `CQ1` | 1579 | 1906 | 1.014 | 1267 | 0.923 | -639 | -312 |
| `CQ2` | 2238 | 1983 | 0.702 | 2309 | 1.050 | +326 | +71 |
| `CQ3` | 3218 | 2411 | 0.943 | 3006 | 0.978 | +595 | -212 |
| `CQ4` | 1942 | 1812 | 0.851 | 2566 | 0.989 | +754 | +624 |
| `CQ5` | 1503 | 3054 | 1.045 | 2559 | 0.979 | -495 | +1056 |
| `CQ6` | 2471 | 2003 | 0.748 | 3509 | 1.059 | +1506 | +1038 |
| `CQ7` | 2426 | 1840 | 0.792 | 2250 | 0.993 | +410 | -176 |
| `CQ8` | 2583 | 2051 | 0.971 | 4459 | 1.524 | +2408 | +1876 |

Aggregate read:

| Variant | Avg words | Median words | Avg duration sec | Median duration sec | Avg final/sections |
| --- | ---: | ---: | ---: | ---: | ---: |
| `R3` | 2245.0 | 2332.0 | 243.4 | 253.2 | n/a |
| `R5` | 2132.5 | 1993.0 | 480.7 | 490.5 | 0.883 |
| `R7` | 2740.6 | 2562.5 | 512.5 | 521.0 | 1.062 |

Paired signs:

- `R7 > R5` on word count: 6 wins, 2 losses, average delta +608.1 words,
  median delta +502.5 words. Exact one-sided sign-test p is 0.1445, so this is
  directional evidence, not statistical proof.
- `R7 > R3` on word count: 5 wins, 3 losses, average delta +495.6 words,
  median delta +347.5 words. Exact one-sided sign-test p is 0.3633.
- `R5 > R3` on word count: 2 wins, 6 losses, average delta -112.5 words,
  median delta -361.5 words. This argues against original `R5` as a product
  default.

Interpretation:

- The original `R5` collapse did not reproduce deterministically. In the paired
  rerun, `CQ1` `R5` produced 1906 words rather than the earlier 799 words. This
  means the failure is better described as instability and compression risk in
  the integration turn, not a guaranteed collapse.
- The `R7` preservation-edit instruction reduced that risk. Its final reports
  retained roughly the same amount of material as the section drafts on median
  (`0.991` final/sections ratio), while `R5` retained less (`0.897` median
  ratio).
- `R7` is not yet a clear product default. It is slower than `R3` by roughly
  269 seconds on average and wins over `R3` only 5 of 8 times by word count.
  It may be useful as a deliberate "long-form / preserve detail" mode, but the
  simpler `R3` path remains the safer default candidate.
- The main product lesson is to separate two concerns: artifact writing is
  useful, but sectional writing needs an explicit preservation-edit integration
  contract. Without that contract, the final turn may treat section drafts as
  notes for a new summary.

## R8 Long-form Follow-up

The next follow-up moved beyond one investigation run per report. `R8` maps the
eight completed AUTO investigation perspectives to eight Parts, parses their
numbered `R7` outline entries into 69 Sections, drafts those Sections in
parallel, preserve-edits each Part, and mechanically assembles one long-form
report.

See:
`../04-longform-report-r8-2026-06-25/README.md`

High-level result: R8 produced a real long-form report of about 63k words and
avoided global summary collapse, but preservation acceptance failed at the Part
level. Some Parts retained nearly all section material, while `P02`, `P04`, and
`P08` compressed heavily. The next report-generation experiment should therefore
focus on Part assembly, not on adding more source/evidence machinery.

## Why This Experiment Exists

The 2026-06-23/24 report prompt follow-up showed that `F4` was the strongest
tested product prompt candidate:

- same provider session as the investigation,
- prior agent answers and controller questions used as working notes, not
  sources,
- source reads through tools instead of prompt-stuffed source packs,
- silent internal synthesis before writing,
- rich report/article wording with labeled uncertainty.

That supports "plan before writing" at the prompt-behavior level. It does not
prove that a separate visible planning turn, persisted as
`report.plan.created` and then fed into a later drafting turn, improves reports.

The concrete concern is rigidity: once a plan is handed back to the writer, the
writer may satisfy headings and coverage bullets mechanically while leaving out
the detailed material discovered in the conversation. The Oda Nobunaga report
case exposed this risk: the mission had Saito Dosan material in the conversation
and source records, but the latest report omitted that axis. A visible plan can
help debug this, but it may also become a shallow checklist.

A second concern is the writing surface. If the final report is returned as the
agent's final response, the model may slip into "answer the user and finish"
behavior and compress a long-form artifact into a short summary. Writing through
MCP artifact tools may instead make composition feel like durable document work,
especially when sections are appended, read back, revised, and finalized.

## Book-writer Observation

`book-writer` does not stop at "make a plan and then write the whole book in one
turn." Its durable loop is:

1. research fan-out,
2. whole-book plan,
3. plan review,
4. chapter-level drafting with local chapter context,
5. style/fact/continuity feedback per chapter,
6. editor integration into a full manuscript,
7. fresh-context whole-manuscript acceptance.

The relevant lesson for Plasma is not EPUB generation or book-specific style.
The lesson is composition control: long output quality may improve when the
system gives each section enough local attention, then separately checks the
whole artifact for coherence and coverage.

For Plasma, this maps to a report workflow that creates a report outline, writes
bounded report sections, integrates the sections into one public article, then
checks coverage and coherence before accepting the final artifact.

## Hypotheses

### H1: Visible Plan Helps Coverage Debugging

A separate visible plan should make omissions easier to diagnose. If the plan
contains "Saito Dosan" but the report omits it, the failure is in drafting. If
the plan omits it, the failure is in coverage discovery.

### H2: Visible Plan May Harm Richness

A separate plan may reduce final richness if the writer treats it as a checklist
rather than a scaffolding. Expected failure mode: headings are present, but each
section is thin and avoids deeper material that was available in the
conversation or source reads.

### H3: Sectional Composition May Recover Detail

A book-writer-style sectional workflow may improve detail density because each
section gets a focused drafting pass. Expected benefit: important axes receive
more concrete facts, source hierarchy, uncertainty labeling, and narrative
texture.

### H4: Sectional Composition May Harm Coherence Or Cost

The same workflow may introduce duplicated context, inconsistent voice,
fragmented transitions, or much higher latency/cost. It should not become a
product default unless it wins on report quality enough to justify those costs.

### H5: MCP Artifact Writing May Reduce Answer-like Brevity

Writing into a report artifact through tools may improve length, detail density,
and willingness to revise because the agent is no longer treating the report as
a final chat answer. Expected benefit: longer sections, more complete local
coverage, and better continuation behavior.

### H6: MCP Artifact Writing May Add Tool Friction

Artifact writing may harm quality if the agent spends attention on tool protocol
instead of source reading and prose. It may also create partial artifacts that
need lifecycle cleanup. A product path should adopt it only if the quality gain
outweighs the added state and failure modes.

## Variant Matrix

All variants use the same completed investigation session or sanitized
transcript/source-copy package. Original sources remain the only citeable source
material. Prior answers, controller turns, generated plans, section drafts, and
intermediate summaries are working results, not sources.

| Variant | Composition Strategy | Writing Surface |
| --- | --- | --- |
| `R0` | F4 single-pass | final response |
| `R1` | F4 single-pass | MCP artifact |
| `R2` | visible plan -> single draft | final response |
| `R3` | visible plan -> single draft | MCP artifact |
| `R4` | sectional composition -> integration | final response |
| `R5` | sectional composition -> integration | MCP artifact |

### R0: F4 Single-pass / Final Response

This is the current validated prompt-shape baseline.

- Resume or fork from the completed investigation session.
- Ask for one final Markdown report.
- Prompt includes F4 principles: working-memory reuse, silent synthesis plan,
  rich article guidance, uncertainty labeling, and leakage suppression.
- No separate visible plan is created.

Purpose: preserve the strongest known single-pass baseline.

### R1: F4 Single-pass / MCP Artifact

This keeps the same composition strategy as `R0`, but changes the writing
surface.

- Resume or fork from the completed investigation session.
- Ask the agent to create a Markdown report artifact through MCP report-writing
  tools.
- The tool surface should be document-shaped, not schema-heavy. A first
  experiment surface can be `report.create_draft`, `report.write_body` or
  `report.append`, `report.read_draft`, and `report.finalize`.
- The agent's final chat response should only summarize the artifact path/status,
  not contain the whole report body.

Purpose: isolate whether artifact writing alone improves report richness.

### R2: Visible Plan Then Single Draft / Final Response

This matches the recently productized report path.

- Same session/fork source as R0.
- Turn 1: produce a visible report generation plan.
- Persist the plan in experiment artifacts.
- Turn 2: pass the plan to the same report writer session and ask for the final
  Markdown report.

Purpose: test whether visible planning improves coverage without making the
report rigid.

### R3: Visible Plan Then Single Draft / MCP Artifact

This keeps the visible-plan composition strategy but changes the final writing
surface.

- Turn 1: produce and persist a visible report generation plan.
- Turn 2: ask the same report writer session to create the final Markdown report
  through MCP report-writing tools.
- The final chat response should point to the artifact, not duplicate the full
  body.

Purpose: test whether plan rigidity is specific to response-writing, or whether
the visible plan itself is the limiting factor.

### R4: Sectional Composition Loop / Final Response

This is the book-writer-inspired comparison.

- Turn 1: produce a report outline with section briefs. Each section brief must
  name:
  - purpose,
  - expected source clusters,
  - working-memory cues from the investigation,
  - known uncertainty or conflict,
  - why this section belongs in the report.
- Section drafting: draft each section as a bounded work item.
  - A section drafter receives the section brief, source index, source-reading
    tools or source-copy files, and the sanitized investigation transcript as
    working memory.
  - It writes only that section, with citations to original sources.
  - It must not cite the plan, transcript, controller questions, or prior
    section drafts as sources.
- Integration: a final editor reads all section drafts, the outline, and the
  original sources as needed, then writes one coherent Markdown article.
- Acceptance: a fresh-context reviewer checks the integrated report against the
  outline and salient coverage checklist. It does not rewrite the report; it
  marks ACCEPT/BLOCK and records missing coverage, rigidity, duplicated prose,
  unsupported claims, and citation problems.

Purpose: test whether local attention per section improves detail while the
integration and acceptance steps prevent fragmentation.

### R5: Sectional Composition Loop / MCP Artifact

This keeps the same sectional composition strategy as `R4`, but each durable
drafting action writes through MCP artifact tools.

- Turn 1: create the outline as a report-work artifact.
- Section drafting: each section is written or appended through MCP artifact
  tools, then read back before the next section starts.
- Integration: the editor reads the section artifacts and writes a final report
  artifact.
- Acceptance: the reviewer reads the final artifact and records ACCEPT/BLOCK.

Purpose: test the strongest long-form product candidate: bounded section work,
durable intermediate artifacts, and a final integrated artifact.

## Parallel Execution Design

Do not run multiple forks from the same provider session at the same time. Prior
fork-mode experiments observed that parallel TUI forks from one source session
can stall before the forked session file appears.

Parallelization rules:

- Parallelize across source runs or missions.
- Within one source run, create the six variant forks sequentially before
  running them. Once each fork session exists, `R0`-`R5` may run in parallel
  because they no longer mutate the same provider session.
- For product-shaped validation, keep `R4`/`R5` section drafting sequential
  inside each variant session. This preserves the intended product behavior:
  the same report writer builds section context over time.
- Section-level parallel drafting is allowed only as a separate exploratory
  stress test, not as the primary product-shaped comparison.
- Judging can be fully parallelized after all candidate reports are generated.

Recommended first batch:

- 18 completed G0 source sessions from the 2026-06-23 follow-up set.
- 6 variants per source session: `18 * 6 = 108` final reports.
- If the full batch is too expensive, first run a 6-session pilot with all
  variants: `6 * 6 = 36` final reports.
- Expand to 18 sessions only if the pilot shows that `R4`/`R5` are not obviously
  dominated by cost, failure rate, or coherence loss.

## Inputs

For each source run, prepare an isolated workspace:

```text
source_index.json
sources/
investigation_transcript.md
mission_metadata.json
salient_coverage_checklist.md
```

`salient_coverage_checklist.md` is created before report generation and is used
only by the judge/acceptance harness. It should list user-steered or repeatedly
discussed axes, not desired wording. For the Oda-style mission, examples would
include Saito Dosan, Okehazama, Nagashino, Honnoji, source hierarchy, transmitted
stories, and uncertainty.

The checklist must not be included in the writer prompt unless the variant
explicitly tests checklist exposure. In this experiment, it is a judging
instrument, not a writer aid.

## Metrics

### Primary Quality Metrics

- Salient coverage: how many checklist axes are materially covered.
- Detail density: concrete source-backed details per salient axis.
- Source-groundedness: whether source-backed paragraphs cite original sources.
- Synthesis depth: whether the report explains relations, causes, tensions, and
  consequences rather than listing facts.
- Reader coherence: whether the final article reads as one coherent piece.

### Rigidity Metrics

- Plan overfit: headings are followed but sections are thin.
- Missing detail despite planned coverage.
- Mechanical wording that exposes the plan structure instead of serving the
  reader.
- Unnatural transitions caused by outline adherence.

### Safety And Cost Metrics

- Unsupported conclusion rate.
- Overclaim rate.
- Internal ID/path/experiment-label leakage.
- Runtime and token cost.
- Failure rate.
- Number of tool/source reads actually used.
- Artifact lifecycle failures: partial drafts, missing finalization, duplicate
  writes, or stale draft IDs.

## Judging

Use blinded judge packets. Judges should see:

- original source snapshots or copied source files,
- sanitized investigation transcript as non-source working context,
- salient coverage checklist,
- candidate report,
- variant-hidden run metadata.

Judges should not see the variant label, prompt text, run directory, or
experiment name.

The judge should score both final output quality and failure mode:

| Dimension | What It Measures |
| --- | --- |
| coverage | Did it include the important user-steered axes? |
| detail | Did covered axes get enough concrete substance? |
| coherence | Does it read like one article rather than stitched notes? |
| flexibility | Did the structure help the writing rather than constrain it? |
| groundedness | Are claims supported by original sources? |
| uncertainty handling | Are weak signals and conflicts labeled well? |
| leakage | Did internal labels, paths, IDs, or experiment details leak? |
| cost | Is the quality gain worth the extra work? |

## Hard-fail Rules

A generated report is invalid and must be rerun or excluded if it:

- cites the transcript, plan, controller questions, section drafts, or previous
  answers as if they were original sources,
- leaks variant labels, experiment labels, run directories, or temporary paths,
- mutates copied source material,
- produces no final report,
- uses a different source set than its paired variants without recording an
  infrastructure reason.

Hard-fail rules are experiment controls, not product UX. Product behavior should
surface recoverable errors; the experiment harness should fail fast to keep
comparisons clean.

## Interpretation Rules

If `R1` beats `R0` without a rigidity penalty, treat MCP artifact writing as a
promising product surface even when the composition strategy remains
single-pass. Continue improving the artifact-writing tool surface before
assuming visible planning is required.

If `R2` beats `R0` without a rigidity penalty, keep visible planning as the next
response-based product candidate and continue improving the plan UI/debug
surface.

If `R2` improves debuggability but not final quality, keep plan logging as a
diagnostic or optional/debug feature rather than assuming it should be on the
default report path.

If `R3` beats `R2`, visible planning is not necessarily wrong; response-based
writing may be the weaker surface.

If `R4` or `R5` beats the single-draft variants on detail and coverage but costs
substantially more, treat sectional composition as a "long-form / high-effort
report" mode, not the default lightweight report path.

If `R5` beats `R4`, the artifact surface is likely valuable for multi-step
composition. If `R4` beats `R5`, tool friction may be outweighing artifact
benefits.

If `R4`/`R5` improve detail but harm coherence, test an improved integration pass
before rejecting sectional composition.

If `R0` remains strongest, revert the default product report generation toward
F4 single-pass and use visible planning only for debugging or explicit user
requests.

## Product Boundaries

This experiment must not revive legacy evidence/claim/confidence/AST machinery
as the report-quality solution. The tested path remains:

- mission plus same provider session where product-shaped,
- source reads through MCP or copied source files,
- generated plans and section drafts as results or report-work artifacts, not
  sources,
- final Markdown report as the user-facing artifact.

The sectional workflow may produce intermediate files in the experiment
workspace, but product adoption would require a separate design decision about
how much of that workspace should be visible in Plasma.

## Expected Deliverables

For each run:

```text
run_manifest.json
{stage}.prompt.md
final_report.md
budget_summary.json
hard_fail_audit.json
```

For MCP artifact variants (`R1`, `R3`, `R5`, `R7`):

```text
experiment.db
artifact_ids in run_manifest.json
experiment.report.artifact.created events inside experiment.db
```

For visible-plan variants (`R2`, `R3`):

```text
visible_plan.md
```

For sectional variants (`R4`, `R5`, `R6`, `R7`):

```text
section_outline.md
section_titles.json
section_{N}.md
integration.prompt.md
```

For the whole experiment:

```text
run_index.csv
runner_results.json
future judged batch: score_matrix.csv, pairwise_stats.json, decision_memo.md
```

## Open Questions Before Execution

- Should the first run use transcript/source-copy mode for speed, fork mode for
  product realism, or both in stages?
- What is the minimal MCP artifact write surface needed for the experiment:
  whole-body write only, append/read/finalize, or section-aware draft tools?
- How many sections should `R4`/`R5` allow by default? A first cap of 4-6
  sections is likely enough for reports without turning the experiment into a
  book project.
- Should the acceptance step be allowed to request one integration repair pass,
  or should it only judge the first integrated output?

Recommended answer for the first batch:

- Start with transcript/source-copy mode for a 6-source pilot.
- Pre-create variant forks sequentially, then run `R0`-`R5` in parallel per
  source run when fork mode is used.
- Use a small MCP artifact surface: create/append/read/finalize.
- Use 4-6 section caps for `R4`/`R5`.
- Allow one sectional integration repair pass only if acceptance BLOCK is caused by
  coherence/transition problems, not by unsupported claims.
- If the pilot is not dominated by cost/failure, expand to fork-mode or
  product-shaped runs on the full 18-session set.
