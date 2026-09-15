# Backend refactor handoff

## Closure update — 2026-09-15

The user approved closing #483 after implementation, main integration and live
product regression. PR #488 merged the refactor; #489/#490 separately retired
OAuth execution and legacy H5. The old H5 skipped test was removed with the
feature. #492 removed the arbitrary 128-block ceiling; #494 aligned continuity
list editing and reader acceptance. Active pre-canonical style editing remains.

Live QA verified new planned and classic long-form reports, new standard IL,
long-form IL checkpoint recovery, and finally a fresh long-form IL generation
on dc345ea (45 minutes 34 seconds). Markdown/HTML/PDF for the last run were
retrieved over product HTTP endpoints with matching byte counts and SHA-256.
These are single-fixture functional checks, not exhaustive content/layout review.

PR #496 corrects the checkout-only revision test helper (ordinary checkout,
linked worktree, detached HEAD and packed refs) without changing runtime code.
The corrected articleexperiment package passes in both checkout layouts. The
most recent full-suite run has a separate reproducible Chrome PDF Mermaid label
extraction failure, tracked in #497. By user decision #497 stays open and is not
part of this closeout; the current full suite is NOT claimed green.

Earlier sections below are historical checkpoint evidence. In particular, the
old pending OAuth/H5 cleanup, skipped H5 test and no-live-smoke statements no
longer describe current main. Worktrees from this refactor and its follow-ups
are retired only after merged-revision checks and preservation of any dirty
recovery/helper material. Unrelated worktrees, QA missions, artifacts and user
databases are retained. No release is authorized by this closure.

Refs #483. Final acceptance record, 2026-09-11. Implementation checkpoint be5f5a5;
user-approved legacy H5 test retirement checkpoint 63e3fae.

## Delivered scope

App capability ownership, Web execution composition/recovery, all selected MCP
feature execution/state boundaries and reporting role/API cleanup are implemented.
See app-followup-483.md and transport-closure-483.md for the retained responsibilities.
No mutex redesign, universal file-size target, wholesale test migration, OAuth
implementation removal or legacy H5 feature removal is included in this handoff.

## Integrated review

Scoped Luna comparisons covered Web and reporting plus each migrated MCP family.
Final host integration checks confirmed root dispatch/schema/tool definitions are
unchanged from 169fcd2, request-local state is supplied to the actual feature owners,
root gates and tracing remain root-owned, and no mutable draft maps were duplicated.
Compatibility DTO aliases and shared mutex pointers are deliberate, not pending
requirements to redesign. Historical Web projection helpers are test-only and are
not counted as validation of current typed workflow recovery.

## Contract comparison

A temporary Go test overlay ran identical probes against an isolated git archive
of 169fcd2 and the final branch without adding probe files to either product tree.
Fifteen serialized output cases matched exactly:

- ListTools: default, legacy research, operator mutation, experiment, patch and
  empty allowlist configurations (6).
- Disabled source-read, patch-start and IL-source-read error envelopes (3).
- Artifact, source-read, patch-finalize, IL-read, editorial-memory and document
  DTO representations (6).

This is representative differential coverage, not an exhaustive enumeration of
all valid stage bindings or every result payload. Existing integration/privacy/
replay/concurrency tests provide the wider behavioral coverage.

## Final validation

- Full uncached `go test -count=1 ./...`: PASS.
- `go test -race ./internal/web ./internal/mcp/... ./internal/reportexecution
  ./internal/reporting/... ./internal/reportworkflow/...`: PASS.
- Architecture boundary tests: PASS with explicitly recorded contract relocation.
- TestReportDraftKeepsOriginalWhenHumanizeGuardFails: intentionally SKIPPED by
  user decision, not repaired. Active long-form style-edit tests remain enabled.
- `go vet ./...`: known two unreachable-code diagnostics in disabled Confluence
  OAuth CLI implementation remain. They are deferred in #486.
- Separate live-server smoke: NOT RUN; no development/release server or user DB
  was changed. Main merge and release require separate authorization.

## Deferred issues and review status

- #486: remove disabled Confluence OAuth implementation while preserving token
  connections; DB/schema migration excluded unless separately decided.
- #487: remove deprecated post-canonical H5 compatibility code/tests; do not reopen
  the retired race as active remediation or remove current style-edit behavior.

Prior checkpoint notes describing H5 as an active unresolved race gate are
historical. This record supersedes those status statements: H5 is an approved
skip, full Web race passes with that skip, OAuth vet remains explicitly non-green.
The refactor is delivered for PR review, not automatically merged or released.
