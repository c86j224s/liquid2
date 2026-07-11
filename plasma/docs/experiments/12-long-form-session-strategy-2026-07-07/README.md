# Long-Form Report Session Strategy Experiment - 2026-07-07

This experiment checks how Plasma should run long-form Markdown report
generation after a mission has accumulated enough source material.

The product problem is practical: the current long-form report path can fail
late in section generation when the report session becomes too heavy. Earlier
R8-style experiments suggested that section-level divide-and-conquer can avoid
that failure, but the product path must still preserve source fidelity, readable
flow, and stable report structure.

## Status

- Raw archive root:
  `~/research-artifacts/liquid2/plasma/experiments/12-long-form-session-strategy-2026-07-07/`
- Planning file:
  `.fleet/plans/long-form-report-session-strategy-experiment.md`
- Harness:
  `plasma/scripts/experiments/long_form_session_strategy_experiment.py`
- Current decision:
  C4 is supported as an assembly-stage candidate, not as a new section-writing
  strategy. Six paired samples have completed; the evidence supports heading
  normalization and final assembly cleanup, but not a claim that C4 itself makes
  section drafts denser.

## Product Question

Can an independent-section long-form report strategy avoid same-session context
pressure while preserving section bodies and producing a coherent, structurally
clean final report?

The experiment compares:

1. A same-session product-style chain.
2. An independent-section candidate path.
3. Follow-up refinements that preserve section bodies while improving heading
   assembly.

## Variants

| Variant | Meaning |
|---|---|
| `A0-current-chain` | Current product-style long-form path. One forked report session is reused across planning, sections, part assembly, and final framing. |
| `B-independent-sections` | R8-aligned candidate. Each section is drafted in an independent provider conversation, and the final report is assembled from durable section outputs. |
| `C1-reframe` | Reframe-only pass over B outputs to improve connective flow. |
| `C2-denser-sections` | Denser section drafting prompt. |
| `C3-no-duplicate-heading` | Assembly rule that prevents adjacent duplicate headings. |
| `C4-normalized-section-headings` | Final current candidate. It reuses the independent section files, assembles parts, and emits deterministic normalized final headings. |

## Current Result

The completed paired samples are `s01` through `s06`.

| Sample | B final words | C4 final words | C4/B final | B section words | C4 source section words | C4/B section |
|---|---:|---:|---:|---:|---:|---:|
| `s01` | 15,815 | 15,882 | 1.00x | 13,815 | 13,815 | 1.00x |
| `s02` | 15,612 | 15,463 | 0.99x | 13,515 | 13,515 | 1.00x |
| `s03` | 15,196 | 15,413 | 1.01x | 13,360 | 13,360 | 1.00x |
| `s04` | 9,688 | 10,286 | 1.06x | 8,881 | 8,881 | 1.00x |
| `s05` | 10,786 | 11,680 | 1.08x | 9,689 | 9,689 | 1.00x |
| `s06` | 12,819 | 13,612 | 1.06x | 11,510 | 11,510 | 1.00x |

Mean C4/B final word ratio: `1.04x`.

Mean C4/B source section word ratio: `1.00x`.

C4 removed the heading-structure problems observed in the earlier B/C outputs:
no h3+ heading drift and no adjacent duplicate headings were observed in the
six completed samples.

## Token-Cost Read

C4 is not a standalone replacement for the independent-section B run. It uses
the section outputs already produced by the B-style section phase and replaces
the later part/final assembly behavior. Product-like cost should therefore be
read as:

`B section drafting + C4 assembly`

Against full B, this composite averaged `0.99x` input tokens, `1.00x` uncached
input tokens, and `1.01x` output tokens across the six completed samples.

This supports the direction that long-form section drafting should stay split,
and that final assembly can be normalized without materially increasing the
cost profile.

## Interpretation

C4 is the best current assembly candidate. It preserves section bodies and
fixes final heading structure while keeping B-level cost.

The evidence is mixed by metric. Final length improved in five of six paired
samples, which is directional but not a formal `p < 0.05` sign-test result.
Section length did not improve because C4 deliberately preserves the section
files. Heading cleanliness passed on all six samples.

## Known Caveat

`s01` and `s02` C2 outputs were produced after the no-duplicate-heading runner
patch had already landed, so those intermediate C2 outputs are contaminated by
C3-like behavior. The final candidate should therefore be read as C4, not as a
pure C2 result.

During the extension run, two A0 smoke runs were briefly started in parallel
before the archive-process preflight could observe the first process. They used
separate archive DBs, provider homes, and loopback ports and completed, but the
harness should use a stronger external lock before future broad parallel runs.

## Files

- Protocol: [`protocol.md`](protocol.md)
- Sample manifest: [`sample-manifest.md`](sample-manifest.md)
- Analysis summary: [`analysis-summary.md`](analysis-summary.md)
- Decision memo: [`decision-memo.md`](decision-memo.md)
