# Process Quality Statistical Plan

This plan is pre-registered before scored investigation runs.

## Primary Endpoint

Process score on a 100-point scale:

- depth: 30
- breadth: 25
- goal preservation: 30
- investigation discipline: 15

The score is based on transcript and tool-trace behavior. Final result prose quality is not a
primary endpoint.

## Paired Unit

A paired block is one fixture and one seed with both `S0-current` and `S1-layered` completed
under the same source corpus, model/config, budget, and source access policy.

Primary delta: `S1-layered - S0-current`.

## Primary Success

- Mean process-score delta >= +3 points.
- Paired bootstrap 95% CI lower bound >= 0.
- Broad-open narrowing harm >= 2 is not more frequent for `S1-layered` than `S0-current` and
  absolute rate for `S1-layered` is <= 10%.
- Broad-open narrowing harm 3 occurs 0 times for `S1-layered`.
- `S1-layered` hard-failure rate is not higher than `S0-current` and does not exceed 2%.
- Goal preservation average does not decline.

## Strong Reject

- `S1-layered` broad-open narrowing harm >= 2 in >= 20% of broad-open paired blocks.
- Any `S1-layered` broad-open narrowing harm 3.
- Repeated cases where `run_goal` wording is treated as higher priority than raw instruction.
- Apparent depth gain is explained only by tool-count inflation or harness compliance.
- `step_instruction` behaves like a strong controller by injecting status judgments, facts,
  conclusions, citations, or report prose.

## Inconclusive

- CI includes 0 widely.
- Fixture-class effects point in conflicting directions.
- Contamination or infrastructure failures invalidate >= 10% of paired blocks.
- Runtime/tool constraints prevent the 120-run primary design.

## Tests

- Paired bootstrap 95% CI over paired deltas, 10,000 resamples with fixed seed.
- Paired sign test over non-zero deltas.
- Fixture-class sensitivity summary. If `statsmodels` is available, a mixed-effects model may be
  added, but product adoption does not depend on a model unavailable in the execution environment.
