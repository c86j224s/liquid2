# Report IL experimental handoff

- Tracking scope: accepted Report IL product work
- Product status: accepted by the user and closed on 2026-09-02
- Product baseline: accepted implementation as of 2026-09-02
- Pipeline family: `report_il_experimental`
- Compiler: `plasma.report_il.phase0_compiler.v68`
- Product manifest: `plasma.report_il.product_manifest.experimental.v36`
- Semantic document: `plasma.report_il.experimental.v2`
- Checkpoint: `plasma.report_il.product_checkpoint.experimental.v1`
- Provider contract: Codex, `gpt-5.6-luna`, `xhigh`; no DeepSeek

This document is the starting point for a new high-context session. It records the final product boundary, implementation map, operational state, verification evidence, and the work that was deliberately left outside Issue #361. Historical experiments and failed gates remain preserved; do not reinterpret them as current failures.

## 1. Final disposition

Issue #361 delivered an independent, opt-in report family rather than replacing the classic report pipelines. It is available in the Plasma product UI and supports both standard and long-form authoring, three validation profiles, source-backed images, and multi-format compilation.

The accepted product behavior is:

- existing planned and classic long-form report families remain independent;
- an IL failure does not fall back to a classic writer;
- source facts and prose are authored through server-bound contracts rather than compiler invention;
- Markdown, self-contained HTML, and PDF are compiled from one accepted semantic document;
- provider, document, artifact, source, and retry lineage are durable;
- long-form retries resume from verified checkpoints;
- repairable publication-quality failures receive one bounded repair pass;
- structural, source-binding, and artifact-integrity failures remain terminal;
- report-level humanize remains disabled for this family;
- no public snapshot, tag, or release was created as part of Issue #361.

The user verified the final runtime and explicitly approved closing the issue.

## 2. Current product topology

```text
accepted mission sources
  -> candidate + image source catalogs
  -> optional source selection
  -> validation-profile authoring plan
  -> standard authoring OR long-form hierarchical authoring
  -> publication reader
  -> optional factual continuity editor
  -> source-backed image placement
  -> target-neutral semantic document validation
  -> Markdown + self-contained HTML + Chrome PDF
  -> atomic artifact bundle storage
```

### Standard authoring

`AuthoringModeStandard` uses one of two source-bound paths depending on the validation profile:

- `unverified`: direct source authoring without editorial memory or publication passes;
- `exploratory` / `strict`: editorial memory followed by report-first authoring and publication reading;
- `strict`: adds the final continuity pass.

### Long-form authoring

`AuthoringModeLongForm` is hierarchical:

```text
source selection
  -> editorial memory
  -> long-form plan
  -> Section authors (bounded fan-out)
  -> Part editors
  -> complete-manuscript final author
  -> publication reader
  -> strict-profile continuity editor
  -> images / document / render / store
```

The plan owns stable Part and Section keys. Section and Part artifacts are independently finalized and verified before assembly. The final author receives verified Part artifacts, not a prompt-expanded copy of every source.

### Validation profiles

The durable values are:

- `unverified`
- `exploratory`
- `strict`

Their stage topology is owned by `internal/reportilphase0/validation_profile.go`. Do not infer behavior from UI labels alone; the manifest records both `authoring_mode` and `validation_profile`.

## 3. Important contracts and ownership boundaries

### Source access

Source access is request-local and server-bound through `reportilcontract.SourceAccessBinding`.

- The provider receives opaque catalog and artifact bindings.
- Reads are bounded and traced.
- Selection and authoring operate against frozen source snapshots.
- Raw source bodies, provider responses, prompts, and private locators must not be persisted in failure payloads.
- Source selection is optional only when the complete catalog already fits the target budget and contains no exclusions.

Primary files:

- `internal/reportilcontract/source_access.go`
- `internal/reportilphase0/product_sources.go`
- `internal/mcp/report_il_source_tools.go`

### Editorial memory

