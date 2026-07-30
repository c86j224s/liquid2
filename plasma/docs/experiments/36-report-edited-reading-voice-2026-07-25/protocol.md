# Protocol

## Fixed Inputs

- Issue: #179.
- Baseline profile: `narrative-contract`.
- Reference voice profile: `curiosity-natural-voice`.
- Reference tight profile: `curiosity-tight-voice`.
- Edited candidate profile: `edited-reading-voice`.
- Initial sample size: three topics across four long-form arms.
- Execution strategy: `section_fanout`.
- Source fixture corpus: the shared locked report experiment fixture corpus
  unless a later run explicitly supplies another `--source-fixtures` archive.

## What The Edited Candidate Changes

The candidate adds prompt guidance only:

- keep the curiosity-led and natural-voice guidance;
- treat the report as edited reading material rather than a shorter report;
- plan the first real subject move instead of planning a report-introduction
  sentence;
- keep headings in the user's report language unless a term is a proper noun,
  product name, code identifier, or source title that should stay visible;
- avoid self-framing sentences when the sentence can begin with the subject;
- reduce repeated source-frame language without hiding the boundary that limits
  a claim;
- keep useful detail even when compressing repetitive background.

It still uses existing plan fields only. It does not add `editor_pass`,
`title_language`, `prose_rhythm`, `self_framing_check`, or other custom schema
fields.

## Run Procedure

1. Prepare the archive and binary.
2. Run baseline, natural voice, tight voice, and edited candidate missions with
   a fixed seed.
3. Analyze completion, words, Parts, Sections, and wall-clock time.
4. Generate blinded judging packets.
5. Inspect representative reports directly for subject-first openings, heading
   consistency, source-frame rhythm, caveat placement, and detail retention
   before drawing a product conclusion.

## Reading Rubric

Preferred reports should:

- start from the subject, mechanism, tension, or consequence;
- sound like edited Korean prose rather than an investigation log;
- keep caveats exactly where they change claims;
- keep source boundaries visible without making them the prose rhythm;
- preserve concrete source-backed detail and citation discipline;
- remain useful for a reader who will not open every source.

Penalize reports that:

- hide uncertainty or source boundaries;
- become shorter by dropping important material;
- introduce casual chatter;
- expose hidden experiment terms;
- keep inconsistent heading language or depth patterns;
- repeatedly describe what the report is about to do.

## Stopping Rules

Stop the run and do not interpret quality if:

- any arm has systemic terminal failures;
- a runner bug changes the source fixture or report mode across arms;
- the candidate produces invalid plans by adding custom fields;
- raw artifacts enter the public repository.

Record the failure as an experiment outcome rather than patching around it
inside the analysis.
