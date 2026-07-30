# Protocol

## Fixed Inputs

- Issue: #189.
- Sample size: six selected long-form topics.
- Execution strategy: `section_fanout`.
- Wave 1 source fixtures: the six locked fixtures used by experiment 38. These
  are English single-source controls for causal isolation only.
- Arms:
  - `current_default` -> `part-connective-economy-voice`
  - `rich_control` -> `section-brief-cluster-memory-narrative-contract`
  - `subject_direct_candidate` -> `part-connective-subject-direct-synthesis-voice`

## Controlled Change

The candidate adds one planning guidance block and one Section-writing guidance
block. Planning guidance shapes existing `writing_contract.tone_and_shape`,
`writing_contract.reading_path`, Part purposes, and Section purposes toward the
first subject move. It adds no schema fields.

Section-writing guidance asks the writer to make the source-backed subject
claim the sentence's actual subject and predicate. Source identity remains in
surface prose only when it materially changes the claim. Otherwise provenance
stays in citations and evidence references.

Part assembly and final editing are unchanged. The default profile remains
`part-connective-economy-voice`.

## Corpus Boundary

Wave 1 does not satisfy issue #189's full Korean-original and
multi-source-conflict acceptance corpus. Before any default-promotion decision,
Wave 2 must add local archived fixtures for Korean original sources,
multi-source conflict, market research, and source-sparse material. This
protocol does not fabricate those fixtures and does not claim those cases are
covered by the experiment-38 controls.

## Measurement

For each completed run, classify raw Markdown artifacts as Section, Part, or
final. Record stage-level source-as-narrator advisory counts, citation counts,
characters, words, headings, Part/Section counts, Part connective character
deltas, final character deltas, and Section-to-final size ratios.

Automated locator counts are advisory only. They identify passages for human
review and are not acceptance gates, keyword policy, or authoritative
classification. Citation counts include supported inline citation shapes plus
Markdown footnote references and footnote definition lines.

## Decision Rules

Stop interpretation if an arm has systemic terminal failures, if the candidate
guidance reaches Part assembly or final editing, if the plan schema changes, or
if raw artifacts enter the repository.

Reject the candidate on any lost source-backed mechanism, example, number,
caveat, comparison, unresolved tension, necessary source identity, or citation
versus the relevant control.
