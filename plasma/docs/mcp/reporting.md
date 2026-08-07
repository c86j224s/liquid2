# Plasma MCP Reporting Guide

URI: `plasma://docs/mcp/reporting`

This guide describes the boundary for Plasma MCP reporting tools. A report is
not a source or an agent transcript. It is a result assembled from saved
knowledge and evidence so the reader can understand the subject.

## Shared Boundary

- Use report tools only in their assigned report stage.
- Do not rewrite mission, session, pending-event, or artifact bindings supplied
  by the runner or server.
- Do not reclassify source, evidence, or saved knowledge for writing
  convenience.
- Do not put provider responses, credentials, private URLs, or runtime
  identifiers into report prose.

## Plan and Requirement Tools

`plasma.report.plan.submit` durably submits the report structure and intent. One
plan submission should represent one plan that the runner can promote. Reuse the
same idempotency key only for the same meaningful retry.

The requirement mapping tool attaches explicit user output requirements to an
already fixed outline. It does not redesign the outline or change the research
direction.

## Part Assembly and Edit

Part assembly tools write connective Markdown around immutable Section bodies.
They handle introductions, transitions, and closing text instead of rewriting
Section bodies.

Part edit tools open one assembled Part as an isolated draft and apply bounded
edits. They do not directly mutate other Part or Section artifacts.

## Long-form finalization

Long-form finalization and final edit stage tools operate only inside the stage
binding supplied by the runner. Writer, reader, style, gate, and evidence-gate
stages have different responsibilities. Do not use one stage's tool to apply
another stage's policy.

Finalization is not a simple file save. It is a product state transition recorded
through the ledger boundary. If an error occurs, read the current draft or stage
state, decide whether the next call is the same retry or a new change, and then
continue.

## Patch Tool

Report patch tools modify an existing Markdown report artifact through bounded
slices and small edit operations instead of pasting the whole report into a
prompt. Patch tools are not research tools for reading new source material. Use
them only inside the safe edit boundary of the existing report.
