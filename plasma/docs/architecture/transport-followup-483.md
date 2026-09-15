# Transport and reporting ownership follow-up

Issue #483, baseline 7b3f0c6. Three bounded changes were selected from the approved
Web, MCP and reporting follow-up. This is not a claim of complete transport or
reporting decomposition.

## Web recovery

The exact legacy Web pending-recovery gate and decoder now belong to
`reportexecution`. Web supplies its compatibility defaults and retains runner
binding, failure append and background dispatch. The generic execution decoder
remains unchanged. Legacy Web omission of `rigor_label` and `pipeline_graph` is
preserved intentionally: forwarding them would be a separate behavior change.
No request route, streaming path, generator callback or prompt/model changed.

## MCP frozen source resolution

`reportilsource.ResolveFrozenSource` owns snapshot identity/state checks, live
read completeness, artifact receipt verification and canonical readable-content
comparison. MCP supplies ordered reader callbacks, including a lazy bound-session
live-read adapter. Tool decoding, allowlists, budgets, errors and content-free
traces remain unchanged. The existing repeated single-artifact read is preserved.
A direct test locks removed-state rejection before artifact access.

## Reporting/workflow cohesion

Durable replay, semantic acceptance and Markdown contracts remain in reporting.
Final-edit duplicate JSON formatting, filename and session validation helpers now
reuse existing `longformutil` implementations. No prompt literal changed. Added
direct tests cover JSON bytes/fallback, filenames and session mismatch clearing.
An unused final-edit plan predicate was removed and the reasoning-effort method
made private after caller verification. No new utility package was introduced.

## Validation

Three Luna proposal seats and the same three resumed verification seats were used;
host implemented and adjudicated. All three bounded comparisons found no concrete
regression. Host rejected forwarding previously ignored Web recovery fields.

Full uncached `go test -count=1 ./...` passed. Focused race tests passed for MCP,
reportilsource, reportexecution, reportworkflow and reporting packages. The final
private-API cleanup also passed reportworkflow tests. These results do not repair
or waive the previously recorded CLI vet or H5 Web race assertion failures.
No separate runtime server smoke, main merge, release or issue closure occurred.

Remaining broad work includes Web runner/generator orchestration, other MCP tool
families and large reporting replay files. They were not moved merely to reduce
line counts; only the three verified ownership boundaries above are delivered.
