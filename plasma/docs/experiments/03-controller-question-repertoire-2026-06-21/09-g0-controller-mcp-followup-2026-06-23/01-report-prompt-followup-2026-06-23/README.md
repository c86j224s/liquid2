# Report Prompt Follow-up Experiment

Status: full official Codex fork batch completed and judged on 2026-06-23.

This follow-up tests final report generation prompts after the completed G0
investigation experiments. It does not rerun the investigation phase and does
not mutate the original G0 run directories.

## Boundary

The default target is the completed `AUTO` controller runs and `R1` random-seek
runs, because those are the investigation surfaces currently worth carrying
forward.

The experiment compares final-report prompt variants:

- `F0`: baseline wording close to the prior final-report prompt.
- `F1`: permits previous agent answers and controller questions as working
  notes, but not as sources.
- `F2`: asks the agent to internally synthesize facts, interpretations,
  hypotheses, conflicts, and structure before writing.
- `F3`: asks for a richer report/article while labeling weak signals,
  interpretation, and uncertainty.
- `F4`: combines the useful parts of `F1`, `F2`, and `F3`: working-memory reuse,
  silent synthesis planning, rich report/article writing, uncertainty labels,
  and explicit suppression of internal experiment labels or paths.

Original sources remain the source snapshots listed in the source catalog. The
investigation transcript is a working-memory result, not a source.

## Parallel Execution

Dry-run the planned report generation:

```bash
python3 plasma/scripts/experiments/g0-controller-mcp-followup/report_prompt_experiment.py \
  run --limit 2 --variants F0,F1 --dry-run
```

Run transcript/source-copy report generation in parallel:

```bash
python3 plasma/scripts/experiments/g0-controller-mcp-followup/report_prompt_experiment.py \
  run --jobs 4 --variants F0,F1,F2,F3
```

Run official Codex session-fork report generation in parallel:

```bash
python3 plasma/scripts/experiments/g0-controller-mcp-followup/report_prompt_experiment.py \
  run --mode fork --jobs 4 --variants F0,F1,F2,F3
```

In `fork` mode, variants for the same source session are generated
sequentially. Parallelism is applied across different source sessions only,
because launching multiple TUI forks from the same provider session at the same
time can stall before a forked session file is created.

Judge generated reports in parallel:

```bash
python3 plasma/scripts/experiments/g0-controller-mcp-followup/report_prompt_experiment.py \
  judge --jobs 4
```

Outputs are written under:

```text
plasma/docs/experiments/03-controller-question-repertoire-2026-06-21/09-g0-controller-mcp-followup-2026-06-23/01-report-prompt-followup-2026-06-23/
```

## 2026-06-23 Fork Result

The full fork run generated 72 reports: 18 completed investigation sessions
times four final-report prompt variants. The runner recorded 8 cached validation
reports and 64 newly generated reports. All 72 rows passed the hard-fail audit.

The judge run produced `analysis/report_prompt_score_matrix.csv` with 72 judged
rows. Average composite scores were close:

| variant | n | composite | note |
| --- | ---: | ---: | --- |
| `F0` | 18 | 0.935 | strongest paired-win count by the current composite formula |
| `F1` | 18 | 0.949 | highest overall composite after leakage penalties |
| `F2` | 18 | 0.945 | close to `F1`, especially strong on random-seek runs |
| `F3` | 18 | 0.928 | richest content signal, but two rows leaked experiment/path-like labels |

If leakage penalties are excluded, `F3` has the highest average composite
(`0.949`). The interpretation is therefore not "choose `F3` as-is"; it is that
rich-report guidance helps, but product prompts must still suppress experiment
labels, temporary paths, and internal working-directory language.

## Product Carry-forward Note

When this result is reflected in Plasma product work, carry forward the tested
principles rather than the experiment harness itself:

- Generate the final report in the same agent session that performed the
  investigation, unless a later experiment proves a better product path.
- Prefer rich report/article guidance over thin summary guidance.
- Treat prior agent answers and controller questions as working notes, not
  sources.
- Keep source, result, saved knowledge, and report roles separate.
- Allow weak signals, interpretation, and uncertainty when clearly labeled.
- Suppress experiment labels, internal IDs, temporary paths, and working
  directory language in product-facing output.
