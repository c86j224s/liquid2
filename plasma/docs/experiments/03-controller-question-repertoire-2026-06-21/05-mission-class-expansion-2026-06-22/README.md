# Controller Mission-Class Expansion - 2026-06-22

This expansion validates the adaptive controller hypothesis beyond the M5 code-analysis runs.

The first attempt was discarded before commit because resumed turns were launched from the repository root and several runs read files outside their source corpus. This clean run launches every resumed turn from the mission source directory and records source-corpus violations separately.

Mission classes:

- M1: narrow-source
- M3: broad-topic
- M6: source-conflict

Source revision: `7ac8fb7`
Source corpora: `<experiment-source-root>`

## Result Summary

This is a mission-class expansion slice, not the full 24-run experiment from
the Kirov plan. It ran three mission classes, one seed each, across all four
variants:

- M1 narrow-source: V2 won the blind final-report judge.
- M3 broad-topic: V3 won the blind final-report judge.
- M6 source-conflict: V3 won the blind final-report judge.

The result strengthens the adaptive-controller hypothesis beyond the earlier
M5-only code-analysis work. The useful conclusion is not "always ship V3" or
"adaptive always wins." The useful conclusion is that question repertoire
selection changes report quality, and the adaptive variants produced the best
final reports in this slice.

The Kirov plan still applies: because the sample is small and mission behavior
can split by class, Plasma should not promote one universal controller default
from this slice alone. The next step should compare V2 and V3 more directly and
test a small repertoire-selection rule.

## Blind Judge Winners

| Mission | Class | Winner | Variant | Notes |
| --- | --- | --- | --- | --- |
| M1 | narrow-source | K1 | V2 | Best at turning a narrow corpus into concrete artifact/recovery/trust risks. |
| M3 | broad-topic | K2 | V3 | Best coverage across C1 loop, UI-less research IDE, source/result boundaries, automatic investigation, and report artifact checks. |
| M6 | source-conflict | K4 | V3 | Best separation of C1 default, legacy read-only, experiment-only, and future investigation boundaries. |

## Contamination Review

The clean runner emitted conservative `final_report_references_outside_source_catalog`
flags for M1-V0, M3-V0, M3-V1, and M3-V2. Manual review found no material use
of outside source material:

- M1-V0 mentioned `legacy-ledger-loop.md` only to say it was not inspected.
- M3 flagged runs cited `product-flow.md` and `automatic-investigation.md`,
  both of which were part of the M3 source catalog.
- M3 flagged runs mentioned `legacy-ledger-loop.md` or
  `evidence-signal-model.md` only as absent/uninspected documents.
- Tool traces used files under the run's source corpus; no repo-root source
  reads like `plasma/internal/...`, `server.go`, or `app.js` were found in the
  reviewed traces.

The flags remain in the raw manifests as conservative signals, but the decision
memo treats them as filter false positives rather than primary contamination.

## Limits

- One seed per mission class is not statistically significant.
- Only final-report-only judging was generated in this slice; transcript-quality
  judging remains a follow-up.
- The runner still lacks a robust hard process timeout around long final-report
  generations.
- Token usage is high even with read-only source corpora, so the next experiment
  should reduce variants or use a smaller V2/V3 follow-up.
