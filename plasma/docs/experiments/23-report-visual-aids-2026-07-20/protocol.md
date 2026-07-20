# Protocol

## Scope

This experiment uses the product report-generation path with isolated Plasma
servers, isolated SQLite databases, isolated provider work directories, and a
local Liquid2 stub. It does not run against the development or release servers.

The source fixture lock is copied from:

`research-artifacts/liquid2/plasma/experiments/17-report-plan-mcp-focused-2026-07-14/fixtures.lock.json`

The public runner is:

`plasma/scripts/experiments/report_visual_aids_experiment.py`

## Arms

| Arm | `generation_guidance_profile` | Product-path effect |
| --- | --- | --- |
| `baseline` | `g2` | Current G2 report writing guidance. |
| `visual_supplement` | `visual-supplement` | Writing step receives weak visual-aid guidance. |
| `visual_plan` | `visual-plan` | Planning and writing steps receive visual-aid intent guidance. |

The plan schema stays unchanged. Planned visual-aid intent must be written into
existing `purpose` or `coverage_notes` text, not into new JSON fields.

## Modes

Run both current Markdown report modes:

- `planned`: normal planned Markdown report path.
- `long_form`: long-form Markdown report path. The default runner strategy is
  `section_fanout`, because it is the current fast long-form option and exposes
  section-level writing behavior. Use `--long-form-strategy serial` for a slower
  follow-up if needed.

## Execution

Prepare the isolated binary and fixture lock:

```bash
python3 plasma/scripts/experiments/report_visual_aids_experiment.py --action prepare
```

Smoke a small set first:

```bash
python3 plasma/scripts/experiments/report_visual_aids_experiment.py --action run --limit 2 --workers 2
python3 plasma/scripts/experiments/report_visual_aids_experiment.py --action analyze
python3 plasma/scripts/experiments/report_visual_aids_experiment.py --action packets
```

If smoke passes, expand the topic count:

```bash
python3 plasma/scripts/experiments/report_visual_aids_experiment.py --action run --limit 6 --workers 2
python3 plasma/scripts/experiments/report_visual_aids_experiment.py --action analyze
python3 plasma/scripts/experiments/report_visual_aids_experiment.py --action packets
```

The full 24-topic fixture set is available when the early run shows no
operational failure.

## Evaluation

Automatic observation signals:

- terminal success rate;
- report word count ratio against baseline;
- table count;
- Mermaid fence count;
- visual aids per 1,000 words;
- Mermaid validation mention signal;
- unvalidated-Mermaid warning signal.

Human reading criteria:

- Does the visual aid answer a reader question?
- Does it supplement, rather than replace, source-grounded prose?
- Does the nearby prose introduce and interpret the visual aid?
- Does the report remain natural Korean article prose?
- Are the visual aids sparse and useful, not decorative?

## Failure Handling

Do not use mid-run failures as a quality gate for this experiment. A failed run
is recorded as an operational failure, then excluded from whole-report quality
pairs that require completed baseline and candidate artifacts.

Do not productize from this experiment if the completed reports show:

- any candidate frequently generates Mermaid that cannot validate or render;
- visual aids repeat adjacent prose without adding structure;
- reports become shorter or thinner by replacing explanation with visuals;
- planning starts forcing visual aids into topics that are clearer as prose.
