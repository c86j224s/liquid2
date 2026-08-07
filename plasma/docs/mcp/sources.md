# Plasma MCP Sources Guide

URI: `plasma://docs/mcp/sources`

This guide defines how Plasma MCP separates sources, source candidates, source
snapshots, and live local path observations.

## Term Boundary

- A source is original research material, such as a document, URL, file, PDF, or
  external repository.
- Evidence is a specific part of a source used as support.
- A result is an agent-produced summary, comparison, answer, conclusion, or
  draft. A result is not a source.
- Saved knowledge is a result or claim that Plasma deliberately stores for a
  mission.
- A report is an output assembled from saved knowledge and evidence for a
  reader.

Agent answers, controller outputs, and report drafts must not be reclassified as
sources. When needed, explain which sources and evidence a result depends on.

## Source snapshot

`plasma.sources.list` and `plasma.sources.read` operate on source snapshots that
the user has accepted into a mission. Soft-removed or superseded sources are not
used for default reading or new report writing unless the user explicitly asks
for audit or history review.

For PDF and upload sources, the original file is the source. MCP read tools
return bounded extracted text and extraction metadata instead of raw original
bytes.

## Connector Search and Candidates

`plasma.sources.search` results are candidates, not sources. Search results, grep
matches, and connector titles are not evidence or saved knowledge until they are
read and judged.

When original material is worth user review, record it with
`plasma.sources.candidates.propose`. This records a review candidate. It does not
create a source snapshot and does not add the material to the default report
source set.

`plasma.sources.candidates.read` reads staged unapproved candidates for
conversation and investigation. Candidate content must become a user-approved
source snapshot before it is used as default report support.

## Live local path

Live local path sources are addressed only through a configured root's `root_id`
and a `relative_path`. Do not use absolute filesystem paths as MCP tool inputs
or documentation examples.

Reading live material records bounded observation metadata instead of creating a
new source body snapshot. If a report depends on live material, cite observation
event metadata such as observed time, relative path, content hash, and git
metadata when available. Do not cite only the source id.

## Safe Reporting

When summarizing source reads, use only the short parts needed for the reader.
Do not paste credentials, private URLs, provider responses, or full document
bodies into reports or error explanations.
