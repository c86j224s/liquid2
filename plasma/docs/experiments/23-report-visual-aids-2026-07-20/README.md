# Report Visual-Aid Experiment - 2026-07-20

This experiment tests one narrow report-quality question:

Can Plasma use Markdown tables and Mermaid diagrams as reading aids without
making reports feel decorative, repetitive, or less source-grounded?

The experiment is intentionally not a renderer project. Mermaid rendering and
Markdown preview support already exist and are observed here only as product
surface constraints. This experiment compares generation guidance candidates.

## Status

- State: six-topic expansion completed; user reading selected `visual_plan`
  for productization
- Public protocol: [`protocol.md`](protocol.md)
- Smoke summary: [`smoke-summary.md`](smoke-summary.md)
- Issue: #153
- Runner: `plasma/scripts/experiments/report_visual_aids_experiment.py`
- Raw archive target:
  `research-artifacts/liquid2/plasma/experiments/23-report-visual-aids-2026-07-20/`

## Product Question

Plasma reports are mostly prose-first. That is usually desirable, but some
source material is easier to understand when a table or diagram supplements the
argument: comparisons, sequences, decision paths, dependency chains, timelines,
and trade-offs.

The risk is that a simple instruction such as "use more visuals" can produce
filler tables, fragile Mermaid syntax, or diagrams that repeat the paragraph
beside them. This experiment asks whether light guidance can improve reader
understanding while preserving coherent prose.

## Compared Arms

| Arm | Meaning |
| --- | --- |
| `baseline` | Current product-equivalent G2 report guidance. |
| `visual_supplement` | Adds weak writing guidance: use tables or Mermaid only when they help the reader understand structure better than prose alone. |
| `visual_plan` | Adds the same writing guidance, and asks the planning step to place visual-aid intent inside existing plan fields when useful. |

The arms use the existing `generation_guidance_profile` request field. Product
defaults were not changed during the experiment run itself.

## Product Decision

After the six-topic expansion, the user read rendered report samples and judged
`visual_plan` to be the better product default. The judgment was that
`visual_supplement` made the visual change stronger but often lengthened reports,
while `visual_plan` added tables and occasional Mermaid diagrams more selectively
and kept report length and flow closer to the baseline.

The productization step applies the same `visual_plan` profile to normal planned
reports and long-form reports by default. It does not add a new plan schema, a
visual-aid gate, renderer changes, or a requirement that every report include a
table or diagram.

## Decision Shape

A candidate is promising only if whole-report reading shows:

- the table or diagram helps understanding rather than repeating adjacent prose;
- the surrounding prose explains why the visual exists and what to take from it;
- source-backed detail, caveats, and uncertainty remain visible;
- Mermaid diagrams are simple enough to validate and render reliably;
- the report still reads as an edited article, not as a list of visual widgets.

Automatic metrics such as visual count, Mermaid fence count, and validation
mentions are observation signals only. They cannot prove quality by themselves.

## Non-Goals

- Do not change the product default report prompt from automatic metrics alone;
  the product change depends on user whole-report reading.
- Do not expose a new UI option during the experiment.
- Do not force a table or Mermaid diagram into every report or every section.
- Do not change report artifact types or HTML artifact generation.
- Do not use development or release databases.
- Do not commit raw reports, copied fixtures, provider logs, prompt packets,
  session IDs, screenshots, or generated previews.

## Files

- Protocol: [`protocol.md`](protocol.md)
- Smoke summary: [`smoke-summary.md`](smoke-summary.md)
- Runner: `plasma/scripts/experiments/report_visual_aids_experiment.py`
- Raw outputs: local archive only.
