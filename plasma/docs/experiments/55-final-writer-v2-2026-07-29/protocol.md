# Protocol: Final Writer V2 Fixed-Input Quality Experiment

Date: 2026-07-29

Status: experiment comparison mixed/inconclusive; product adoption approved separately by explicit user decision

The preregistered report generation below was executed as W6-B after the earlier
W6-A run was rejected for synthetic, undersized Part inputs. A later audit found
that the first W6-B adjudication was not durably bound to its private mapping and
that upstream Part provenance needed stronger replay validation. The generated
reports were reused for a corrected sealed blind reread. Two independent model
readers completed all four pairs but disagreed on three pairs, so the sealed
quality comparison remains inconclusive. The later product adoption is a
separate explicit user decision based on responsibility separation and no
observed regression, not a retroactive claim that this protocol produced a
conclusive blind preference result. Current evidence and decision states are
recorded in [README.md](README.md) and [decision-memo.md](decision-memo.md).

## Question

Does adding deterministic final assembly plus a bounded final writer improve or
preserve long-form report reading quality compared with the current v1
reader/style/gate path, when both arms receive the same frozen reviewed Part
artifacts?

## 한국어 질문

같은 frozen reviewed Part artifact를 입력으로 줄 때, deterministic final assembly와
bounded final writer를 추가한 v2가 현재 v1 reader/style/gate 경로보다 장문 보고서의
읽기 품질을 개선하거나 최소한 보존하는가?

## Arms

| Arm | Product path | Pipeline | Stages |
| --- | --- | --- | --- |
| A | Current v1 compatibility path | `reader_style_gate_v1` | reader-source assembly, reader editor, optional style editor, corrective gate |
| B | Final-writer v2 path | `assembly_writer_reader_style_gate_v2` | deterministic final assembly, final writer, reader editor, optional style editor, corrective gate |

The experiment must use the product prompts, rigor behavior, durable reporting
contracts, MCP tool partitions, and finalization boundaries already implemented
for those paths. It must not use a separate writer, collector, evaluator-owned
rewrite pass, or prompt copied into an ad hoc runner.

## Fixed Inputs

Each pair freezes one reviewed Part artifact manifest. A and B both consume that
same manifest. The harness records the manifest path once per pair and mirrors
it into both arm inputs so any mismatch is machine-detectable before reading.
Each frozen manifest also has an archive-local SHA-256 receipt. A and B must
present the same receipt path and the same 64-hex SHA-256 digest; a path match
without digest identity is rejected.

The four preregistered pairs are:

| Pair ID | Topic | Rigor |
| --- | --- | --- |
| `wang-anshi-northern-song-exploratory` | Wang Anshi and Northern Song reform memory | Exploratory |
| `wang-anshi-northern-song-strict` | Wang Anshi and Northern Song reform memory | Strict |
| `go-raft-implementation-roadmap-exploratory` | Go Raft implementation roadmap | Exploratory |
| `go-raft-implementation-roadmap-strict` | Go Raft implementation roadmap | Strict |

## Stage Contracts

V1 uses only the stored compatibility final-edit path:

1. server reader-source assembly, no provider session,
2. reader editor with `plasma.report.long_form.reader_edit.*`,
3. optional style editor with `plasma.report.long_form.style_edit.*`,
4. corrective gate with `plasma.report.long_form.final_edit.*`.

V2 uses the approved sequence:

1. deterministic final assembly in product code, no provider session,
2. final writer with `plasma.report.long_form.final_write.*`,
3. independent reader editor with `plasma.report.long_form.reader_edit.*`,
4. optional style editor with `plasma.report.long_form.style_edit.*`,
5. corrective gate with `plasma.report.long_form.final_edit.*`.

The v2 final assembly event must be `report.final_assembly.created`, with kind
`final_assembly`, label `최종 조립`, producer `reporting_final_assembly`, and
schema `plasma.final_assembly.v1`. The writer stage uses kind `final_write`,
label `최종 작성`, and events `report.final_edit.writer.started` and
`report.final_edit.writer.submitted`.

