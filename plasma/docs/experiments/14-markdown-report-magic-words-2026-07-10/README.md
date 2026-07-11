# Markdown Report Magic-Word Experiment - 2026-07-10

This experiment addresses issue 77 in two phases. Phase 1 compares subtle
Korean writing cues under complete-source conditions. Phase 2 adds explicit
unavailable, rejected, and unresolved research results to test whether report
content stays ahead of a separate late limitations section.

## Status

- Raw archive root:
  `~/research-artifacts/liquid2/plasma/experiments/14-markdown-report-magic-words-2026-07-10/`
- Product runtime and defaults: unchanged.
- Phase 1: complete; 20 report candidates and 20 blind comparisons.
- Phase 2 gap stress: complete; 16 report candidates and 20 blind comparisons.
- Decision: do not change the product default from this run.
- Strongest candidate: the explicit separate-late-limitations instruction,
  which beat the baseline 4-0 in phase 2.

The generic `step-calm` wording was not useful. Reader-flow wording was 3-1
under complete sources but 1-3 under gap stress, so it is not a robust default
candidate. The late-limitations instruction produced a separate section in all
four phase-2 reports, with no detected research-process or gap narrative left
outside that section. Combining reader-flow wording with that instruction did
not improve overall preference over the structural instruction alone.

All phase-2 reports still opened with substantive judgment. The observed
failure where a page of research limitations precedes the report body was not
reproduced, so this run validates separation behavior but not a product-path
fix for that exact opening failure.

See [`protocol.md`](protocol.md) for both fixed phases and isolation rules, and
[`analysis-summary.md`](analysis-summary.md) for the results and interpretation.
Raw sources, generated reports, prompt packets, provider logs, judge packets,
and private blind mappings are not committed.

The top-level experiment index is intentionally unchanged while experiment 13
is active on another worktree. The branch that lands second should add both
entries in numeric order instead of creating a concurrent edit at the same
index location.
