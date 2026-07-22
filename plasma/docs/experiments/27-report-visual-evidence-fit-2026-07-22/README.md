# Report Visual Evidence-Fit Experiment - 2026-07-22

This experiment follows experiments 23, 24, and 25. Those experiments made
visual-aid planning part of the report-writing path and showed that agents can
choose more suitable Mermaid and table forms when source structure calls for
them.

This experiment asks a narrower follow-up question:

> Should report agents use Mermaid diagrams, tables, and qualitative charts more
> readily when the source supports a structure, flow, relation, or qualitative
> contrast, even when the source does not provide exact numeric values?

## Status

Completed 24 product-path report generations across six fixtures, two report
modes, and two arms. No product default has been changed.

## Scope

The experiment uses the existing product-shaped report path:

- isolated Plasma database per run;
- local source snapshot attached through the product source path;
- the browser/API report endpoint;
- Codex report agents with MCP source-reading and Mermaid validation tools;
- `generation_guidance_profile` as the only report-writing selector;
- no prompt-only source dump;
- no report schema, renderer, MCP, or durable artifact change.

Raw reports, ledgers, prompt traces, judging packets, and local databases must
remain outside the repository under:

`research-artifacts/liquid2/plasma/experiments/27-report-visual-evidence-fit-2026-07-22/`

## Product Question

The product already asks the writer to plan sparse visual aids and to match the
visual type to the source structure. The observed gap is not syntax support or
renderer support. The gap is judgment: the agent can be too strict and decline a
useful chart or diagram merely because the source lacks exact numeric proof.

For research reports, most visuals are reading aids. A visual should not prove
more than the source can support, but it can still help the reader understand a
source-backed relationship, dependency, process, sequence, direction, or
qualitative contrast.

## Arms

| Arm | `generation_guidance_profile` | Meaning |
| --- | --- | --- |
| `visual_plan` | `visual-plan` | Current product baseline: sparse visual-aid planning plus visual type selection. |
| `visual_evidence_fit` | `visual-evidence-fit` | Candidate: current visual plan plus evidence-strength guidance for exact, qualitative, and interpretive visuals. |

The candidate does not ask for more visuals by count. It asks the agent to match
each visual's claim strength to the source evidence:

- exact numeric charts may reproduce source values;
- qualitative charts should use qualitative labels or relative positioning;
- interpretive diagrams may show source-backed structure, flow, or relation;
- unsupported precision, fabricated numbers, and overconfident causal claims
  remain forbidden.

## Fixture Families

The runner reuses the six synthetic fixture families from experiment 25 because
they already cover the relevant visual judgment cases:

| Fixture | Structure under test | Why it matters here |
| --- | --- | --- |
| `fictional-equity-dashboard` | ordered values and event markers | Exact values exist, so the candidate must not loosen numeric discipline. |
| `industry-capacity-statistics` | regional comparison and bottleneck timing | Mixes numeric and qualitative planning pressure. |
| `agent-benchmark-matrix` | multi-axis trade-offs | Tests qualitative comparison without collapsing to one score. |
| `architecture-dependency-graph` | service dependencies and blast radius | Tests interpretive structure diagrams without fabricated links. |
| `protocol-lifecycle` | actors, handoffs, state transitions | Tests sequence/state diagrams where structure is explicit. |
| `scenario-risk-portfolio` | scenario bands and uncertainty | Tests qualitative risk visuals without pretending exact precision. |

## Decision Criteria

The candidate is useful only if it improves the report as a report. Automatic
counts are observation signals, not the final decision.

Evaluate:

- whether useful visuals appear in cases where the baseline is too conservative;
- whether each visual's precision matches its evidence level;
- whether qualitative or interpretive visuals are clearly labeled without
  repeating boilerplate disclaimers;
- whether Mermaid diagrams validate before final Markdown submission;
- whether visuals supplement prose rather than replace source-grounded
  explanation;
