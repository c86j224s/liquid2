# Report Natural Voice Contrastive Examples Experiment

Status: completed; contrastive example prompt not adopted for issue #207

## Question

Can category-matched contrastive examples make the natural-voice correction
more reliable than experiment 58's simple before/after/preserve examples,
without changing meaning, claim scope, citations, or document structure?

This was a screening experiment on wording and tone only. It did not change
report planning, evidence selection, document structure, the production
prompt, product runtime behavior, or saved reports.

## Why This Follow-up Exists

Experiment 58 produced four clear wins and four slight losses. Its original
seven-win adoption threshold was too strong to treat the 4-4 result as a broad
rejection of examples, and its preservation summary also combined drift from
both arms. A closer audit showed that one drift belonged to the example arm
and three to the instruction-only control.

Experiment 59 therefore kept the earlier result as a useful but unresolved
signal. It changed the example design instead of merely raising or lowering
the old threshold. The control was experiment 58's exact example prompt. The
candidate replaced its example block with six category-matched contrast sets.
Each category showed an edit to make, a similar line to leave alone, a tempting
but forbidden rewrite, and why that rewrite would be harmful. The intent was
to teach the boundary of an edit, not only its desired surface form.

## Locked Design

- Development set: the same two sealed manuscripts used to calibrate
  experiment 58
- Evaluation set: the same eight previously unread final manuscripts used in
  experiment 58
- Topics: mortgages, hand washing, vaccination, road safety, and earthquake
  preparedness
- Arms: experiment 58's exact example prompt versus a contrastive-example
  prompt built on experiment 57's unchanged base instruction
- Model: `gpt-5.5`
- Reasoning effort: `medium`
- Control prompt SHA-256:
  `f8d8362f08086a89e8abd1f388bcbf0ff105ed9b87e7f2544da5364298bac39d`
- Contrastive prompt SHA-256:
  `3520e0f38b18bfadcc543c4b6ad4ca137a9d5fa5a58e4a6743bf61d74e7c8de5`
- Contrastive-only suffix SHA-256:
  `705e8467d894af68ceecb878472ad982a3a0d09f5e170fe7ca5fed5ac2ece64c`

The corpus, prompts, model, schedule, blind decode order, and screening rules
were frozen before pilot calls. The candidate needed five of eight wins,
including three clear or large wins, with no clear or large loss and no
candidate-arm semantic, claim-scope, or citation drift. These were screening
rules for whether to continue the approach, not a product-adoption standard or
a claim of statistical significance.

The evaluation corpus and host reader were reused from experiment 58. Arm
identity was still hidden during A/B reading, but the manuscripts themselves
were not novel to the reader. This makes experiment 59 a paired engineering
screen for the next design choice, not an independent confirmation or an
estimate of population-level preference.

## Method

Both arms used the same line-local selective acceptance mechanism. The model
proposed edits; deterministic guards rejected changes to protected structure,
numbers, citations, links, quoted text, and technical tokens. The assembled
document then passed the same guards again.

The run contained 16 full cells: eight documents by two prompt arms. Candidates
were randomized into A/B packets. The host read every complete pair and locked
preference and magnitude before opening the private mapping. After decoding,
all 286 accepted edits were compared with the original line and adjacent
context for semantic, claim-scope, and citation drift.

Two response-contract defects were handled without rerunning the model or
altering text. One response omitted a diagnosis summary for an edit category;
the runner deterministically reconstructed only that metadata. Another copied
the source line exactly but returned a one-character error in its claimed
SHA-256; after a separate amendment was locked, the runner corrected only the
hash when the copied text was byte-identical to the sealed source. Raw outputs
remain unchanged in the external archive.

## Results

The four pilot cells proposed 69 edits, accepted 61, and rejected 8. Across the
16 full-run cells:

| Arm | Proposed | Accepted | Rejected |
| --- | ---: | ---: | ---: |
| Experiment 58 example control | 172 | 142 | 30 |
| Contrastive-example candidate | 177 | 144 | 33 |

Every assembled document passed the deterministic guards. The locked blind
reading produced:

- Contrastive candidate wins: 2 of 8, one `slight` and one `large`
- Contrastive candidate losses: 6 of 8, two `slight` and four `clear`
- Ties: 0

The large win matters: on one mortgage manuscript, the contrastive examples
prevented conspicuous mistranslation-like phrasing and grammar errors that
appeared in the control. The effect was not stable. On four other manuscripts,
the contrastive prompt clearly reintroduced report-process narration,
formulaic framing, awkward word choice, or broken sentence subjects. Those are
full-document reading failures even where the underlying claims survived.

The post-decode audit found two candidate-arm lines with semantic and
claim-scope drift, both in the hand-washing manuscript. One strengthened
managing a possibility into reducing it; the other removed one layer of
uncertainty from a risk statement. Neither arm changed a citation. The control
had no audited semantic, claim-scope, or citation drift in this run.

## Decision

Do not adopt the contrastive prompt. It failed the screening preference,
clear-win, clear-loss, semantic-preservation, and claim-scope-preservation
rules. This result is stronger than experiment 58's threshold miss because the
candidate also lost six full-document readings and introduced two meaning
changes under an arm-specific audit.

The conclusion remains bounded. Contrastive examples can help a specific
manuscript, as the large mortgage win shows, but this six-category whole-prompt
design is unstable across topics and writing styles. The experiment does not
show that all examples are ineffective, and it does not undo experiment 58's
four clear wins. It shows that adding preserve and forbidden cases in this form
did not make those gains reliable on this reused corpus and reader. No
production behavior changes as a result.

## Archive Evidence

Raw manuscripts, prompt bodies, model output, blind mappings, and line-level
review notes remain in the external experiment archive defined by
[the artifact archive policy](../artifact-archive.md). The public repository
keeps only this redacted decision record and the experiment-only runner.

Stable archive-relative evidence:

- `control/protocol.lock.json`
  (`22b8446f399a5b9dfb523098b459940a420a1b9a3cadeb675ee9aa189ecbc93b`)
- `control/source-manifest.lock.json`
  (`0ec1cdeb55f1297a2146e05ab09c238a13221e9765f7d1c7c51ef38422ed89cb`)
- `blind/host-verdicts.lock.json`
  (`ff5daf395fd20578b8c40c022b2db9f77555b547137759ba5d8d4681f3cbf44a`)
- `analysis/semantic-audit.lock.json`
  (`a60f76d3f0074ffbccbfc99b05dd9fe8b2ed60de308b7580771f6633703f2535`)
- `analysis/public-summary.json`
  (`5866d449c39a33e20c88279ed72100161b03e451c2254422fd6a12806bd23ce0`)

The committed runner preserves the experimental path used for this result. It
is not a product runtime dependency.
