# Plasma MCP Mermaid Guide

URI: `plasma://docs/mcp/mermaid`

This guide defines the checks to apply before adding Mermaid diagrams to Plasma
reports or answers.

## Basic Principle

Use a Mermaid diagram only when it helps the reader understand structure, order,
dependency, or comparison. If a simple Markdown list is enough, do not turn it
into a diagram.

Call `plasma.mermaid.validate` before showing diagram source to the user. The
tool statically checks Mermaid parse risks and compatibility risks that Plasma
knows about.

## Reading validate Results

- `ok: true`: the known static preflight rules passed. This does not guarantee a
  browser render.
- `ok: false`: read `errors` and `warnings`, revise the source, and validate
  again.
- If there are only warnings, still fix patterns that could make the report hard
  to read.

## Writing Rules

- Quote node labels, requirement text, and long descriptions so the parser does
  not reinterpret them.
- Avoid Markdown, HTML, and dense punctuation inside labels.
- Do not put full source bodies or long quotations inside diagrams. Use concise
  labels and summaries.
- Use stable ASCII tokens for IDs. Put reader-facing descriptions in labels.
- A diagram is not evidence by itself. In a report, explain the source and
  evidence connections that the diagram summarizes.

## If Validation Fails

Do not hide validation failure and ship the diagram anyway. Remove the parser
risk with the smallest clear revision and validate again. If validation keeps
failing, drop the diagram and use ordinary Markdown structure.
