# Protocol

## Fixed Inputs

- Issue: #179.
- Baseline profile: `narrative-contract`.
- Supporting candidate profile: `reader-paragraph-contract`.
- Primary candidate profile: `curiosity-led-explanation`.
- Initial sample size: eight topics across three long-form arms.
- Execution strategy: `section_fanout`.
- Source fixture corpus: the shared locked report experiment fixture corpus
  unless a later run explicitly supplies another `--source-fixtures` archive.

## What The Current Supporting Candidate Changes

The already implemented supporting candidate adds prompt guidance only:

- plan the reader's actual question and reading path;
- express each Part as a reader movement;
- express each Section purpose as a natural prose paragraph plan;
- keep source-backed clusters in `coverage_notes`;
- ask the writer and final editor to run a silent paragraph-quality pass.

The candidate does not add a schema field, UI option, database migration, new
report artifact type, citation behavior, or hard-coded postprocessor.

## What The Primary Candidate Changes

The product problem is not only paragraph discipline. Plasma should produce a
processed reading artifact, not a source annotation layer or investigation log.
The primary candidate adds prompt guidance only:

- start from the reader's reason to care, not from the source inventory;
- identify the information gap, tension, surprising contrast, or market/research
  question that makes the material worth reading;
- plan a chain of partial resolution and next questions so the user has a
  reason to keep reading;
- use sources as explanation material, not as the visible organizing principle;
- preserve caveats and citations at the point where they matter, without letting
  them dominate the prose;
- adapt the reading rhythm to the user's purpose: learning, insight synthesis,
  gathered reading, or market research.

It still uses existing plan fields only. `writing_contract.reading_path` carries
the curiosity path, Section `purpose` carries local tension and payoff, and
`coverage_notes` carries source-detail memory.

## Run Procedure

1. Prepare the archive and binary.
2. Run baseline, supporting candidate, and primary candidate missions with a
   fixed seed.
3. Analyze completion, words, Parts, Sections, and wall-clock time.
4. Generate blinded judging packets for full-report reading.
5. Score packets before reading the private arm mapping.
6. Write a short public conclusion after inspecting aggregate metrics and
   representative reports.

## Reading Rubric

Judge the full report, not isolated paragraphs.

Preferred reports should:

- answer the reader's actual question early enough that the structure is clear;
- create a useful reason to continue reading, such as a question, contrast,
  tension, or insight path;
- make each paragraph's job understandable without sounding formulaic;
- carry source details inside explanation rather than dumping a source inventory;
- preserve caveats near the claims they limit;
- make transitions and conclusions feel edited rather than mechanically
  assembled;
- stay specific enough to be useful for a reader who will not open every source.

Penalize reports that:

- become shorter by dropping important source material;
- hide uncertainty or source boundaries;
- repeat the same disclaimer or setup frame;
- expose hidden experiment terms such as reader brief or paragraph quality pass;
- use a planned visual, table, or section structure when prose would be clearer.

## Stopping Rules

Stop the run and do not interpret quality if:

- either arm has systemic terminal failures;
- a runner bug changes the source fixture or report mode across arms;
- the candidate produces invalid plans by adding custom fields;
- raw artifacts enter the public repository.

Record the failure as an experiment outcome rather than patching around it
inside the analysis.
