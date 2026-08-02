# Report Natural Voice Selective Acceptance Experiment

Status: completed; prompt selected as a productization candidate for issue #207

## Question

Can a sealed, line-local editing prompt make finished reports sound more like
natural human prose without changing their information, claims, citations, or
document structure?

This experiment evaluated tone and word choice only. It did not change report
planning, section order, evidence selection, citation placement, product
runtime behavior, or the saved report.

## Locked Inputs

- Corpus: eight final manuscripts copied from experiment 55's sealed blind
  split A/B reading set
- Topics: Wang Anshi and Go Raft
- Rigor levels: exploratory and strict
- Model: `gpt-5.5`
- Reasoning effort: `medium`
- Prompt SHA-256:
  `4922b0cc2774dfe972c5403603f0dd8fe6a0e172ec2ef838fdc54ff039ee565f`
- Structured-response schema SHA-256:
  `a2149b32e844ff73da8e8eb3e456544fc033fed53af3a81887c59dc607ecff29`

The prompt and policy were frozen before the experiment 57 pilot calls. No
prompt iteration or failed full-run retry was allowed inside this experiment.

## Research Basis

The prompt used concrete editing principles rather than a detector-oriented
definition of "human" writing:

- The U.S. government's [plain-language writing guide](https://digital.gov/guides/plain-language/writing)
  recommends familiar wording, active verbs, and replacing nominalizations
  with direct verbs. These informed the word-choice categories, but not a
  general instruction to shorten the manuscript.
- [Evaluating Style Transfer for Text](https://aclanthology.org/N19-1049/)
  treats style strength, content preservation, and naturalness as separate
  evaluation dimensions. The experiment therefore paired blind preference
  with independent preservation gates.
- [A Review of Human Evaluation for Style Transfer](https://aclanthology.org/2021.gem-1.6/)
  found that human-evaluation protocols are often underspecified. Experiment
  57 fixed the corpus, model, threshold, tie treatment, and decode order before
  the full comparison.
- [Improving Iterative Text Revision by Learning Where to Edit](https://aclanthology.org/2022.emnlp-main.678/)
  models revision through explicit edit locations and intents. The runner used
  numbered, hashed source lines and categorized line edits instead of asking
  for a rewritten manuscript.

The design rejected three tempting shortcuts: optimizing an AI-detector score,
globally replacing a forbidden-word list, and rewriting the complete document.
None of those approaches establishes that the result reads better while
preserving the original report.

## Failed Predecessor

Experiment 56 tested the same refined prompt with a preregistered
whole-candidate policy. Its first valid pilot proposed 16 edits, but the
assembled candidate changed protected numbers and exceeded the line-locality
budget. The experiment therefore closed with no adoption and did not continue
to a full run.

A post-hoc diagnosis found that 13 edits passed when checked individually and 3
did not. That observation could not retroactively rescue experiment 56. It
instead justified a separately sealed experiment 57: selective acceptance was
declared as the policy before fresh model calls, and every assembled candidate
still had to pass the aggregate guards. An earlier incompatible response-schema
attempt was recorded as a tooling failure, not a scientific result.

## Method

The model proposed line-local edits for word choice, cadence, formulaic
connections, process narration, inflated abstraction, and redundant framing.
Each proposed edit was tested independently against deterministic guards. An
edit was accepted only if it preserved line and paragraph shape, headings,
citations, links, quoted text, code, lists, numbers, technical tokens, and the
change budget. The assembled candidate then had to pass the same guards again.

After all eight candidates passed, the runner randomized original and candidate
documents into A/B packets. The host read every full pair and locked all eight
preferences before opening the private mapping. A tie counted as a candidate
loss. After decoding, every accepted line edit was compared with the original
and its adjacent context for semantic, claim-scope, citation, heading, and
structure drift.

## Results

The two fixed pilots proposed 41 edits. The selective policy accepted 38 and
rejected 3, and both assembled pilot candidates passed all deterministic gates.

Across the full corpus, the model proposed 161 edits. The policy accepted 132
and rejected 29. All eight assembled candidates passed every deterministic
gate. In the locked blind reading, the candidate won all eight pairs, with no
losses or ties. The post-decode audit reviewed all 132 accepted edits and found
zero semantic, claim-scope, citation, heading, or structure drift.

In the host's post-decode reading, common accepted edits replaced
report-process narration and nominalized phrasing with direct verbs, removed
unnecessary connective phrases, and varied repetitive sentence endings. This
is a qualitative reading note, not an aggregate category measurement.
Technical names, qualifications, numbers, and citations remained unchanged.

## Decision

Adopt the sealed prompt as the productization candidate for the evaluated
corpus and model configuration. This is not yet a product default: experiment
57 changes no production prompt, workflow, storage, API, or UI. Productization
must keep the same narrow responsibility and must be validated through the real
report path.

The qualitative verdict is one host's full-document blind reading, not a
human-subject study. User review remains a productization gate, and broader
topics and languages remain outside this experiment's evidence.

Independent review found one reproducibility gap in the first runner: verdict
entry used the private mapping internally to enumerate packet IDs, although the
mapping's slot assignments were never shown to the reviewer. The committed
runner now validates only the public A/B packets during verdict entry and does
not read the private mapping until summary export. This correction did not
change the already locked verdicts or the experiment result.

## Archive Evidence

Raw manuscripts, prompt bodies, model output, blind mappings, and review notes
remain in the external experiment archive defined by
[the artifact archive policy](../artifact-archive.md). The public repository
keeps only this redacted decision record and the experiment-only runner.

Stable archive-relative evidence:

- `analysis/full-edit-acceptance.json`
  (`cfb6d10eceba36f4f21f8e60487face0e3dda53f133f7c89ac5712e2cbbde417`)
- `analysis/full-hard-gates.json`
  (`d98999df26ffd2cefae273c4eb936bcef177e82375d7abda3d3e7f5170171983`)
- `analysis/public-summary.json`
  (`cf2d1bec14cd32704ddd9e69a9874abbbec6127618af397d6f0ba853e3f8cffa`)
- `analysis/post-decode-semantic-audit.json`
  (`cd6bccd741902a79e1995f74dc349b4ca971ca6f7c045ffabf66b2a5bebd288f`)
- `analysis/adoption-decision.lock.json`
  (`9466667203d36b0f39dc25a94a17166e8d5ec846ccb0dde89bf24035d8903727`)

The committed runner remains experiment-only and preserves the code path used
for this run. It is not a product runtime dependency.
