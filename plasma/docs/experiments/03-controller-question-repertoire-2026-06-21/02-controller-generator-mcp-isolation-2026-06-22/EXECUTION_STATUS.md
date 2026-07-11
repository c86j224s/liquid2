# Execution Status

Status: nine-block execution and judging complete.

Prepared:

- 2x2x2 factor matrix with nine mission-seed blocks.
- Seventy-two planned run cells in `run_index.csv`.
- External original source corpora under
  `<experiment-source-root>/`.
- Per-run manifests and hard-fail audit templates.
- Judge packet view directories and blinding protocol.
- Statistical plan, score matrix headers, pairwise result headers, factor effect
  headers, claim audit headers, and contamination failure log.
- Experiment-local validator:
  `tools/validate_experiment.py`.

Executed:

- 72/72 planned cells completed.
- 9/9 paired mission-seed blocks are complete.
- 0 hard-fail audit failures remain in the primary run set.
- 72/72 completed runs were converted into blinded judge packets.
- 72/72 blinded judge packets were scored into `analysis/score_matrix.csv`.
- `tools/validate_experiment.py` passes after execution and judging.
- `tools/analyze_results.py` produced `analysis/factor_effects.csv`,
  `analysis/pairwise_results.csv`, and `analysis/decision_memo.md`.
- `tools/analyze_g0_slice.py` produced
  `analysis/g0_slice_factor_effects.csv` and
  `analysis/g0_slice_decision_memo.md` for the same-session-only follow-up
  view.

Nine-block result:

- Controller (`C1`) has no statistically supported win and no supported harm.
- Separate report generation session (`G1`) produced a supported depth harm:
  `generator:depth` estimate -0.2500, CI [-0.4167, -0.1111],
  wins=0/6, two_sided_p=0.0312.
- Additional research surface (`M1`) has no statistically supported win and no
  supported harm.
- Provenance gate passes overall, but this is not a product go decision by
  itself.
- The supported `G1` harm is a report-depth loss, not a provenance-risk signal.

Product interpretation:

This nine-block result is enough to stop this look and reject `G1` as the
default report path. Plasma should keep same-session final report generation as
the default for this experiment's decision scope.

This result does not prove that controller-led steering or the additional MCP
research surface is ineffective. Those factors remain unresolved and need a
separate, narrower follow-up if they are to be productized.

The `G0`-only slice fixes final report writing to the same provider session and
does not find supported controller or MCP-surface effects. It is a narrowing
analysis for the next experiment, not a new product decision.
