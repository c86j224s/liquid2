# Analysis Summary

Raw analysis file:

`~/research-artifacts/liquid2/plasma/experiments/12-long-form-session-strategy-2026-07-07/analysis/c4-normalized-section-headings-summary.md`

Extended six-sample analysis files:

- `~/research-artifacts/liquid2/plasma/experiments/12-long-form-session-strategy-2026-07-07/analysis/c4-six-sample-summary.json`
- `~/research-artifacts/liquid2/plasma/experiments/12-long-form-session-strategy-2026-07-07/analysis/c4-six-sample-product-cost-summary.json`

## What Has Been Established

C4 is directionally better than the previous B final assembly on structural
cleanliness. It is not evidence that C4 creates denser section drafts, because
the final C4 candidate intentionally preserves the section files produced by
the independent-section path.

| Metric | Result |
|---|---:|
| Completed paired samples | 6 |
| C4 wins on final length | 5 / 6 |
| C4 wins on source-section length | 0 / 6; all were ties |
| C4 h3+ heading drift | 0 / 6 |
| C4 adjacent duplicate headings | 0 / 6 |
| Mean C4/B final word ratio | 1.04x |
| Mean C4/B source section word ratio | 1.00x |

## Token Cost

The useful product comparison is not standalone C4 versus full B. C4 reuses the
section files and replaces assembly behavior, so the product-like comparison is
`B section drafting + C4 assembly` versus full B.

| Metric | Mean |
|---|---:|
| Composite C4/B input tokens | 0.99x |
| Composite C4/B uncached input tokens | 1.00x |
| Composite C4/B output tokens | 1.01x |

This means C4 is effectively cost-neutral relative to B when measured as a
product path. It should not be described as a token-diet win by itself.

## Statistical Read

The current evidence is enough to treat C4 as a good assembly candidate, but
not enough to claim broad report-quality superiority.

Final length improved in five of six samples. For a two-sided sign test, that is
still not a formal `p < 0.05` result. Section length was unchanged by design.
Heading cleanliness passed all six samples.

## Interpretation

C4 should be treated as the current leading assembly candidate. Productization
should be limited to deterministic heading normalization and section-preserving
assembly unless a separate experiment proves that another section-drafting
prompt improves density, flow, or tone.

## Risks

- Automated measurements do not replace human review of prose flow and tone.
- Intermediate C2 outputs for `s01` and `s02` were contaminated by the later
  heading-dedup behavior, so C2 should not be interpreted as a clean variant.
- Productization should implement C4-like section heading normalization in the
  assembly path, not as a second rewrite pass over the final report.
- The current C4 measurement reuses B section outputs. It must not be reported
  as an independent section-writing improvement.