Every stage trace must include the declared order, pipeline, label, fork source,
`source_artifact`, `canonicalizes`, required events, and enabled MCP tools.
Provider stages must expose exactly their full start/read/patch/submit surface:
`final_write.*`, `reader_edit.*`, `style_edit.*`, or `final_edit.*` as
appropriate. Server-owned assembly stages must expose no MCP tools. A partial
trace, wrong tool partition, wrong source artifact, or wrong canonicalization
flag is a hard failure.

## Blind Reading Packs

The harness produces Markdown reading packs under the archive root. Public pack
metadata may show pair ID, topic, rigor, and neutral labels `report_1` and
`report_2`; it must not reveal A/B identity, pipeline literals, stage names, run
IDs, provider session IDs, or private mapping data. The private mapping is a
separate archive-local JSON file.

Default blind assignment uses unpredictable local randomness. Tests may inject
`--blind-seed`, but the default run must not use a public deterministic seed
that lets a reader reconstruct A/B identity.

The first valid mapping is sealed by reuse: later verification runs load and
validate the existing archive-local mapping instead of drawing a new one. A
new blind assignment requires an explicit new evidence namespace or deliberate
removal of the old local mapping before any reading begins.

Human review should read each report directly before comparing. Diff or metric
views may support later diagnosis, but they are not a replacement for direct
reading.

## Machine Checks

The run is invalid or the candidate is rejected if any hard-fail check detects:

- hard information loss,
- citation loss or citation-to-claim breakage,
- explicit user requirement loss,
- unsupported external facts,
- product prompt or stage-contract mismatch,
- fixed-input pair identity mismatch,
- blinding leak.

The checks are intentionally hard gates, not scoring dimensions. They protect
source, citation, requirement, and product-path integrity before subjective
reading preference is interpreted.

Acceptance evidence must contain exactly four unique preregistered pair records.
Each record must include:

- `winner`: `A`, `B`, or `tie`,
- `hard_fail`: explicit boolean,
- `v2_structural_regression`: explicit boolean,
- all hard-fail count fields as non-negative integers,
- all hard-fail status fields as explicit `pass` or `fail`,
- `stage_trace_errors`: explicit list, empty only when the trace validator
  passed.

Missing, duplicate, extra-substitution, or malformed evidence rejects before the
acceptance rule is calculated.

## Acceptance Rule

The B path can be considered for integration only if all conditions hold:

- B is equal or better in at least three of the four blind pairs, counting ties
  as equal-or-better.
- No hard-fail check fires for either arm in any pair.
- B does not show a repeated structural regression across both rigor levels of
  the same topic.

Failing this experiment does not remove v1 compatibility. Passing it is still a
pre-integration signal, not an automatic merge decision.

For this run, the sealed model readings did not produce a conclusive acceptance
aggregate. The user subsequently approved product adoption under a different,
explicit criterion: responsibility separation plus no observed regression was
sufficient, while immediate quality superiority was not required or claimed.

## Repository And Archive Policy

Committed files are limited to the public protocol, reading-order index updates,
redacted product documentation updates, the reusable harness, and focused
harness tests.

All generated or sensitive material stays outside Git:

```text
~/research-artifacts/liquid2/plasma/experiments/55-final-writer-v2-2026-07-29/
```

That archive holds frozen Part artifacts, generated report Markdown, blind
reading packs, private A/B mapping, machine-check JSON, raw ledgers, prompt
packets, provider outputs, provider traces, session IDs, run databases, and
logs.

## Harness

Use:

```bash
cd plasma
python3 scripts/experiments/report_final_writer_v2_experiment.py --action manifest
python3 scripts/experiments/report_final_writer_v2_experiment.py --action write-manifest
python3 scripts/experiments/report_final_writer_v2_experiment.py --action check-fixed-inputs
python3 scripts/experiments/report_final_writer_v2_experiment.py --action write-blind-packs
```

The harness prepares contracts and reading packs from already generated
archive-local report Markdown. It does not invoke providers, judge reports, or
run W6. Omit `--blind-seed` for real blind assignment; use it only in tests.
