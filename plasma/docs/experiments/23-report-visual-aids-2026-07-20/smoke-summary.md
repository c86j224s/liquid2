# Smoke Summary

## Run Shape

The first smoke run used two fixture topics from the locked source fixture set
and ran both report modes:

- `planned`
- `long_form` with `section_fanout`

Each topic and mode was initially run across four arms:

- `baseline`
- `visual_supplement`
- `visual_plan`
- `visual_gate`

This produced 16 product-path report runs.

After the first smoke, `visual_gate` was removed from the formal comparison.
The gate arm made the experiment more about enforcing an intermediate rule than
about whether visual aids improve reading. Its raw run remains archived for
transparency, but it is not part of the formal analysis or follow-up packet set.

## Operational Result

All 16 initial runs completed. No terminal report-generation failure was
observed. The formal post-correction analysis uses the 12 runs that belong to
`baseline`, `visual_supplement`, and `visual_plan`.

The runner also produced aggregate analysis and blind judging packets:

- `analysis/aggregate.json`
- `judging/packets/`
- `judging/private-mapping.json`

These files live in the local raw archive, not in this repository.

## Automatic Observation Signals

The formal smoke analysis produced four complete baseline-versus-candidate
pairs: two topics across two report modes.

| Candidate arm | Completed pairs | Median visual-aid delta | One-sided visual increase p | Median word ratio vs baseline | Unvalidated Mermaid signal |
| --- | ---: | ---: | ---: | ---: | ---: |
| `visual_supplement` | 4 | 4.5 | 0.0625 | 1.26 | 0 |
| `visual_plan` | 4 | 3.5 | 0.0625 | 1.07 | 0 |

The p-value here is from a tiny smoke sample and must not be treated as
statistical proof. It only confirms that both candidates increased visual aid
usage on this small run without triggering the runner's Mermaid-validation
warning signal.

## Six-Topic Expansion

The follow-up run expanded the formal comparison to six fixture topics while
keeping the same two report modes and three formal arms. The completed shape
was:

- 36 formal report runs: 6 topics x 2 modes x 3 arms.
- 12 complete baseline-versus-candidate pairs.
- 0 terminal report-generation failures.
- 24 blind judging packets: 6 topics x 2 modes x 2 candidate arms.

The expansion deliberately excludes the earlier `visual_gate` smoke arm. The
experiment no longer gates intermediate output; it records completed product
path reports and compares completed same-topic, same-mode pairs.

| Candidate arm | Completed pairs | Median visual-aid delta | One-sided visual increase p | Median word ratio vs baseline | Unvalidated Mermaid signal |
| --- | ---: | ---: | ---: | ---: | ---: |
| `visual_supplement` | 12 | 4.0 | 0.0059 | 1.19 | 0 |
| `visual_plan` | 12 | 2.0 | 0.0020 | 1.03 | 0 |

Automatic signals indicate that both candidates increased visual aids across
the paired set without producing unvalidated Mermaid warning signals.
`visual_supplement` is the stronger surface-change arm: it adds more tables and
usually lengthens reports. `visual_plan` is the subtler arm: it increases visual
aids while keeping report length closer to baseline.

These numbers are still not a product decision. The next step is to read whole
reports and decide whether the added tables or diagrams actually improve
understanding, preserve source-grounded prose, and avoid decorative repetition.

## Whole-Report Reading Decision

After reviewing rendered report samples, the user judged `visual_plan` to be the
better product default. This is a user reading judgment, not an automatic metric
result.

The reading judgment was:

- `visual_supplement` produces the stronger surface change and often adds more
  tables, but it can make reports longer.
- `visual_plan` is subtler. It uses the planning step to decide where a visual
  aid belongs, then lets the writing step use or skip that intent based on the
  source read.
- The user found `visual_plan` surprisingly more useful when reading the full
  rendered reports, especially because it still produced some Mermaid diagrams
  without making the whole report feel like a visual-widget exercise.

## Productization Decision

Use `visual_plan` as the default report generation guidance for both normal
planned reports and long-form reports.

The product change should stay aligned with the experiment:

- keep using the existing `generation_guidance_profile` request field;
- add the visual planning guidance only through the existing plan prompt;
- add the visual writing guidance only through the existing report-writing
  prompts;
- do not add a new report plan schema, gating step, renderer behavior, or
  mandatory visual-aid rule.
