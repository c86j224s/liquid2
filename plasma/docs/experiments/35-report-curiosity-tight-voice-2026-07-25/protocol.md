# Protocol

## Fixed Inputs

- Issue: #179.
- Baseline profile: `narrative-contract`.
- Reference candidate profile: `curiosity-led-explanation`.
- Voice candidate profile: `curiosity-natural-voice`.
- Tight candidate profile: `curiosity-tight-voice`.
- Initial sample size: three topics across four long-form arms.
- Execution strategy: `section_fanout`.
- Source fixture corpus: the shared locked report experiment fixture corpus
  unless a later run explicitly supplies another `--source-fixtures` archive.

## What The Tight Candidate Changes

The candidate adds prompt guidance only:

- keep the curiosity-led and natural-voice guidance;
- use existing `can_summarize` and `move_to_supporting_layer` fields more
  aggressively for repeated background, repeated mechanisms, repeated
  source-boundary notes, and redundant examples;
- keep only reasoning moves that change the reader's understanding;
- merge or delete paragraphs that merely echo the previous point;
- keep one clear caveat where it limits the claim, then avoid restating the same
  source boundary through later paragraphs;
- prefer one specific example over several examples that prove the same point;
- silently compress repeated signposts, caveats, section previews, and tidy
  mini-conclusions before submission.

It still uses existing plan fields only. It does not add `compactness_pass`,
`paragraph_budget`, `caveat_ledger`, `compression_pass`, or other custom schema
fields.

## Run Procedure

1. Prepare the archive and binary.
2. Run baseline, reference, voice, and tight candidate missions with a fixed
   seed.
3. Analyze completion, words, Parts, Sections, and wall-clock time.
4. Generate blinded judging packets.
5. Inspect representative reports directly for reading momentum, length
   control, caveat rhythm, and detail retention before drawing a product
   conclusion.

## Reading Rubric

Preferred reports should:

- keep curiosity without self-announcing the curiosity path;
- sound like edited Korean prose, not a checklist of rhetorical moves;
- carry caveats exactly where they change claims;
- reduce repeated caveats and repeated examples without hiding evidence;
- preserve concrete source-backed detail and citation discipline;
- remain useful for a reader who will not open every source.

Penalize reports that:

- hide uncertainty or source boundaries;
- become shorter by dropping important material;
- expand to explain what is already clear;
- make every paragraph end with a tidy lesson;
- expose hidden experiment terms.

## Stopping Rules

Stop the run and do not interpret quality if:

- any arm has systemic terminal failures;
- a runner bug changes the source fixture or report mode across arms;
- the candidate produces invalid plans by adding custom fields;
- raw artifacts enter the public repository.

Record the failure as an experiment outcome rather than patching around it
inside the analysis.
