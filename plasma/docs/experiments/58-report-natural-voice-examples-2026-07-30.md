# Report Natural Voice Examples Experiment

Status: completed; example-augmented prompt not adopted, with a promising but
inconclusive signal for issue #207

## Question

Can concrete before/after/preserve examples make experiment 57's narrow Korean
tone correction visibly more natural without changing the report's meaning,
claim scope, citations, or structure?

Experiment 57 had established a safe instruction-only baseline, but the user's
direct reading found its improvement too small to feel meaningful. Experiment
58 changed one variable: the candidate received six target-voice example sets,
while the control kept experiment 57's exact prompt. It did not change report
planning, evidence selection, document structure, product runtime behavior, or
the saved report.

## Rationale

The preceding research and experiment established four boundaries:

- Korean prose problems should be diagnosed in context instead of handled by a
  global forbidden-word or replacement table.
- Tone strength, content preservation, and naturalness are separate evaluation
  dimensions.
- Line-local edit proposals make preservation failures inspectable.
- Experiment 57's abstract instructions were directionally useful, but gave the
  model no concrete target-voice examples and produced only a small perceived
  improvement.

The candidate therefore added one paired example for each existing diagnosis:
translation-like nominalization, formulaic connection, uniform cadence,
process narration, inflated abstraction, and redundant framing. Every set had
a `Before`, `After`, and `Preserve` case. The preserve case was intended to show
that a similar surface form can still carry necessary meaning and must remain
unchanged.

## Locked Design

- Development set: two manuscripts from experiment 57, used only to calibrate
  the example wording
- Evaluation set: eight previously unread final manuscripts from experiment 33
- Evaluation topics: mortgages, hand washing, vaccination, road safety, and
  earthquake preparedness
- Arms: experiment 57's exact instruction-only prompt versus the same prompt
  plus six example sets
- Model: `gpt-5.5`
- Reasoning effort: `medium`
- Control prompt SHA-256:
  `4922b0cc2774dfe972c5403603f0dd8fe6a0e172ec2ef838fdc54ff039ee565f`
- Example prompt SHA-256:
  `f8d8362f08086a89e8abd1f388bcbf0ff105ed9b87e7f2544da5364298bac39d`
- Example-only suffix SHA-256:
  `5625274fc18b133c1f844b299e40c38fae17b63ed527c8b17f9893dfa9c9aae7`

The corpus, prompts, schedule, model, structured-response schema, tie handling,
decode order, and success thresholds were frozen before pilot calls. The first
example draft met the preregistered development stop rule, so no second prompt
candidate was made.

The candidate had to win at least seven of eight blind comparisons, including
at least six clear or large wins, with no clear or large loss. It also required
zero semantic, claim-scope, and citation drift. A tie counted as a candidate
loss.

## Method

Both arms used experiment 57's selective acceptance policy. The model proposed
line-local edits, each edit passed deterministic preservation checks on its
own, and only passing edits were assembled. The complete candidate then passed
the same checks again.

The full run contained 16 cells: eight documents by two prompt arms. The runner
randomized control and example candidates into A/B packets. The host read each
complete pair and locked both preference and magnitude (`none`, `slight`,
`clear`, or `large`) before the private mapping was opened. After decoding, all
276 accepted edits were compared with the source line and adjacent context for
semantic, claim-scope, and citation drift.

## Results

The four pilot cells proposed 79 edits, accepted 70, and rejected 9. Every
assembled pilot passed the deterministic guards.

Across the 16 full-run cells:

| Arm | Proposed | Accepted | Rejected |
| --- | ---: | ---: | ---: |
| Instruction-only control | 165 | 141 | 24 |
| Example-augmented candidate | 164 | 135 | 29 |

Every assembled full-run candidate passed the deterministic guards. The blind
reading then produced:

- Example candidate wins: 4 of 8, all `clear`
- Example candidate losses: 4 of 8, all `slight`
- Ties: 0
- Clear or large losses: 0

The examples helped when they discouraged nominalized framing and report
process narration. They hurt when the model imitated the surface shape too
aggressively, introduced awkward connective or process language, or replaced a
precise phrase with a more colloquial but less exact one. The direction was not
stable across topics or source styles.

The post-decode audit found four accepted lines with both semantic and
claim-scope drift, and no citation drift. One of those four lines belonged to
the example arm: it weakened a legal distinction around collateral. The other
three belonged to the instruction-only control: two strengthened uncertainty
into prevention or a firmer risk claim, and one weakened the stated importance
of an earthquake-preparedness principle. Deterministic token and structure
checks could not detect those sentence-level meaning shifts.

## Decision

Do not adopt the example-augmented prompt from this experiment. It missed the
preregistered preference thresholds: a 4-4 split did not establish consistent
superiority under a gate requiring seven wins and six clear or large wins.
However, the original preservation summary aggregated drift from both arms.
Because only one of the four drift lines belonged to the example candidate,
that aggregate gate cannot support the stronger claim that the candidate
itself introduced four meaning failures.

Experiment 57 remains the adopted productization candidate until a successor
earns a broader decision. Experiment 58 does not change a production prompt,
workflow, storage model, API, UI, or product default. Its useful result is
narrow but positive: the examples produced four clear wins and no clear or
large loss, while also producing four slight losses and one candidate-arm
meaning drift. That is promising evidence worth refining, not proof that the
approach works reliably and not grounds to discard it.

A successor should test a more precise example design without treating the old
threshold miss as a verdict on examples in general. It must report safety by
arm, keep full-document reading separate from edit activity, and preserve the
one useful question left open here: whether examples can guide a narrow edit
without becoming a template that the model imitates mechanically.

## Archive Evidence

Raw manuscripts, prompt bodies, model output, blind mappings, and line-level
review notes remain in the external experiment archive defined by
[the artifact archive policy](../artifact-archive.md). The public repository
keeps only this redacted decision record and the experiment-only runner.

Stable archive-relative evidence:

- `control/protocol.lock.json`
  (`886a057e2cdc340812c9cf4a7216b8e3afc9c8994e2c36c66280fccc941bc423`)
- `control/source-manifest.lock.json`
  (`8b9a6ae0256c33bb310d2b375b4175f60911d78c33df7b061a0e627734d948d3`)
- `blind/host-verdicts.lock.json`
  (`97aa1e8e4f8b8e0c7ef2c5598ef3c828b52f2711b88864e0fac7121a18dde703`)
- `analysis/semantic-audit.lock.json`
  (`495e68304cb5dfd54f0de7bfc2e162ce7c946faf6ba8a793d789af6df7b07de6`)
- `analysis/public-summary.json`
  (`b29149903e7504d80572279b29b9bd141efb5178259a28339bb12af85d0ff46f`)

The committed runner preserves the experiment path used for this result. It is
not a product runtime dependency.
