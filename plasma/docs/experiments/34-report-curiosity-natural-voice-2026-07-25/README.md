# Report Curiosity Natural Voice Experiment

## Status

Small-run generation, aggregate analysis, and judging-packet preparation
completed for issue #179. This experiment keeps the
`curiosity-led-explanation` direction from experiment 33 and adds a narrower
voice-control candidate that tries to reduce AI-like signposting, repeated
caveats, and mechanically neat paragraph endings. No product default has
changed.

## Question

Direct reading of experiment 33 suggested that the curiosity-led candidate was
often more promising than the paragraph-contract candidate, but it could still
sound like an AI explaining its own structure. The follow-up asks:

> Can Plasma keep the curiosity and processed-reading-artifact gains while
> making the prose feel less self-announcing and less caveat-driven?

## Arms

| Arm | Guidance profile | Meaning |
|---|---|---|
| Baseline | `narrative-contract` | Current product writing baseline. |
| Reference candidate | `curiosity-led-explanation` | Experiment 33 primary candidate: source-grounded processed reading artifact, curiosity path, and reader payoff. |
| Follow-up candidate | `curiosity-natural-voice` | Adds natural voice constraints on top of the curiosity-led candidate: fewer repeated signposts, caveats stated once where they matter, more varied paragraph starts and endings, and no horizontal-rule separators unless requested. |

The follow-up candidate still uses the existing report plan schema and existing
long-form assembly path. It does not add UI choices, database state, report
artifact types, citation behavior, postprocessors, or custom plan fields.

## Runner

Use:

```bash
cd plasma
python3 scripts/experiments/report_curiosity_natural_voice_experiment.py --action prepare --limit 3
python3 scripts/experiments/report_curiosity_natural_voice_experiment.py --action run --limit 3 --workers 2
python3 scripts/experiments/report_curiosity_natural_voice_experiment.py --action analyze --limit 3
python3 scripts/experiments/report_curiosity_natural_voice_experiment.py --action packets --limit 3
```

The first pass is intentionally small: three topics across three long-form arms,
or nine report generations before packet preparation. If the result is
promising, a later run may expand the sample before productization discussion.

## Small Run Check

The small run completed across three topics and three arms, producing nine
report records and three complete baseline-versus-candidate pairs. No arm
failed. Packet preparation produced six comparison files: each candidate against
the baseline for each completed topic.

| Metric | Baseline | Curiosity Candidate | Natural Voice Candidate |
|---|---:|---:|---:|
| Completed reports | 3 | 3 | 3 |
| Mean words | 5,066.3 | 4,725.7 | 5,574.0 |
| Mean word ratio over baseline | 1.000 | 0.928 | 1.101 |
| Word-ratio range | 1.000-1.000 | 0.731-1.076 | 0.877-1.276 |
| Mean sections | 10.7 | 9.0 | 11.3 |
| Mean wall seconds | 646.6 | 648.1 | 660.6 |

Direct reading and phrase scans showed a mixed result. The natural-voice
candidate reduced repeated emphasis markers such as "core point" style
openers, reduced repeated "therefore" transitions, and removed an unrequested
horizontal-rule separator in the transport-safety case. It also opened several
reports with more concrete reader-facing prose.

The same candidate often expanded the explanation. It averaged longer than both
the baseline and the experiment 33 curiosity candidate, and some reports still
leaned on caveat-like language. This is not an adoption result. The useful
signal is narrower: voice-control guidance can reduce visible AI-like
signposting, but the next candidate must control expansion and repeated
source-boundary language more explicitly without hiding uncertainty.

## Reading Focus

Judge the follow-up candidate against the experiment 33 curiosity-led output,
not only against the baseline.

Prefer the follow-up only if it:

- keeps the reader's reason to continue reading;
- reduces repeated stock emphasis frames and source-boundary disclaimers;
- starts paragraphs with concrete claims, questions, contrasts, mechanisms, or
  consequences rather than meta-explanation;
- preserves source-backed facts, caveats, citations, examples, and unresolved
  tensions;
- avoids becoming shorter by hiding material that should stay in the report.

Reject or revise the follow-up if it becomes bland, hides uncertainty, reduces
useful detail, or merely replaces one formulaic style with another.

## Public Artifact Policy

Commit only this protocol, small aggregate metrics, and redacted conclusions.
Keep generated reports, prompt packets, judging packets, databases, logs, and
copied source bundles under the local experiment archive described in
`plasma/docs/artifact-archive.md`.
