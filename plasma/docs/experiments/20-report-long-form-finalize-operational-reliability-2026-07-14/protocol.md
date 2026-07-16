# Experiment 20 Protocol

## Frozen products and conditions

- Smoke baseline: `1b6239805f2dde41f7aaab36d8025812623da5a6`.
- Current candidate: `8a054e6d7d1e50a9ebeb72b6bf6b933303264dc1`.
- Candidate binaries are built only from an independent `git archive` snapshot
  of the current candidate. Requested commit, source archive hash, build
  manifest commit, executable version output, and binary SHA-256 are locked.
- Codex only, using experiment 18's frozen model, `high` effort, `long_form`,
  `same_session`, mission sources only, 120,000 tokens, and 7,200 seconds.
- Fixtures and the external Codex authentication seed are recursively checked
  with `lstat`, copied as independent regular files, and checked again after
  prepare. Symlinks and regular files with link count other than one are fatal.

## Execution and gate

The only action order is `prepare-operational -> smoke -> reliability -> audit`.
Smoke executes the baseline and current candidate once each with two workers
through `plasma serve -> Web report API -> Codex -> MCP -> artifact download`.
Both cells must pass before reliability starts.

Reliability executes the current candidate once for each of the same 12
long-form topics, with at most six workers. There are no baseline quality,
planned, Claude, H5, designed HTML, judge, recovery, fault, or extra replicate
cells. A started failure is never rerun, replaced, or excluded.

Every run must show source reading, plan binding and session lineage, successful
`plasma.report.long_form.finalize`, exact `REPORT_FINALIZED` acknowledgement,
one canonical artifact event and downloadable matching artifact, finalizer trace
provenance, no legacy canonical path, and isolated process, port, database,
temporary, and provider-home state. The gate requires corrected smoke plus 12 of
12 candidate runs with every invariant. A failed gate permanently stops this
experiment ID.

The controller preregistration commit is locked separately from both product
commits. Raw evidence stays outside Git under the repository archive policy.
