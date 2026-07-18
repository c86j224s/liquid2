# Pilot Summary

## Scope

The pilot tested whether a long-form `section-contract` guidance profile can
make section-fanout reports read with a clearer local center.

The product path stayed close to the current Web long-form route:

- archive-local Plasma database;
- local source fixture attachment;
- Web report API request;
- `plasma.report.plan.submit` for planning;
- MCP/source reads for section drafting;
- section-preserving part assembly;
- existing long-form finalization.

No development or release database was used. Raw reports and ledgers remain in
the local archive.

## Runs

### v1 archive

Archive:
`research-artifacts/liquid2/plasma/experiments/22-report-section-contract-2026-07-17/`

| Topic | Arm | Parts | Sections | Words | Wall seconds |
| --- | --- | ---: | ---: | ---: | ---: |
| public-health-guidance-a | baseline | 3 | 10 | 4582 | 296.6 |
| public-health-guidance-a | section_contract | 3 | 9 | 3756 | 297.5 |
| public-health-guidance-b | baseline | 5 | 13 | 7000 | 419.2 |
| public-health-guidance-b | section_contract | 4 | 9 | 4578 | 322.2 |
| transport-safety-a | baseline | 5 | 12 | 6659 | 371.2 |
| transport-safety-a | section_contract | 3 | 10 | 4910 | 396.2 |

Completed paired topics: 3. Terminal failures: 0.

### r2 archive

After v1, the candidate guidance was amended to say that a sharper section
contract must not collapse source clusters or reduce necessary coverage. A
single paired smoke was run in a separate archive:

`research-artifacts/liquid2/plasma/experiments/22-report-section-contract-2026-07-17-r2/`

| Topic | Arm | Parts | Sections | Words | Wall seconds |
| --- | --- | ---: | ---: | ---: | ---: |
| public-health-guidance-a | baseline | 4 | 9 | 4529 | 342.6 |
| public-health-guidance-a | section_contract | 3 | 7 | 3218 | 313.5 |

The r2 wording did not fix the compression problem. On the smoke topic it made
the candidate even shorter.

## Editorial Reading

The candidate moved in the intended direction on section focus:

- section titles were more explicit;
- opening paragraphs usually stated the section's local point sooner;
- sections were less likely to become broad source inventories;
- transitions often felt cleaner because the section role was clearer.

The candidate also showed a systematic downside:

- it shortened reports substantially across all v1 topics;
- it often reduced the number of parts or sections;
- some baseline reports were richer, slower, and more patient in historical or
  policy context;
- the candidate's stronger local focus sometimes came from dropping or merging
  material rather than organizing the same material better.

This means the current candidate confounds two effects:

1. clearer section-level intent;
2. shorter report plans with lower coverage.

The first effect is promising. The second prevents a product recommendation.

## Current Interpretation

Do not productize this candidate yet.

The experiment produced useful evidence that a more concrete `purpose` can help
section writers. It also showed that the current wording pushes the planner
toward tighter and shorter outlines. Because Plasma reports need to stay rich
and useful, the next candidate should separate the two controls:

- keep or explicitly target the current long-form coverage range;
- strengthen each section's local thesis, reader takeaway, evidence path, and
  boundary;
- evaluate whether section focus improves when coverage is held roughly stable.

## Next Candidate

The next candidate should not simply say "write a better purpose." It should
probably add a planning constraint such as:

- normal long-form reports should keep roughly the same coverage density as the
  current baseline unless the source packet is genuinely small;
- each section purpose must contain a local thesis and evidence path;
- adjacent source clusters should not be merged merely to make the outline
  cleaner.

That candidate should be run as a new arm or successor archive before any
larger statistical run.