For exploratory and strict paths, connected source accounts are captured before prose. The memory preserves actor, role, action, relationship, date, duration, uncertainty, and exact source anchors as one connected account. Later author and continuity stages must preserve those relationships rather than merely retaining nearby facts.

Primary files:

- `internal/reportilcontract/editorial_memory.go`
- `internal/reportilphase0/editorial_memory.go`
- `internal/mcp/report_il_memory_tools.go`

### Server-owned author workspaces

The provider edits server-owned workspaces through MCP tools. A stage is accepted only when its workspace is finalized exactly once and the resulting raw artifact matches the trace receipt by mission, producer, media type, SHA-256, byte size, revision, stage, and schema.

Primary files:

- `internal/reportilcontract/author_document.go`
- `internal/mcp/report_il_document_tools.go`
- `internal/mcp/report_il_long_form_tools.go`
- `internal/app/report_il_author_document.go`

### Semantic document and compilation

The compiler owns structure, IDs, evidence references, target syntax, and render validation. It does not invent prose or source facts.

Primary files:

- `internal/reportilphase0/model.go`
- `internal/reportilphase0/report_first.go`
- `internal/reportilphase0/long_form.go`
- `internal/reportilphase0/product_run.go`
- `internal/reportilphase0/render.go`

### Images

Images come from the frozen image catalog and are selected after prose finalization. The product supports repeated image bytes across report bundles while retaining source-ingestion deduplication. Image artifacts are bundle-local and appear in the product manifest.

Primary files:

- `internal/reportilphase0/product_images.go`
- `internal/reportilphase0/product_sources.go`
- `internal/app/report_il_store.go`

### Mermaid and equations

Self-contained HTML embeds shared renderer assets. Chrome PDF generation waits for the renderer completion contract and fails closed on Mermaid or equation render errors. Raw Mermaid source must not appear in the rendered PDF when rendering succeeds.

Primary files:

- `internal/reportilphase0/render.go`
- `internal/reportmermaidassets/`
- `internal/reportmathassets/`
- `internal/reportilphase0/render_chrome_smoke_test.go`

## 4. Checkpoint and retry model

Long-form checkpoints are content-addressed lineage, not hints. A checkpoint must pass envelope, catalog, source-selection, editorial-memory, artifact, finalization, and reader-finalization validation before reuse.

Supported checkpoint stages:

- `il_long_form_parts`
- `il_long_form_final`
- `il_reader`
- `il_continuity`

### Parts checkpoint

After every planned Section and Part has finalized and passed validation, the product writes `il_long_form_parts` before the complete-manuscript final-author pass. A retry can therefore reuse:

- source selection;
- editorial memory;
- long-form plan;
- every Section artifact;
- every Part artifact.

It reruns only the final author and later stages.

Pre-checkpoint historical runs can be promoted only after their trace slice and every plan/Section/Part artifact are verified. The recovery CLI is:

```text
plasma-report-il-recover-parts -db PATH -mission mis_... -pending evt_... [-write]
```

Primary files:

- `internal/reportilcontract/checkpoint.go`
- `internal/reportilphase0/checkpoint.go`
- `internal/reportilphase0/legacy_checkpoint.go`
- `internal/app/report_il_checkpoint.go`
- `cmd/plasma-report-il-recover-parts/`

### Reader checkpoint and bounded repair

`il_reader` now distinguishes repairable quality failure from contract failure.

Repairable quality codes include:

- `reader_facing_content`
- `reader_opening`
- `reader_audit_voice`
- `reader_ordinary_si`
- `reader_process`
- `reader_metadata`
- `reader_internal_machinery`

For one of these failures, the product performs at most one repair pass. That pass opens the exact finalized publication candidate, receives the safe validation code, makes the smallest necessary text edit, rereads the complete document, and finalizes once.

The following remain terminal without repair:

- malformed or unfinalized workspace;
- duplicate finalization;
- document schema or structure change;
- source-binding change;
- artifact hash, byte-size, producer, or mission mismatch;
- transport failure;
- any unclassified semantic failure.

