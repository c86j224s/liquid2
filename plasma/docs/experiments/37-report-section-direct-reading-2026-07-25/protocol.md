# Protocol

## Fixed Inputs

- Issue: #179.
- Baseline profile: `edited-reading-voice`.
- Candidate profile: `section-direct-reading-voice`.
- Sample size: six selected topics across two long-form arms.
- Execution strategy: `section_fanout`.
- Model and effort: the shared runner defaults unless explicitly recorded
  otherwise by the run manifest.
- Source fixtures: the existing locked report experiment corpus, filtered to
  the six topics listed in the experiment README.

## Controlled Change

The candidate adds guidance only to the Section drafting prompt:

- write the subject instead of explaining the Section's role in the report;
- let the heading carry structural position;
- connect ideas through their substance instead of document-position phrases;
- avoid previews and recaps that exist only to expose the outline;
- preserve source boundaries and concrete detail rather than compressing them.

Planning, Part assembly, final editing, MCP tools, citations, report storage,
and plan schema remain unchanged.

## Run Procedure

1. Prepare an isolated binary and copy the locked fixtures to the external
   experiment archive.
2. Run the edited baseline and Section-direct candidate with a fixed seed and
   two concurrent report workers.
3. Verify that all reports used `section_fanout` and reached final narrative
   editing.
4. Analyze completion, words, Parts, Sections, and wall-clock time.
5. Count document-position markers separately in Section, Part, and final
   artifacts.
6. Generate blinded paired packets and directly compare the six final reports.

## Decision Rules

Advance the candidate only when:

- all six topic pairs complete without a systemic arm failure;
- tracked outline narration falls in most topics and in aggregate;
- direct reading prefers the candidate in at least four of six topic pairs;
- important concrete detail, citations, caveats, and unresolved tensions remain;
- any material length reduction reflects removed framing rather than lost
  explanation.

Do not treat a raw phrase count as sufficient evidence. A lower count can still
produce abrupt prose, hidden uncertainty, or equivalent outline narration with
different wording.

## Stopping Rules

Stop interpretation if the candidate instruction reaches plan, Part, or final
prompts; if the two arms use different fixtures or execution strategies; if an
arm has systemic terminal failures; or if raw artifacts enter the repository.

Record the source-form limitation explicitly: this corpus does not test Korean
original sources, multi-source disagreement, market research, or source-sparse
investigations.
