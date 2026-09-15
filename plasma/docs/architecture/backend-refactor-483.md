# Backend Refactor #483 Architecture Record

> Recovery reconstruction: this record was rebuilt from the successful source, artifact, and mission closeout transcripts after the original uncommitted document was lost. It is faithful to the recovered scope, but not claimed byte-exact. The recovery incident and reconstructed-document status are part of this record.

## Status and boundary

The recovered work and this follow-on slice complete the source, artifact, mission, research-catalog, evidence-record, question-record, option-record, claim-confidence, and proposal creation/decision ownership portions of #483. It does **not** complete the full #483 application-boundary refactor. `internal/app` still contains research application orchestration, visibility, materialization, workflow, report legacy models, connector facades, and the lookup, persistence, and atomic I/O orchestration; proposal storage and atomic commit orchestration remain there.

The recovery-lineage slice moves only `ReportRecoveryLineage` and its direct unit-test closure from Web to `internal/reportexecution`. Web calls the canonical function directly; reporting and reportrun lineage semantics remain separate.

## Before → after capability map

| Capability | Before #483 slice | After recovered closeout | Remaining `internal/app` responsibility |
| --- | --- | --- | --- |
| Source | App-owned source snapshot builders and policy-shaped models | `internal/source` owns snapshot creation contracts, validation, locator parsing, retrieval policy, ordered hashing, and non-persisting assembly | Atomic artifact/snapshot/event commit orchestration and application service wiring |
| Artifact | App-owned raw-artifact construction and identity/content policy | `internal/artifact` owns construction, storage URI, identity validation, hash verification, normalization, timestamps, and defensive copying | Persistence and atomic transaction orchestration |
| Mission | App-owned mission metadata/lifecycle contracts and append policy | `internal/mission` owns metadata, lifecycle-change contracts, append-request construction, normalization, and idempotency policy | Conditional append, projection rebuild, active-work validation, and related I/O orchestration |
| Research catalog | Research IDE references, summaries, pages, kind constants, and pure list/reference algorithms lived in app | `internal/researchcatalog` owns the service, `ObjectRef`, `ObjectSummary`, `Page`, `References`, exact kind constants, outline/list orchestration, source/artifact/ledger/research-record summaries, report block reads, locator metadata projection, reference deduplication, and normalization/gating/limit/cursor/pagination/reference algorithms; it collaborates only with canonical root mission/source/artifact/ledger/researchrecords contracts and product errors, with no proposal or app reverse dependency | App retains DTOs, visibility, persistence callbacks/adapters, upload/read policy, payload materialization, and transport orchestration |
| Research inspection read and grep | Bounded object-read validation, UTF-8 chunking, and literal grep sequencing lived in app alongside materialization | `internal/researchinspection` owns exact `ReadRequest`/`ObjectRead` and `GrepMatch`/`GrepResult` contracts, validation, callback order, byte-preserving `ClampBytes`/`ChunkBytes`, case-insensitive non-overlapping literal matching, snippets, and pagination; app wires callbacks and retains PDF/local-path observation, security/visibility, and payload materialization | App retains callback implementations and candidate construction; no generic source I/O or query engine is introduced |
| Evidence records | Evidence identity, source-ref validation, confidence normalization, and creation lifecycle lived in app | `internal/researchrecords` owns evidence models, the narrow source-snapshot reader port, validation/normalization, and evidence builder tests; it imports only exact `internal/ledger`, `internal/producterror`, and `internal/source` roots | App retains record/source lookup, visibility, materialization, and persistence/atomic I/O orchestration |
| Claim confidence updates | Claim confidence request, event payload, append validation, origin normalization, and projections lived in app | `internal/researchrecords` owns the confidence request/payload/projection types, validation builder, origin normalization, and event projections; app supplies I/O callbacks | App retains record/evidence lookup and persistence/atomic append I/O |
| Question and option records | Question/option identity, creation validation, normalization, and builders lived in app | `internal/researchrecords` owns question/option models, schema/object constants, requirement ports, builders, and builder tests | App retains mission-event/record lookup, visibility, materialization, and persistence/atomic creation I/O |
| Research proposal surface | Proposal and claim/question policy lived in app and proposal/MCP adapters | `internal/researchproposal` owns proposal creation, submission allowlists, decision payload validation, and terminal transitions; `internal/mcp/research` owns transport adaptation | App retains ledger/record lookup, storage, visibility, materialization, and persistence/atomic commit I/O; evidence creation is delegated to `internal/researchrecords` |

