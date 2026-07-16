# 보고서 계획 제출

Web 일반 계획형 및 장문 보고서는 계획과 작성 사이에 내구성 MCP 제출 경계를
사용합니다. 보고서 실행기는 공급자 계획 turn에 결합된 도구 세션을 만들고 기존
자료 조사 도구와 함께 `plasma.report.plan.submit`을 노출합니다. 이 바인딩은 공급자
세션을 새로 만들거나 선택하지 않습니다. 공급자 세션은 기존 `same_session` 또는
`isolated_fork` 정책이 계속 결정합니다.

상태 전이는 다음과 같습니다.

```text
report.draft.pending
  -> report.plan.submitted
  -> 공급자 종료 검증(응답 전체가 정확히 PLAN_SUBMITTED이고 세션 계보가 일치)
  -> report.plan.created
  -> 기존 작성과 조립
```

엄격한 도구 입력의 공통 필드는 `mission_id`, `session_id`, `pending_event_id`,
`report_mode`, `idempotency_key`, `producer`, `plan`입니다. root와 모든 중첩
객체에서 알 수 없는 필드를 거부합니다. `planned` 계획은 기존 `summary`,
`sections`, `coverage_notes`, `planned_omissions`만 사용합니다. section은 `title`,
`purpose`, `target_refs`를 사용하며 summary 또는 sections 중 하나는 있어야 합니다.
기존 planned whitespace 의미는 바꾸지 않습니다. `long_form` 계획은 `summary`,
`parts`, `coverage_notes`, `planned_omissions`를 사용합니다. 각 part는 `title`, 선택적
`purpose`, sections를 가지며 section은 `title`, 선택적 `purpose`, `target_refs`를
가집니다. 장문 문자열은 trim하고 빈 항목은 제거하며 coverage와 omission은 각각
최대 24개를 유지합니다.

이슈 #110의 의도적 변경은 불완전한 장문 part/section을 합성하지 않고 거부하는 것과,
참조된 claim, evidence, snapshot, question, option 모두에 기존 미션 및 보고서 사용 가능
규칙을 적용하는 것으로 제한됩니다. 새 approval 상태, plan ID, 의미적 크기·개수 정책은
추가하지 않습니다.

공개 입력의 `session_id`와 producer는 서버에 결합된 MCP 도구 세션을 뜻하며 공급자
세션이 아닙니다. 서버는 미션, pending 이벤트, 모드, 멱등성 키, executor/model/effort와,
turn 전에 실제로 존재하는 경우에만 이전 공급자 세션을 선택적으로 결합합니다. 도구는
이 값과 계획 구조, 모든 자료 참조 종류를 검증하고 서버 소유 event producer로 제출
provenance만 기록합니다. 도구는 정식 계획을 만들 수 없습니다. 실행기는 `Invoke`가
반환한 뒤 실제 공급자 세션 계보를 검증하고, 그 실제 세션은 정식
`report.plan.created` provenance에만 기록합니다. 그 뒤 현재 도구 세션의
유효한 제출 하나만 원자적으로 승격합니다. 제출만 남은 시도는 진행 상태를 전진시키지
않으며, 재시도는 새 도구 세션을 사용해 오래된 제출을 무시합니다. 정식 계획 이후의
복구는 기존 규칙대로 정식 이벤트에서 계속됩니다.

계획 프롬프트에는 구체적인 미션, 도구 세션, pending 이벤트, 모드, 멱등성 키와 도구
세션 producer가 들어갑니다. 에이전트는 승인된 제출 하나를 만들어야 합니다. 정상
parse된 제출 호출은 성공과 멱등 재생을 포함해 모두 도구 세션의 3회 예산을
소비합니다. stdio는 dispatch 전에 JSON-RPC가 정확히 `2.0`인지, request ID가 문자열
또는 숫자인지 검증합니다. version 누락·불일치와 잘못된 ID 형태는 JSON-RPC
invalid-request 오류로 끝나며 도구 예산을 소비하지 않습니다. 기존 notification은
응답 없이 유지됩니다. 다른 envelope/protocol parse 실패도 소비하지 않습니다. 재시도 가능한
validation 실패는 고쳐 다시 제출할 수 있지만 세 번째 validation 실패와 네 번째 이후
parse 호출은 저장소에 도달하기 전에 재시도 불가로 거부됩니다. binding, conflict,
storage 실패도 재시도할 수 없습니다.

변환 범위는 Web `planned`와 Web `long_form`의 응답 JSON 계획 경로뿐입니다. CLI
planned는 Markdown `plan_text` 계약을 유지하고 CLI long-form 거부도 유지합니다.
보고서 작성, 장문 section/part 조립, 자료 읽기 정책, 보고서 세션 선택, one-take,
H5 패치, G2 지침, designed HTML은 바뀌지 않습니다.

향후 CLI 또는 MCP 보고서 시작 어댑터도 같은 reporting lifecycle을 호출해야 합니다.
도구 제출을 직접 승격하거나 최종 공급자 응답을 fallback 계획으로 해석하면 안 됩니다.
