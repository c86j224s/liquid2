# Question Navigator Experiment - 2026-06-26

This directory records the valid C0/PAL2/NAV investigation-controller
experiment after fixing the working-directory contamination found in the first
run.

The experiment asked a narrow product question:

> Should Plasma productize a stronger investigation controller that reads the
> latest answer and sends structured steering messages to the main agent?

The answer from this run is no. The experiment did not produce a controller
variant worth adopting as a default product feature. It did produce a useful
negative result: stronger always-on steering can make investigation worse.

## Variants

- `C0`: neutral continuation. This is closest to the current product baseline:
  let the same agent session continue and ask it to keep investigating.
- `PAL2`: a rhythm-aware question controller from the previous experiment.
- `NAV`: an investigation navigator that states current status, next-turn
  intent, judgment criteria, optional hints, and a user-style request.

The controller is not a researcher, source producer, judge, or report writer.
Controller output is treated as a result-like steering turn, not as source
material.

## Missions

- `M3` product-flow: UI-less Plasma research IDE and C1 product flow.
- `M5` codebase-analysis: Plasma C1 controller strategy integration surfaces.
- `W1` web-research: current autonomous/deep research product and paper trends.

Each mission/variant pair ran five seeds. The final valid corpus is therefore:

- 3 missions
- 3 variants
- 5 seeds
- 45 total completed runs

The first 27-run batch in `10-question-navigator-2026-06-26/` is an invalid
audit artifact. `codex exec resume` had run from the repository root instead of
the fixed source corpus, so some turns could inspect unrelated repo state. The
experiment script was fixed to run resumed turns with the source corpus as the
subprocess working directory and to isolate controller calls in an empty
controller workspace. This directory is the corrected run.

## Validity Checks

- `run_index.csv` contains 45 rows.
- Each mission/variant pair has 5 rows.
- Investigation turn log contamination count: 0.
- Controller logs are separate from investigation turn logs.
- The score is a screening metric, not a final product decision by itself.

## Overall Result

| Variant | Runs | Mean score | Median score | Mean sources | Mean tools | Mean coverage | Mean repetition |
|---|---:|---:|---:|---:|---:|---:|---:|
| C0 | 15 | 97.17 | 98.73 | 26.07 | 123.87 | 7.53 | 0.171 |
| PAL2 | 15 | 96.20 | 98.41 | 25.67 | 98.40 | 7.47 | 0.177 |
| NAV | 15 | 93.43 | 97.07 | 25.53 | 112.13 | 7.07 | 0.204 |

The strongest comparison is `NAV - C0`:

- paired blocks: 15
- wins: 3/15
- mean diff: -3.74
- bootstrap 95% CI: [-7.32, -0.53]
- sign-test p-value: 0.0352

This supports rejecting NAV as a product default.

`PAL2 - C0` remained inconclusive:

- wins: 5/15
- mean diff: -0.97
- bootstrap 95% CI: [-3.56, 1.73]
- sign-test p-value: 0.3018

PAL2 should not become a default either. It remains a possible future
conditional steering experiment.

## Mission-Level Result

| Mission | Variant | Mean score | Mean sources | Mean tools | Mean coverage | Mean repetition |
|---|---:|---:|---:|---:|---:|---:|
| M3 | C0 | 78.66 | 6.20 | 41.20 | 8.00 | 0.189 |
| M3 | PAL2 | 74.25 | 6.00 | 37.60 | 7.60 | 0.197 |
| M3 | NAV | 73.59 | 6.00 | 44.80 | 7.60 | 0.217 |
| M5 | C0 | 115.69 | 58.40 | 213.20 | 7.20 | 0.175 |
| M5 | PAL2 | 115.79 | 57.20 | 170.80 | 7.20 | 0.204 |
| M5 | NAV | 111.10 | 54.40 | 215.20 | 6.80 | 0.244 |
| W1 | C0 | 97.17 | 13.60 | 117.20 | 7.40 | 0.148 |
| W1 | PAL2 | 98.57 | 13.80 | 86.80 | 7.60 | 0.130 |
| W1 | NAV | 95.61 | 16.20 | 76.40 | 6.80 | 0.150 |

M3 favored the baseline. M5 was essentially tied between C0 and PAL2, with NAV
behind. W1 had a mild PAL2 lead, but the paired result was not strong enough to
productize.

## Product Decision

Keep the default Plasma investigation loop close to C0:

1. Continue the same agent provider session.
2. Give only a short mission reminder and the current user/workflow steering
   turn.
3. Let the agent use MCP/source read tools to inspect sources and ledger state.
4. Record controller strategy selection as observable telemetry, not as a
   source, evidence item, claim, or saved knowledge item.

Do not productize NAV. Do not make PAL2 a default. Future controller work should
be conditional and weak: intervene only when the run appears stuck, repetitive,
too narrow, or off-mission, and validate the change before shipping it.

## Files

- `analysis.md`: machine-scored summary and paired comparisons.
- `run_index.csv`: one row per run.
- `runner_results.json`: runner-level result list.
- `runs/*`: raw prompts, agent answers, controller messages, metrics, and tool
  trace summaries for each run.
