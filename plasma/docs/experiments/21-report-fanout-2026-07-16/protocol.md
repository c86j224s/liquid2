# Protocol

## Objective

Measure whether section-level fanout can make Plasma long-form report
generation faster without lowering report quality relative to the current
serial long-form path.

This was a planning and experiment issue. The completed result supports an
explicit product option, not a default-path replacement.

## Current Baseline

`A-serial-current` must follow the current product-equivalent long-form
lifecycle as closely as the harness permits:

1. create a long-form report request;
2. create the canonical long-form plan through the existing report-planning
   lifecycle;
3. draft section artifacts;
4. assemble part artifacts from section outputs;
5. assemble the final Markdown report while preserving section bodies and
   applying the existing deterministic wrapper cleanup.

Before implementation, the harness owner must verify the exact current code
path and record any deviation from product execution. A deviation that changes
prompt shape, source access, session policy, final assembly, or retry behavior
must be documented as a blocking issue, not hidden inside the result.

## Experiment Arms

### A. Serial Current

The baseline runs the current long-form lifecycle in product order. It provides
the wall-clock and quality reference for every topic.

### B. Section Fanout

The candidate keeps one canonical plan, then drafts each section in an
independent worker. Each section worker receives only:

- mission objective and report direction;
- the full long-form plan;
- its part title and purpose;
- its own section title, purpose, and target references;
- source/evidence access through the same bounded Plasma research tools used by
  product report generation;
- the same report tone and source-fidelity guidance as the baseline.

Part assembly starts only after every section for that part reaches a terminal
state. Final assembly starts only after every part reaches a terminal state.

The candidate must still preserve section artifacts as the durable child
outputs. Part assembly remains a connector-writing step: it may create part
introduction, transition, and closing text, but it must not rewrite child
section bodies.

### Deferred Arms

Part-level fanout and binary-tree or recursive merge strategies are deliberately
deferred. A part worker that writes an entire part at once would bypass the
current product's section artifact boundary. It may be tested later only after
it is reframed as a part lane whose internal section work still follows the
same section-preserving contract.

## Shared Contracts

Both arms use the same topic packets, source snapshots, model family, model
effort, report direction, and judging rubric.

The plan is the synchronization boundary. Fanout workers may use plan context
and source/evidence tools, but they must not read sibling worker transcripts or
invent a new report structure. Assembly may read child outputs as results, not
as sources.

For the section fanout candidate, worker session lineage is intentionally not
the same as the serial baseline. The baseline chains each section from the
previous stage. The candidate starts each section from the canonical plan
boundary, or from an explicitly recorded fork of that boundary. The experiment
must record this as a designed difference instead of pretending both arms have
the same provider-session shape.

The final report must cite or refer back to original source material. It must
not cite generated section or part outputs as if they were source material.

## Isolation Rules

The experiment uses archive-local state only:

- archive-local SQLite databases;
- archive-local provider homes where needed;
- archive-local run directories;
- loopback-only temporary services if a service is required.

The harness must reject development and release database paths, release ports,
raw provider homes, and any command that would write raw artifacts into the Git
worktree.

Raw outputs stay under the local archive. The repository keeps only the
protocol, redacted aggregate metrics, and decision summary.

## Execution Plan

1. **Code-path audit**
   - Identify the current Web long-form path, report-planning path, section
     runner, part assembly, final assembly, report session policy, and retry
     boundary.
   - Record whether the harness can reuse these components directly or must
     call them through a product-like adapter.

2. **Harness smoke**
   - Run one small topic through A and B.
   - Validate artifact presence, terminal states, source-read traces, section
     and part counts, and final Markdown assembly.
   - Stop if any fanout arm changes source access or report structure in a way
     the baseline does not.

3. **Pilot**
   - Run a small set of diverse topics through both arms if smoke exposes
     scheduling, retry, or judging uncertainty.
   - Use this only to catch scheduling, retry, and judging defects.
   - Do not use the pilot to make a product decision.

