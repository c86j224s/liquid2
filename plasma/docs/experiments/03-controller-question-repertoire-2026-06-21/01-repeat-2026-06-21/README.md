# Controller Question Repertoire Repeat - 2026-06-21

This repeat run follows the initial M5 pilot, but changes the protocol so the
controller must make at least three steering decisions per run. The purpose is
to observe whether V1, V2, and V3 behave differently once the run is long enough
for stagnation or scheduled divergence to appear.

This directory is separate from the first pilot artifacts. It uses fresh source
copies from the current branch after the C1 Markdown artifact browser fix.

## Scope

- Mission: M5 code-analysis mission.
- Variants: V0, V1, V2, V3.
- Seeds: 0002, 0003, 0004 are reserved for repeated runs.
- Minimum controller decisions per completed run: 3.
- Controller role: question-only steering.
- Main agent role: inspect source and produce the analysis/report.

## Decision Rule

Do not select a winning controller variant unless the repeat produces visible
variant separation. If the variants still collapse to the same questions, record
that as a finding instead of forcing a ranking.

## Seed 0002 Result

Seed 0002 completed for all four variants with three controller decisions and a
final report per variant. The run produced visible separation:

- V0 stayed close to the baseline call-path audit.
- V1 stayed in confirm/detect mode and produced the cleanest product-vs-test
  triage.
- V2 fired a creative switch on decision 3 and shifted the atomicity issue into
  a product recovery and observability problem.
- V3 fired its scheduled divergence on decision 3 and produced the clearest
  end-to-end artifact lifecycle map.

This is not a statistical conclusion. Seeds 0003 and 0004 remain available if a
larger repeat is needed.
