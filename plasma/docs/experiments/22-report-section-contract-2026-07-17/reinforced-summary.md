# Reinforced Summary

## Scope

The reinforcement run expanded the section-contract experiment from a small
pilot to a full 24-topic, three-arm comparison.

Arms:

| Arm | Meaning |
| --- | --- |
| `baseline` | Current long-form `g2` report guidance. |
| `section_contract` | Existing `purpose` field carries central point, reader takeaway, evidence path, and boundary. |
| `section_contract_coverage` | Same section contract plus an explicit guard to preserve normal long-form coverage density. |

The run stayed on the product-like path: archive-local Plasma database, local
source fixture attachment, Web report API, MCP plan submission, MCP/source reads
for section drafting, section-preserving part assembly, and long-form
finalization.

Raw reports, ledgers, and judging packets remain in the local artifact archive.

## Execution

Archive:
`research-artifacts/liquid2/plasma/experiments/22-report-section-contract-2026-07-17-3arm-r2/`

Completed reports:

| Item | Count |
| --- | ---: |
| Total reports | 72 |
| Completed reports | 72 |
| Failed reports | 0 |
| Paired topics | 24 |
| Arms per topic | 3 |

Two harness issues were found during the reinforcement run and fixed in the
experiment runner:

- completed runs did not release their reserved loopback ports inside the same
  runner process;
- interrupted runs could leave a directory without a terminal manifest, blocking
  resume.

These were runner-resume issues, not report-generation failures.

## Aggregate Metrics

### `section_contract` vs baseline

| Metric | Result |
| --- | ---: |
| Median word ratio | 0.852 |
| Mean word ratio | 0.893 |
| Shorter than baseline | 18 / 24 |
| Longer than baseline | 6 / 24 |
| One-sided sign-test p for shorter output | 0.0113 |
| Median section ratio | 1.000 |
| More sections than baseline | 10 / 24 |
| Fewer sections than baseline | 10 / 24 |

Interpretation: the existing section-contract arm has a statistically visible
shortening tendency, even though section count does not consistently fall.

### `section_contract_coverage` vs baseline

| Metric | Result |
| --- | ---: |
| Median word ratio | 0.699 |
| Mean word ratio | 0.767 |
| Shorter than baseline | 20 / 24 |
| Longer than baseline | 4 / 24 |
| One-sided sign-test p for shorter output | 0.00077 |
| Median section ratio | 0.894 |
| More sections than baseline | 5 / 24 |
| Fewer sections than baseline | 14 / 24 |
| One-sided sign-test p for fewer sections | 0.0318 |

Interpretation: the coverage-lock arm failed its purpose. It made the shortening
and section-reduction tendency stronger, not weaker.

## Editorial Reading

Direct reading supports a split conclusion.

The `section_contract` arm often improves local section focus:

- headings are more explicit;
- section openings tend to state the local point sooner;
- sections are less likely to read like broad source inventories;
- part flow can feel cleaner when section roles are clear.

The same arm also loses useful breadth in repeated cases:

- baseline sometimes keeps separate treatment of policy, metrics, uncertainty,
  implementation, or application frames;
- the candidate often merges those frames into a shorter final part;
- the result can read smoother but less patient and less richly structured.

The `section_contract_coverage` arm did not solve that problem. On several
topics it compressed even more aggressively despite the coverage guard.

## Decision

Do not productize either arm.

The underlying idea remains useful: section writers benefit when the plan tells
them what a section is trying to do. The current prompt form is not enough,
because it changes the planning behavior toward tighter, shorter reports.

The next candidate should not merely add another warning about preserving
coverage. It should change the planning contract so that section focus and
coverage density are evaluated separately before the plan is accepted.

Possible next direction:

- keep the existing report plan schema unless a separate issue explicitly
  approves schema work;
- ask the planner to identify source-backed clusters first;
- require each cluster to map to a section, coverage note, or planned omission;
- then ask for section purposes that make the mapped sections easier to write;
- reject or retry plans that reduce coverage without a concrete source-size
  reason.
