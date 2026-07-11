# Scorer Construct-Validity Patch

Date: 2026-07-01

This patch updates only the workflow step-instruction experiment harness before
the 2-parallel smoke and 6-parallel primary execution.

## What Changed

- Keyword-based process scoring now uses a masked response text that removes
  direct S1 prompt scaffolding echoes such as `user_instruction_raw`,
  `run_goal`, `step_instruction`, precedence wording, mission reminder wording,
  and current progress request wording.
- The raw unmasked score is still computed and emitted beside the masked score.
  `score_matrix.csv` includes `raw_process_score`,
  `raw_facet_coverage_rate`, `raw_generated_as_source_incidents`,
  `scorer_mask_applied`, and `scorer_mask_score_delta`.
- `scorer_mask_audit.csv` is written during scoring so pilot and primary rows can
  be inspected for raw-vs-masked scorer behavior.
- Generated-as-source incident detection is symmetric across S0 and S1. It now
  includes S0-equivalent labels such as mission reminder, current progress
  request, prior turn, and previous answer.

## What Did Not Change

- Fixture definitions, source corpus, run structure, variants, seeds, and primary
  design are unchanged.
- Existing pilot run outputs and previous score files are not rewritten by this
  patch. The new fields appear when the scorer is run again.
- The pilot gate semantics remain unchanged; pilot still runs single-job only.
