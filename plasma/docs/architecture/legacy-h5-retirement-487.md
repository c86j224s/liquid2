# Legacy H5 retirement

Refs #487. Removes manual/post-canonical H5 execution, not current long-form
pre-canonical style editing. Existing ledgers and stored artifacts are not deleted.

## Removed

- reporthumanize execution/recovery/prompt package and package-only tests.
- Classic Web, legacy workflow and CLI post-canonical H5 calls and result plumbing.
- CLI `reports draft -humanize` flag and browser creation/retry function/action.
- Web H5 provider callback and start implementation.
- Obsolete H5 execution tests, including the previously skipped race test.

## Compatibility and historical closure

The historical HTTP launch URL returns 410 Gone, without accessing stored content
or launching a provider. Existing humanized artifacts remain readable/downloadable
and historical event classification/cancellation remains supported.

Per explicit user approval, stale report.humanize.pending entries are terminally
closed via AppendReportTerminalIfOpen with report.humanize.failed and kind
humanized_markdown_report_retired. The terminal preserves parent pending ID, title,
source artifact/hash, model/session metadata and transport; it marks retired=true,
internal_failure_detail=feature_retired and original Markdown preservation.
Repeated reconciliation cannot duplicate the terminal. No patch recovery/promotion
or provider replay occurs. This happens on normal recovery, not as a DB migration.

Runner StartHumanize/ResumeHumanize/RunHumanize remain small fail-closed compatibility
entrypoints, not workers. The H5 cancellation payload decoder remains for historical
records. The shared Markdown fidelity validator remains used by patch/style editing.

## Current functionality retained

Legacy final artifact generation/completion remains, with only its H5 tail removed.
V1/V2/V3 pre-canonical style/semantic/evidence stages and PostReportHumanize settings
that select those stages are retained. Token connections, databases and schema are
untouched. The OAuth vet cleanup is a separate branch/PR (#486 / #489).

## Acceptance

Retirement tests cover 410 without service access, no provider or finalized-patch
recovery, one terminal on repeated recovery and preservation of original lineage.
Historical artifact UI access and current workflow parity checks remain. Obsolete
launch expectations were removed rather than broadly skipping suites.

Final validation: full uncached `go test -count=1 ./...` passed. Race tests for
Web, reportexecution and reportworkflow (including current style-edit stages)
passed. This branch still starts before the separate OAuth cleanup; its two
pre-existing OAuth vet diagnostics are resolved by PR #489, not by this change.