The research-record move covers evidence, claim, question, and option record creation plus claim confidence in `internal/researchrecords`; proposal creation and decision live in `internal/researchproposal`. App retains lookup, visibility, materialization, and persistence/atomic I/O orchestration rather than these record and proposal policies.

The app import surface is guarded by focused architecture tests; this evidence slice adds no baseline debt.

| Application facade | Broad compatibility hub, including capability re-exports | The completed source/artifact/mission models and policy bodies, plus the catalog, research-record, confidence, and proposal creation/decision contracts and algorithms, are no longer app-owned | Workflow, research application orchestration, lookup/materialization, report legacy models, persistence/atomic I/O, and connector services, plus the source/artifact/mission orchestration listed above |

The old app proposal policy bodies and source builder wrappers were removed rather than re-exported. The research catalog move is intentionally partial: app-owned visibility, lookup, materialization, report legacy models, and persistence/atomic I/O remain for a later independently verified slice. `internal/researchrecords` owns evidence, claim, question, and option record creation plus claim confidence; `internal/researchproposal` owns proposal creation and decision. The six source contracts are concrete source-owned types. The remaining legacy aliases outside this slice are retained only where their migration is not part of this closeout.
## Measured change

Counts are direct non-test, non-generated Go files and lines under each package. The
following before/after table is a historical stage comparison recorded at the
`d027ab6` checkpoint; it is not a complete `d027ab6`-only snapshot and is not the
current tree. It is retained to preserve the recovered closeout record.

| Package | Before files / lines | After files / lines | Delta |
| --- | ---: | ---: | ---: |
| `internal/app` | 78 / 15,961 | 75 / 15,402 | -3 / -559 |
| `internal/source` | 3 / 180 | 9 / 679 | +6 / +499 |
| `internal/artifact` | 3 / 51 | 4 / 123 | +1 / +72 |
| `internal/mission` | 7 / 627 | 8 / 760 | +1 / +133 |

The historical architecture import-debt baseline decreased from 145 to 118
entries; the catalog and evidence boundaries added no new baseline debt. The three
deleted app model files contained only `package app` (with one deprecated comment in
`projection_models.go`) and had no declarations.

### Current measured scope at `6d89457` (2026-09-08)

The current direct non-test, non-generated Go-file counts and line counts are:

| Package | Current files / lines |
| --- | ---: |
| `internal/app` | 77 / 12,639 |
| `internal/researchcatalog` | 6 / 921 |
| `internal/researchinspection` | 4 / 271 |
| `internal/researchrecords` | 10 / 1,117 |
| `internal/researchproposal` | 5 / 786 |

The architecture import-debt baseline is now 105 entries, down from the original
145. The current 105 consist of 101 `app-hub` entries and 4
`transport-to-adapter` entries; there are zero `capability-to-adapter` entries. These
are current measurements, not a claim that the full application-boundary refactor is
complete.

## Checkpoint and handoff

This CLI source-file partition is mechanical and same-package only: it improves readability by separating Confluence dispatch, auth, connections, browse, snapshot, update, client, local-source, read, and output responsibilities without changing package ownership. The original `source_commands.go` retains `runSources`, shared repeated-string flags, read-size constants, and upload.

The moved declarations are: `runSourcesConfluence` and `printSourcesConfluenceUsage` → `source_commands_confluence_dispatch.go`; OAuth command/config helpers → `source_commands_auth.go`; connection/site commands → `source_commands_connections.go`; spaces/pages/children/search and page printing → `source_commands_browse.go`; preview/snapshot → `source_commands_snapshot.go`; check-update/update-preview/update → `source_commands_update.go`; client/browser/site/cloud discovery helpers → `source_commands_client.go`; local roots/tree/attach/list/show/remove/restore and local config helpers → `source_commands_local.go`; read/live/snapshot/grep and artifact content reads → `source_commands_read.go`; raw-artifact metadata/response and shared output/error helpers → `source_commands_output.go`.

