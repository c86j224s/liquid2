# Report Part Connective-Economy Experiment

## Status

Completed as an issue #179 follow-up to experiment 37. All 12 report runs
finished, producing six complete blinded pairs. The candidate passed every
advancement rule. After end-to-end reading, the user explicitly approved the
cumulative candidate as the default for new long-form reports.

Experiment 37 improved direct Section writing, but the Part assembler added
17,632 characters on top of 108,725 characters of immutable candidate Section
drafts, an increase of about 16.2%. Tracked document-position phrases also rose
from 11 in Section drafts to 50 in assembled Parts. Final editing then changed
total length only slightly.

## Result

The Part-only instruction reduced connective overhead in every topic. Because
the two arms generated different plans, the primary comparison is the ratio
inside each run rather than final report length alone.

| Part assembly measure | Baseline | Candidate |
|---|---:|---:|
| Completed reports | 6 | 6 |
| Immutable Section characters | 105,989 | 94,843 |
| Added Part connective characters | 17,182 | 3,015 |
| Connective characters / Section characters | 16.2% | 3.2% |
| Position markers introduced by Part assembly | 38 | 2 |

Both Part measures improved in all six topics. The final editor then added
about 4.7% to the candidate Part manuscripts, mainly through report-level
opening and closing material, while removing one tracked position marker. It
did not restore the repetitive Part previews and recaps.

| Final report measure | Baseline | Candidate |
|---|---:|---:|
| Average words | 4,950 | 4,113 |
| Average wall time | 578.8 s | 491.2 s |
| Topics with fewer candidate words | - | 5 of 6 |
| Final tracked position markers | 42 | 11 |

The candidate was preferred in all six blinded comparisons. Direct reading
found that Part titles and substantive Section openings were enough to carry
the structure. The reports did not feel mechanically concatenated, and their
conclusions retained source limits, caveats, mechanisms, examples, and
unresolved questions.

A later end-to-end reread after the arm mapping had been revealed confirmed the
same six preferences, but it is a mapped confirmation rather than a second
independent blinded judgment. That reread also identified the Section-level
repetition and evidence-relative length work now tracked in issue #189.

## Decision

Adopt `part-connective-economy-voice` as the default for new long-form reports.
This is the cumulative experiment lineage: reader-centered writing contract,
curiosity-led explanation, natural voice, edited reading, direct Section
writing, and Part connective economy. One-take and planned reports keep their
existing default, explicit older profiles remain reproducible, and stored
report events with a recorded profile are not migrated or reinterpreted.

This decision resolves the measured Part-assembly verbosity. It does not claim
that Section-level repetition or evidence-relative report length is solved.
Those questions, together with Korean original sources, multi-source conflict,
market research, and source-sparse investigations, continue in issue #189.

## Question

> Can Plasma keep the direct, detailed Sections from experiment 37 while
> removing Part introductions, transitions, and closings that merely preview,
> repeat, or recap them?

## Arms

| Arm | Guidance profile | Meaning |
|---|---|---|
| Baseline | `section-direct-reading-voice` | Experiment 37 candidate. |
| Candidate | `part-connective-economy-voice` | The same planning, Section, and final guidance plus a Part-only rule that connective text must perform a necessary reading job. |

The candidate leaves Section bodies mechanically preserved. It does not impose
a report word limit, shorten source-backed explanation, change plan schema, or
add a cleanup pass.

## Runner

Use:

```bash
cd plasma
python3 scripts/experiments/report_part_connective_economy_experiment.py --action prepare --limit 6
python3 scripts/experiments/report_part_connective_economy_experiment.py --action run --limit 6 --workers 2
python3 scripts/experiments/report_part_connective_economy_experiment.py --action analyze --limit 6
python3 scripts/experiments/report_part_connective_economy_experiment.py --action packets --limit 6
```

The analyze action writes ordinary paired report metrics and a separate stage
aggregate. The stage aggregate compares each run's immutable Section characters
with its assembled Part characters, so independent plan variation does not get
mistaken for Part verbosity.

## Reading Focus

Advance the candidate only if:

- all six topic pairs complete without a systemic arm failure;
- Part connective-character ratio falls in most topics and in aggregate;
- Part-introduced document-position phrases fall in most topics and in
  aggregate;
- at least four of six blinded comparisons prefer the candidate;
- Parts do not feel abrupt, mechanically concatenated, or deprived of a useful
  cross-Section synthesis;
- citations, caveats, examples, mechanisms, and unresolved tensions remain in
  the immutable Sections.

Do not treat total report word count alone as the decision metric. Separate
runs can produce different plans and Section counts even when their planning
prompts are the same.

## Source Limitation

The six fixtures are the same English Wikipedia single-source bundles used in
experiment 37. This run can isolate Part assembly across several subject areas,
but it cannot establish generalization to Korean original sources, multi-source
conflict, market research, or source-sparse investigations.

## Public Artifact Policy

Commit only this protocol, small aggregate metrics, and redacted conclusions.
Keep generated reports, intermediate manuscripts, judging packets, databases,
logs, and copied source bundles under the local experiment archive described in
`plasma/docs/artifact-archive.md`.