4. **Main run**
   - Run up to twenty-four diverse topics through A and B.
   - Counterbalance run order by topic so one arm does not always benefit from
     cache warmth or provider drift.
   - Increase parallelism gradually: one smoke run, then two concurrent runs,
     then up to the safe limit confirmed by the harness.
   - Treat twenty-four paired topics as the confirmatory target. If environment
     or provider failures reduce the completed paired set, analyze the
     intention-to-treat set and explicitly state whether the statistical claim
     is underpowered.

5. **Analysis**
   - Compute wall-clock total time, critical path time, per-stage time, retry
     counts, terminal failure rate, and token usage where available.
   - Blind-judge final reports by topic and arm-hidden artifact id.
   - Inspect representative reports manually as an editor, not only by scoring
     excerpts.

## Quality Rubric

Each completed report receives scores on a 1-5 scale:

| Dimension | What to check |
| --- | --- |
| Source fidelity | Claims stay within source/evidence support and uncertainty is explicit. |
| Coverage | Important source material is not dropped because of fanout boundaries. |
| Flow | Sections and parts connect naturally and the final report has a readable arc. |
| Tone consistency | The report sounds like one coherent writer rather than multiple stitched writers. |
| Repetition control | The same claim or caveat is not repeated in a tiring way. |
| Specificity | The report keeps concrete details instead of collapsing into generic summaries. |
| Assembly cleanliness | Headings, numbering, and transitions are structurally clean. |

The primary quality endpoint is the mean paired difference against
`A-serial-current`. Flow, tone consistency, and source fidelity are guardrail
dimensions: a speed win cannot override a meaningful loss in those areas.

## Decision Rules

A fanout arm may be recommended for a later productization issue only if:

- median wall-clock time improves by at least 25% versus `A-serial-current`;
- quality is non-inferior with a lower confidence bound no worse than `-0.25`
  on the 1-5 paired score scale;
- source fidelity has no serious regression;
- terminal failure and retry behavior are no worse than the baseline;
- manual reading does not find systematic stitched-fragment prose.

If B passes, recommend a separate productization issue for an opt-in fanout
strategy. If B does not pass, leave the current serial path unchanged and record
what failed. Deferred arms are not treated as failed; they simply remain
untested by this run.

## Stop Conditions

Stop the experiment and report the blocker if:

- the harness cannot stay close to the current product path;
- fanout workers need source bodies pasted into prompts instead of MCP/source
  reads;
- raw artifacts would need to be committed for the experiment to be understood;
- failures are caused by provider authentication, unrelated environment drift,
  or missing local source material rather than the fanout strategy;
- the experiment starts adding unrelated report-writing, tone, or UI changes.
- the candidate requires weakening the existing long-form finalization binding
  or reclassifying generated section/part results as sources.

## Deliverables

- Public protocol and index entry in this repository.
- Issue comments that explain each phase in Korean for human review.
- Local raw archive with runs, logs, generated reports, and judge packets.
- Public aggregate analysis and decision memo after the experiment completes.

## Completed Run Notes

The executed confirmatory run used 24 paired topics and compared only:

- `A-serial-current`
- `B-section-fanout`

No part-level fanout, recursive merge, tone rewrite, UI change, or product
default change was included. A harness defect initially copied the full Codex
home into each run-local provider directory and exhausted local disk space. That
was corrected by seeding only minimal Codex auth/config references while keeping
session/log/temp files run-local. The affected partial run directories were
deleted from the local archive, completed reports were preserved, and the full
run was resumed.

Final aggregate metrics are summarized in `README.md`; raw run outputs remain
in the local archive according to the repository artifact policy.

The accepted productization keeps the default serial long-form path and exposes
section fanout only as an explicit long-form execution strategy. This preserves
the current report mode, planning MCP boundary, finalization MCP boundary,
section artifact model, and C4 section-preserving assembly contract.
