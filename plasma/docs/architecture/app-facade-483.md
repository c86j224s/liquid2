# App facade ownership checkpoint

Issue: #483. This checkpoint continues the app-only policy extraction after the
60a9621 measurement. It does not change report schemas, retry rules, visibility,
provider behavior, or persistence ordering.

## Moved ownership

- `source/liquid2source`: snapshot content selection, rune ranges, payload and
  locator encoding (checkpoint 1ba7ea2).
- `source/confluencesource`: snapshot payload, selected range validation, preview
  ranges, version normalization and ordering. Connector IO and update commits
  remain in app.
- `reporting/reportdocument`: legacy report models, block construction,
  block/reference validation, Markdown/HTML/JSON export and footnotes. This child
  package has no dependency on app or the reporting orchestration root.
- `reportexecution`: retry request and event-history decision helpers. App still
  evaluates them inside the same conditional-append snapshot; no validation was
  moved outside the atomic boundary.
- `reportilsource`: source-read trace verification. The event callback preserves
  validation-before-read ordering and does not introduce an additional read.

Report/approval, retry-request and Confluence range aliases were removed from
app; callers reference the actual owner. Retry lineage unit tests moved with the
policy. Existing app integration tests remain; direct reportdocument export and
block validation tests were added. The unrelated test-only SHA helper remains
in app test code, not production.

## Measurement

Direct, non-test Go files, physical lines including comments and whitespace:

| Baseline | Files | Lines |
| --- | ---: | ---: |
| Issue start d027ab6 | 78 | 15,961 |
| Before resumed app work 60a9621 | 77 | 12,639 |
| Intermediate a00d72f | 74 | 10,648 |
| App ownership closure | 72 | 9,932 |

These are package-size measurements, not a completion percentage or deleted-code
count. Source owners gained the moved code. The measured four policy groups have
been addressed, but this is not a claim that every app method is a thin wrapper.

## Retained and residual responsibilities

App retains transactions, conditional append, record loading, projection wiring,
connector IO, observations, research materialization and visibility checks.
The closing pass moved uploaded-file classification, filename and metadata rules
into `source`; connection normalization into `confluenceaccess`; terminal
compatibility and IL receipt/lineage validation into `reportexecution`; and pure
scope normalization, promotion-event validation and format allowlists into
`reportdocument`. Existing app-backed integration tests remain, with direct
scope/promotion characterization tests added at the document owner.

Record-backed report scope/approval resolution and download eligibility remain
in app deliberately: they enforce mission-scoped reads, visibility/security and
historical ledger compatibility. Conditional append, bundle transactions, stage
companion ordering and projection defaults remain at their original application
boundaries. No generic owner or alternate persistence layer was introduced.

This closes the bounded app policy extraction, not an assertion that every
application method is a one-line delegate or that every file meets the soft
size target. Twelve app files still exceed 200 physical lines and retain the
application responsibilities above; future splitting is not part of this closure.

## Validation

- Full uncached `go test -count=1 ./...`: passed after direct-owner caller migration.
- Focused race tests: app, reportexecution, reportilsource,
  source/confluencesource and reporting/reportdocument passed.
- Read-only Luna comparison of the initial implementation moves found no concrete
  behavior regression or import cycle. The subsequent alias/caller migration was
  checked by compilation, full tests and the architecture boundary test.
- The final closure also passed focused race tests for source, confluenceaccess,
  reportexecution, reportdocument, app and SQLite, and the full uncached suite.
- A second bounded Luna comparison found no concrete regression in the closing
  pass. Only formatting issues were reported and corrected before commit.
- Architecture baseline removed two resolved Web-to-app rows and one MCP-to-app
  row across both passes; no new boundary exception was accepted.
- This checkpoint does not resolve or waive the previously recorded CLI vet
  unreachable-code errors or H5 race assertion failure. No separate server smoke,
  main merge, release or issue closure was performed.
