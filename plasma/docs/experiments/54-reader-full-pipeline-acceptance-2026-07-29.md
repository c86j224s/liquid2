# Reader Prompt Full-Pipeline Acceptance

Date: 2026-07-29

Status: accepted for Issue #190 integration; product-path preservation verified
in one exploratory and one strict long-form report

Related issue: #190

## Question

Does the corrected reader-edit prompt still provide useful whole-report
orientation when it runs inside the complete product workflow, and do the later
style, integrity, and canonicalization stages preserve rather than undo that
effect?

This was a candidate-only integration acceptance. The preceding fixed-draft
experiment isolated the prompt difference; this run tested the selected prompt
through the complete product path. It was not another A/B comparison.

## Product Path

Two isolated reports used the same public HTTP, job-runner, durable ledger, and
MCP boundaries as the product:

1. section fan-out and Part editing,
2. whole-report reader editing,
3. style editing when enabled,
4. corrective integrity gate,
5. canonical finalization.

The cases deliberately covered both report modes requested for acceptance:

| Topic | Rigor | Style pass |
| --- | --- | --- |
| Public-health guidance | Strict | Enabled |
| MCP server safety | Exploratory | Enabled; valid no-op |

Both reports completed, every required stage was durably submitted exactly
once, and each run produced exactly one canonical artifact.

## Execution Evidence

The product-reported word counts and edit operation counts were:

| Case | Part assembly | Reader | Style | Gate | Canonical |
| --- | ---: | ---: | ---: | ---: | ---: |
| Public health / strict | 5,201 | 5,341 (8) | 5,340 (3) | 5,277 (5) | 5,277 |
| MCP safety / exploratory | 6,918 | 6,973 (6) | 6,973 (0) | 6,993 (4) | 6,993 |

Numbers in parentheses are submitted edit operations. In both cases the
canonical artifact was byte-identical to the accepted gate output. Stage
lineage, input/output hashes, restart-safe state, and terminal product gates all
validated.

## Direct Reading

The host read each complete Part-edited assembly and then the complete textual
change made by every downstream stage.

In the strict report, the reader stage added a useful report-level question and
answer, normalized duplicated Part headings, and added a closing synthesis. The
style pass was narrow. The integrity gate removed or qualified unsupported
generalizations without breaking the argument. The final report remained
caveat-heavy because the input was one broad source, but the downstream stages
did not erase the reader improvement.

In the exploratory technical report, the reader stage added a concise
whole-report orientation, removed internal provenance identifiers from
reader-facing citations, and made small clarity corrections. The style stage
correctly made no change. The gate replaced unsupported examples with cases
present in the source and qualified a few inferences while preserving the
report's practical synthesis.

The prompt therefore passed the integration question: its useful orientation
survived the complete workflow in both rigor modes. No additional prompt or
workflow correction was introduced from this run.

## Boundary Of The Result

This result verifies execution parity and downstream preservation for two full
reports. It does not establish statistical superiority, broad topic coverage,
or finished prose quality. The reader stage made a modest intervention and did
not remove every repeated Part introduction or evidence-boundary phrase. Those
remaining writing limitations are not converted into new Issue #190 scope by
this acceptance.

## Archive Evidence

Generated reports, stage artifacts, run databases, ledgers, and the detailed
host reading note remain outside the repository under:

`~/research-artifacts/liquid2/plasma/experiments/54-reader-full-pipeline-acceptance-2026-07-29/`

This repository stores only the redacted protocol, aggregate evidence, and
decision boundary.
