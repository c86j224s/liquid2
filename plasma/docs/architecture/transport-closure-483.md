# Web, MCP and reporting closure

Issue #483. Baseline: 169fcd2. This closes the approved transport execution and
reporting ownership work rather than delivering only one small helper move.

## Web

HTTP route binding is separate from report start/preflight, execution composition,
artifact access, exports, request policy, prompt adapters and workflow adapters.
The execution runner owns newest-first stale draft/patch/humanize dispatch and
legacy draft resumption. Finalized humanize recovery still precedes provider
replay; nonrecoverable pending events remain visible and active.

Classic one-take/planned wrapper duplication was removed. Classic RunDraft keeps
its historical post-canonical H5 tail; RunLongForm keeps workflow-owned final
editing/humanization. Experimental IL, unverified and designed-renderer adapters
retain their existing specialized providers and stores rather than broadening the
lifecycle runner's port. Dead legacy wrapper removed.

The old Web recovery projection graph has no production callers. It now compiles
only as historical test support, preserving its fixtures without pretending those
fixtures exercise current typed workflow recovery. Production recovery uses the
existing workflow stage owners.

## MCP

Root retains Call/dispatch, global and stage gates, idempotency placement, tool
registration, schemas, response/error policy and trace writing. Feature execution
and request-local state belong to:

- source: accepted source inspection/search and operator-approved adapters;
- sourcecandidate: proposal, staging kickoff and candidate reads;
- reportil: source budgets/quotes, editorial memory, document and long-form workspaces;
- reportplan and reportrequirements: strict submission and parsed-call accounting;
- reportparts: assembly, section reads and editing;
- reportpatch: complete patch draft lifecycle;
- reportfinaledit: legacy editing, stage editing, read-only gates and finalization;
- reportexperiment: opt-in draft composition.

Existing research/mission/workflow handlers remain. DTO aliases preserve root
trace and test type assertions without duplicating implementations. Shared wire
contracts own the mutation envelope, artifact and extraction shapes. Quote-only
trace fields remain json-excluded. Feature handlers use the existing shared mutex
where cross-feature serialization was previously observable. Dynamic bindings and
deferred capability callbacks preserve lookup and validation timing.

## Reporting

Fourteen role files separate durable progress/result replay, semantic/evidence
replay, artifact persistence, payload decoding, Markdown structure, assembly replay,
plan normalization/hash and part-parent handling. Reporting remains the durable
contract owner. Four internally consumed APIs became private; externally used
contracts and external-test entrypoints remain. Direct tests cover evidence replay,
fenced source lines and humanization drift.

## Boundary accounting

Direct non-test physical lines (comments/whitespace included), at closure:

| Root package | Files | Lines |
| --- | ---: | ---: |
| Web | 65 | 14,181 |
| MCP root | 33 | 4,551 |
| Reporting root | 69 | 9,834 |

Child package code is not included in root counts. These are ownership measurements,
not deletion or completion percentages. Known import rows changed from 102 to 47.
Five rows represent relocated app request/service contracts (plan, requirements,
source handler/port and Web artifact access), not new feature policy dependencies.
Error-only app imports were replaced with the existing producterror sentinel owner.

The wire boundary now explicitly allows the exact ledger package for Producer in
the shared mutation envelope; ledger children remain forbidden and a regression
test enforces that distinction. This is a deliberate contract-boundary update, not
an arbitrary test waiver. No adapter or Web dependency was permitted in wire.

## Validation

Host ran full uncached `go test -count=1 ./...` successfully after architecture
reconciliation. Focused race suites passed for MCP and all child handlers,
reportexecution, reporting and reportworkflow. Root dispatch, schemas and tool
definitions were unchanged. Existing privacy, concurrency, rollback, binding,
replay and output integration tests were retained, with direct handler tests added.
Luna's scoped Web/reporting and per-feature MCP comparisons found no concrete
behavioral regression. Host checked and corrected proposal inaccuracies rather
than accepting them as requirements.

Previously recorded CLI vet unreachable-code diagnostics and the H5 Web race
assertion failure remain separate baseline issues; this is not an all-gates-green
claim. No separate runtime smoke, main merge, release or issue closure occurred.
