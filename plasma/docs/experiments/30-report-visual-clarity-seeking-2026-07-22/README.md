# Report Visual Clarity-Seeking Experiment - 2026-07-22

This experiment follows experiments 28 and 29 on issue 174.

Experiment 28 showed that a broad preference for compact visual aids can make
reports use Mermaid diagrams and tables more actively. Experiment 29 showed
that a reader-task framing can reduce one meta-diagram failure, but it also made
some reports too conservative and did not improve aggregate visual alignment.

This follow-up tests a more active framing:

> While planning and writing each section, actively look for visual aids that
> make the reader understand the source-backed point faster or more clearly.

The candidate is intentionally not a list of prohibitions. It asks the report
agent to treat a visual aid as part of good explanation when the visual makes a
pattern, sequence, comparison, dependency, trade-off, scenario, range,
uncertainty, or category structure easier to grasp.

## Status

Completed. Product default has not been changed.

## Scope

The experiment uses the same product-shaped report path as experiments 28 and
29:

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

`research-artifacts/liquid2/plasma/experiments/30-report-visual-clarity-seeking-2026-07-22/`

## Product Question

The product question is whether positive metacognitive guidance helps report
writers choose better visual aids than the current `visual_plan` baseline.

The candidate should:

- look for places where a compact visual makes the source-backed point faster
  or clearer to understand;
- use visuals as explanation surfaces rather than decoration;
- match precision to the source's resolution, including exact values, ranges,
  directional movement, qualitative strength, or interpretive structure;
- preserve natural prose around the visual so the final report does not expose
  the internal selection process.

## Arms

| Arm | `generation_guidance_profile` | Meaning |
| --- | --- | --- |
| `visual_plan` | `visual-plan` | Current product baseline: sparse visual-aid planning plus visual type selection. |
| `visual_clarity_seeking` | `visual-clarity-seeking` | Candidate: current visual plan plus active clarity-seeking guidance for using visuals as explanatory surfaces. |

## Fixture Families

The runner reuses the six synthetic fixture families from experiments 28 and
29:

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

- whether visual aids become more useful without becoming decorative;
- whether numeric and ordered fixtures preserve source-near charts, timelines,
  and compact tables where they reduce reader effort;
- whether structure-heavy fixtures still get appropriate diagrams;
- whether scenario and uncertainty fixtures use timelines, matrices, or other
  compact aids when they make the comparison easier to read;
- whether Mermaid diagrams validate before final Markdown submission;
- whether the report remains readable, natural, and source-grounded.

## Result

The candidate increased the number of visual aids, but it did not improve
visual-aid fit. It is therefore recorded as a useful negative result, not as a
product-default change.

The full paired run completed 24 report generations, covering 6 fixture
families x 2 report modes x 2 arms. There were 12 completed baseline/candidate
pairs and no generation failures.

| Metric | Result |
| --- | ---: |
| Completed pairs | 12 |
| Median visual-aid delta | +2.0 |
| Visual-aid increase sign test, one-sided p | 0.0327 |
| Median alignment delta | 0.0 |
| Alignment increase sign test, one-sided p | 0.875 |
| Median word ratio over baseline | 1.086 |

This means the active clarity-seeking wording made reports add more visual
surfaces with statistical signal, but it did not make those surfaces better
matched to the fixture. The candidate also made the median report about 8.6%
longer.

## Reading Notes

The candidate helped in one important long-form dashboard case. For
`fictional-equity-dashboard / long_form`, it used a source-near `xychart-beta`
comparison for close vs. 5-day average and added a timeline for event markers.
The alignment score improved from 2 to 3.

The candidate also preserved good structure-heavy behavior. Architecture and
protocol fixtures kept their flowchart, sequence, and state diagrams, and the
candidate did not introduce invalid Mermaid blocks.

The weakness appeared in planned-mode timeline fixtures. For
`industry-capacity-statistics / planned` and `scenario-risk-portfolio /
planned`, the baseline used a timeline, while the candidate fell back to tables
only. In both cases the alignment score dropped from 2 to 1. This is the main
reason the candidate should not replace `visual_plan`.

The result suggests that "actively think about visual clarity" is not enough
by itself. A better next candidate should prime natural visual affordances
without turning into a prohibition list: chronology invites timeline,
dependency invites graph, lifecycle invites state or sequence, trade-off
invites matrix, and numeric movement invites a source-near chart.

## Runner

The public runner is:

`plasma/scripts/experiments/report_visual_clarity_seeking_experiment.py`

Prepare the isolated binary and fixture lock:

```bash
python3 plasma/scripts/experiments/report_visual_clarity_seeking_experiment.py --action prepare
```

Smoke the two cases that best expose the previous trade-off:

```bash
python3 plasma/scripts/experiments/report_visual_clarity_seeking_experiment.py --action run --topics fictional-equity-dashboard scenario-risk-portfolio --modes planned long_form --workers 2
python3 plasma/scripts/experiments/report_visual_clarity_seeking_experiment.py --action analyze
python3 plasma/scripts/experiments/report_visual_clarity_seeking_experiment.py --action packets
```

If smoke passes, run the paired comparison:

```bash
python3 plasma/scripts/experiments/report_visual_clarity_seeking_experiment.py --action run --limit 6 --workers 2
python3 plasma/scripts/experiments/report_visual_clarity_seeking_experiment.py --action analyze
python3 plasma/scripts/experiments/report_visual_clarity_seeking_experiment.py --action packets
```

Use fewer workers if local provider stability becomes the limiting factor.