`reader_process` detection is intentionally limited to explicit report/section navigation language. Normal subject prose such as “먼저 검토되었고 … 다음 판단” or “독자가 따로 확인해야 할 질문” must not be rejected.

Primary files:

- `internal/reportilphase0/reader_patch.go`
- `internal/reportilphase0/reader_quality.go`
- `internal/reportilcontract/source_access.go`

## 5. Product UI and persistence

The route is integrated into the normal Plasma report controls. The user selects the IL family, authoring mode, and rigor/validation profile; progress is projected from durable events.

Long-form progress includes:

- source packet;
- source selection;
- editorial memory;
- plan;
- Section nodes;
- Part edit nodes;
- final author;
- reader / continuity;
- images;
- document / render / store;
- final artifact.

Retry controls have separate capabilities:

- `resume_failed`: resume from the newest verified checkpoint;
- `restart`: start from the beginning.

A missing resume checkpoint must not disable a full restart. A completed attempt exposes neither control.

Primary files:

- `internal/web/report_il.go`
- `internal/web/report_il_retry_checkpoint_test.go`
- `internal/ledgerstate/report_progress.go`
- `internal/app/report_runs.go`

The runtime SQLite database is stored outside Git under:

```text
research-artifacts/liquid2/plasma/runtime/dev-6002/plasma-ui-user.db
```

Generated reports, provider traces, screenshots, PDFs, and failure databases must remain outside Git.

## 6. Acceptance evidence

Two late-stage missions are particularly useful regression anchors.

### Checkpoint resume anchor

- Mission: `mis_20260831090047_f8cdd5b5`
- Title: `신장의 야망 신생 게임 초보자 팁`
- Original failed pending: `evt_20260901091358_a1a90a61`
- Recovered checkpoint: `il_long_form_parts`, 12 Sections, 4 Parts
- Retry pending: `evt_20260901125419_85b9be99`
- Attempt: 2
- Final state: completed
- Completion time: 2026-09-01 13:07:18 UTC

This proves that a pre-checkpoint final-author failure can be recovered and resumed without reauthoring Sections or Parts.

### Reader false-positive and repair anchor

- Mission: `mis_20260812142856_dae5878b`
- Title: `재미있게 읽히는 글의 특징`
- Origin pending: `evt_20260901150749_e824075d`
- Failed reader attempts: 1–3, `reader_process`
- Accepted retry pending: `evt_20260901212517_df192f2d`
- Attempt: 4
- Final state: completed
- Completion time: 2026-09-01 21:30:50 UTC

The rejected candidates had been finalized correctly; the validator misclassified normal subject prose. PR #457 narrowed the detector and added bounded candidate repair. The user verified that the changed runtime succeeds.

### Broader dogfood evidence

The durable Issue #361 evidence tree contains the full sequence of protocol locks, failed changed-runtime gates, corrections, screenshots, output bundles, and accepted gates:

```text
$RESEARCH_ARTIFACTS/liquid2/plasma/experiments/issue-361-report-il-dogfood/
```

On the original development machine, `$RESEARCH_ARTIFACTS` is the user-owned `research-artifacts` root outside Git.

Start with its `HANDOFF.md`, then read `protocol/protocol.lock.md` and the numbered `checkpoints/` only when historical detail is needed. Failed checkpoints are retained intentionally; they are provenance, not cleanup candidates.

## 7. Test and runtime state at closure

Focused suites passed on the final product change:

```text
go test ./internal/reportilcontract ./internal/reportilphase0
go test ./internal/mcp ./internal/agentexec ./internal/app \
  ./internal/reportexecution ./internal/ledgerstate ./internal/web
```

The full `go test ./...` still contains known clean-main baseline failures outside the final change:

- `internal/architecturecheck/TestPackageBoundaries`: baseline drift for existing report IL app-hub imports;
- `internal/mcp/research/TestLegacyMutationCommonInputErrorsPreservePreExtractionStrings`;
- existing `internal/reportworkflow` failure-contract tests;
- existing `internal/reportworkflow/internal/finaledit` failure-contract test.

