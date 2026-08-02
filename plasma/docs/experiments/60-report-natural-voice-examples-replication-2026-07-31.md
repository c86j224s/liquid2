# Report Natural Voice Examples Replication

Status: completed; experiment 58's simple-example effect contradicted on a
fresh corpus, with no product adoption decision

## Question

Does experiment 58's exact simple before/after/preserve example prompt improve
the naturalness of Korean report wording on fresh manuscripts while preserving
meaning, claim scope, citations, and document structure?

This experiment repeated the original comparison rather than redesigning the
prompt. It separated three questions that earlier conclusions had partially
combined:

- reading efficacy: which complete manuscript is easier and more natural to
  read;
- semantic safety: whether accepted edits preserve meaning, claim scope, and
  citations, reported separately for each arm; and
- product readiness: whether the behavior survives the actual product path.

The replication evaluated the first two. It did not run the product path, so
product readiness remained explicitly unevaluated.

## Why This Replication Exists

Experiment 58 produced four clear wins and four slight losses for its example
prompt. That was a useful signal, but not stable evidence of a general effect.
Experiment 59 changed the examples and reused the earlier corpus, so it tested
a different prompt design rather than independently checking experiment 58.

Experiment 60 returned to experiment 58's exact two prompts and used previously
unread manuscripts. This isolates the question that remained open: whether the
simple examples themselves transfer to new topics and source styles.

## Locked Design

- Development topics: community education and intangible cultural heritage
- Evaluation topics: adult education, inflation, requests for proposal, the
  Silk Road, public transport, token usage metering, flood management, and
  multifactor authentication
- Arms: experiment 57's instruction-only prompt versus experiment 58's exact
  prompt with six before/after/preserve example sets
- Model: `gpt-5.5`
- Reasoning effort: `medium`
- Control prompt SHA-256:
  `4922b0cc2774dfe972c5403603f0dd8fe6a0e172ec2ef838fdc54ff039ee565f`
- Example prompt SHA-256:
  `f8d8362f08086a89e8abd1f388bcbf0ff105ed9b87e7f2544da5364298bac39d`
- Example-only suffix SHA-256:
  `5625274fc18b133c1f844b299e40c38fae17b63ed527c8b17f9893dfa9c9aae7`

The corpus, prompt hashes, model, reasoning effort, randomized schedule, retry
policy, blind decode order, and interpretation rules were frozen before the
full run. Reading efficacy was classified directionally rather than treated as
a population estimate. Semantic safety could be called clean only if the
example arm had zero drift. No combined pass/fail or product-adoption threshold
was used.

## Protocol Amendment

The first pilot calls exposed a response-contract defect in two control cells:
the model copied the source line byte for byte but returned the wrong
`original_line_sha256`. Both example cells used correct hashes, and no full
call had started.

The raw responses were preserved. A protocol amendment was locked before the
full run and allowed only one deterministic correction: recalculate the line
hash when the reported line number exists and the copied line is byte-identical
to the sealed source. It could not change a line number, original text,
replacement text, category, or decision, and it did not retry a model call.
Only the two pilot control cells triggered the rule; no full-run cell did.

## Method

Both arms used the same line-local selective acceptance policy. The model
proposed edits, deterministic guards rejected edits that changed protected
structure or tokens, and the assembled manuscript passed the same guards.

The full run contained 16 cells: eight manuscripts by two arms. Each pair was
randomized into an A/B packet. The host read every complete pair and locked the
winner and magnitude before opening the private mapping. After decoding, all
261 accepted edits were checked against the source line and adjacent context
for semantic, claim-scope, and citation drift.

## Results

The four pilot cells proposed 88 edits, accepted 79, and rejected 9. Every
assembled pilot passed its deterministic guards.

Across the 16 full-run cells:

| Arm | Proposed | Accepted | Rejected |
| --- | ---: | ---: | ---: |
| Instruction-only control | 158 | 126 | 32 |
| Simple-example candidate | 171 | 135 | 36 |

Every assembled full-run manuscript passed the deterministic guards. The
locked blind reading produced:

- Instruction-only control wins: 5 of 8
- Simple-example candidate wins: 3 of 8
- Clear or large wins: 1 for each arm
- Ties: 0
- Signed magnitude score from the example arm's perspective: `-2`
- Reading classification: `directional_contradiction`

The examples still helped individual manuscripts. They improved several
awkward constructions and removed some report self-framing. The effect did not
transfer reliably. On other manuscripts they introduced fresh process
narration, formulaic framing, awkward grammar, or less precise subjects. This
is the opposite direction from experiment 58's promising signal, not proof of
a population-wide preference for the control.

The arm-specific semantic audit found:

| Arm | Semantic drift | Claim-scope drift | Citation drift |
| --- | ---: | ---: | ---: |
| Instruction-only control | 1 | 0 | 0 |
| Simple-example candidate | 3 | 3 | 0 |

The control changed existing knowledge being updated into knowledge being
learned anew. The example arm weakened one evidence boundary, strengthened a
possible public-funding effect into a direct claim, and turned an incentive to
seek leverage into a statement that leverage was secured. No accepted edit in
either arm changed a citation.

## Decision

Do not advance experiment 58's simple-example prompt as a general natural-voice
stage on the strength of the earlier 4-4 result. On fresh manuscripts it lost
five of eight complete readings and produced more semantic and claim-scope
drift than the instruction-only control.

This is not a one-dimensional rejection of all examples. Three candidate wins
show that examples can still help a particular sentence or source style. The
product goal, however, is not to maximize the number of visible rewrites. It is
to make the report sound more naturally written while leaving its information,
claims, citations, and structure intact. This fixed six-category example block
did not satisfy those two demands reliably together.

No production prompt, workflow, storage model, API, UI, or report default
changes as a result. Product readiness is `not_evaluated` because this
replication did not exercise the product path. A future attempt should start
from the failure modes observed here instead of adding more global examples:
it should make an edit only when the local sentence benefits, reject
modality-changing rewrites, and then prove the behavior through the product
path before adoption.

## Archive Evidence

Raw manuscripts, prompt bodies, model output, blind mappings, and line-level
review notes remain in the external experiment archive defined by
[the artifact archive policy](../artifact-archive.md). The public repository
keeps only this redacted decision record and the experiment-only runner.

Stable archive-relative evidence:

- `control/protocol.lock.json`
  (`e50aedc0b491d502f03351478faecfdf5c5671dbfdaf3a43698f07061bd13ef8`)
- `control/source-manifest.lock.json`
  (`c3db09f50209d566c2c30a9223371bb548b6de64237f2b584000fb33873b04bd`)
- `control/protocol-amendment-01.lock.json`
  (`d1060ad4bd68270c59dafe81bfe9a056d080136213c74b3ef727317c62092ea9`)
- `blind/host-verdicts.lock.json`
  (`8b2135dea11e731575eede2a28475f6a9d92572e8561f90b22cf30b1ffb28591`)
- `analysis/semantic-audit.lock.json`
  (`5b5095317c60fb2f19007f202e04d9ed6e38adb3c25de7bf7c98b36bc3667893`)
- `analysis/public-summary.json`
  (`942098230d9865712ba85bdcbf0242e0311aed79e1664e35ba5b022fbb78c830`)

The committed runner preserves the experimental path used for this result. It
is not a product runtime dependency.
