# Plasma 용어집

이 문서는 Plasma에서 쓰는 제품 용어와 실험 코드명을 설명합니다. 오래된 축약어나 실험명이 현재
동작처럼 잘못 읽히지 않도록, 문서를 쓸 때는 이 용어집의 의미를 기준으로 맞춥니다.

## 제품 용어

| 용어 | 의미 |
|---|---|
| Mission | 하나의 주제, 목표, 질문을 다루는 지속적인 연구 작업 공간입니다. 대화, source event, workflow event, report artifact가 이 안에 쌓입니다. |
| Mission ledger | 미션의 append-only 이벤트 장부입니다. 대화 턴, source lifecycle event, MCP call, workflow event, report event가 모두 여기에 기록됩니다. |
| Connector | Liquid2, Confluence, 향후 settings-managed local filesystem root처럼 외부 원천에 접근하는 어댑터입니다. |
| Source | URL, PDF, 업로드 파일, Liquid2 문서, Confluence 페이지, media URL, local path 파일/디렉터리처럼 미션에 붙은 원본 연구 재료입니다. |
| Source candidate | 사용자가 검토할 수 있게 제안된 source 후보입니다. 사용자가 승인하기 전까지 정식 source가 아닙니다. |
| Staged source candidate | fetch 또는 추출이 성공해 candidate-only raw artifact가 생긴 후보입니다. Agent가 candidate-read 도구로 읽을 수 있지만, 여전히 승인된 source는 아닙니다. |
| Raw artifact | 저장된 본문, 추출 텍스트, 업로드 바이트, 생성된 보고서 Markdown, 내부 렌더링 재료 같은 저장 객체입니다. raw artifact가 자동으로 source가 되지는 않습니다. |
| Source snapshot | 사용자가 승인한 미션 단위 source 기록입니다. retrieval policy에 따라 raw artifact 또는 live reference locator를 가리킵니다. |
| Live reference | 변할 수 있는 원본 재료를 다루는 source policy입니다. 현재는 승인된 local path source에 쓰이며, 바이트를 고정하지 않고 locator와 observation을 남깁니다. |
| Evidence | source의 특정 인용 부분입니다. 현재 C1 기본 루프에는 포함되지 않지만, 나중에 비게이팅 reference/index 계층으로 돌아올 수 있습니다. |
| Claim | evidence가 뒷받침할 수 있는 주장이나 해석입니다. claim record는 legacy 또는 future design 영역이며, 현재 기본 workflow state가 아닙니다. |
| Result | agent가 만든 답변, 비교, 요약, 중간 결론, 초안 같은 출력입니다. result는 source를 참조할 수 있지만 source 자체는 아닙니다. |
| Saved knowledge | 미션에 의도적으로 남기는 지식입니다. 현재 C1은 이를 가볍게 유지하며, 예전 claim/evidence gate를 기본값으로 되살리지 않습니다. |
| Report | 미션 작업을 바탕으로 조립한 Plasma 소유 output artifact입니다. Markdown이 기본 보고서 artifact이고, HTML은 렌더링/내보내기 결과입니다. |
| Designed HTML | Markdown 보고서에서 JSON content model과 deterministic renderer를 거쳐 만드는 self-contained interactive HTML 보고서입니다. |
| MCP research surface | Agent가 거대한 prompt pack 없이 미션 상태를 보고, source를 검색/읽고, reference를 따라갈 수 있게 하는 UI 없는 도구 표면입니다. |

## 실험 코드명

| 코드 | 의미 | 현재 제품 상태 |
|---|---|---|
| C1 | 현재 기본 Plasma 제품 루프입니다. mission, 같은 provider session, user/controller steering, MCP/source read, conversation result, report artifact가 중심입니다. | 현재 제품 방향입니다. |
| C0 | controller 실험의 neutral baseline입니다. 강한 controller 개입 없이 같은 session을 이어가는 흐름에 가깝습니다. | 보수적인 controller 기본값의 근거로 사용합니다. |
| PAL2 | C0, NAV와 비교한 rhythm-aware question controller 변형입니다. | 결론이 충분하지 않아 기본값으로 쓰지 않습니다. |
| NAV | 상태, 다음 턴 의도, 방향을 더 강하게 제시한 investigation navigator 변형입니다. | 실험 뒤 기본값에서 제외했습니다. |
| G2 | 보고서 작성 시점에 적용한 한국어 말투 guidance입니다. | 기본 보고서 guidance 방향으로 제품화했습니다. 장문 보고서 작성에는 후속 검증에서 얻은 human-writer guidance도 함께 적용합니다. |
| H5 | 기존 Markdown 보고서를 bounded MCP patch 도구로 수정하는 한국어 humanization 후처리입니다. | 선택적/보조 후처리입니다. planning이나 source selection에는 참여하지 않습니다. |
| DH23 | agent-authored JSON content model과 deterministic renderer를 사용하고, 첫 화면의 강한 visual unit을 강조한 designed HTML 실험 경로입니다. | 현재 designed HTML 후보입니다. 한계는 남아 있습니다. |
| C4 | 장문 보고서 조립 전략입니다. section body를 보존하고 전체 재작성 대신 제한적인 heading normalization만 수행합니다. | 장문 보고서 조립 방식으로 제품화했습니다. |
| F4 | 실험에서 이어받은 보고서 작성 guidance입니다. 이전 조사 결과를 working memory로 쓰되 내부 run detail을 노출하지 않고 풍부한 Markdown 보고서를 쓰는 방향입니다. | 기본 Markdown 보고서 스타일 guidance입니다. |
| R-series | 보고서 생성 실험 변형입니다. 정확한 의미는 각 실험 문서 안에서 정의됩니다. | 과거 실험 기록입니다. 적용 전 local protocol을 읽어야 합니다. |
| M-series | controller와 workflow 실험에 쓰인 mission/corpus 변형입니다. | 과거 실험 기록입니다. 적용 전 local protocol을 읽어야 합니다. |
| DH-series | designed HTML 렌더링 실험 변형입니다. | 과거 실험 기록입니다. DH23이 주로 이어받은 변형입니다. |

## 현재와 Legacy를 구분하는 규칙

문서가 evidence, claim, confidence update, proposal bundle, AST report, report block을 만든다고 설명한다면,
그 문서가 현재 C1 경로를 말하는지 legacy ledger loop를 말하는지 먼저 확인해야 합니다.

현재 기본 규칙은 다음과 같습니다.

- source는 원본 재료입니다.
- result는 agent 출력입니다.
- report는 output artifact입니다.
- evidence/claim은 나중에 유용할 수 있지만, 조사나 보고서 생성을 막는 gate가 되어서는 안 됩니다.

## 작성 규칙

제품 문서를 추가할 때는 다음 순서를 따릅니다.

1. 현재 동작을 먼저 이름 붙입니다.
2. 과거 동작은 `Historical note` 또는 `Legacy note` callout에 둡니다.
3. 실험 코드명은 이 용어집으로 연결하거나 해당 문서에서 직접 설명합니다.
4. agent result를 source로 바꾸어 부르지 않습니다.
5. 미래 백로그 항목을 현재 제품 경로처럼 표현하지 않습니다.
