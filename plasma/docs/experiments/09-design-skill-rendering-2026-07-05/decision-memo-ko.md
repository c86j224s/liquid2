# 외부 디자인 스킬 장점 흡수 결정 메모

Issue: [#19 외부 디자인 스킬 장점 흡수](https://github.com/c86j224s/liquid2/issues/19)

## 결정

제품 반영한다.

Plasma Designed HTML renderer는 첫 화면의 연결형 관계도는 유지하되, 이후 visual
unit을 모두 같은 관계도로 렌더링하지 않는다. content model의 `kind`를 정규화한 뒤
timeline/flow/decision/dependency ladder, evidence chain, trade-off matrix, loop,
relationship map 중 하나로 dispatch한다. renderer version은
`dh25-visual-grammar-20260705`, content model contract는 `dh25_visual_grammar`로
올린다.

## 외부 스킬에서 흡수한 점

- Anthropic `frontend-design`: 디자인을 장식이 아니라 brief, 독자, 목적에서
  도출한다. Plasma에서는 보고서의 정보 구조를 visual grammar 선택 기준으로
  번역했다.
- Taste Skill: anti-slop의 핵심을 "landing page처럼 보이게 하기"가 아니라
  반복 문법을 피하고 pre-flight gate를 두는 방식으로 흡수했다.
- Vercel `web-design-guidelines`: 결과물을 실제 렌더링 기준으로 점검한다는 원칙을
  DOM smoke와 static syntax check로 반영했다.
- Impeccable/DESIGN.md 계열: 디자인 기준을 코드 안 암묵 규칙으로만 두지 않고
  실험 프로토콜과 제품 문서에 남겼다.

참고한 공개 자료:

- <https://github.com/anthropics/skills/blob/main/skills/frontend-design/SKILL.md>
- <https://platform.claude.com/docs/en/agents-and-tools/agent-skills/overview>
- <https://github.com/Leonxlnx/taste-skill>
- <https://github.com/vercel-labs/agent-skills/tree/main/skills/web-design-guidelines>
- <https://github.com/pbakaus/impeccable>
- <https://github.com/VoltAgent/awesome-design-md>

## 제품 반영

- `plasma/internal/web/report_design_visuals.go`
  - visual `kind` 정규화
  - visual grammar label
  - timeline/flow/decision/dependency ladder
  - evidence chain
  - trade-off matrix
  - loop
  - relationship map fallback
- `plasma/internal/web/server.go`
  - renderer version bump
  - content-model prompt에 non-hero visual kind 선택 규칙 추가
  - visual unit renderer dispatch
  - renderer CSS에 새 visual grammar 스타일 추가
- `plasma/internal/web/static/app.js`
  - browser cache-state 계산용 renderer version 동기화
- 제품 문서
  - Designed HTML이 source가 아닌 추가 report artifact라는 경계를 유지하면서
    visual grammar dispatch가 현재 제품 경로에 들어왔음을 반영

## 검증 결과

- `go test ./internal/web`
- `go test ./...`
- `node --check internal/web/static/app.js`
- `git diff --check`
- archive-backed smoke experiment:
  - 3 representative generated HTML samples
  - desktop screenshots at 1440x1200
  - mobile screenshots at 390x1000
  - Chrome DevTools Protocol viewport metrics

추가 테스트:

- `TestDesignedReportVisualKindNormalization`
- `TestDesignedReportVisualUnitsDispatchGrammar`
- `TestDesignedReportHTMLDOMSmoke`

DOM smoke는 generated designed HTML을 파싱해 `hero-map-svg`,
`visual-timeline`, `visual-evidence-chain`, `visual-matrix`, `visual-loop`,
`sources-panel`이 존재하고 외부 script/link/iframe/http image auto-load가 없는지
확인한다.

실제 스크린샷 실험에서는 처음에 mobile viewport 문제가 드러났다. Chrome CLI
`--screenshot` 경로가 390px CSS viewport를 정확히 만들지 않아 잘린 PNG를 만들었고,
renderer도 mobile hero 영역에서 overflow 방어가 충분하지 않았다. 그래서 Chrome
DevTools Protocol로 viewport를 강제해 다시 캡처했고, renderer CSS에는 mobile에서 SVG
hero map을 숨기고 readable node list를 보여주는 규칙, 강한 텍스트 줄바꿈, max-width /
min-width guard를 추가했다. 최종 `viewport-metrics.json`에서는 세 샘플 모두 desktop과
mobile에서 `scrollWidth == clientWidth`로 확인했다.

## 남은 한계

이번 변경은 "기본 렌더링 능력"을 올리는 첫 반영이다. 첫 화면은 여전히 연결형
관계도이고, swimlane, cost ladder, 더 풍부한 decision route 같은 grammar는 아직
별도 renderer로 만들지 않았다. agent가 부정확한 `kind`를 고르면 renderer는 map으로
fallback하므로, 이후에는 content model 품질 평가와 screenshot 비교 실험을 더 넓혀야
한다.

## Artifact 경계

raw generated HTML, screenshots, judge packets, logs는 Git에 넣지 않는다. 필요한 경우
다음 archive root 아래에 둔다.

```text
~/research-artifacts/liquid2/plasma/experiments/09-design-skill-rendering-2026-07-05/
```