The CLI split added ten files and seven additional app-import baseline rows. The
original `source_commands.go` still retains the shared `runSources`/flag/upload edge
described above; the seven moved files are the rows that continue to import
`internal/app`.
This is an existing app-hub edge relocated across same-package files: no new package
edge and no new package ownership boundary were introduced.

The current known CLI vet diagnostics are in `source_commands_auth.go` at lines 36
and 90 (`go vet ./...` unreachable-code diagnostics). They are recorded as known
verification debt, not silently treated as a clean vet result.

`internal/reporthumanize` documents the legacy/manual post-canonical H5
compatibility contract: same-session binding, validation, terminal event application,
safe failure/no-op preservation, and restart recovery. It is not the active long-form
path. The active long-form pre-canonical style-edit stage remains separate in the
`internal/reportworkflow` reporting pipeline. Web retains request normalization,
executor/model selection, route locks, and orchestration for the relevant paths; CLI
invokes its transport-neutral path without importing Web.

The Phase 0 PDF test accounting is also explicit: the original ten tests were
restored as nine tests in `internal/reportilpdf` plus one externally authored Phase 0
test. The bridge file is wiring, not itself a test.

The legacy/manual H5 humanization duplicate-rejection failure/recovery test remains
unresolved. It surfaced during the auto-compaction refactor, and the baseline and
changed implementations both reproduced the failure in 20/20 runs under the tested
race condition. Later targeted normal tests passed, but that does not establish
globally deterministic behavior or an all-race-green result. The recent full-module
test pass and the selected PDF race pass are distinct verification results.

The checkpoint trail through `6d89457` and the prior recovery incident remain part of
this record. Historical snapshots above are dated steps in that trail and do not
represent the current measurements.

The recovered implementation lineage is, in order:

`f359b47` → `a93a1ef` → `260e6c4` → `4e59d0e` → `63c143c` → `5da2e3a` → `630018a` → `ddb56e2`

`a1106b0` is the required research-catalog starting checkpoint for this slice. The source/artifact/mission closeout lineage remains `ddb56e2`; this slice does not rewrite that lineage. No main merge, commit, GitHub operation, or issue-closure action is included here. The next owner should treat the completed source/artifact/mission ownership as stable, and continue any remaining app-facade migration as separate, independently verified work rather than claiming #483 is fully complete.

## Verification record

Focused catalog, app, MCP, SQLite, web, and architecture tests pass for this slice. The catalog package depends only on the standard library and `internal/producterror`; no source adapter or transport import was added. `go vet ./...` still has the two known unreachable-code diagnostics in the CLI source command file. Real provider execution and browser validation were not run.

No user-machine paths, local runtime state, provider sessions, or raw recovery artifacts are part of this public record.

## Liquid2 source capability boundary

The Liquid2 consumer contract now lives in `internal/source/liquid2source`, including its request/result/document models, snapshot nested contracts, connector identity constants, and externally used normalization functions. The package depends only on the exact `internal/source`, `internal/artifact`, `internal/ledger`, and `internal/producterror` contracts. The parent `internal/source` package does not import its child; HTTP behavior remains in `internal/connectors/liquid2`, which depends on the child contract and product error sentinel but not on `internal/app`. Application snapshot content-range, producer, artifact, event, and atomic orchestration remain in `internal/app`.

## Confluence source capability boundary

`internal/source/confluencesource` now owns the Confluence source connector ports, request/result contracts, normalization rules, external identity helpers, and structured failures: the stable `ConfluenceError` type, category/code constants, safe status/message helpers, HTTP/transport constructors, and their private cause closure. It depends on the exact `internal/source` and `internal/producterror` contracts; the parent `internal/source` does not import this child. Stored credentials and accessible-site projections remain in `internal/confluenceaccess`, while snapshot/artifact/event and update orchestration remain in `internal/app`.


## Phase 0 PDF renderer boundary

The Phase 0 consumer package owns the `PDFRenderer` port and `PDFResult` contract. Chrome execution remains in `internal/reportilpdf.Chrome`, which owns the executable path, 45-second timeout, temporary profile, readiness checks, and renderer identity. CLI and Web composition points preserve existing Chrome-path resolution timing and inject the adapter. A nil renderer fails explicitly at the render stage without skipping the provider or adding an automatic Chrome fallback.
