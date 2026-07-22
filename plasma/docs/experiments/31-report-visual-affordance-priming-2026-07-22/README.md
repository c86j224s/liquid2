# Report Visual Affordance-Priming Experiment - 2026-07-22

This experiment follows experiments 25 through 30 on issue 174.

Experiment 25 already answered the broad type-selection question. The product
`visual-plan` profile now knows that chronology can use `timeline`,
dependencies can use `flowchart`, actor handoffs can use `sequenceDiagram`,
lifecycle or status changes can use `stateDiagram-v2`, ordered numeric values
can use `xychart-beta`, and exact or multi-axis comparison can stay in Markdown
tables.

The unresolved question is not "which Mermaid type exists for which case?".
The unresolved question is whether report writers consistently recall that
mapping while planning and writing.

Experiment 30 showed that simply asking writers to seek clearer visual aids
increased visual-aid count, but did not improve visual alignment. This
follow-up tests a more specific but still non-prohibition framing:

> Let the source's shape suggest the visual aid before defaulting to prose.

## Status

Completed with one additional full repeat. The affordance reminder was then
productized into the existing `visual-plan` family rather than exposed as a new
browser option.

## Scope

The experiment uses the same product-shaped report path as experiments 28, 29,
and 30:

- isolated Plasma database per run;
- local source snapshot attached through the product source path;
- the browser/API report endpoint;
- Codex report agents with MCP source-reading and Mermaid validation tools;
- `generation_guidance_profile` as the only report-writing selector;
- no prompt-only source dump;
- no report schema, renderer, MCP, durable artifact, UI, or product default
  change.

Raw reports, ledgers, prompt traces, judging packets, and local databases must
remain outside the repository under:

`research-artifacts/liquid2/plasma/experiments/31-report-visual-affordance-priming-2026-07-22/`

## Product Question

The product question is whether a light affordance reminder can make the
existing visual-type mapping fire more reliably than the current `visual_plan`
baseline.

The candidate should:

- help the writer notice the dominant source shape in a section;
- softly connect that shape to the visual surface it naturally affords;
- make chronological sections more likely to use Mermaid `timeline` when order,
  lag, or pending decisions are the central evidence;
- avoid adding a new quota, checklist, schema field, or visible meta
  explanation;
- keep prose natural and keep visuals source-grounded.

## Arms

| Arm | `generation_guidance_profile` | Meaning |
| --- | --- | --- |
| `visual_plan` | `visual-plan` | Current product baseline: sparse visual-aid planning plus productized visual type selection. |
| `visual_affordance_priming` | `visual-affordance-priming` | Candidate: current visual plan plus source-shape affordance priming. |

## Fixture Families

The runner reuses the six synthetic fixture families from experiments 25, 28,
29, and 30:

| Fixture | Structure under test | Natural affordance to watch |
| --- | --- | --- |
| `fictional-equity-dashboard` | ordered values and event markers | source-backed chart, timeline, compact value table |
| `industry-capacity-statistics` | regional comparison and bottleneck timing | table, chart, timeline |
| `agent-benchmark-matrix` | multi-axis trade-offs | matrix/table, source-backed chart if axes are defensible |
| `architecture-dependency-graph` | service dependencies and blast radius | flowchart dependency graph |
| `protocol-lifecycle` | actors, handoffs, state transitions | sequence diagram, state diagram |
| `scenario-risk-portfolio` | scenario bands and uncertainty over time | matrix/table, timeline |

## Decision Criteria

The candidate is useful only if it improves the report as a report.

Evaluate:

- whether timeline-heavy planned reports recover the timelines missed by
  experiment 30;
- whether numeric or ordered fixtures keep source-near charts when they reduce
  reader effort;
- whether architecture and lifecycle fixtures keep using stable Mermaid types;
- whether the candidate improves visual alignment without merely increasing
  visual count or report length;
- whether Mermaid diagrams validate before final Markdown submission;
- whether the report remains readable, natural, and source-grounded.

Automatic counts are observation signals. Manual reading still decides whether
the candidate improves the actual report.

## Result

The final candidate is promising, but this experiment alone is not a strict
statistical productization basis.

An initial smoke pass showed that the agent could name `timeline` in the plan
but still produce only Markdown timeline tables in the final report. The
candidate was narrowed once before the full run: when timing anchors are the
section's central evidence, prefer a Mermaid `timeline` as the orientation
surface unless exact lookup is the main reader task.

The calibrated full paired run completed 24 report generations, covering 6
fixture families x 2 report modes x 2 arms. There were 12 completed
baseline/candidate pairs and no generation failures.

| Metric | Result |
| --- | ---: |
| Completed pairs | 12 |
| Terminal failures | 0 |
| Median visual-aid delta | 0.0 |
| Visual-aid increase sign test, one-sided p | 0.5 |
| Median alignment delta | 0.0 |
| Alignment increase sign test, one-sided p | 0.0625 |
| Median word ratio over baseline | 0.963 |

Arm-level signals:

| Arm | Median visual aids | Median alignment score | Mermaid type totals |
| --- | ---: | ---: | --- |
| `visual_plan` | 5.0 | 1.5 | `flowchart`: 5, `xychart-beta`: 4, `timeline`: 1, `sequenceDiagram`: 3, `stateDiagram-v2`: 3 |
| `visual_affordance_priming` | 6.5 | 2.0 | `flowchart`: 4, `sequenceDiagram`: 4, `timeline`: 6, `xychart-beta`: 2, `stateDiagram-v2`: 3 |

The candidate did not win by simply adding more visual aids. The paired median
visual-count delta was 0.0 and the median report was slightly shorter than the
baseline. Its main effect was making timelines fire more reliably.

