# Decision

## Decision

Adopt an adaptive controller policy for Plasma C1 research. Do not adopt a fixed
winner among V0, V1, V2, and V3.

## Why

More seeds changed the conclusion from "V2/V3 look best" to "different question
moves are useful at different moments."

- V2-style creative reliability switching is valuable when the agent is stuck
  inside a local implementation mechanism.
- V3-style lifecycle divergence is valuable when the agent needs to connect
  implementation details to user-visible states and affordances.
- V0/V1-style confirmation remains valuable because the agent still needs a
  precise implementation map before broader questions produce useful answers.

The controller should therefore monitor the intermediate answer and select the
next question type. The product should encode a repertoire, not a single rigid
mode.

## Implementation Implication

The next Plasma controller prototype should:

1. Log each controller question and the reason for choosing it.
2. Classify intermediate answers for signs of over-narrowing, shallow breadth,
   unresolved boundary, or missing product implication.
3. Ask one short steering question at a time.
4. Let the main agent keep reading through MCP/tools.
5. Generate the final report only after the intermediate material has enough
   implementation grounding and product/lifecycle coverage.

## Confidence

Confidence is sufficient for product direction. It is not a statistical proof.
