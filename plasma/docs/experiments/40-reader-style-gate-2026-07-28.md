# Reader/Style/Gate Product-Path Comparison

## Status

The W7 product comparison and a fixed-draft reader diagnosis are complete.
The product matrix passed all execution gates, but the candidate won only 2 of
4 blind pairs and lost both public-health pairs by a moderate margin. The
candidate is not promoted. A subsequent same-MCP fixed-draft smoke diagnosis
also failed its semantic gate, so the remaining three diagnosis pairs were not
run and no further reader-prompt iteration is approved inside Issue 190.

## Question

Does the full Issue 190 finalization package, evaluated as a whole, improve the
active long-form Web `section_fanout` path versus the old-compatible
finalization path while preserving source reads, fixed request settings, and
exactly-one canonical completion?

The comparison does not attribute any result to reader edit, style edit, or
corrective gate individually.

## Locked Inputs

- Baseline ref: `feat/issue-190-long-form-author-workflow`
- Baseline commit: `2f8be94c224b364207e1b4eabc65d6b7f4d6f097`
- Candidate committed HEAD: `066858ba69240c925732c01d833528b9a305b004`
- Shared committed tree: `082c32f6260222db0d7f04dc6db119b26db59812`
- Candidate status porcelain SHA-256:
  `0b8d86577659d09b02927e5f48e92675d80146a21e91f770a58095b000fcc6b6`
- Candidate git-diff binary SHA-256:
  `c1e4bdd907ce2a633cdcd128dc8e68b6330eaae67c54b2c440ceb77ed88ffcdd`
- Baseline binary SHA-256:
  `2ac01dcef3c652b2e61dca89e7c135c5a65e29ac8a8bb7aba8faf35462744211`
- Candidate binary SHA-256:
  `0ae31cce9c7ab0e27d6056e3bc6d69b7c01f867301541cc0f05d8a4a30b4fe23`

The baseline and candidate committed trees matched before the candidate's
uncommitted Issue 190 changes were built.

## Product Boundary

The runner used public product boundaries only: CLI mission creation, local
source attachment, Web long-form report POST, event polling and artifact
download, and MCP source/research reads. The fixed Web request settings were:

- `report_mode=long_form`
- `execution_strategy=section_fanout`
- `agent_executor=codex`
- `agent_model=gpt-5.5`
- `agent_reasoning_effort=medium`
- `generation_guidance_profile=narrative-contract`
- `report_session_policy=same_session`
- `post_report_humanize=enabled`

The planned main matrix was two arms across two topics and two rigor levels
(`exploratory`, `strict`), with at most two workers. The smoke gate had to pass
before the matrix could run.

## Product Comparison Result

The main matrix used two topics, two rigor levels, and baseline/candidate arms.
All 8 cells passed the hard execution gates. In a blinded direct read, the
candidate won both technical-topic pairs strongly, but lost both public-health
pairs moderately. No visible unique source fact, citation, number, code or
technical identifier, requirement, or uncertainty boundary was lost.

The promotion rule required at least 3 of 4 wins and no moderate-or-worse loss,
so the candidate is not promoted. Because baseline and candidate generated
independent upstream plans, sections, parts, and manuscripts, this result is an
end-to-end comparison and cannot isolate the reader stage.

## Fixed-Draft Reader Diagnosis

The diagnosis held one candidate upstream manuscript fixed and ran the real
product reader MCP tools under two prompts: the current reader instruction and
one author-owner instruction. Both arms used the same input SHA, model,
reasoning effort, report-plan fork, and exactly the reader `start`, `read`,
`patch`, and `submit` tools. QA passed, and no style, gate, or canonicalization
event occurred.

The current arm made 15 edits; the author-owner arm made 11. Direct reading
found no unique-information loss in either arm. The current arm slightly
improved this pair by removing a repeated closing conclusion and repairing
source-process residue. The author-owner arm mostly corrected headings and
citation labels, leaving the duplicated closing structure and failing to make a
paragraph- or section-level explanatory improvement. Both retained frequent
source-process narration instead of consistently speaking as the report's
author to its reader.

This fixed-draft semantic gate therefore failed. The remaining three diagnosis
pairs were intentionally not run, and neither prompt is promoted. The result
does not justify a new MCP tool or persistence surface; it indicates that the
current reader stage is acting as a cautious copyeditor rather than a final
author. Any next workflow change requires a separately approved Issue 190
scope decision.

## Archive Evidence

Local archive root:
`~/research-artifacts/liquid2/plasma/experiments/40-reader-style-gate-2026-07-28/`

Stable archive-relative evidence:

- `control/protocol.lock.json`
- `control/paired-matrix.json`
- `analysis/host-blind-reading-notes.md`
- `runs/main/` redacted manifests and reports for the 8 product cells
- `diagnostics/fixed-draft-reader-mcp/qa-smoke.json`
- `diagnostics/fixed-draft-reader-mcp/outputs/` source/reader pairs and
  sanitized traces for the fixed-draft smoke

Raw databases, provider rollouts, event ledgers, and generated report bodies
remain only in the local archive. The public repository contains only this
redacted decision summary.

## Decision

Do not promote the current reader prompt or the author-owner variant. Preserve
the W7 evidence and stop prompt iteration in #190 until the workflow role
boundary is reconsidered. This public record contains only redacted conclusions;
raw databases, provider traces, and report bodies remain in the local archive.

## Subsequent User-Approved Prompt Experiment

The stop decision above remains the result of W7. On 2026-07-29, the user
explicitly approved a separate, narrower reader-prompt experiment based on the
recorded human reading criteria. That work does not retroactively change the W7
verdict. See
[Reader Orientation Boundary Prompt Experiment](53-reader-orientation-boundary-prompt-2026-07-29.md)
for the corrected hypothesis, bounded product-path comparison, and current
promotion boundary.