## Reading Notes

The target regression from experiment 30 improved. In `scenario-risk-portfolio
/ planned`, the baseline used only tables, while the candidate added a Mermaid
timeline for Month 1, Month 2, Month 3, and Month 6 decision anchors. Alignment
improved from 1 to 2 without increasing visual count.

`industry-capacity-statistics / long_form` also improved. The baseline used a
flowchart where the source's central movement was quarterly bottleneck timing.
The candidate used a Mermaid timeline for 2025 Q4 through 2026 Q3 and alignment
improved from 1 to 2. The report was also shorter.

`fictional-equity-dashboard / long_form` stayed strong. The candidate kept
source-backed `xychart-beta` line charts and added timelines for event markers,
improving alignment from 2 to 3 while reducing visual count and length.

Architecture and lifecycle fixtures were stable. Architecture stayed on
flowcharts, and protocol lifecycle reports kept sequence and state diagrams.
No candidate pair lost alignment, and no Mermaid validation problem appeared.

The remaining weaknesses are important:

- `agent-benchmark-matrix` still did not produce source-backed chart or
  quadrant-style visuals in either mode.
- `fictional-equity-dashboard / planned` added tables but still missed the
  expected source-backed chart and timeline.
- The alignment sign test landed at p = 0.0625, just above the stricter 0.05
  threshold, because only four pairs improved and none regressed.

Initial decision:

`visual_affordance_priming` is a better follow-up candidate than
`visual_clarity_seeking`. It solves the intended timeline activation problem
without visible regressions in this fixture set. However, because strict
statistical significance was not reached and benchmark/chart behavior remains
weak, this experiment records the candidate as promising rather than adopting
it as the default.

## Additional Repeat

After the initial result landed just above the strict p < 0.05 threshold, the
same runner was executed once more against a separate local archive:

`research-artifacts/liquid2/plasma/experiments/31-report-visual-affordance-priming-2026-07-22-repeat2/`

The repeat used the same product-shaped path and the same two arms. No product
default, schema, UI, or report renderer behavior was changed for the repeat.

Repeat result:

| Metric | Repeat result |
| --- | ---: |
| Completed pairs | 12 |
| Terminal failures | 0 |
| Median visual-aid delta | 1.0 |
| Visual-aid increase sign test, one-sided p | 0.171875 |
| Median alignment delta | 0.0 |
| Alignment increase sign test, one-sided p | 0.125 |
| Median word ratio over baseline | 0.956 |

Combined result across the initial run and the repeat:

| Metric | Combined result |
| --- | ---: |
| Completed pairs | 24 |
| Terminal failures | 0 |
| Alignment improvements | 7 |
| Alignment regressions | 0 |
| Alignment ties | 17 |
| Alignment increase sign test, one-sided p | 0.0078125 |
| Median visual-aid delta | 0.5 |
| Median word ratio over baseline | 0.960 |

Combined Mermaid type totals:

| Arm | Mermaid type totals |
| --- | --- |
| `visual_plan` | `flowchart`: 10, `sequenceDiagram`: 5, `stateDiagram-v2`: 5, `timeline`: 3, `xychart-beta`: 5 |
| `visual_affordance_priming` | `flowchart`: 11, `sequenceDiagram`: 8, `stateDiagram-v2`: 6, `timeline`: 12, `xychart-beta`: 4 |

The additional repeat strengthens the conclusion. The candidate still does not
win by visual-count alone: visual-aid increases were mixed, and median length
remained slightly shorter than baseline. The consistent signal is alignment.
Across 24 paired comparisons, seven pairs improved and no pair regressed.

Manual reading matches that direction. Timeline-shaped reports became more
likely to use Mermaid timelines as orientation aids while keeping tables for
exact lookup. The clearest examples were scenario monitoring and industry
bottleneck timing. The candidate also kept architecture and lifecycle fixtures
stable by staying with flowcharts, sequence diagrams, or state diagrams where
those were already the natural shape.

The main remaining weakness is unchanged: multi-axis benchmark material still
leans on Markdown tables and does not reliably discover a chart or quadrant
view. This experiment therefore supports adopting the affordance reminder for
general visual-aid planning, but it does not close the separate benchmark/chart
selection problem.

Productization:

The affordance reminder is now part of the existing product `visual-plan`
family: `visual-plan`, `section-brief-visual-plan`, and
`section-brief-cluster-memory-visual-plan`. The old
`visual-affordance-priming` profile remains accepted for stored event replay
and experiment reproducibility, but it is not a new active browser choice.

## Runner

The public runner is:

`plasma/scripts/experiments/report_visual_affordance_priming_experiment.py`

Prepare the isolated binary and fixture lock:

```bash
python3 plasma/scripts/experiments/report_visual_affordance_priming_experiment.py --action prepare
```

Smoke the two cases that exposed the experiment 30 regression:

```bash
python3 plasma/scripts/experiments/report_visual_affordance_priming_experiment.py --action run --topics industry-capacity-statistics scenario-risk-portfolio --modes planned --workers 2
python3 plasma/scripts/experiments/report_visual_affordance_priming_experiment.py --action analyze
python3 plasma/scripts/experiments/report_visual_affordance_priming_experiment.py --action packets
```

If smoke passes, run the paired comparison:

```bash
python3 plasma/scripts/experiments/report_visual_affordance_priming_experiment.py --action run --limit 6 --workers 2
python3 plasma/scripts/experiments/report_visual_affordance_priming_experiment.py --action analyze
python3 plasma/scripts/experiments/report_visual_affordance_priming_experiment.py --action packets
```

Use fewer workers if local provider stability becomes the limiting factor.
