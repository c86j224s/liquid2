# Plasma Evidence Signal Model

상태: 이 문서는 현재 C1 기본 제품 루프가 아니라 future/legacy design note입니다. 현재 기본 루프는 source
중심이며 evidence, claim, confidence update, proposal record를 만들지 않습니다. 이 모델을 다시 살리더라도
source 위의 reference, index, traceability 계층이어야 합니다. 조사나 보고서 생성을 허가하거나 차단하는 gate가
되어서는 안 됩니다.

## 원칙

Plasma는 어떤 정보가 확정된 사실이 아니라는 이유만으로 유용한 연구 재료를 막으면 안 됩니다. Rumor,
reaction, interpretation, community discussion, code example, formula, benchmark, conflicting claim은 모두
유용한 research signal이 될 수 있습니다.

핵심 제어 지점은 “이 signal이 미션에 들어와도 되는가”가 아닙니다. 중요한 것은 signal을 얼마나 명확하게
label하고, source와 연결하고, confidence, limit, report-use value를 보여줄 수 있느냐입니다.

Agent result는 source가 아닙니다. Result가 signal을 식별하거나 요약할 수는 있지만, 저장된 evidence는
original source snapshot, user assertion, code location, formula source처럼 provenance가 분명한 대상을
가리켜야 합니다.

Source 하나에서 여러 evidence record가 나올 수 있습니다. Evidence record는 source identity를 바꾸지 않고
추가, 편집, 제거, supersede될 수 있어야 합니다.

## Signal Kinds

Evidence가 항상 엄격한 fact만 표현해야 하는 것은 아닙니다.

- `fact`: 날짜, 스펙, 출연진, API behavior, release metadata처럼 source가 직접 말한 사실.
- `interpretation`: analyst, critic, author, agent의 해석. 유용할 수는 있지만, 그 자체가 검증된 사실은
  아닙니다.
- `reaction`: community, market, press, audience response.
- `rumor`: 확인되지 않은 보도, leak, speculation, circulating claim.
- `controversy`: recurring disagreement, backlash axis, contested framing.
- `market_signal`: presales, traffic, view count, adoption signal, ranking 같은 attention indicator.
- `code`: source code, example code, test, snippet, API usage pattern, implementation detail.
- `formula`: mathematical expression, model, algorithmic equation, derived calculation.
- `benchmark`: measured performance result, comparison, experiment, reproducibility note.
- `open_question`: gap, unresolved contradiction, missing verification target.

이 kind는 approval 여부를 결정하지 않습니다. Display, confidence 영향, report wording을 돕는 metadata입니다.

## Confidence와 Usefulness

Plasma는 두 판단을 분리해야 합니다.

- Confidence: signal이 세계에 대한 진술로서 얼마나 믿을 만한가.
- Report value: signal이 약하거나 논쟁적이거나 추측이라도, 미션을 설명하는 데 얼마나 유용한가.

Rumor는 confidence가 낮아도 public reaction을 다루는 미션에서는 report value가 높을 수 있습니다. Code example은
“이 repository는 API를 이렇게 쓴다”는 면에서는 confidence가 높습니다. 하지만 version, runtime, license
constraint가 불분명하면 portability는 낮을 수 있습니다.

보고서는 이 차이를 보존해야 합니다. 약하거나 충돌하는 signal을 포함할 수는 있지만, 그것을 확정된 사실처럼
평평하게 만들면 안 됩니다.

예시:

> Casting은 primary source로 확인되지 않았지만, 여러 secondary source가 controversy axis로 다루기 때문에
> reaction signal로 중요하다.

## Rigor Levels

Strictness는 report-generation control입니다. Research-domain taxonomy가 아닙니다. 즉, 유용한 material이
미션에 들어올 수 있는지를 결정하는 장치가 아니라, 보고서가 evidence를 어떻게 사용하고 가중하고 표현할지
조정하는 장치입니다.

초기 수준은 다음과 같습니다.

- `exploratory`: 넓은 signal을 수집하고 사용합니다. Rumor, reaction, interpretation, controversy,
  market signal, code example, formula, benchmark, open question이 명확히 label된다면 보고서를 풍부하게
  만들 수 있습니다. 약한 material은 약한 material로 보이게 남겨야 합니다.
