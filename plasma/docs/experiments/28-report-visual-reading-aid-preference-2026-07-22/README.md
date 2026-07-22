# Report Visual Reading-Aid Preference Experiment - 2026-07-22

This experiment follows experiment 27 on issue 174.

Experiment 27 tested whether explicit evidence-strength guidance would make
report agents use Mermaid diagrams, tables, and qualitative charts more
appropriately. The result was useful but too conservative: the candidate avoided
unsupported precision, but it did not reliably make the writer use helpful
interpretive visuals when prose could responsibly explain the same structure.

This follow-up tests a more direct reading-aid preference:

> When a section has source-backed structure, prefer a compact visual aid over
> another explanatory paragraph if the visual would make the structure easier to
> scan.

## Status

Completed 24 product-path report generations across six fixtures, two report
modes, and two arms. No product default has been changed.

## Scope

The experiment uses the same product-shaped report path as experiment 27:

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

`research-artifacts/liquid2/plasma/experiments/28-report-visual-reading-aid-preference-2026-07-22/`

## Product Question

The product already supports Mermaid and Markdown tables. The question is not
whether rendering works. The question is whether the report writer can be guided
to treat visuals as reader aids when a section contains structure that prose
would otherwise explain at length.

The candidate should not draw decorative or unsupported charts. It should
recognize that research, business, and IR-style documents often use approximate,
indexed, directional, or qualitative visuals. The visual's implied precision
must match the source's own resolution.

## Arms

| Arm | `generation_guidance_profile` | Meaning |
| --- | --- | --- |
| `visual_plan` | `visual-plan` | Current product baseline: sparse visual-aid planning plus visual type selection. |
| `visual_reading_aid_preferred` | `visual-reading-aid-preferred` | Candidate: current visual plan plus a conditional preference for compact visuals when source-backed structure would otherwise become long prose. |

The candidate asks the agent to:

- prefer a compact visual aid for relationships, sequences, dependencies,
  lifecycles, comparisons, trade-offs, timelines, and uncertainty structures;
- match visual precision to the source resolution, not to stock-chart-grade
  exactness;
- use exact values, ranges, indexed movement, directional labels, relative
  strength, qualitative labels, or interpretive structure as appropriate;
- skip visuals that only decorate or duplicate a simple point.

## Fixture Families

The runner reuses the six synthetic fixture families from experiment 27:

| Fixture | Structure under test |
| --- | --- |
| `fictional-equity-dashboard` | ordered values and event markers |
| `industry-capacity-statistics` | regional comparison and bottleneck timing |
| `agent-benchmark-matrix` | multi-axis trade-offs |
| `architecture-dependency-graph` | service dependencies and blast radius |
| `protocol-lifecycle` | actors, handoffs, state transitions |
| `scenario-risk-portfolio` | scenario bands and uncertainty |

## Decision Criteria

The candidate is useful only if it improves the report as a report.

Evaluate:

- whether structure-heavy sections use compact visuals where prose would become
  harder to scan;
- whether the visual's precision matches the source resolution;
- whether approximate, directional, or qualitative visuals are clearly labeled
  without repetitive disclaimers;
- whether Mermaid diagrams validate before final Markdown submission;
- whether the report remains readable and source-grounded;
- whether normal and long-form reports avoid decorative filler.

## Result

The run completed without terminal failures:

| Signal | `visual_plan` | `visual_reading_aid_preferred` |
| --- | ---: | ---: |
| Completed runs | 12 | 12 |
| Completed paired comparisons | 12 | 12 |
| Median visual aids | 5.5 | 7.0 |
| Median visual alignment score | 2.0 | 2.0 |
| Unvalidated Mermaid signals | 0 | 0 |

Candidate deltas:

| Signal | Value |
| --- | ---: |
| Median visual-count delta | 0.0 |
| Visual increase sign-test p-value, one-sided | 0.623046875 |
| Median alignment-score delta | 0.0 |
| Alignment increase sign-test p-value, one-sided | 0.5 |
| Median word ratio over baseline | 0.9266847506581806 |

Mermaid type totals:

| Arm | Mermaid types |
| --- | --- |
| `visual_plan` | `flowchart`: 5, `timeline`: 3, `xychart-beta`: 2, `sequenceDiagram`: 2, `stateDiagram-v2`: 2 |
| `visual_reading_aid_preferred` | `flowchart`: 9, `xychart-beta`: 1, `timeline`: 4, `sequenceDiagram`: 2, `stateDiagram-v2`: 2 |

Interpretation:

- The candidate made reports shorter overall. Median word ratio was about 0.93
  of baseline, which is consistent with the intended "compact visual instead of
  more prose" pressure.
- The candidate increased Mermaid usage, especially flowcharts, but the paired
  count and alignment improvements were not statistically strong.
- Direct reading showed useful wins on structure-heavy cases. In
  `scenario-risk-portfolio` planned mode, the candidate replaced a longer
  table-heavy explanation with a cleaner timeline for Month 1-6 observation
  anchors. In `industry-capacity-statistics` planned mode, the candidate made
  the bottleneck sequence easier to scan with a timeline while keeping the
  capacity table grounded.
- Direct reading also showed a failure mode. In
  `fictional-equity-dashboard` long-form mode, the candidate lost the baseline
  timeline and spent one flowchart on the boundary between allowed comparison
  and forbidden investor-psychology inference. That diagram was not fabricated,
  but it was less useful than a source-backed chart or timeline for the numeric
  dashboard fixture.
- In `agent-benchmark-matrix`, the candidate became more compact and sometimes
  added a decision flow, but it still did not reliably choose the expected
  trade-off chart or quadrant-style view.

Decision:

Do not productize `visual-reading-aid-preferred` as a universal default. It is
better than `visual-evidence-fit` for structural and qualitative material, but
it can overuse flowcharts for explanation boundaries and underuse
source-backed charts or timelines for numeric dashboards.

The next candidate should preserve the conditional visual preference but add a
stronger type-selection guard:

- for numeric or ordered observations, prefer a source-backed chart, timeline,
  or compact table over a meta-level explanation diagram;
- for architecture, process, lifecycle, dependency, scenario, and uncertainty
  structures, prefer compact diagrams when they reduce prose load;
- do not use a diagram mainly to explain what cannot be inferred if a more
  source-near visual would help the reader more.

## Runner

The public runner is:

`plasma/scripts/experiments/report_visual_reading_aid_preference_experiment.py`

Prepare the isolated binary and fixture lock:

```bash
python3 plasma/scripts/experiments/report_visual_reading_aid_preference_experiment.py --action prepare
```

Smoke one structure-heavy topic first:

```bash
python3 plasma/scripts/experiments/report_visual_reading_aid_preference_experiment.py --action run --topics architecture-dependency-graph --modes planned --workers 1
python3 plasma/scripts/experiments/report_visual_reading_aid_preference_experiment.py --action analyze
python3 plasma/scripts/experiments/report_visual_reading_aid_preference_experiment.py --action packets
```

If smoke passes, run the paired comparison:

```bash
python3 plasma/scripts/experiments/report_visual_reading_aid_preference_experiment.py --action run --limit 6 --workers 2
python3 plasma/scripts/experiments/report_visual_reading_aid_preference_experiment.py --action analyze
python3 plasma/scripts/experiments/report_visual_reading_aid_preference_experiment.py --action packets
```

Use fewer workers if local provider stability becomes the limiting factor.
