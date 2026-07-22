# Report Narrative Contract Product Check

## Decision

Issue #175 adopts the reader-facing narrative contract as a common Web
report-writing baseline, not a separate user choice. The existing visible
choices remain visual planning, section-centered writing, and section-centered
writing with richer cluster memory. New requests combine each choice with the
common contract. Previous profile values remain readable for stored-event and
restart compatibility rather than being reinterpreted.

This is a product-path check, not a claim that one report proves universal
quality superiority. The implementation was accepted because the actual serial
Web path completed, preserved the required source details, made a substantive
final editorial pass, and did not show the shortening regression seen in prior
rewrite experiments.

## Product Contract

- Planned and long-form planners add a small `writing_contract` to the existing
  plan: central question, reader takeaway, reading path, must-keep details,
  compressible material, supporting-layer candidates, visual role, and tone.
- The compatibility `one_take` Web API has no plan stage, so it receives the
  same reader-facing writing guidance without inventing a writing contract.
- Writers digest the original sources and explain the subject directly to a
  reader who may not read those sources.
- A Part editor can bounded-read only the immutable Sections bound to that Part.
  It may edit Part intro, transitions, and closing, but not Section bodies.
- The final editor reads a server-owned manuscript assembled from immutable Part
  artifacts. It receives exact bounded editing tools and Mermaid validation, but
  no source or research tools.
- The edited manuscript is finalized atomically. Existing Section and Part
  artifacts remain unchanged.
- Missing or invalid writing contracts are rejected at plan submission with a
  retryable validation error; the final rewrite does not start without a valid
  contract. Legacy profile values retain their preserved-assembly semantics for
  historical replay and interrupted work.
- Pending events created before guidance profiles were persisted recover through
  the legacy preserved-assembly path instead of being reinterpreted through the
  new default after a restart.

## Actual Serial Check

The development server used an existing mission with four unchanged source
snapshots. Its latest prior `visual-plan` long-form report and the candidate
report therefore shared the same source state, although their plans and request
wording were not randomized or identical.

| Observation | Prior `visual-plan` | `narrative-contract` |
|---|---:|---:|
| Parts | 4 | 4 |
| Sections | 11 | 12 |
| Final words | 8,806 | 9,050 |
| Unicode characters | 45,054 | 48,046 |
| Lines | 407 | 393 |
| Literal `candidate`-meaning mentions (`후보`) | 53 | 39 |
| Literal approval-state mentions (`승인`) | 42 | 36 |
| Literal source-management mentions (`소스`, `source`, `원천`) | 26 | 18 |

The literal counts are only a small diagnostic for process-language repetition;
they are not readability scores. Direct reading found that the candidate opened
with the reader's decision path, removed duplicated numbered headings, connected
the four Parts, and ended with user-specific actions instead of a second generic
summary. Some repeated caveats and user-tier tables remain, so this does not
close all long-form repetition work.

All five must-keep groups survived the final edit:

- the 2026-06-18 transition date and affected consumer paths;
- the consumer versus Enterprise/paid-API distinction;
- the prior npm installation and `gemini` command baseline;
- the boundary between approved snapshots and candidate live documentation;
- the multi-agent, Go, asynchronous-work, and shared-harness product rationale.

The run created 12 immutable Section artifacts and 4 immutable Part artifacts.
Every Part editor read its 3 bound Sections before submitting connective edits.
The final editor read the manuscript in bounded slices, applied 17 exact edits,
reread it, and submitted one canonical report artifact. It called no source or
research tools during final editing.

One reread guessed a stale byte offset after edits and landed inside a UTF-8
character. The tool rejected that call without changing state, the agent
recovered by rereading from a valid boundary, and the prompt was tightened to
restart at offset zero and follow returned `next_offset` values after edits.

## Latest-Build Product Checks

After restarting the development server with the latest build, two additional
HTTP checks omitted `generation_guidance_profile` from the request. Both pending
events selected `narrative-contract` and recorded its guidance hash.

The planned-report check created a three-section plan with a complete writing
contract, then produced a 6,759-character Markdown report. Direct reading found
that all five required details remained and that the report explained the date,
user-path split, prior command baseline, and platform rationale as a reader
decision path rather than a source-by-source tour.

The section-fanout check used two Parts with one Section each. Both Sections ran
in parallel. Each Part editor read its one bound Section before editing and
submitting connective text. The final editor then:

- started one server-owned manuscript;
- read it in three bounded slices;
- applied four exact edits;
- reread it in two bounded slices; and
- submitted one canonical artifact atomically.

The final event recorded 2 Sections, 2 Parts, 1,763 Section words, and 2,062 final
words with `sectional_narrative_edit` and `narrative_contract_final_edit`. It did
not contain the legacy `preservation_ratio` field. No source or research tool was
available or called during final editing.

Direct reading found one residual quality limit in both latest-build checks: the
report-level boundary that detailed migration documents were not in the approved
material appeared more than once. The reports remained complete and readable,
but the candidate does not yet guarantee that every repeated evidence-limit
caveat is collapsed to one location. This is recorded as a residual rather than
hidden behind the successful contract and tool checks.

## Verification Boundary

- Serial and section-fanout HTTP route tests assert the same writing-contract,
  bound Part-read, final-editor allowlist, lineage, and exact-artifact behavior.
- Reporting tests cover exact manuscript finalization, conflict replay, atomic
  creation, and SQLite restart replay.
- MCP tests cover closed schemas, tool visibility, bound Section reads, in-memory
  final editing, and canonical submission.
- Codex and Claude executor acceptance tests connect provider-protocol shims to
  the built MCP server; the candidate path submits a required writing contract
  and uses the bound final editor tools.
- `go test ./...` passes for Plasma after the default-profile and recovery
  compatibility changes.
- Raw prompts, tool payloads, generated manuscripts, and runtime databases stay
  outside the public repository under the experiment artifact policy.

These checks establish product-surface parity and bounded behavior. They do not
claim universal statistical quality superiority.