Do not report the full suite as green. Do not “fix” the architecture baseline as a side effect of unrelated work; open a dedicated issue if that boundary is intentionally changed.

After the final product merge, the repository-owned full development stack was restarted. Liquid2 API, Liquid2 Web, Plasma API, and Plasma Web all returned HTTP 200. The deployed Plasma binary matched a fresh build from product-code commit `37a181cf9b3cdd8e636fc849c143e6ad523d4c61` by SHA-256.

Operational command:

From the internal repository root:

```text
./dev-browser.sh restart
```

Runtime logs:

```text
/tmp/liquid2-dev-api-6011.log
/tmp/liquid2-dev-web-6001.log
/tmp/plasma-browser-6002.log
/tmp/plasma-browser-6002.err
```

## 8. Merge history map

The complete PR history is linked from Issue #361. Use these groups to orient rather than replaying every patch chronologically.

- #363–#382: initial product path, source-tool migration, server-owned structure, quality and terminology grounding.
- #383–#404: selected-source grounding, evidence receipts, lossless IL compilation, continuity and reader corrections.
- #405–#424: publication quality, connected accounts, editorial memory, source-backed images, PDF, storage, UI integration, validation profiles.
- #425–#432: long-form mode, hierarchical Section/Part graph, language/conclusion constraints, progress projection.
- #433–#439: long-form depth, equations, publication repair, technical locators, PDF code readability, image ingestion.
- #440–#453: durable checkpoints, legacy recovery, retry/run projection, storage diagnostics, checkpoint lineage, reader artifact promotion.
- #454–#457: self-contained Mermaid rendering, deterministic reprojection authorization, Parts resume, reader false-positive and bounded repair.

The last product-code merge is #457. This handoff document is a documentation-only follow-up.

## 9. Durable constraints for future work

These constraints survived the full dogfood cycle and should be treated as product decisions unless a new issue explicitly changes them.

1. Keep `report_il_experimental` independent from classic report families.
2. Use Luna only for this path; do not add DeepSeek fallback.
3. Keep source access frozen, bounded, and trace-verifiable.
4. Do not send raw source bodies through expanded prompts when MCP reads are available.
5. Do not let the compiler rewrite prose or invent source facts.
6. Keep failure payloads content-free and typed.
7. Preserve exact connected-account causation through publication and continuity.
8. Keep report artifacts atomic and bundle-local while preserving source-ingestion deduplication.
9. Keep HTML self-contained and make PDF generation wait for equation/Mermaid rendering.
10. Resume only from fully verified checkpoints and artifacts.
11. Repair only classified reader-quality failures and cap the repair loop.
12. After every internal-main merge, restart the full development stack and verify actual HTTP plus deployed binary revision/hash.
13. Do not publish a public snapshot, tag, or release without a separate explicit release decision.

## 10. Work deliberately left outside Issue #361

Issue #361 is complete. The following are possible follow-ups, not closure blockers:

- optimize provider/token cost without reducing source or reader quality;
- revisit image candidate ranking, selection, and placement quality;
- clean the known full-suite architecture and failure-contract baselines;
- decide whether and how the experimental family should graduate, be renamed, or become a default;
- create release notes and a public snapshot only after an explicit release gate.

The prior session’s local task IDs (for example its cost and image-review tasks) are not durable project authority. A new session must open or claim a dedicated GitHub issue before changing code for any follow-up. Do not reopen #361 merely to host unrelated optimization.

## 11. New-session bootstrap

A new 1M-context session can start with:

1. Read this document.
2. Read Issue #361’s final handoff comment and closure reason.
3. Confirm internal `main` and runtime state rather than assuming the recorded commit is still current.
4. If investigating a historical regression, read the matching numbered checkpoint under the durable evidence tree.
5. If implementing follow-up work, create/claim a new issue and a new sibling worktree from current `origin/main`.
6. Keep Issue #361 worktrees and branches deleted; use Git history and durable artifacts for archaeology.

Issue #361 should remain closed unless its accepted behavior itself regresses and reopening is preferable to a focused regression issue.