- whether normal and long-form reports stay readable and specific.

## Result

The run completed without terminal failures:

| Signal | `visual_plan` | `visual_evidence_fit` |
| --- | ---: | ---: |
| Completed runs | 12 | 12 |
| Completed paired comparisons | 12 | 12 |
| Median visual aids | 6.0 | 5.0 |
| Median visual alignment score | 2.0 | 2.0 |
| Unvalidated Mermaid signals | 0 | 0 |

Candidate deltas:

| Signal | Value |
| --- | ---: |
| Median visual-count delta | 0.0 |
| Visual increase sign-test p-value, one-sided | 0.85546875 |
| Median alignment-score delta | 0.0 |
| Alignment increase sign-test p-value, one-sided | 0.3125 |
| Median word ratio over baseline | 0.9886309289799095 |

Mermaid type totals:

| Arm | Mermaid types |
| --- | --- |
| `visual_plan` | `flowchart`: 6, `xychart-beta`: 1, `timeline`: 3, `sequenceDiagram`: 2, `stateDiagram-v2`: 2 |
| `visual_evidence_fit` | `flowchart`: 6, `quadrantChart`: 1, `sequenceDiagram`: 3, `timeline`: 4, `xychart-beta`: 1, `stateDiagram-v2`: 2 |

Interpretation:

- The candidate did not increase visual count overall. It slightly reduced the
  median count while keeping validation clean.
- The candidate improved some concrete cases. In the long-form benchmark report
  it added a `quadrantChart` for the speed/error trade-off and preserved exact
  benchmark figures in tables. In the long-form equity report it used a timeline
  plus an `xychart-beta` instead of a less relevant explanatory flowchart. In
  the planned industry report it added a timeline for source-provided events.
- The candidate was not consistently better. In the planned scenario-risk
  report it removed the baseline timeline and replaced the structure with
  tables, reducing visual alignment for that pair.
- Direct reading suggests the candidate is good at avoiding unsupported
  precision and explaining evidence boundaries, but it still does not reliably
  solve the original product concern: the writer can remain too conservative
  about useful interpretive visuals.

Decision:

Do not productize `visual-evidence-fit` as-is. The result is useful evidence for
a follow-up candidate, but this arm is not a statistically or qualitatively
strong enough replacement for the current `visual-plan` default.

The next candidate should keep the useful evidence-strength distinction but
frame the permission more directly: if a relationship, sequence, lifecycle,
dependency, qualitative contrast, or uncertainty structure can be responsibly
explained in prose, it may be shown as a clearly labeled reading aid. The prompt
should avoid turning the writer's attention mainly toward caveats and restraint.
It should also avoid treating stock-chart-grade exactness as the normal standard:
business, research, and IR-style documents often use approximate, indexed,
directional, or qualitative visuals, and the real constraint is that the visual's
implied precision must match the source's own resolution.

## Runner

The public runner is:

`plasma/scripts/experiments/report_visual_evidence_fit_experiment.py`

Prepare the isolated binary and fixture lock:

```bash
python3 plasma/scripts/experiments/report_visual_evidence_fit_experiment.py --action prepare
```

Smoke one structure-heavy topic first:

```bash
python3 plasma/scripts/experiments/report_visual_evidence_fit_experiment.py --action run --topics architecture-dependency-graph --modes planned --workers 1
python3 plasma/scripts/experiments/report_visual_evidence_fit_experiment.py --action analyze
python3 plasma/scripts/experiments/report_visual_evidence_fit_experiment.py --action packets
```

If smoke passes, run the paired comparison:

```bash
python3 plasma/scripts/experiments/report_visual_evidence_fit_experiment.py --action run --limit 6 --workers 2
python3 plasma/scripts/experiments/report_visual_evidence_fit_experiment.py --action analyze
python3 plasma/scripts/experiments/report_visual_evidence_fit_experiment.py --action packets
```

Use fewer workers if local provider stability becomes the limiting factor.
