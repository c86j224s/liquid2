# Long-Form Section Contract Experiment - 2026-07-17

This experiment tests one narrow writing-quality question:

Can Plasma make long-form report sections read with a clearer local center by
making the existing `purpose` field carry a more concrete section-writing
contract?

The experiment is intentionally not a broad report rewrite. It does not add a
new report artifact type, does not change the long-form plan schema, does not
make generated drafts into sources, and does not change the part/final assembly
contract.

## Status

- State: statistical reinforcement completed; section-brief follow-up completed
- Public plan: [`protocol.md`](protocol.md)
- Pilot summary: [`pilot-summary.md`](pilot-summary.md)
- Reinforced summary: [`reinforced-summary.md`](reinforced-summary.md)
- Section intent pilot summary: [`intent-pilot-summary.md`](intent-pilot-summary.md)
- Plan richness pilot summary: [`plan-richness-pilot-summary.md`](plan-richness-pilot-summary.md)
- Section brief statistical expansion: [`section-brief-stat-summary.md`](section-brief-stat-summary.md)
- Raw archive target:
  `research-artifacts/liquid2/plasma/experiments/22-report-section-contract-2026-07-17-section-brief-stat/`

## Product Question

Recent long-form reports can contain source-backed sections that are individually
accurate but feel weakly centered. The section writer receives a title, a
purpose, and the overall plan, but the purpose may be too broad. The part
assembler then preserves section bodies, so it cannot repair a weak section
without breaking the current contract.

This experiment asks whether a stronger planning/section-writing contract,
encoded inside the existing `purpose` string, improves readability without
making the report less faithful to the sources.

## Compared Arms

| Arm | Meaning |
| --- | --- |
| `baseline` | Current product-equivalent long-form path with the existing `g2` long-form guidance. |
| `section_contract` | Same path, but the planner writes each section purpose as a compact contract: central point, reader takeaway, evidence path, and boundary. The section writer treats that contract as binding. |
| `section_contract_coverage` | Same section contract, with an explicit guard to preserve normal long-form coverage density and keep major source-backed clusters from being collapsed out of the outline. |
| `section_intent` | Same path, but the planner writes each section purpose as quiet reader-facing intent: what the reader should come to notice, understand, or question by the end of the section. This arm deliberately avoids hard contracts and coverage locks. |

The first pilot showed that `section_contract` can sharpen section focus but
also tends to shorten reports. The reinforcement run therefore keeps the
original candidate and adds the coverage-locked candidate so the two effects can
be separated instead of hidden inside one arm.

The default first run uses the existing long-form `section_fanout` execution
strategy because the observed problem is most visible when section writers run
from the plan boundary. The harness can also run `serial` for follow-up checks.

## Decision Shape

The candidate is promising only if manual reading and blinded packets show:

- clearer section-level topic sentences and section arcs;
- less source-inventory prose;
- fewer repeated caveat frames;
- no loss of concrete source detail;
- no source/result boundary regression;
- no structural break in plan, section, part, or final report events.

Automatic metrics such as length and duration are guardrails only. The primary
judgment is editorial reading of complete reports, not isolated excerpts.

The reinforced run completed 72 reports across 24 topics and found no terminal
failures. The original candidate improved section focus in some samples but
shortened reports in 18 of 24 paired topics. The coverage-locked follow-up arm
shortened reports in 20 of 24 paired topics and therefore failed its purpose.
Neither arm is recommended for productization.

The follow-up `section_intent` arm tests a softer hypothesis: section writers
may need a felt direction of travel more than a binding mini-contract. It is a
pilot candidate until enough topics are read and scored.

After the `section_intent` pilot still showed a shortening tendency, the next
pilot narrows the question to planning richness. It compares three new
candidates against baseline:

| Arm | Meaning |
| --- | --- |
| `source_cluster_first` | Identify source-backed clusters before outlining, then map each important cluster into a Section, planned omission, or out-of-scope note. |
| `section_brief` | Put a light prose writing brief into the existing `purpose` string: reader movement, concrete details, tension or caveat, and adjacent-topic boundary. |
| `plan_review` | Ask the planner to review its own outline for thinness before the first successful MCP plan submission. This is pre-submit self-review, not a new workflow stage. |

The section-brief statistical expansion then compared `baseline`,
`section_brief`, and `section_brief_cluster_memory` across 24 topics and 72
reports. All runs completed. `section_brief` remained the safer candidate in
manual reading, but its quality gain was not statistically proven by this run.
`section_brief_cluster_memory` produced a statistically visible length increase
and should not become the default guidance. Both candidates are suitable as
explicit long-form writing options: `section_brief` for cleaner section focus,
and `section_brief_cluster_memory` for richer coverage pressure when the user
deliberately wants it.

## Non-Goals

- Do not add fields to `SectionalReportPlan`.
- Do not change the default product prompt from this experiment alone.
- Do not make part assembly rewrite section bodies.
- Do not introduce tree merging, recursive synthesis, or another fanout shape.
- Do not use development or release databases.
- Do not commit raw reports, prompt packets, provider logs, session IDs, or
  copied source fixtures.

## Files

- Protocol: [`protocol.md`](protocol.md)
- Pilot summary: [`pilot-summary.md`](pilot-summary.md)
- Reinforced summary: [`reinforced-summary.md`](reinforced-summary.md)
- Section intent pilot summary: [`intent-pilot-summary.md`](intent-pilot-summary.md)
- Plan richness pilot summary: [`plan-richness-pilot-summary.md`](plan-richness-pilot-summary.md)
- Section brief statistical expansion: [`section-brief-stat-summary.md`](section-brief-stat-summary.md)
- Runner: `plasma/scripts/experiments/report_section_contract_experiment.py`
- Raw outputs: local archive only.
