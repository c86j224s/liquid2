# Section Intent Pilot Summary

## Scope

This follow-up pilot tested a softer alternative after both section-contract
arms showed a shortening tendency.

The new arm is `section_intent`.

It keeps the same product-like long-form path:

1. archive-local Plasma database;
2. local source fixture attachment;
3. Web report API request;
4. MCP report-plan submission;
5. MCP/source reads during section drafting;
6. section-preserving part assembly;
7. long-form finalization.

The only intended change is the report generation guidance profile. The planner
is asked to place quiet reader-facing intent inside the existing `purpose`
string: what the reader should come to notice, understand, or question by the
end of the section. The arm deliberately avoids a hard section contract,
coverage lock, new schema field, section-count target, or part-assembly rewrite.

## Execution

Archive:
`research-artifacts/liquid2/plasma/experiments/22-report-section-contract-2026-07-17-intent-pilot/`

Completed reports:

| Item | Count |
| --- | ---: |
| Topics | 6 |
| Arms per topic | 4 |
| Total reports | 24 |
| Completed reports | 24 |
| Failed reports | 0 |

Arms:

| Arm | Meaning |
| --- | --- |
| `baseline` | Current long-form `g2` guidance. |
| `section_contract` | Strong section-purpose contract. |
| `section_contract_coverage` | Strong contract plus explicit coverage guard. |
| `section_intent` | Soft reader-intent guidance in the existing purpose string. |

## Aggregate Metrics

This is a shaping pilot, not a statistical adoption run.

| Arm | Median word ratio vs baseline | Mean word ratio vs baseline | Shorter than baseline | Median section ratio vs baseline |
| --- | ---: | ---: | ---: | ---: |
| `section_contract` | 0.662 | 0.748 | 5 / 6 | 0.817 |
| `section_contract_coverage` | 0.678 | 0.648 | 6 / 6 | 0.775 |
| `section_intent` | 0.732 | 0.762 | 5 / 6 | 0.804 |

The softer arm did not remove the shortening tendency. It was usually less
severe than `section_contract_coverage`, but it still produced shorter reports
on five of six topics.

## Direct Reading

The plan artifacts show that the soft prompt lands in the intended place. The
`purpose` strings often describe what the reader should come to understand,
rather than listing a rigid central point, evidence path, and boundary.

Positive observations:

- section purposes read more like editorial direction;
- openings often state the report's reader path more naturally;
- some sections feel less like source inventories and more like explanations;
- the arm does not require a schema change or a new report artifact type.

Negative observations:

- output still often becomes shorter than baseline;
- some reports still front-load source-boundary caveats heavily;
- at least one sampled report produced an awkward single-paragraph list in the
  reading guide, so the softer guidance can still create presentation defects;
- richer baseline reports sometimes preserve more policy, evidence, or
  application breadth than the candidate.

## Decision

Do not productize `section_intent` from this pilot alone.

The candidate is useful as a direction: it suggests that section writers respond
better to a reader-facing sense of intent than to a rigid mini-contract. But the
pilot does not show enough evidence that this solves the original problem. It
still tends to shorten reports, and direct reading finds mixed quality.

If this line continues, the next run should compare `baseline` and
`section_intent` over more diverse topics and judge full-report readability
directly. The acceptance question should be narrow: does soft intent improve
human readability without reducing useful breadth?
