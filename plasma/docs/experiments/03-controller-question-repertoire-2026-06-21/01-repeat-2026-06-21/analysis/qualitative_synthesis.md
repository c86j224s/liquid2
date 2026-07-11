# Qualitative Synthesis

## Result

Seed 0002 produced visible controller separation. The variants did not collapse
into identical final reports:

- V0 gave the most direct implementation map.
- V1 gave the cleanest classification of product gaps versus test-only gaps.
- V2 produced the strongest reliability framing after the creative switch.
- V3 produced the strongest user artifact lifecycle framing after scheduled
  divergence.

This supports keeping a controller-led C1 research loop, but it does not prove a
single best controller policy. Only one repeat seed was executed.

## Product Findings Surfaced by the Run

The repeat surfaced the same core product holes from different angles:

1. C1 Markdown report artifact creation is not atomic. The code saves the raw
   artifact and then appends `report.artifact.created`; failure between the two
   can leave an undiscoverable artifact.
2. Report failure events are thin compared with successful report events and
   normal agent turn failures. They do not carry enough stage, duration, session,
   log, or tool-trace context for easy diagnosis.
3. `Markdown 보기` currently shows raw Markdown in a `<pre>`, not a rendered
   document. The label and UI contract need a decision.
4. Korean report titles can collapse to `source.md` through `safeFilename`.
5. Report artifact cards do not show the eventual download filename.
6. Server read/download routes are tested, but browser-level button, filename,
   preview, and card affordance behavior is mostly untested.

## Controller Behavior

V0 and V1 remained useful, but mostly stayed inside the code/test frame. V2 and
V3 made the analysis more product-shaped:

- V2 changed the atomicity issue from "orphan row" into "the product appears to
  lose a report the user just generated." This is the more useful framing for
  prioritization.
- V3 made the artifact lifecycle explicit: create, wait, refresh, discover, view,
  download, and find later. This made UI contract gaps easier to see.

The controller should not always force divergence. The useful behavior in this
run was conditional: first let the agent build a precise map, then shift
perspective once the analysis starts to over-concentrate on one local mechanism.

## Follow-up for Plasma

The next product wave should fix reliability before polishing the browser UI:

1. Move Markdown artifact creation into an app-service atomic write path.
2. Strengthen `report.draft.failed` payloads with stage, duration, agent session,
   tool session, and log excerpt where available.
3. Decide the `Markdown 보기` contract: raw source view or rendered document view.
4. Fix Korean filename handling or use an explicit report artifact filename
   policy.
5. Add browser-level tests for artifact cards, view/download buttons,
   `Content-Disposition`, raw-vs-rendered preview, and failure state rendering.
