# Protocol

## Question

Experiment 23 showed that `visual-plan` is the better default for normal report
writing. Experiment 22 showed that the long-form options `section-brief` and
`section-brief-cluster-memory` are useful as explicit writing modes, but for
different reasons.

This experiment tests only the missing intersection:

> Should the long-form section-writing options also receive visual-aid planning
> guidance?

## Product-Path Rule

The experiment must stay close to the product path:

- create an isolated mission;
- attach a source through the product source path;
- request a long-form report through the product HTTP endpoint;
- let report agents read sources through MCP/source tools;
- keep the plan schema unchanged;
- keep generated reports, ledgers, prompts, and DBs out of Git.

The experiment must not use a prompt-only source dump as the evidence path.

## Arms

| Arm | Profile | Role |
| --- | --- | --- |
| `section_brief` | `section-brief` | Current focused long-form writing option. |
| `section_brief_visual_plan` | `section-brief-visual-plan` | Candidate with focused writing plus visual-aid planning/writing guidance. |
| `section_brief_cluster_memory` | `section-brief-cluster-memory` | Current richer-coverage long-form writing option. |
| `section_brief_cluster_memory_visual_plan` | `section-brief-cluster-memory-visual-plan` | Candidate with richer coverage plus visual-aid planning/writing guidance. |

## Profile Semantics

The two candidate profiles are hidden experiment profiles:

- accepted only when `report_mode = long_form`;
- not exposed in the Web UI;
- no new report JSON field;
- no new plan schema field;
- visual-aid intent is written into existing section purpose or coverage notes.

## Execution

Smoke:

```bash
python3 plasma/scripts/experiments/report_section_visual_plan_experiment.py --action prepare
python3 plasma/scripts/experiments/report_section_visual_plan_experiment.py --action run --limit 1 --workers 2
python3 plasma/scripts/experiments/report_section_visual_plan_experiment.py --action analyze
python3 plasma/scripts/experiments/report_section_visual_plan_experiment.py --action packets
```

Full pass:

```bash
python3 plasma/scripts/experiments/report_section_visual_plan_experiment.py --action run --limit 24 --workers 4
python3 plasma/scripts/experiments/report_section_visual_plan_experiment.py --action analyze
python3 plasma/scripts/experiments/report_section_visual_plan_experiment.py --action packets
```

## Metrics

Automatic metrics are observation signals:

- terminal success;
- word-count ratio against the matching current option;
- section-count ratio;
- table count;
- Mermaid fence count;
- total visual-aid count;
- unvalidated Mermaid signal.

Automatic metrics do not decide productization by themselves.

## Manual Reading

Read whole reports, not isolated snippets. The useful question is whether the
candidate is a better report, not whether it contains more visuals.

Manual criteria:

- section center remains clear;
- prose flow remains natural;
- source-backed details and caveats remain visible;
- visual aids clarify comparison, sequence, dependency, hierarchy, or trade-off;
- visual aids are introduced and interpreted by nearby prose;
- length increase, if any, remains useful rather than padded.

## Stop Conditions

Do not productize if candidates often:

- force visuals into prose-only topics;
- add tables that merely repeat adjacent paragraphs;
- use Mermaid without a clear reader payoff;
- shorten or thin the explanation by replacing prose with visuals;
- make long-form options harder to choose because the visible options overlap.

## Outcome Notes

The run used the product-shaped long-form report path and did not dump source
text directly into prompts. Agents read the attached source through the existing
MCP/source tools.

The focused section-writing candidate passed the useful threshold:

- 24 of 24 paired reports completed.
- Median word ratio against `section_brief` was 1.029.
- Median visual delta was +4.
- No unvalidated Mermaid signals were detected.
- Manual reading found the added visual aids generally helped readers follow
  comparisons, sequences, and trade-offs without making the report feel padded.

The rich-coverage candidate stayed inconclusive:

- 23 of 24 paired reports completed because one matching baseline run failed.
- Median word ratio against `section_brief_cluster_memory` was 0.929.
- Median visual delta was +5.
- No unvalidated Mermaid signals were detected.
- Manual reading found useful visuals, but also enough shortening and table
  density to avoid silently replacing the existing rich-coverage option.

This means the two long-form options should not be treated as one decision. The
focused option can absorb sparse visual planning more safely than the rich
coverage option.

Follow-up product decision:

- The focused `section-brief` Web UI choice is replaced by
  `section-brief-visual-plan`.
- The rich cluster-memory visual variant is exposed as its own user-selected
  long-form option for product testing, not treated as a silent replacement for
  every rich report path.
