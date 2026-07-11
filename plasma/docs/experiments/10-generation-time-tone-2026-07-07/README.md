# Generation-Time Tone Experiment - 2026-07-07

This experiment checks whether Plasma should improve Korean report wording during
report generation, rather than relying only on the existing H5 post-generation
humanize pass.

## Status

- Raw archive root:
  `~/research-artifacts/liquid2/plasma/experiments/10-generation-time-tone-2026-07-07/`
- Planning file:
  `.fleet/plans/plasma-generation-time-tone-experiment.md`
- Product runtime: unchanged by this experiment.
- Decision: do not replace H5 with generation-time tone guidance alone.

## Product Question

Can the report writer produce a naturally written Korean report from the start,
without a separate H5 tone patch pass?

The tested answer is nuanced:

- Plain generation-time tone guidance is not enough.
- A stronger generation-time instruction that protects coverage and source
  safety is useful.
- H5 remains useful as a tone-focused post-generation patch.

## Variants

- `B0`: current-like source-to-report generation without special tone guidance.
- `G1`: generation-time Korean tone guidance only.
- `H5post`: `B0` followed by the existing H5 whole-report tone patch pass.
- `G2`: generation-time guidance that combines Korean tone direction with an
  explicit anti-compression and coverage-preservation constraint.

`G2` should be read as a substance-preserving generation guide, not as a
replacement humanizer.

## Sample Corpus

The local experiment used four archived source packets:

1. Phone purchase comparison.
2. Sengoku history / Oda Nobunaga context.
3. OAuth/OIDC server design.
4. Ollama UI operational comparison.

The raw source packets, generated candidates, judge packets, and detailed logs
remain outside Git under the raw archive root. This repository keeps only the
decision summary and aggregate metrics.

## Result Summary

The first generation-time attempt, `G1`, was unstable. It split evenly against
both `B0` and `H5post`, and it heavily compressed some samples.

`H5post` remained a strong positive control over `B0`: it was preferred 14
times, lost once, and tied 5 times across 20 repeated model-judge decisions.

`G2` then tested the important missing condition: report fluency must not be
achieved by dropping concrete details, caveats, numbers, source links, or
procedural steps. That changed the result.

| Comparison | Overall result | Main reason |
|---|---:|---|
| `G1` vs `B0` | 10-10 | No stable win; compressed some samples |
| `G1` vs `H5post` | 10-10 | No stable win; sample-dependent |
| `H5post` vs `B0` | 14-1, 5 ties | Better tone, stable structure |
| `G2` vs `B0` | 18-2 | Better coverage and source safety |
| `G2` vs `H5post` | 18-2 | Better coverage and source safety |
| `G2` vs `G1` | 20-0 | G2 avoided G1's compression |

The important caveat is that `G2` did not win because it had the best pure tone.
On the tone-only axis, `H5post` still beat `G2` 11-7 with 2 ties in the
`G2`-vs-`H5post` comparison.

## Decision

Adopt this as the product direction to test next:

1. Add a short `G2`-style generation guide to report writing so the report does
   not trade substance for fluency.
2. Keep H5 as the tone-focused patch pass.
3. Do not describe `G2` as a humanize replacement.
4. Before changing the default runtime behavior, verify `G2 + H5post` in the
   live Plasma report runner path.

## Limits

- The corpus had four samples.
- Each sample was judged through five repeated model-judge passes; these are not
  the same as twenty independent corpus samples.
- The experiment used an offline source-to-report harness rather than the full
  live Plasma report runner.
- `G2` sometimes produced much longer reports. That can be useful coverage, but
  product integration should keep a verbosity audit.

See [`analysis-summary.md`](analysis-summary.md) for the aggregate detail.

