# Protocol

## Question

Experiments 23 and 24 established that sparse visual-aid planning can be useful.
They did not test whether the agent chooses the right visual type for different
source structures.

This experiment compares the current `visual-plan` default with a candidate
`visual-type-manual` profile that gives a compact type-selection guide.

## Product-Path Rule

The experiment must stay close to the product path:

- create an isolated mission;
- attach a source through the product source path;
- request a planned or long-form report through the product HTTP endpoint;
- let report agents read sources through MCP/source tools;
- keep the submitted report plan schema unchanged;
- keep generated reports, ledgers, prompts, judging packets, and DBs out of Git.

The experiment must not use a prompt-only source dump as the evidence path.

## Arms

| Arm | Profile | Role |
| --- | --- | --- |
| `visual_plan` | `visual-plan` | Current product default. |
| `visual_type_manual` | `visual-type-manual` | Candidate with a compact guide for choosing visual type by source structure. |

## Candidate Semantics

The candidate guidance is intentionally weak. It should:

- tell the planner to name visual intent inside existing `purpose` or
  `coverage_notes`;
- tell the writer to match visual type to structure;
- prefer a table for exact dense numbers;
- allow source-backed chart types when values and axes are explicitly supported;
- prefer stable `flowchart`/`graph` dependency diagrams for architecture
  dependency graphs;
- use `sequenceDiagram`, `stateDiagram-v2`, and `timeline` for interactions,
  lifecycle, and chronology when they fit;
- avoid compatibility-sensitive grammars unless needed and validated;
- preserve prose as the primary explanation.

It should not:

- add a report plan schema field;
- force visuals into prose-only topics;
- invent numeric values, axes, causal links, or architecture dependencies;
- make C4, sankey, block, packet, or requirement diagrams the default.

## Fixtures

The six fixtures are synthetic and source-backed inside the local archive:

1. Fictional equity dashboard: OHLC-style table, volume, and event markers.
2. Industry capacity statistics: regional capacity, utilization, lead time, and
   bottleneck timing.
3. Agent benchmark matrix: accuracy, latency, tool-call reliability, context,
   and cost trade-offs.
4. Complex architecture dependency graph: services, stores, workers, event
   streams, read models, criticality, and failure impact.
5. Protocol lifecycle: actors, happy path, states, and terminal decisions.
6. Scenario risk portfolio: broad probability bands, risk dimensions, and time
   anchors.

These fixtures are designed to catch one-pattern behavior. A report that only
adds Markdown tables everywhere is not enough; a report that forces fragile
Mermaid everywhere is also not enough.

## Modes

Run both:

- `planned`: normal report generation;
- `long_form`: long-form generation with `section_fanout`.

## Metrics

Automatic metrics are diagnostic only:

- terminal success;
- final word count;
- table count;
- Mermaid fence count;
- Mermaid type counts;
- Mermaid validation-call signal;
- expected visual-family alignment score;
- visual and alignment deltas against `visual_plan`.

## Manual Reading

Read whole reports. The main question is whether the candidate produces a more
useful report, not whether it produces more visuals.

Manual criteria:

- the visual type matches the source structure;
- exact numeric data remains exact;
- quantitative charts do not invent unsupported values;
- architecture dependency graphs make critical paths and blast-radius risk
  easier to understand;
- Mermaid diagrams are stable enough to render in Plasma;
- visuals are introduced and interpreted by nearby prose;
- prose remains coherent and source-grounded.

## Stop Conditions

Do not productize if the candidate often:

- adds decorative visuals;
- uses the same table shape for every source structure;
- draws charts from inferred or invented numbers;
- produces architecture diagrams with unsupported dependencies;
- reaches for compatibility-sensitive Mermaid grammar when a stable flowchart
  would work;
- makes reports less readable even if visual counts increase.
