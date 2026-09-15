# App follow-up ownership

Issue #483; follow-up to ed573d0. All five approved improvement areas were
addressed without changing persistence, visibility rules or product behavior.

## Implemented

- Connector readability: same-package role files for Confluence browse, preview,
  snapshot operations and local-path attach/browse, observation and lifecycle.
  Existing transaction bodies and method signatures remain intact.
- Owner API hygiene: 15 package-local helpers made private after checking all Go
  callers. No shared validation framework was introduced.
- Approval: researchproposal owns lazy created-event/approval-event evaluation;
  reportdocument owns scope resolution and plan reference decisions. App supplies
  raw store getters for scope and validated facade getters for plan references,
  intentionally preserving their distinct error precedence. Projection defaults
  and atomic persistence remain in app.
- Downloads: sourcecandidateevents owns rejection, consumption and historical
  identity rules; source owns the single-artifact snapshot selector. App keeps
  mission binding, reads, read order and indistinguishable not-found mapping.
  Explicit selectors still do not permit multi-artifact snapshots.
- Research: catalog owns canonical discovery predicates and reference sequencing.
  Target visibility runs before payload reads. Existing lazy callbacks and repeated
  reads remain to preserve timing, ordering and errors. PDF/local-path reading,
  observation and payload materialization remain app-owned.
- Direct characterization tests added for terminal compatibility, strict IL
  receipts, approval fast paths/error precedence, reference gate ordering,
  single-artifact downloads, upload classification/metadata, connection URL
  boundaries, source-trace validation-before-IO and Confluence payload ranges.

## Validation and limitations

Luna ran five bounded proposal/audit seats and three follow-up verification seats
using the same agent contexts. Host implemented and adjudicated all changes.
Research, approval and download verification found no concrete regression. One
proposal incorrectly described the existing multi-artifact test as contradictory;
host read the fixture, rejected that recommendation and preserved rejection.

Full uncached `go test -count=1 ./...` passed. Focused race gates passed for app,
researchcatalog, researchproposal, reportdocument, source and both connector child
packages, sourcecandidateevents, confluenceaccess, reportexecution, reportilsource
and SQLite. These are not a claim that the previously failing H5 Web race gate or
CLI vet gate was repaired. No server smoke, main merge or release was performed.

Direct app non-test physical lines: 9,932 -> 9,456; files: 72 -> 77. File count
increased deliberately due to role separation. This is not a completion-rate
metric. The new owner tests characterize selected high-value branches, not every
combination proposed by the audit. Existing integration tests were retained.
Optional further identity/producer relocation and full payload DTO redesign were
not needed for these boundaries and were not folded into this checkpoint.
