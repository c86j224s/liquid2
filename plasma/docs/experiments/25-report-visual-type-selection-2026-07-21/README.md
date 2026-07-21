# Report Visual Type Selection Experiment - 2026-07-21

This experiment follows experiments 23 and 24. Those experiments showed that
sparse visual-aid planning can help reports add tables and Mermaid diagrams
without turning visuals into decorative filler.

This experiment asks the next narrower question:

> Can a report agent choose the right visual type for the source structure,
> including dense quantitative data, benchmark matrices, and complex
> architecture dependency graphs?

## Status

Completed:

- planned mode: six fixtures, six paired comparisons after one
  candidate-guidance calibration pass;
- long-form mode: six fixtures, six paired comparisons;
- total: 24 generated reports and 12 paired comparisons;
- terminal failures: zero;
- unvalidated Mermaid signal: zero.

The implementation first added an experiment-only `visual-type-manual` guidance
profile and a product-path runner with synthetic source fixtures. After the
completed pass, the type-selection guidance was folded into the product
`visual-plan` profile.

No UI option, report schema, source-reading path, or durable artifact type
changes are part of this productization.

## Scope

The experiment uses the existing product-shaped report path:

- isolated Plasma database per run;
- local source snapshot attached through the normal source path;
- the browser/API report endpoint;
- Codex report agents with MCP source-reading tools;
- `generation_guidance_profile` as the only profile selector;
- no prompt-only source dump;
- raw reports, ledgers, prompt traces, judging packets, and local databases kept
  outside Git.

Raw material lives under:

`research-artifacts/liquid2/plasma/experiments/25-report-visual-type-selection-2026-07-21/`

## Arms

| Arm | `generation_guidance_profile` | Meaning |
| --- | --- | --- |
| `visual_plan` | `visual-plan` | Baseline at experiment time: plan sparse table/Mermaid intent when useful. |
| `visual_type_manual` | `visual-type-manual` | Candidate: current visual planning plus a compact type-selection guide. |

The candidate tells the agent to match the visual type to the source structure.
It does not require more visuals and does not add a new plan schema field.
After productization, that behavior is part of `visual-plan`; the
`visual-type-manual` spelling remains accepted for experiment replay and old
events.

## Fixture Families

The synthetic fixtures are intentionally structured so that one all-purpose
visual pattern is not enough.

| Fixture | Structure under test | Visuals that may be useful |
| --- | --- | --- |
| `fictional-equity-dashboard` | OHLC-style values, volume, event markers | table, source-backed chart, timeline |
| `industry-capacity-statistics` | regional capacity, utilization, lead time, bottleneck timing | table, chart, timeline |
| `agent-benchmark-matrix` | accuracy, latency, tool error, context, cost trade-offs | table, quadrant/chart if source-backed |
| `architecture-dependency-graph` | services, stores, workers, read models, critical dependencies | dependency graph, table |
| `protocol-lifecycle` | actors, happy path, state model, terminal states | sequence diagram, state diagram, timeline |
| `scenario-risk-portfolio` | scenario bands, risk dimensions, timeline anchors | table, chart, timeline |

The architecture fixture is deliberately complex enough to test whether the
agent can draw grouped dependencies and failure impact without reaching for a
fragile C4 grammar or inventing relationships.

## Decision Criteria

The candidate is useful only if it improves the report as a report.

Evaluate:

- whether the selected visual type matches the source structure;
- whether dense quantitative or benchmark data stays source-backed;
- whether architecture dependency diagrams make relationships and blast-radius
  risk easier to understand;
- whether compatibility-sensitive Mermaid grammars are avoided or simplified;
- whether tables and diagrams supplement prose rather than replace explanation;
- whether the prose remains coherent and readable.

Automatic counts are observation signals. They do not decide productization by
themselves.

## Result

The first candidate was too conservative. It kept reports mostly table-centered
and did not reliably move stock-style or lifecycle material into more suitable
chart/state visuals. The candidate guidance was then narrowed: exact values
still stay in Markdown tables, but the writer is asked to add one simple
source-backed chart when an ordered series, comparison axis, or lifecycle
structure is explicit in the source.

After that calibration, the six-fixture planned pass and six-fixture long-form
pass completed without terminal failures.

