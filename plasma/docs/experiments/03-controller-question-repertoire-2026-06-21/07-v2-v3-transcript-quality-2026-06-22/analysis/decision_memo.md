# Decision Memo

## Decision

The V2/V3 follow-up strengthens the adaptive-controller hypothesis.

The key result is that transcript-quality judging agrees with the prior
final-report-only judging on all three mission classes:

- M1 narrow-source: V2 wins.
- M3 broad-topic: V3 wins.
- M6 source-conflict: V3 wins.

This reduces the risk that the prior result was only a final-report-generation
artifact. The better variants also produced better auditable research
conversations.

## What This Means

The evidence now supports a small selection rule rather than a universal
controller default:

- Use V2 as the safer candidate for narrow missions. It gives the agent one
  creative switch when needed, then forces direction recovery.
- Use V3 as the stronger candidate for broad-topic and source-conflict missions,
  where repeated lens shifts can expose more product boundaries and unresolved
  tensions.

Do not promote V3 globally yet. V3 can be better, but it is also costlier and
more likely to produce long transcripts. It should be selected when the mission
needs breadth, conflict preservation, or repeated reframing.

## Evidence

| Mission | Class | Final-report-only winner | Transcript-quality winner | Main reason |
| --- | --- | --- | --- | --- |
| M1 | narrow-source | V2 | V2 | The conversation narrowed into artifact identity, recovery, discovery, download target, and provenance instead of only broad metadata. |
| M3 | broad-topic | V3 | V3 | The conversation moved through source map, ownership, automatic investigation handoff, UI-less MCP contract, and controller boundary. |
| M6 | source-conflict | V3 | V3 | The conversation preserved conflict instead of resolving too quickly, then recovered into current C1, read-only legacy, experiment, and future boundaries. |

## Product Boundary

This does not change the C1 product contract:

- The controller remains question-only.
- The controller does not create evidence, claims, confidence, source
  candidates, saved knowledge, report body, or citations.
- The main agent still reads sources through MCP/source-read tools.
- The controller's value is adaptive steering, not data modeling.

## Remaining Limits

- This follow-up reused existing clean runs instead of running new seeds.
- The judge score scales were not uniform across missions, so only qualitative
  winners and rationales should be used.
- Transcript packets still included some filesystem path noise from generated
  report citations. It did not reveal variant labels, but the next runner should
  sanitize these paths.
- More seeds are needed before treating the selection rule as stable.

## Next Step

The next implementation-shaped experiment should not add old ledger machinery.
It should add a controller selection surface that logs:

- mission class or observed state;
- selected question repertoire;
- whether the turn was continuation, creative switch, scheduled divergence, or
  recovery;
- why the controller chose that question.

That should be tested as a product slice only after one more replay with new
seeds confirms the V2-for-narrow and V3-for-broad/conflict pattern.