- `balanced`: main narrative는 source-backed fact와 medium/high confidence claim에 둡니다. 약하거나
  해석적인 signal은 이해를 개선할 때 context, competing account, unresolved question, explanatory color로
  사용합니다.
- `strict`: major conclusion은 source-backed fact와 medium/high confidence evidence에 둡니다. 약한 signal은
  uncertainty, background discourse, risk, coverage gap으로 명시될 때만 사용합니다.

이 수준은 saved evidence 사용 방식에 영향을 주고, collection breadth에는 간접적으로만 영향을 줍니다. 유용한
signal을 조용히 버리거나 approval hurdle을 높이면 안 됩니다.

## Code Evidence

Code evidence에는 단순 텍스트 이상의 metadata가 필요합니다.

- language와 framework
- source type: official docs, GitHub repository, issue, PR, Stack Overflow, blog, local code 등
- repository, commit, path, line range, URL, package version
- role: API example, production implementation, test, workaround, anti-pattern, migration note 등
- execution status: runnable, partial snippet, pseudocode, illustrative
- runtime과 dependency constraint
- license 또는 copyright risk
- portability caveat

보고서는 code를 pattern과 constraint의 evidence로 인용해야 합니다. Universal truth처럼 쓰면 안 됩니다. 긴 code는
license와 quoting limit이 허용되지 않으면 그대로 복사하지 말고, 짧은 excerpt, link, explanation을 선호합니다.

## Formula Evidence

Formula evidence에는 다음 정보가 필요합니다.

- LaTeX, MathML, plain text 형식의 원래 formula
- variable definition
- unit
- assumption과 boundary condition
- derivation source
- example calculation
- applicability limit
- confidence와 verification status

보고서는 formula의 assumption을 함께 설명해야 합니다. 설명 모델로만 쓰는 formula라면 그렇게 label해야 합니다.

## Report Behavior

보고서 생성은 더 풍부해져야 하지, 더 느슨해지면 안 됩니다.

보고서는 다음을 해야 합니다.

- confirmed fact, interpretation, reaction, rumor, conflict가 미션에 유용하면 포함합니다.
- prose 또는 collapsible metadata에서 signal kind와 confidence를 label합니다.
- competing account를 너무 빨리 하나로 해결하지 말고 competing account로 보여줍니다.
- 중요한 gap이 있으면 missing-information과 coverage-gap section을 둡니다.
- high-value weak signal을 fact로 승격하지 않고 표시합니다.
- Markdown report artifact를 C1 기본 output으로 유지합니다. AST report record는 나중에 제품 결정이 바뀌기
  전까지 legacy 또는 explicit experiment machinery입니다.

Wiki-like topic에서 Plasma는 fact sheet와 section scaffold 같은 predictable coverage pattern을 benchmark할 수
있습니다. 하지만 generic wiki가 되는 것이 목적은 아닙니다. Plasma의 장점은 evidence lineage, confidence,
uncertainty, correction, mission-specific synthesis입니다.

## Product Guardrails

- 약하다는 이유만으로 유용한 정보를 막지 않습니다.
- weak signal을 saved fact로 자동 승격하지 않습니다.
- source candidate, evidence, claim, report를 하나의 평평한 bucket으로 만들지 않습니다.
- confidence update를 approval hurdle로 만들지 않습니다.
- 보고서를 깔끔하게 보이게 하려고 uncertainty를 숨기지 않습니다.
- 미션이 조사를 요구한다면 agent가 search할 수 있게 합니다. Optional source/evidence/claim/report promotion
  point가 있다면 approval은 그 경계에 둡니다. 모든 search step마다 approval을 요구하지 않습니다.
- fact sheet를 evidence와 분리된 free-text surface로 만들지 않습니다.

## Implementation Slices

제안된 구현 순서는 다음과 같습니다.

1. Approval rule을 바꾸지 않고 evidence와 proposal surface에 signal-kind와 source-quality vocabulary를
   추가합니다.
2. Mission/report generation context에 report rigor level을 추가합니다.
3. Report drafting이 fact, interpretation, reaction, rumor, conflict, code, formula를 표현할 때 signal kind와
   confidence를 사용하게 합니다.
4. Approved evidence와 claim 위의 projection으로 fact sheet와 coverage map을 추가합니다.
5. 새 approved evidence가 section confidence나 coverage를 바꿀 때 report freshness signal을 추가합니다.
