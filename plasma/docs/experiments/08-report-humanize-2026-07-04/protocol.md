# Protocol

## Archive Setup

Use the local archive for raw and generated experiment material:

```bash
mkdir -p ~/research-artifacts/liquid2/plasma/experiments/08-report-humanize-2026-07-04/{samples,candidates,analysis}
```

Do not commit raw report samples, generated candidates, or unredacted review
notes.

## Sample Export

Samples may be exported from a local Plasma database, but the exported Markdown
files must remain outside Git.

Suggested sample names:

- `s1-short-report.md`
- `s2-long-report.md`
- `s3-section-or-part.md`

For each sample, record only the following public metadata in this directory:

- sample id;
- mission title or redacted mission category;
- artifact kind;
- byte size;
- source database path class, for example `dev-6002`, not a private absolute
  path;
- reason it was selected.

## Humanize Profile

Use the `im-not-ai` material as a style reference, but apply these Plasma
constraints:

1. Preserve all Markdown headings exactly by level and order.
2. Preserve links, URLs, footnotes, code fences, tables, images, and source
   labels.
3. Preserve numbers, dates, model names, product names, citations, and quoted
   passages.
4. Preserve uncertainty level. Do not turn "may" into "is", or "is" into "may",
   unless the original report already warrants that change.
5. Prefer local sentence-level edits over outline-level rewrites.
6. Do not add new facts, examples, interpretations, or jokes.
7. Keep the genre as a report, not a column, essay, post, or marketing article.

Allowed edits:

- remove redundant connective phrases;
- vary repeated sentence endings;
- reduce stiff translationese;
- simplify awkward nominalizations;
- make Korean sentence rhythm more natural while preserving meaning.

Disallowed edits:

- changing report structure;
- merging sections;
- deleting caveats;
- hiding source limits;
- rewriting tables or code examples as prose;
- replacing source references with generic prose.

## Structure Gate

The original local run used a structure checker before reading candidates for
quality:

```bash
python3 <artifact-archive>/experiments/08-report-humanize-2026-07-04/tools/markdown_structure_check.py \
  --original ~/research-artifacts/liquid2/plasma/experiments/08-report-humanize-2026-07-04/samples/s1-short-report.md \
  --candidate ~/research-artifacts/liquid2/plasma/experiments/08-report-humanize-2026-07-04/candidates/s1-h1.md \
  --json-out ~/research-artifacts/liquid2/plasma/experiments/08-report-humanize-2026-07-04/analysis/s1-h1-structure.json
```

The checker and per-candidate JSON outputs are local experiment artifacts, not
part of the public source tree. Public readers should use the summarized
structure-gate outcome in this experiment directory rather than expecting to
rerun the archived helper from a fresh clone.

Any hard failure blocked product adoption for that variant in the original run.

The checker intentionally treats non-empty Markdown block count changes as a
hard failure. This catches a class of tone-pass errors where the rewrite keeps
headings, tables, code fences, links, and numbers intact but silently drops a
normal paragraph.

## Review Rubric

Score each candidate on a 1-5 scale:

- tone naturalness;
- report-genre fit;
- content fidelity;
- structure fidelity;
- source/citation safety;
- usefulness as a product post-pass.

The adoption decision should prefer conservative reliability over occasional
prettier prose.
