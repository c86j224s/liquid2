# Plan Richness Pilot Summary

## Scope

This pilot followed the `section_intent` result. The previous soft-intent arm
made Section purposes more editorial, but it still shortened most reports.

This run tested whether richer planning, not stronger local section control,
helps preserve useful breadth while keeping section writing centered.

The run stayed on the same product-like path:

1. archive-local Plasma database;
2. local source fixture attachment;
3. Web report API request;
4. MCP report-plan submission;
5. MCP/source reads during section drafting;
6. section-preserving part assembly;
7. long-form finalization.

No plan schema, API, MCP tool, artifact type, or product default changed.

## Compared Arms

| Arm | Meaning |
| --- | --- |
| `baseline` | Current long-form `g2` guidance. |
| `source_cluster_first` | Identify source-backed clusters before outlining, then map important clusters to Sections, planned omissions, or out-of-scope notes. |
| `section_brief` | Put a light prose writing brief into the existing `purpose` string: reader movement, concrete details, tension or caveat, and adjacent-topic boundary. |
| `plan_review` | Ask the planner to review its own outline for thinness before the first successful MCP plan submission. This is pre-submit self-review only. |

## Execution

Archive:
`research-artifacts/liquid2/plasma/experiments/22-report-section-contract-2026-07-17-plan-richness-pilot/`

Completed reports:

| Item | Count |
| --- | ---: |
| Topics | 6 |
| Arms per topic | 4 |
| Total reports | 24 |
| Completed reports | 24 |
| Failed reports | 0 |

## Aggregate Metrics

This is a shaping pilot, not a statistical adoption run.

| Arm | Median word ratio vs baseline | Mean word ratio vs baseline | Shorter than baseline | Median section ratio vs baseline | Fewer sections than baseline |
| --- | ---: | ---: | ---: | ---: | ---: |
| `source_cluster_first` | 0.926 | 0.886 | 5 / 6 | 1.000 | 2 / 6 |
| `section_brief` | 0.950 | 0.931 | 6 / 6 | 1.000 | 1 / 6 |
| `plan_review` | 0.869 | 0.903 | 5 / 6 | 0.958 | 3 / 6 |

## Direct Reading

`section_brief` is the best candidate in this pilot.

The section purposes are more useful than plain intent, but they do not read as
rigid checklists. In sampled reports, the openings remain natural and the
report usually keeps the same section count as baseline. The arm still shortens
all six reports, but the shortening is much less severe than the earlier
section-contract and section-intent failures.

`source_cluster_first` is mixed.

It works well on some source packets. For example, the vaccination report
preserved the policy, evidence, access, and communication clusters while keeping
a coherent structure. But it failed on at least one short source packet:
`transport-safety-b` became much shorter than baseline and lost breadth. The
cluster map can help when the source has enough distinct material, but it can
also make the model conclude that a short source justifies a thin report.

`plan_review` is not promising in its current form.

The pre-submit self-review did not reliably prevent thin planning. It often
kept the report readable, but it reduced sections in three of six topics and
shortened five of six reports. If plan review is revisited, it should probably
be a real separate review stage with explicit accept/revise behavior, not only
a self-check inside the planning prompt.

## Decision

Do not productize any arm from this pilot yet.

The next narrow candidate should be based on `section_brief`, possibly combined
with a gentler source-cluster memory. The important lesson is that the Section
writer benefits from a usable writing brief, but stronger planning interventions
can still invite the model to compress the report.

Recommended next experiment:

- compare `baseline` vs `section_brief` over more topics;
- optionally add `section_brief_cluster_memory`, where the brief can refer to
  important source clusters but does not require a separate cluster-map
  discipline;
- continue judging full-report readability by direct reading, not only length
  or section count.
