# Plasma Glossary

This glossary explains Plasma product terms and experiment code names. Use it to
avoid reusing old shorthand in a way that makes current behavior ambiguous.

## Product Terms

| Term | Meaning |
|---|---|
| Mission | The durable research workspace for one topic, goal, or question. A mission owns conversation, source events, workflow events, and report artifacts. |
| Mission ledger | The append-only event record for a mission. Conversation turns, source lifecycle events, MCP calls, workflow events, and report events are all ledger producers. |
| Connector | An adapter for reaching an external origin, such as Liquid2, Confluence, or a future settings-managed local filesystem root. |
| Source | Original research material attached to a mission, such as a URL, PDF, uploaded file, Liquid2 document, Confluence page, media URL, or local path file/directory. |
| Source candidate | A source-like item proposed for user review. It is not an accepted source until the user approves it. |
| Staged source candidate | A source candidate whose fetch or extraction succeeded enough to create a candidate-only raw artifact. Agents may read it through candidate-read tools, but it is still not an accepted source. |
| Raw artifact | Stored content, extracted text, uploaded bytes, generated report Markdown, or internal rendering material. Raw artifacts are storage objects, not automatically sources. |
| Source snapshot | The accepted mission-level source record. It points to raw artifacts or to a live reference locator, depending on retrieval policy. |
| Live reference | A source policy for mutable original material, currently used for accepted local path sources. Plasma stores a locator and records observations instead of pinning bytes. |
| Evidence | A specific cited part of a source. Evidence is not part of the current C1 default loop, but may return as a non-gating reference/index layer. |
| Claim | A statement or interpretation that may be supported by evidence. Claim records are legacy/future design material, not current default workflow state. |
| Result | Agent-produced output such as an answer, comparison, summary, intermediate conclusion, or draft. Results may refer to sources, but they are not sources. |
| Saved knowledge | Deliberately retained mission knowledge. Current C1 keeps this lightweight and does not revive the old claim/evidence gate by default. |
| Report | A Plasma-owned output artifact assembled from mission work. Markdown is the primary report artifact; HTML is a rendering/export. |
| Designed HTML | A self-contained interactive HTML report export produced from a Markdown report through a JSON content model and deterministic renderer. |
| MCP research surface | The UI-less tool surface that lets an agent inspect mission state, search/read sources, and traverse references without receiving a huge prompt pack. |

## Experiment Code Names

| Code | Meaning | Current Product Status |
|---|---|---|
| C1 | The current default Plasma product loop: mission, same provider session, user/controller steering, MCP/source reads, conversation results, and report artifacts. | Active product direction. |
| C0 | Neutral controller baseline from controller experiments. It is closest to continuing the same session without strong controller intervention. | Used as evidence for conservative controller defaults. |
| PAL2 | A rhythm-aware question controller variant tested against C0 and NAV. | Inconclusive; not a default. |
| NAV | An investigation navigator variant that stated status, next-turn intent, and direction more strongly. | Rejected as a default after experiments. |
| G2 | Generation-time report tone guidance that improved Korean report style when applied during report writing. | Productized as default report guidance direction. |
| H5 | Post-report Korean humanization pass that patches an existing Markdown report through bounded MCP patch tools. | Optional/secondary post-processing; not part of planning or source selection. |
| DH23 | Designed HTML experiment path using an agent-authored JSON content model and deterministic renderer with a strong first-viewport visual unit. | Current designed HTML product candidate, with known limitations. |
| C4 | Long-form report assembly strategy that preserves section bodies and performs limited heading normalization instead of rewriting the whole report. | Productized for long-form report assembly. |
| F4 | Report-writing guidance carried forward from experiments: use prior investigation as working memory, synthesize privately, then write a rich Markdown report without leaking internal run details. | Productized as default Markdown report style guidance. |
| R-series | Report-generation experiment variants. Exact meanings are local to each experiment document. | Historical experiment records; read local protocol before applying. |
| M-series | Mission/corpus variants used in controller and workflow experiments. | Historical experiment records; read local protocol before applying. |
| DH-series | Designed HTML rendering experiment variants. | Historical experiment records; DH23 is the main carried-forward variant. |

## Current-Vs-Legacy Rule

If a document says a feature creates evidence, claims, confidence updates,
proposal bundles, AST reports, or report blocks, check whether that document is
describing the current C1 path or the legacy ledger loop.

The current default rule is:

- sources are original material
- results are agent output
- reports are output artifacts
- evidence/claim may be useful later, but must not gate investigation or report
  generation

## Writing Rule

When adding product documentation:

1. Name the current behavior first.
2. Put historical behavior in a `Historical note` or `Legacy note` callout.
3. Link experiment code names to this glossary or define them locally.
4. Do not convert agent results into sources.
5. Do not imply that a future backlog item is already the current product path.