| Scope | Completed pairs | Candidate Mermaid types | Interpretation |
| --- | ---: | --- | --- |
| Planned six-fixture pass | 6 | `flowchart`, `xychart-beta`, `sequenceDiagram`, `stateDiagram-v2` | Candidate improved type variety without increasing median visual count. |
| Long-form six-fixture pass | 6 | `quadrantChart`, `flowchart`, `timeline`, `xychart-beta`, `sequenceDiagram`, `stateDiagram-v2` | Candidate chose more fitting visual types for benchmark, numeric, risk, architecture, and lifecycle structures. |

Aggregate signals across 12 paired comparisons:

| Metric | `visual_plan` | `visual_type_manual` |
| --- | ---: | ---: |
| Completed runs | 12 | 12 |
| Median visual aids | 5.0 | 5.0 |
| Median visual alignment score | 1.0 | 2.0 |
| Mermaid type totals | `flowchart`: 4, `sequenceDiagram`: 3, `stateDiagram-v2`: 2 | `flowchart`: 5, `xychart-beta`: 4, `timeline`: 3, `sequenceDiagram`: 2, `stateDiagram-v2`: 2, `quadrantChart`: 1 |
| Unvalidated Mermaid signals | 0 | 0 |

Candidate deltas:

| Signal | Value |
| --- | ---: |
| Median visual-count delta | 0.0 |
| Median alignment-score delta | +0.5 |
| Alignment increase sign-test p-value, one-sided | 0.015625 |
| Median word ratio over baseline | 0.98 |

Interpretation:

- The candidate did not win by adding more visuals. Median visual count stayed
  the same, median length stayed slightly below baseline, and type alignment
  improved.
- The strongest improvements were in long-form benchmark and stock-style
  numeric fixtures. The benchmark candidate used a source-backed
  `quadrantChart`; the stock-style candidate used `xychart-beta` line charts and
  a timeline instead of an unrelated explanatory flowchart.
- Architecture remained on stable `flowchart` diagrams, which is the desired
  outcome for compatibility. The candidate added separate dependency views in
  long-form mode without moving to fragile C4 syntax.
- Lifecycle reports already had useful sequence/state diagrams in the baseline,
  so the candidate mainly preserved quality rather than changing the outcome.
- Industry and risk reports improved more modestly. They used timelines where
  the source explicitly had event anchors, but still kept exact dense values in
  Markdown tables. That is acceptable because the candidate guidance is meant to
  choose suitable visuals, not force charts.
- Manual reading matched the aggregate direction. The candidate reports were
  generally easier to scan where the source had time series, scenario timing,
  benchmark trade-offs, or architecture dependencies, while prose remained
  source-grounded and did not become decorative.

Decision:

The `visual-type-manual` profile had enough evidence to productize. The change
was integrated narrowly into the existing `visual-plan` report-generation
profile family, without adding a new report plan schema field and without
changing source-reading or artifact contracts.

## Runner

The public runner is:

`plasma/scripts/experiments/report_visual_type_selection_experiment.py`

Prepare the isolated binary and fixture lock:

```bash
python3 plasma/scripts/experiments/report_visual_type_selection_experiment.py --action prepare
```

Smoke the architecture dependency fixture first:

```bash
python3 plasma/scripts/experiments/report_visual_type_selection_experiment.py --action run --topics architecture-dependency-graph --modes planned --workers 2
python3 plasma/scripts/experiments/report_visual_type_selection_experiment.py --action analyze
python3 plasma/scripts/experiments/report_visual_type_selection_experiment.py --action packets
```

Run the full six-fixture pass:

```bash
python3 plasma/scripts/experiments/report_visual_type_selection_experiment.py --action run --limit 6 --workers 2
python3 plasma/scripts/experiments/report_visual_type_selection_experiment.py --action analyze
python3 plasma/scripts/experiments/report_visual_type_selection_experiment.py --action packets
```

Use fewer workers if local provider stability becomes the limiting factor.

Run only selected fixtures or arms when recalibrating a candidate:

```bash
python3 plasma/scripts/experiments/report_visual_type_selection_experiment.py --action run --topics architecture-dependency-graph fictional-equity-dashboard --modes long_form --workers 2
python3 plasma/scripts/experiments/report_visual_type_selection_experiment.py --action run --limit 6 --modes planned --arms visual_type_manual --workers 2
```
