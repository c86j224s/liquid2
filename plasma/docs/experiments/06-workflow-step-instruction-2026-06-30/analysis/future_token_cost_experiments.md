# Future Token Cost Experiment Notes

Status: future experiment candidate. This note does not change the current
workflow step-instruction experiment design or interpretation.

## Observation

The current experiment keeps one Codex provider session alive across multiple
investigation turns. This preserves conversational continuity, but source reads,
tool outputs, agent answers, and intermediate reasoning context accumulate in
the resumed session. As a result, later turns can have much larger input-token
usage than their prompt size alone would suggest.

This is a product-relevant signal, but it is not the same question as the current
experiment. The current experiment compares step-instruction structure. Token
cost reduction should be tested separately so it does not become a hidden
confounder.

## Candidate Direction

Test a "ledger-recalled short session" workflow against a "long resumed session"
workflow.

The short-session workflow would not rely on the agent session history as the
primary memory surface. Instead, each turn would receive a compact mission
reminder and access to Plasma ledger/MCP tools with guidance such as:

- use the ledger to inspect what has already been done;
- use source-read tools to reopen original sources when needed;
- do not assume previous generated answers are sources;
- write any durable findings back to the ledger rather than relying on chat
  history alone.

The goal is to preserve investigation continuity while reducing token cost from
ever-growing provider-session history.

## Why Separate

This would change a major independent variable:

- memory surface: provider session history vs. Plasma ledger/MCP recall;
- context construction: implicit resumed context vs. explicit tool-mediated
  retrieval;
- failure mode: forgotten conversation history vs. missing or poor ledger reads;
- product implication: when to compact, reset, or fork agent sessions.

Because those variables are distinct from step-instruction quality, they should
not be mixed into the current experiment.

## Suggested Future A/B Shape

- A: long resumed provider session, current behavior.
- B: short-session turns with ledger/MCP recall and source random-seek tools.
- Optional C: periodic compacted session plus ledger/MCP recall.

Primary measurements:

- investigation depth and breadth;
- raw user intent preservation;
- source grounding quality;
- number of source/ledger reads;
- per-turn and total input/output token usage;
- wall-clock runtime;
- whether the final report preserves useful details from earlier turns.

The important product question is not only whether token usage falls, but
whether the agent can still recover enough context from Plasma's ledger and
source tools to continue the same mission without becoming shallow or repetitive.
