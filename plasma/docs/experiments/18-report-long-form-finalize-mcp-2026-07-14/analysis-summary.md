# Experiment 18 Analysis Summary

## Execution Outcome

The corrected two-arm smoke passed. The baseline completed without calling the
long-form finalizer. The candidate completed with one successful finalizer call,
one matching canonical artifact event, and the exact sentinel acknowledgement.

The frozen quality matrix then ran exactly 24 Codex long-form cells: 12 topics,
two arms, one replicate, and no planned or Claude cells. Twenty-three cells
completed. One candidate cell failed after the product mutation boundary and
was retained as an intention-to-treat failure. It was not replaced. The machine
gate therefore failed.

Eleven complete report pairs were blinded and scored with the experiment 17
rubric and judge adapter. The controller then stopped during analysis because it
called the final endpoint calculation before applying the existing
intention-to-treat low-score assembly step to the failed record. No confirmatory
bootstrap bound, completeness guardrail, or sensitivity p-value is reported.
Computing those values with an unregistered one-off command would violate the
frozen controller boundary.

## Gate Result

| Gate | Result |
| --- | --- |
| Corrected two-arm smoke | Pass |
| Exact 24-cell quality matrix | Pass |
| Product-path machine gate | Fail: one post-boundary candidate failure |
| Blind score completion | 11 complete pairs scored |
| Preregistered final analysis | Not completed: controller stopped |
| Adoption | False |

The local archive preserves the first stopped smoke attempt, the corrected
smoke, all 24 quality runs, blind packets, private mapping, scores, and an
immutable stopped record. No raw material is committed here.
