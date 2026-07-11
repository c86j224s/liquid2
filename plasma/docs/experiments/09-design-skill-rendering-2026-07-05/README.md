# Design Skill Rendering Experiment

Issue: [#19 외부 디자인 스킬 장점 흡수](https://github.com/c86j224s/liquid2/issues/19)

This experiment translates external frontend design skill patterns into Plasma's
designed HTML report renderer. It does not copy external skill text or assets.
The product target is better deterministic rendering of an existing Markdown
report artifact as an additional designed HTML report artifact.

Decision memo: [`decision-memo-ko.md`](decision-memo-ko.md)

## Product Hypothesis

Plasma designed HTML quality improves when the renderer uses the report's
information shape to choose a visual grammar instead of rendering every visual
unit as the same relationship map.

The first productized slice is intentionally narrow:

- keep Markdown reports and basic HTML exports unchanged
- keep designed HTML as a derived report artifact, not a source
- keep the agent responsible for a JSON content model only
- keep final HTML deterministic, self-contained, and cache-versioned
- add renderer-side visual grammar dispatch for non-hero visual units

## External Skill Translation

The useful pattern across the reviewed skills is workflow discipline, not a
particular aesthetic.

| External pattern | Plasma translation |
| --- | --- |
| Read the brief before choosing style | infer `visual_identity`, `composition_shape`, and visual `kind` from the report artifact |
| Avoid template-like UI | use multiple report visual grammars instead of one repeated relationship map |
| Use explicit design tokens and quality gates | keep renderer CSS tokenized and test for emitted grammar classes |
| Iterate against rendered output | require DOM or screenshot smoke before treating renderer changes as product-ready |
| Separate creator and reviewer | route implementation through Sentinel after local tests |
| Preserve product context | prioritize evidence visibility, citation preservation, long-text handling, and mobile safety over marketing-style hero treatment |

Primary references:

- Anthropic `frontend-design` skill:
  <https://github.com/anthropics/skills/blob/main/skills/frontend-design/SKILL.md>
- Anthropic Agent Skills overview:
  <https://platform.claude.com/docs/en/agents-and-tools/agent-skills/overview>
- Taste Skill / `design-taste-frontend`:
  <https://github.com/Leonxlnx/taste-skill>
- Vercel `web-design-guidelines` skill:
  <https://github.com/vercel-labs/agent-skills/tree/main/skills/web-design-guidelines>
- Impeccable:
  <https://github.com/pbakaus/impeccable>
- DESIGN.md ecosystem:
  <https://github.com/VoltAgent/awesome-design-md>

## Productization Protocol

1. Update the content-model prompt so visual units ask for a precise grammar:
   timeline, evidence chain, dependency path, trade-off matrix, decision route,
   loop, or relationship map.
2. Update deterministic rendering so non-hero visual units dispatch by `kind`.
3. Keep a relationship-map fallback for unknown or thin kinds.
4. Bump the designed HTML renderer version so stale cached artifacts are not
   reused.
5. Store only this public protocol and small redacted fixtures in Git. Store raw
   generated HTML, screenshots, and judge packets under:

```text
~/research-artifacts/liquid2/plasma/experiments/09-design-skill-rendering-2026-07-05/
```

## Acceptance Gates

- The rendered HTML remains self-contained and uses no external scripts, fonts,
  images, iframes, or fetches.
- The Markdown source artifact remains the source report artifact; designed HTML
  remains an additional report artifact.
- Source notes, URLs, caveats, tables, and uncertainty text are preserved.
- The renderer emits at least three non-map visual grammar classes in focused
  tests.
- The browser UI recognizes the new renderer version for cache state.
- `go test ./internal/web` passes.
- Static JS syntax check passes when browser static files change.
- `git diff --check` passes.

## Decision Rule

Adopt the renderer change if it improves visual variety without weakening
source traceability or mobile safety. If it fails, keep the protocol and leave
the current DH23 relationship-map renderer as the product default while moving
visual-grammar dispatch back into the experiment backlog.

## Result

Adopted for the product renderer.

The 2026-07-05 slice keeps the first-viewport connected relationship map, then
dispatches later visual units to deterministic timeline/flow/decision/dependency
ladder, evidence-chain, trade-off matrix, loop, or relationship-map renderers.
The renderer version and content-model contract were bumped so existing cached
designed HTML artifacts do not mask the changed behavior.

After initial implementation, a real archive-backed smoke experiment caught a
mobile screenshot problem in the capture/rendering path: the CLI screenshot path
was not enforcing a 390px CSS viewport, and the renderer also needed stronger
mobile overflow guards. The adopted renderer now hides the SVG hero map on
mobile and shows the readable node list, applies mobile text wrapping, and keeps
the page scroll width equal to the viewport in the checked samples.

Archive outputs, outside Git:

- `manifest.json`
- `viewport-metrics.json`
- `renders/{engineering-dependency-path,research-evidence-digest,decision-route}.html`
- `screenshots/*-{desktop,mobile}.png`

Checked samples:

| Sample | Desktop | Mobile | Result |
| --- | --- | --- | --- |
| `engineering-dependency-path` | 1440x1200 | 390x1000 | pass |
| `research-evidence-digest` | 1440x1200 | 390x1000 | pass |
| `decision-route` | 1440x1200 | 390x1000 | pass |

For all three samples, Chrome DevTools Protocol capture reported
`scrollWidth == clientWidth` in both desktop and mobile viewports.
