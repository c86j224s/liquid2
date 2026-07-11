# V2/V3 Transcript-Quality Follow-Up - 2026-06-22

This follow-up uses the clean mission-class expansion transcripts to compare V2 and V3.

It does not run new main-agent research turns. It creates blind transcript-quality judge packets from existing V2/V3 runs, evaluates whether the conversation flow was auditable and useful, and compares that with the prior final-report-only result.

Source experiment: `../05-mission-class-expansion-2026-06-22/`

## Result Summary

The transcript-quality result matched the prior final-report-only result for
all three missions:

| Mission | Class | Final-report-only winner | Transcript-quality winner | Decision |
| --- | --- | --- | --- | --- |
| M1 | narrow-source | V2 | V2 | V2 remains the safer narrow-source candidate. |
| M3 | broad-topic | V3 | V3 | V3 remains stronger for broad product-flow exploration. |
| M6 | source-conflict | V3 | V3 | V3 remains stronger for conflict-preserving exploration. |

This makes the adaptive-controller signal stronger. The better reports were not
only final-generation artifacts; the transcript judge also saw better steering
flow in the same winning variants.

The result still does not justify one universal default. It supports a
selection rule: prefer V2 when the mission is narrow or when a single creative
switch plus recovery is enough; consider V3 when the work needs repeated lens
changes across broad-topic or source-conflict analysis.

## Audit Notes

- Judge packets hide `V2`, `V3`, and run IDs.
- Some transcript content exposes temporary filesystem paths from the original
  generated reports. Judges flagged this as sanitation noise, not as a hidden
  variant clue.
- The evaluation score scales were inconsistent across missions, so numeric
  scores are not aggregated. The qualitative winner and written rationale are
  the decision input.
