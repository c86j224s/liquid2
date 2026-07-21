# Protocol

## Objective

Compare the current long-form Part assembly path against a candidate where the
Part assembly agent submits only connective tissue through MCP edit tools.

## Fixed Inputs

- Source fixture archive: experiment 17 fixtures.
- Report mode: `long_form`.
- Default execution strategy: `section_fanout`.
- Agent executor: Codex.
- Model default: `gpt-5.5`.
- Reasoning effort default: `medium`.
- Post-report humanize: disabled.
- Report session policy: same session.

## Arms

1. `visual_plan`
   - `generation_guidance_profile`: `visual-plan`
   - Existing JSON-return Part assembly behavior.

2. `part_assembly_edit_tools`
   - `generation_guidance_profile`: `part-assembly-edit-tools`
   - Planning and section-writing guidance mapped to `visual-plan`.
   - Part assembly must use the bound MCP tools and finalize through
     `report.part_assembly.submitted`.

## Product-Path Requirements

Each run must:

- start an isolated Plasma server from the experiment binary;
- use an isolated DB under the local experiment archive;
- attach a local source through the product source path;
- create the report through the HTTP report endpoint;
- collect ledger events from the mission API;
- download the final Markdown artifact only after `report.artifact.created`.

The runner must not create prompt-only source dumps or bypass the report
endpoint.

## Fixed-Plan Follow-Up

The first product-path smoke confirmed execution but did not isolate the
assembly change: the two arms independently created different long-form plans.
The follow-up therefore freezes one product-created plan per topic and resumes
both arms from cloned state.

The fixed-plan run must:

- create the seed plan through the normal report endpoint;
- stop the isolated server after `report.plan.created` and before any
  `report.section.created`, `report.part.created`, or `report.artifact.created`
  event;
- clone the seed DB, fixture, and Codex provider session state for each arm;
- change only the cloned pending/plan guidance profile metadata before resume;
- resume through the normal mission-detail recovery path, not a direct internal
  function call;
- require the paired arms to have the same `plan_signature` before quality
  comparison.

This follow-up excludes plan quality from the comparison. It tests whether the
same plan produces a better final report when the Part assembly stage uses the
MCP connective-edit tools.

## Acceptance Checks

Before comparing quality:

- both arms complete for the same topic;
- the candidate has one `report.part_assembly.submitted` event per created Part;
- the candidate event contains only connective assembly fields, not section
  bodies;
- no terminal report failure occurs in a paired run.

Then compare:

- final word count ratio;
- preservation-ratio delta;
- wall-clock ratio;
- direct reading of whole reports for flow, naturalness, detail retention, and
  caveat placement.

## Stop Conditions

Stop before productization if:

- the candidate frequently fails to submit MCP assembly events;
- the candidate rewrites or summarizes section bodies instead of adding
  connective tissue;
- terminal failures concentrate in the candidate arm;
- direct reading shows smoother but less faithful reports.
