# Protocol

## Fixed Inputs

- Issue: #179.
- Baseline profile: `section-direct-reading-voice`.
- Candidate profile: `part-connective-economy-voice`.
- Sample size: six selected topics across two long-form arms.
- Execution strategy: `section_fanout`.
- Source fixtures: the same six locked fixtures used in experiment 37.

## Controlled Change

The candidate adds guidance only to Part assembly:

- leave the Part introduction empty when the first Section already begins the
  subject clearly;
- otherwise use at most one short, two-sentence introductory paragraph;
- default to no transition and add one sentence only when the relationship
  between adjacent Sections would otherwise be unclear;
- leave the closing empty unless the Sections support a genuinely new
  Part-level synthesis;
- never recap, preview, or restate source-backed material already present in the
  immutable Sections.

Planning, Section drafting, final editing, MCP tools, citations, report storage,
and plan schema remain unchanged.

## Measurement

For each completed run, classify raw Markdown artifacts as Section drafts,
assembled Parts, or final report. Record:

- characters and tracked document-position phrases at each stage;
- `Part characters - Section characters` as connective characters;
- connective characters divided by Section characters;
- Part marker count minus Section marker count;
- final characters minus Part characters.

The within-run Part ratio is the primary length measure. Paired final word count
is secondary because independent runs may generate different plans.

## Decision Rules

Advance only when all pairs complete, both Part-overhead measures improve in
most topics and in aggregate, blinded reading prefers the candidate in at least
four of six comparisons, and useful connective synthesis is not lost.

Stop interpretation if the candidate guidance reaches planning, Section, or
final prompts; if Section bodies are rewritten; if an arm has systemic terminal
failures; or if raw artifacts enter the repository.
