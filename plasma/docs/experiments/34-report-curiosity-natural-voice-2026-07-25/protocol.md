# Protocol

## Fixed Inputs

- Issue: #179.
- Baseline profile: `narrative-contract`.
- Reference candidate profile: `curiosity-led-explanation`.
- Follow-up candidate profile: `curiosity-natural-voice`.
- Initial sample size: three topics across three long-form arms.
- Execution strategy: `section_fanout`.
- Source fixture corpus: the shared locked report experiment fixture corpus
  unless a later run explicitly supplies another `--source-fixtures` archive.

## What The Follow-Up Candidate Changes

The candidate adds prompt guidance only:

- keep the experiment 33 curiosity path and processed-reading-artifact framing;
- plan fewer visible signposts in `writing_contract.tone_and_shape`;
- mark only caveats that materially change a claim or reader understanding;
- prefer concrete questions, contrasts, scenes, mechanisms, and consequences
  over abstract labels in Part and Section purposes;
- ask writers and final editors to remove repeated stock emphasis frames,
  repeated source-boundary disclaimers, mechanical paragraph starts, mechanical
  paragraph endings, and unrequested horizontal-rule separators.

It still uses existing plan fields only. It does not add `voice_pass`,
`style_pass`, `caveat_budget`, `signpost_map`, or other custom schema fields.

## Run Procedure

1. Prepare the archive and binary.
2. Run baseline, reference candidate, and follow-up candidate missions with a
   fixed seed.
3. Analyze completion, words, Parts, Sections, and wall-clock time.
4. Generate blinded judging packets for full-report reading.
5. Inspect representative reports directly for voice, caveat rhythm, and
   detail retention before drawing a product conclusion.

## Reading Rubric

Judge whole-report readability.

Preferred reports should:

- keep the curiosity path useful rather than merely cleaner;
- sound like edited Korean prose, not a checklist of rhetorical moves;
- use caveats exactly where they limit a claim;
- avoid repeated emphasis formulas and repeated section-introduction formulas;
- preserve concrete source-backed detail and citation discipline.

Penalize reports that:

- hide uncertainty or source boundaries;
- become shorter by dropping important material;
- make every paragraph end with a tidy lesson;
- expose hidden experiment terms;
- remove the curiosity-led momentum that made experiment 33 promising.

## Stopping Rules

Stop the run and do not interpret quality if:

- any arm has systemic terminal failures;
- a runner bug changes the source fixture or report mode across arms;
- the candidate produces invalid plans by adding custom fields;
- raw artifacts enter the public repository.

Record the failure as an experiment outcome rather than patching around it
inside the analysis.
