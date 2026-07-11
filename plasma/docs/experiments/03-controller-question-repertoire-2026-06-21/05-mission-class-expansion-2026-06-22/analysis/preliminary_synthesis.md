# Mission-Class Expansion Synthesis

## What Changed

This run expanded the controller-question repertoire experiment beyond the
earlier M5 code-analysis slice. It tested three additional mission classes:

- M1 narrow-source.
- M3 broad-topic.
- M6 source-conflict.

Each mission ran V0, V1, V2, and V3 under the same source corpus for that
mission. The first attempt was discarded because resumed turns ran from the
repository root and some agents read outside the intended source corpus. The
clean attempt launched resumed turns from the source corpus directory.

## Results

| Mission | Class | Winner | Variant | Summary |
| --- | --- | --- | --- | --- |
| M1 | narrow-source | K1 | V2 | Best concrete artifact/recovery/trust risk report from a narrow corpus. |
| M3 | broad-topic | K2 | V3 | Best multi-angle product-flow and UI-less research IDE report. |
| M6 | source-conflict | K4 | V3 | Best product-boundary report across C1 default, legacy read-only, experiment-only, and future investigation. |

## Contamination Handling

The runner marked M1-V0, M3-V0, M3-V1, and M3-V2 with
`final_report_references_outside_source_catalog`. Manual review found these to
be conservative filter hits rather than material contamination:

- Some reports mentioned documents only to say they were not inspected.
- Some reports mentioned `product-flow.md` and `automatic-investigation.md`,
  which were valid M3 source-corpus files.
- Tool traces did not show repo-root source reads in the reviewed commands.

The raw flags remain in manifests. The synthesis does not treat them as
disqualifying contamination.

## Adaptive Hypothesis

The adaptive hypothesis is stronger after this run. V0 did not win any of the
three mission classes, and the winning variants were adaptive:

- V2 won M1.
- V3 won M3 and M6.

The evidence does not yet justify one universal controller default. It supports
a smaller selection-rule experiment: decide when to use V2's one-time creative
switch plus recovery, and when to use V3's scheduled divergence.

## Next Action

Do not add evidence/claim/confidence/source-candidate machinery back into the
product. Keep C1 read-first and question-only.

Next, run a V2/V3-focused follow-up with transcript-quality judging. The product
candidate should be a controller that logs which question repertoire it selected
and why, while leaving source reads and report writing to the main agent.
