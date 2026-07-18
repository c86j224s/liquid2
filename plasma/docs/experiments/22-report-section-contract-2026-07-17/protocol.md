# Protocol

## Objective

Measure whether strengthening the existing long-form section `purpose` into a
section-writing contract improves report readability while preserving the
current product path.

## Current Baseline

The baseline follows the current Web long-form path as closely as the harness
permits:

1. create a mission in an archive-local database;
2. attach a local source fixture through the existing local-source path;
3. request a long-form report through the Web report API;
4. create the plan through `plasma.report.plan.submit`;
5. draft section artifacts with MCP/source reads;
6. assemble parts while preserving section bodies;
7. finalize the report through the existing long-form finalization path.

The experiment must stop if it needs to paste source bodies into prompts, alter
the plan schema, or treat generated section/part text as source material.

## Candidate

The candidates keep the same product path and change only the writing guidance
profile.

### Arm 1: section contract

- `generation_guidance_profile = "section-contract"`
- the plan prompt asks the agent to put the section contract inside the
  existing `purpose` string;
- the section prompt receives the same plan shape and uses the purpose as the
  section's writing contract;
- part and final assembly still preserve section bodies.

The contract is not a new schema. It is prose in the existing purpose field.
It should include:

- the section's central point;
- what the reader should understand after the section;
- the evidence path the section should inspect;
- what adjacent topic or broad background the section should avoid.

### Arm 2: section contract with coverage lock

The second candidate keeps the same section-contract idea but adds an explicit
coverage guard:

- `generation_guidance_profile = "section-contract-coverage"`
- keep normal long-form coverage density unless the source packet is genuinely
  small;
- do not reduce Parts or Sections merely because the section purposes are more
  concrete;
- every major source-backed cluster should appear in a Section, coverage note,
  or planned omission.

This arm is intended to separate two effects that were confounded in the first
pilot: clearer section intent and shorter/lower-coverage outlines.

### Arm 3: section intent

The third candidate keeps the current plan schema and weakens the intervention:

- `generation_guidance_profile = "section-intent"`
- the plan prompt asks the agent to put quiet editorial intent inside the
  existing `purpose` string;
- the intent describes what the reader should come to notice, understand, or
  question by the end of the section;
- it does not ask for a central-point/evidence-path/boundary contract;
- it does not add coverage counts, section-count targets, or a hard lock.

This arm tests whether section writers can be guided by a sense of reader
movement without making the planner compress the report into a cleaner but
thinner outline.

### Arm 4: source cluster first

The fourth candidate changes where planning begins:

- `generation_guidance_profile = "source-cluster-first"`
- before outlining, the planner identifies major source-backed clusters:
  definitions, mechanisms, examples, numbers, tensions, caveats, comparisons,
  and missing evidence;
- the outline is built after that cluster pass;
- `coverage_notes` records how each important cluster is handled: planned
  Section, planned omission, or out-of-scope reason.

This arm tests whether the report becomes less thin when the planner preserves
the material map before it chooses the neat outline.

### Arm 5: section brief

The fifth candidate is a softer version of a section contract:

- `generation_guidance_profile = "section-brief"`
- each existing `purpose` string becomes a light writing brief;
- the brief should carry reader movement, concrete details to keep visible, a
  tension or caveat to handle, and an adjacent-topic boundary;
- it must stay natural prose, not a labeled checklist or new schema.

This arm tests whether section writers need more useful writing orientation
than `section_intent`, without the rigid contract that shortened reports.

### Arm 6: plan review

The sixth candidate keeps the workflow shape unchanged but asks the planner to
review its own outline before the first successful plan submission:

- `generation_guidance_profile = "plan-review"`
- before submitting, the planner checks whether the outline became too narrow,
  whether a major source-backed cluster disappeared, whether the Part/Section
  count is artificially low, and whether caveats are detached from the sections
  that need them;
- if the plan is thin, the planner revises before submitting;
- `coverage_notes` briefly states what the review preserved or why the source
  packet is genuinely small.

This first experiment does not add a separate post-submit review stage. That
would be a larger workflow change and should be tested only if pre-submit review
looks promising.

### Arm 7: section brief with cluster memory

The seventh candidate keeps the best pilot direction and adds only a gentle
source-memory cue:

- `generation_guidance_profile = "section-brief-cluster-memory"`
- each Section purpose remains a light prose writing brief;
- while researching, the planner notices important source-backed clusters:
  mechanisms, examples, numbers, caveats, comparisons, policy tensions, and
  missing-evidence boundaries;
- the brief may mention the most important clusters that should stay visible;
- `coverage_notes` may record inspected clusters as a memory aid, but the arm
  does not require a rigid cluster map.

This arm is meant to test whether `section_brief` can preserve breadth better
without inheriting the compression failure of `source_cluster_first`.

## Execution Plan

1. Build one experiment binary from this worktree.
2. Run one paired smoke topic through `baseline` and `section_contract`.
3. Check terminal events, plan artifact presence, final report artifacts, and
   report readability by direct reading.
4. If smoke is clean, run a small diverse pilot.
5. Expand to a three-arm run over the available diverse fixtures:
   `baseline`, `section_contract`, and `section_contract_coverage`.
6. If both contract arms shorten or thin reports, run a narrow follow-up pilot
   with `section_intent` to test a softer intent-guidance approach before any
   full statistical expansion.
7. If `section_intent` still shortens reports, run a narrow follow-up pilot over
   `baseline`, `source_cluster_first`, `section_brief`, and `plan_review`.
8. If `section_brief` is the best pilot candidate, run a statistical expansion
   over 24 topics with `baseline`, `section_brief`, and
   `section_brief_cluster_memory`.
9. Use aggregate metrics and direct editorial reading to decide whether either
   candidate improves section focus without damaging coverage or flow.

## Quality Rubric

| Dimension | What to check |
| --- | --- |
| Section center | Each section has a clear local point and does not drift into generic background. |
| Reader takeaway | A reader can tell why the section exists in the report. |
| Evidence path | Concrete source details support the section instead of appearing as a list. |
| Flow | Paragraphs move from claim to evidence to implication naturally. |
| Repetition control | Caveats and claims are not repeated in a tiring formula. |
| Source fidelity | Claims stay within source support and uncertainty remains visible. |
| Assembly fit | Part intros, transitions, and final report structure still read coherently. |

## Stop Conditions

Stop and report if:

- the candidate requires a new plan field;
- report generation fails for a harness or provider reason that prevents a fair
  paired comparison;
- the candidate improves local prose but drops source-backed detail;
- the result depends on rewriting section bodies in part assembly;
- raw experiment material would need to enter the Git worktree.

## Deliverables

- Public protocol and running summary in this directory.
- Local raw archive with run directories, plans, reports, ledgers, and blind
  packets.
- A concise Korean briefing after smoke and after any expanded run.
