# Product-Path G2 Analysis Summary

Raw archive:
`~/research-artifacts/liquid2/plasma/experiments/11-product-path-g2-2026-07-07/`

## Files Of Record

- Run summary:
  `analysis/run-summary.json`
- Product-path audit:
  `analysis/product-path-audit.jsonl`
- Blind packet summary:
  `analysis/blind-packet-summary.json`
- Preference summary:
  `analysis/preference-summary.json`
- Preference results:
  `analysis/preference-results.jsonl`

## What Was Actually Tested

The run did not use the development or release Plasma DB. It created isolated
per-run databases under the raw archive, attached local source roots, and drove
the real `plasma reports draft` CLI path with `-mcp-mode auto`.

This matters because the previous generation-time tone experiment was an
offline source-to-report harness. This experiment checks whether the same
direction survives the product path where the agent must discover and read
registered sources through Plasma tools.

## Hard Gates

| Check | Result |
|---|---:|
| Variant runs | 32 |
| Runs with MCP source-read trace | 32 |
| Event-audit failures | 0 |
| Candidate quality failures | 0 |
| Judge failures | 0 / 200 |

The experiment is usable for interpretation. The main limitation is corpus size,
not execution failure.

## Aggregate Generation Metrics

| Variant | Mean bytes | Median bytes | Mean duration | Median duration |
|---|---:|---:|---:|---:|
| `P0-current` | 6,508 | 5,438 | 432.2s | 408.8s |
| `P0-H5` | 7,420 | 6,131 | 678.3s | 658.6s |
| `G2-current` | 9,317 | 8,779 | 545.9s | 493.9s |
| `G2-H5` | 7,788 | 7,428 | 709.2s | 544.1s |

`G2-current` produced the longest reports on average. H5 added time and sometimes
reduced the final artifact size relative to the raw G2 output.

## Pairwise Results

| Pair | First wins | Second wins | Ties | p-value | Read |
|---|---:|---:|---:|---:|---|
| `G2-H5` vs `P0-H5` | 4 | 4 | 0 | 0.63671875 | no win |
| `G2-current` vs `P0-current` | 7 | 1 | 0 | 0.03515625 | raw G2 win |
| `P0-H5` vs `P0-current` | 6 | 2 | 0 | 0.14453125 | positive but not decisive |
| `G2-H5` vs `G2-current` | 4 | 4 | 0 | 0.63671875 | no win |
| `G2-current` vs `P0-H5` | 5 | 3 | 0 | 0.36328125 | no win |

The product question was not whether `G2-current` can beat `P0-current`; it was
whether `G2 + H5` should become the product default. That question was not
supported by this run.

## Axis Detail

### `G2-H5` vs `P0-H5`

- Overall: `20-20` by judge pass, `4-4` by case.
- Tone: `P0-H5` led `21-15`, with 4 ties.
- Coverage: `G2-H5` led `21-18`, with 1 tie.
- Source safety: `G2-H5` led `17-9`, with 14 ties.

This is the central mixed result. `G2-H5` preserved slightly more coverage and
source safety, but the overall report preference tied.

### `G2-current` vs `P0-current`

- Overall: `G2-current` led `35-5` by judge pass and `7-1` by case.
- Tone: `P0-current` led `23-10`, with 7 ties.
- Coverage: `G2-current` led `32-7`, with 1 tie.
- Source safety: `G2-current` led `30-7`, with 3 ties.

This confirms that `G2` is doing useful work before H5. It improves substance,
not surface tone.

### `G2-H5` vs `G2-current`

- Overall: split `19-21` by judge pass and `4-4` by case.
- Tone: `G2-H5` led `24-15`, with 1 tie.
- Coverage: `G2-current` led `21-19`.
- Source safety: mostly tied.

H5 improves tone after G2, but the overall result is not better because some of
the coverage advantage is lost or becomes less visible.

## Interpretation

The result is not a rejection of `G2`. It is a rejection of the simpler product
claim: "add G2 before H5 and the final report will clearly improve."

The stronger interpretation is:

1. `G2` improves the raw report by making the writer preserve concrete detail.
2. H5 can improve wording, but it is not guaranteed to preserve all raw-report
   advantages.
3. The product needs a final-pass preservation strategy, not just more
   generation-time guidance.

## Product Recommendation

Keep `G2` behind experimental flags. Do not enable it as the default final
report behavior yet.

The next product candidate should be one of:

1. A patch-only H5 pass that edits wording while preserving raw report structure
   and content.
2. A final-pass audit that compares raw and H5 outputs for dropped details before
   accepting the final artifact.
3. A two-artifact UI where users can inspect raw and humanized versions when the
   preservation audit is uncertain.

The first option best matches the current product direction because it avoids
turning the report generator into a rigid harness while still protecting the
substance that `G2` improved.

## Limits

- Four source topics is small.
- Two replicates per topic are useful, but not equivalent to eight independent
  topics.
- Model judges are useful for iteration, not a substitute for user judgment.
- The exact sign test treats sample-replicate cases as units; read p-values as
  directional evidence, not final statistical truth.

