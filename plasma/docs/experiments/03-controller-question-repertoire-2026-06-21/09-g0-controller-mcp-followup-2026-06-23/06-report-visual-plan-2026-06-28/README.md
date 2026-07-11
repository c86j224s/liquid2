# Report Visual Plan Experiment

This experiment keeps completed Markdown reports immutable and tests whether optional visual-plan JSON can make self-contained HTML more useful without changing the report body.

| Variant | Runs | Mean visual blocks | Block stdev | Mean visual items | Mean HTML bytes | Quote misses | Line misses | Block types |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| `VP0` | 17 | 0.0 | 0.0 | 0.0 | 28206.06 | 0 | 0 | - |
| `VP1` | 7 | 5.0 | 0.0 | 35.14 | 27562.0 | 0 | 0 | callout_grid, comparison_table, relationship_map, stat_cards, timeline |
| `VP2` | 3 | 4.33 | 0.47 | 23.67 | 17983.0 | 0 | 0 | callout_grid, comparison_table, relationship_map, timeline |
| `VP3` | 16 | 5.06 | 0.24 | 31.62 | 34772.38 | 0 | 0 | callout_grid, comparison_table, relationship_map, stat_cards, timeline |

Hard-fail summary:

- `VP2`: 4 hard/runner failures (support_quote_not_in_report: 4)

Variant meanings:

- `VP0`: deterministic Markdown-only HTML baseline.
- `VP1`: agent-created free visual plan, rendered deterministically.
- `VP2`: agent-created constrained visual plan requiring exact `support_quote` strings from the report.
- `VP3`: agent-created constrained visual plan requiring `support_lines` anchors into numbered report lines.

Product adoption remains blocked until judged output quality and nondeterminism are reviewed.
