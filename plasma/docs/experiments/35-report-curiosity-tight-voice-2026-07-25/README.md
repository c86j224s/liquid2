# Report Curiosity Tight Voice Experiment

## Status

Completed as an issue #179 follow-up. Experiment 34 showed that
`curiosity-natural-voice` can reduce visible AI-like signposting, but it often
expands the report and can still repeat caveat-like language. This experiment
added a tighter candidate that keeps the curiosity and natural-voice direction
while explicitly compressing repeated setup, repeated caveats, repeated
examples, and tidy mini-conclusions.

The tight candidate controlled length clearly, but the reading result is mixed.
It kept some concrete opening momentum and removed much of the visible
signposting, while also showing risks: occasional report-framing sentences,
inconsistent title language, and a few places where compactness felt closer to
outline compression than stronger edited prose. It is not a product-default
adoption result. No product default changed.

## Question

> Can Plasma keep the more natural curiosity-led reading experience while
> avoiding the length expansion and repeated source-boundary language seen in
> experiment 34?

## Arms

| Arm | Guidance profile | Meaning |
|---|---|---|
| Baseline | `narrative-contract` | Current product writing baseline, kept because the shared harness analyzes candidates against a baseline. |
| Reference candidate | `curiosity-led-explanation` | Experiment 33 primary candidate. |
| Voice candidate | `curiosity-natural-voice` | Experiment 34 candidate that reduced visible signposting but expanded some reports. |
| Tight candidate | `curiosity-tight-voice` | Adds explicit compression discipline on top of the curiosity and natural-voice guidance. |

The tight candidate still uses the existing report plan schema and existing
long-form assembly path. It does not add UI choices, database state, report
artifact types, citation behavior, postprocessors, or custom plan fields.

## Runner

Use:

```bash
cd plasma
python3 scripts/experiments/report_curiosity_tight_voice_experiment.py --action prepare --limit 3
python3 scripts/experiments/report_curiosity_tight_voice_experiment.py --action run --limit 3 --workers 2
python3 scripts/experiments/report_curiosity_tight_voice_experiment.py --action analyze --limit 3
python3 scripts/experiments/report_curiosity_tight_voice_experiment.py --action packets --limit 3
```

With `--limit 3`, the runner creates three topics across four long-form arms,
or 12 report generations before packet preparation.

## Small Run Result

The completed small run used three paired topics and four long-form arms. One
`curiosity-natural-voice` run failed without a reported error on the first pass;
the failed run directory was preserved in the local archive and that single
case was rerun. The final paired set completed 12 of 12 report generations with
zero remaining failures.

| Arm | Average words | Average word ratio vs. baseline | Ratio range | Average section ratio vs. baseline | Average wall seconds |
|---|---:|---:|---:|---:|---:|
| `curiosity-led-explanation` | 4,826 | 0.962 | 0.853-1.122 | 0.976 | 684.0 |
| `curiosity-natural-voice` | 5,431 | 1.084 | 0.992-1.250 | 1.077 | 589.0 |
| `curiosity-tight-voice` | 3,819 | 0.765 | 0.599-0.872 | 0.912 | 597.8 |

The tight candidate averaged about 70% of the `curiosity-natural-voice` word
count across the same topics. Direct reading supports the metric: it reduced
repeated setup, repeated "therefore" framing, and some caveat loops. It did not
fully solve the product-quality problem, because some outputs still announced
the report frame directly and a few headings retained English titles or
inconsistent depth. Treat the result as evidence that compactness guidance is a
useful constraint, not as evidence that this exact prompt should become the
default.

## Reading Focus

Judge the tight candidate primarily against `curiosity-natural-voice`, then
against `curiosity-led-explanation`.

Prefer the tight candidate only if it:

- keeps the reader's reason to continue reading;
- keeps the concrete, less self-announcing opening style from experiment 34;
- reduces repeated caveats without hiding uncertainty;
- compresses repeated examples and repeated paragraph machinery;
- stays near the curiosity-led candidate's density unless the sources genuinely
  require more coverage;
- preserves source-backed facts, caveats, citations, examples, counterpoints,
  and unresolved tensions.

Reject or revise the tight candidate if it becomes bland, drops useful detail,
hides source boundaries, or merely becomes shorter.

## Public Artifact Policy

Commit only this protocol, small aggregate metrics, and redacted conclusions.
Keep generated reports, prompt packets, judging packets, databases, logs, and
copied source bundles under the local experiment archive described in
`plasma/docs/artifact-archive.md`.