- Do not reintroduce claim/confidence/AST/evidence-ledger structures as the
  core path for report quality unless they are revalidated separately.

## 2026-06-24 F4 Fork Follow-up

The follow-up added `F4` and ran it on the same 18 fork-mode source sessions
used by the 2026-06-23 comparison. All 18 reports passed the hard-fail audit.
The run index now contains 90 rows: the original 72 `F0`-`F3` rows plus 18 `F4`
rows.

Using the normalized local composite summary across fork-mode rows, `F4` was the
strongest candidate:

| variant | n | composite | leakage |
| --- | ---: | ---: | ---: |
| `F0` | 18 | 0.964 | 0.000 |
| `F1` | 18 | 0.972 | 0.000 |
| `F2` | 18 | 0.970 | 0.000 |
| `F3` | 18 | 0.967 | 0.111 |
| `F4` | 18 | 0.977 | 0.000 |

Paired composite comparison against `F4`:

| compared variant | n | F4 mean diff | F4 wins | losses | ties |
| --- | ---: | ---: | ---: | ---: | ---: |
| `F0` | 18 | +0.0127 | 11 | 6 | 1 |
| `F1` | 18 | +0.0044 | 11 | 7 | 0 |
| `F2` | 18 | +0.0065 | 11 | 7 | 0 |
| `F3` | 18 | +0.0093 | 10 | 5 | 3 |

Interpretation: `F4` is a better product candidate than `F1`/`F2`/`F3` as-is,
because it kept the richness signal without the leakage failures seen in `F3`.
The margin over `F1` and `F2` is modest, so it should be treated as the next
default to try in product, not as a final proof that the report problem is fully
solved.

Average report length also moved in the desired direction: `F4` averaged 1,082
words, compared with `F1` at 777, `F2` at 823, and `F3` at 1,417. That means
`F4` increased depth and breadth over the stable prompts without becoming as
verbose or leakage-prone as `F3`.

## Session Resume Constraint

Codex exposes both `codex exec resume` and `codex fork`. `resume` continues the
same session, so sending multiple final-report prompt variants to the same
provider session would contaminate the session history. `fork` creates a new
thread that preserves the original transcript and can be used for product-shaped
A/B report generation.

The harness still uses `transcript` mode by default because it is simple and
non-interactive. It copies the completed investigation transcript and original
source snapshots into a fresh workspace for each variant, which is safe to
parallelize.

`resume` mode exists for product-shaped spot checks, but it allows only one
variant per source run:

```bash
python3 plasma/scripts/experiments/g0-controller-mcp-followup/report_prompt_experiment.py \
  run --mode resume --variants F2 --jobs 2 --limit 4
```

Use `resume` mode to check whether the same-session product path behaves
similarly, not to run a full prompt A/B over the same session.

`fork` mode uses the official TUI command, so the harness runs it in a
controlling PTY, answers minimal terminal capability queries and the directory
trust prompt when needed, waits for the forked session's final answer to appear in
`~/.codex/sessions`, then records that answer as the candidate report. The
forked session id is stored as
`fork_provider_session_id.txt` for traceability. The PTY transcript is kept as a
bounded tail log in `final-events.txt` so large TUI redraws do not dominate
experiment artifacts.

## Hard-fail Checks

Each generated report is checked for:

- missing final report output
- leaked experiment labels or prompt variant labels
- leaked repository, run, or temporary paths
- mutation of copied source snapshots

Judge packets include the original source snapshots, the investigation
transcript as non-source working context, and the candidate report.

## Planned Composition Follow-up

The next planned report experiment is the 2026-06-25 composition follow-up:
[`../05-report-composition-followup-2026-06-25/`](../05-report-composition-followup-2026-06-25/).
It crosses three composition strategies with two writing surfaces. The
composition strategies are `F4` single-pass writing, the productized
visible-plan-then-draft path, and a `book-writer`-inspired sectional composition
loop. The writing surfaces are final chat response and MCP artifact writing.
That creates six variants, `R0` through `R5`. The purpose is to test whether
visible planning improves coverage, whether it makes reports rigid, whether
section-level drafting can recover detail, and whether MCP artifact writing
reduces answer-like brevity without adding too much tool friction.
