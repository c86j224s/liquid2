# R8 Long-form Hierarchical Report Experiment

Status: execution completed on 2026-06-25; preservation acceptance failed for
three Parts.

R8 tests a different report-composition structure from `R0` through `R7`.
The previous variants mostly asked one writer turn to produce or integrate a
whole report. That made it too easy for the final integration step to collapse
rich section material back into a short summary. R8 instead treats the completed
AUTO investigation set as one long-form report project:

1. map each completed `CQ1` through `CQ8` AUTO investigation to one Part;
2. parse only numbered entries from the existing `R7` section outlines;
3. write each section independently and in parallel;
4. preserve-edit sections into Part manuscripts;
5. generate only front matter and closing globally;
6. assemble the final report mechanically, preserving Part manuscripts.

The important design rule is that section briefs, Part drafts, editor notes, and
front matter are intermediate results. They are not sources. Report claims
should still cite original source snapshots such as `CQ1-S1.md`, not generated
experiment artifacts.

## Runner

Script:

```bash
python3 plasma/scripts/experiments/g0-controller-mcp-followup/longform_report_experiment.py \
  --clean \
  --force \
  --jobs 6 \
  --timeout-sec 1500
```

The runner is intentionally separate from
`report_composition_experiment.py`. R8 is not another single-run prompt
variant; it is a hierarchy experiment over the whole completed AUTO set.

Generated output:

- `runs/R8-longform-hierarchical/final_report.md`
- `runs/R8-longform-hierarchical/sections/`
- `runs/R8-longform-hierarchical/part_manuscripts/`
- `runs/R8-longform-hierarchical/global/`
- `runs/R8-longform-hierarchical/run_manifest.json`
- `runs/R8-longform-hierarchical/section_results.csv`
- `runs/R8-longform-hierarchical/part_results.csv`

The runner uses a local temporary workspace during execution. The durable review
artifacts are copied under the `runs/` directory above, so the experiment does
not depend on the temporary workspace for later inspection.

## Result

The execution completed and produced a report, but the run is marked as failed
against the preservation acceptance check because three Parts fell below the
`0.7` Part preservation threshold.

| Metric | Value |
| --- | ---: |
| Parts | 8 |
| Sections | 69 |
| Section draft words | 76,497 |
| Part manuscript words | 60,839 |
| Final report words | 63,121 |
| Final / section words | 0.825 |
| Final / Part words | 1.038 |
| Duration | 2,017 sec |
| Execution status | completed |
| Preservation audit status | failed |

Runtime note: the `2,017 sec` duration is the measured runner time for the
original R8 hierarchy experiment. It does not include later manual analysis,
debugging, audit, or the MCP patch retry.

Section draft distribution:

| Metric | Value |
| --- | ---: |
| Minimum | 796 |
| Median | 1,100 |
| Mean | 1,108.7 |
| Maximum | 1,535 |

Part preservation:

| Part | Part words | Section words | Preservation ratio |
| --- | ---: | ---: | ---: |
| `P01-CQ1` | 7,307 | 8,875 | 0.823 |
| `P02-CQ2` | 6,453 | 10,901 | 0.592 |
| `P03-CQ3` | 9,511 | 9,291 | 1.024 |
| `P04-CQ4` | 4,888 | 11,456 | 0.427 |
| `P05-CQ5` | 9,298 | 9,309 | 0.999 |
| `P06-CQ6` | 8,537 | 8,412 | 1.015 |
| `P07-CQ7` | 9,999 | 9,820 | 1.018 |
| `P08-CQ8` | 4,846 | 9,031 | 0.537 |

## Interpretation

R8 confirms that the section-first strategy can produce substantially richer
raw material. The 69 section drafts averaged about 1,109 words each, so the
first stage did not collapse into short answer-style prose.

The final report also avoided the worst R5/R7 failure mode. It is a real
long-form artifact at roughly 63k words, and the global step did not rewrite the
whole report into a short summary. This supports the mechanical-preservation
assembly idea: front matter and closing can be generated separately while the
body remains the Part manuscripts.

However, the Part preservation step is still unstable and failed the acceptance
check. Four Parts preserved around all section material (`P03`, `P05`, `P06`,
`P07`), while `P02`, `P04`, and `P08` compressed heavily. This means R8 is not
yet a finished product workflow. The remaining issue is no longer only "global
integration collapses everything"; it is also "some Part editors still rewrite
too aggressively."

## Product Lesson

For production report generation, the promising direction is:

- keep same investigation/generation session where product context matters;
- generate a visible long-form outline before writing;
- split the outline into Parts and Sections;
- draft Sections in parallel;
- preserve-edit locally;
- avoid whole-report rewrite passes that can collapse detail;
- use mechanical assembly plus small front/closing synthesis for the final step.

The next experiment should focus on the Part preservation boundary. Likely
variants:

- split large Parts into smaller section groups before Part assembly;
- make Part assembly append section text first, then ask the editor to add
  transitions in-place;
- add a hard acceptance check on `part_words / section_words`;
- rerun low-ratio Parts only instead of rerunning all sections.

Those checks should improve preservation without adding the kind of rigid
claim/evidence machinery that previously made reports thinner rather than
richer.

## MCP Patch Retry

The next retry tested exactly that preservation boundary for the failed Parts.
Instead of asking the model to return a new Part manuscript, the retry gave
Codex an experiment-only MCP surface that treated Section bodies as immutable
blocks. The agent could read Sections, add intro/transition/closing text, and
finalize the Part by mechanical assembly, but it could not rewrite Section
bodies.

See:
`runs/R8-part-mcp-patch-retry-2026-06-25/README.md`

Result: all three failed Parts passed. `P02` improved from `0.592` to `1.025`,
`P04` from `0.427` to `1.024`, and `P08` from `0.537` to `1.034`. The retry
final report grew from about 63k words to about 79k words because the previously
compressed Section detail was preserved.

This is the strongest evidence so far that the report problem is not lack of
stored structure or lack of source material. The failure mode is free-form
rewrite during assembly. A productized report workflow should therefore use MCP
document-editing tools with immutable blocks and explicit patch/diff operations,
then let the system assemble the final artifact.

Measured runner time for the retry was `91.91 sec`. The original R8 run plus
the successful retry runner time is therefore about `2,108.91 sec`, or about
`35 min 09 sec`, excluding manual debugging, review, and Sentinel audit time.

## Proven Structure

The experiment supports the following report-generation structure:

1. Keep investigation and report generation in the same agent session when the
   investigation conversation carries important context.
2. Use controller turns to steer investigation depth and breadth, not to write
   the final report.
3. Split the report into Parts and Sections before drafting.
4. Draft Sections as rich, durable body blocks.
5. Assemble Parts through an MCP editing surface that can add introductions,
   transitions, and closings around immutable Section bodies.
6. Let the system mechanically assemble the final artifact instead of asking
   the model to rewrite the whole report.

The validated boundary is narrow but important: when Section bodies were kept
immutable and the agent could only add connective tissue, the failed R8 Parts no
longer collapsed. This argues for MCP-based document editing surfaces over
larger prompts or more intermediate claim/evidence machinery for long-form
report preservation.
