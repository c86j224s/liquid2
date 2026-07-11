# Product-Path G2 Report Experiment - 2026-07-07

This experiment checks whether the `G2` generation-time report guidance from
experiment 10 still helps when the report is generated through the real Plasma
CLI report path, with sources available through Plasma/MCP instead of being
prompt-pasted into an offline harness.

## Status

- Raw archive root:
  `~/research-artifacts/liquid2/plasma/experiments/11-product-path-g2-2026-07-07/`
- Planning file:
  `.fleet/plans/plasma-product-path-g2-report-experiment.md`
- Harness:
  `plasma/scripts/experiments/product_path_g2_report_experiment.py`
- Decision: do not make `G2 + H5` the default product path yet.

## Product Question

Experiment 10 showed that `G2` helped an offline source-to-report harness avoid
substance loss. That was useful, but it did not prove the same result in the
product path.

This experiment asks:

1. Does `G2` still help when the report writer must read registered Plasma
   sources through MCP?
2. Does the benefit survive the H5 post-generation Korean tone patch?
3. Is there enough evidence to change the product default?

## Product-Path Shape

Each run used an isolated SQLite database under the experiment archive. The
development or release Plasma databases were not used.

For each sample and replicate, the harness:

1. Created a new isolated mission DB.
2. Attached one local-path source root with `plasma sources attach-local`.
3. Ran `plasma reports draft` against that isolated DB.
4. Passed `-mcp-mode auto` and `-local-source-root sample=...`.
5. Required the agent to read the registered source through Plasma/MCP.
6. Extracted the raw and final report artifacts from the isolated DB.
7. Audited the ledger for source-read traces and report-event consistency.

The harness did not paste source text directly into the report prompt.

## Variants

| Variant | Generation guidance | H5 post-generation tone pass |
|---|---|---|
| `P0-current` | current product-like generation | off |
| `P0-H5` | current product-like generation | on |
| `G2-current` | `G2` substance-preserving generation guidance | off |
| `G2-H5` | `G2` substance-preserving generation guidance | on |

`G2` is not a humanizer. It is a short generation-stage instruction that tells
the report writer not to improve fluency by dropping concrete conditions,
numbers, caveats, source distinctions, URLs, code, commands, or procedural
details.

## Corpus And Scale

The full run used four archived source packets and two replicates per packet:

1. Phone purchase comparison.
2. Sengoku history / Oda Nobunaga context.
3. OAuth/OIDC server design.
4. Ollama UI operational comparison.

This produced:

- 8 sample-replicate blocks.
- 4 report variants per block.
- 32 generated report artifacts.
- 40 blind pairwise comparison cases.
- 5 judge passes per case.
- 200 total blind judge passes.

The corpus has four topics, not eight independent topics. The two replicates are
useful for stochastic stability, but they should not be treated as fully
independent corpus samples.

## Generation Audit

All generated candidates passed the product-path hard gates.

| Variant | Runs | Source-read traces | Event failures | Candidate quality failures | Mean bytes | Mean seconds |
|---|---:|---:|---:|---:|---:|---:|
| `P0-current` | 8 | 8 | 0 | 0 | 6,508 | 432.2 |
| `P0-H5` | 8 | 8 | 0 | 0 | 7,420 | 678.3 |
| `G2-current` | 8 | 8 | 0 | 0 | 9,317 | 545.9 |
| `G2-H5` | 8 | 8 | 0 | 0 | 7,788 | 709.2 |

`G2-current` expanded reports substantially. That is partly the intended
coverage-preservation effect, but it remains a verbosity-risk area.

## Blind Preference Results

Pair results are counted by sample-replicate case after five blind judge passes.
The p-value column is an exact one-sided sign test for the first variant beating
the second variant, excluding ties.

| Pair | Case result | Judge-pass result | p-value | Interpretation |
|---|---:|---:|---:|---|
| `G2-H5` vs `P0-H5` | 4-4 | 20-20 | 0.63671875 | no evidence that `G2-H5` beats current H5 |
| `G2-current` vs `P0-current` | 7-1 | 35-5 | 0.03515625 | `G2` helps raw generation |
| `P0-H5` vs `P0-current` | 6-2 | 31-9 | 0.14453125 | H5 trends positive, but this run is not decisive by case count |
| `G2-H5` vs `G2-current` | 4-4 | 19-21 | 0.63671875 | H5 does not reliably improve a `G2` report |
| `G2-current` vs `P0-H5` | 5-3 | 25-15 | 0.36328125 | not decisive |

The only statistically notable win in this product-path run is
`G2-current` over `P0-current`.

## Axis Breakdown

The detailed judge axes explain the product decision.

| Pair | Tone signal | Coverage signal | Source-safety signal |
|---|---|---|---|
| `G2-H5` vs `P0-H5` | `P0-H5` 21, `G2-H5` 15, tie 4 | `G2-H5` 21, `P0-H5` 18, tie 1 | `G2-H5` 17, `P0-H5` 9, tie 14 |
| `G2-current` vs `P0-current` | `P0-current` 23, `G2-current` 10, tie 7 | `G2-current` 32, `P0-current` 7, tie 1 | `G2-current` 30, `P0-current` 7, tie 3 |
| `P0-H5` vs `P0-current` | `P0-H5` 19, `P0-current` 17, tie 4 | `P0-H5` 33, `P0-current` 5, tie 2 | `P0-H5` 18, `P0-current` 7, tie 15 |
| `G2-H5` vs `G2-current` | `G2-H5` 24, `G2-current` 15, tie 1 | `G2-current` 21, `G2-H5` 19 | `G2-current` 13, `G2-H5` 11, tie 16 |

`G2` improves coverage and source safety before H5, but H5 does not preserve a
clear final advantage when compared against current generation plus H5.

## Decision

Do not enable `G2-H5` as the product default from this run alone.

Keep these conclusions:

1. `G2` is useful as a substance-preservation direction.
2. The current H5 final report path should not simply inherit `G2` as a default.
3. Product work should focus on preserving `G2`'s coverage gain through the final
   H5 patch, not on adding more prompt text.
4. The next experiment should isolate why H5 erases or destabilizes the
   generation-stage advantage.

## Follow-Up

Recommended next checks:

1. Compare patch-only H5 against whole-report H5 on the same product-path
   artifact set.
2. Add a regression check that reports whether final H5 dropped sections,
   source distinctions, URLs, numbers, commands, or caveats from the raw report.
3. Run at least one more corpus with more topics before declaring a product
   default.

See [`analysis-summary.md`](analysis-summary.md) for the aggregate detail.

