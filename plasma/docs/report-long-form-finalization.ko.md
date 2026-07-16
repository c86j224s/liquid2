# 장문 보고서 최종화

Web 장문 보고서는 기존 계획, 섹션 작성, 파트 조립, 세션 정책, H5, 디자인
HTML 흐름을 그대로 사용한다. 바뀌는 부분은 마지막 인계뿐이다. 보고서
에이전트가 여는 글과 맺는 글을 `plasma.report.long_form.finalize`로 제출하면,
서버가 내구 저장된 파트 artifact와 조립하고 기존 raw Markdown artifact와
`report.artifact.created` event를 한 트랜잭션에서 만든다.

계획형 보고서와 CLI 보고서 동작은 이 명령을 사용하지 않는다.

## 공개 도구 계약

도구는 완전한 숨은 실행기 binding과 명시적 도구 활성화가 있는 장문 최종
세션에서만 노출된다. 닫힌 입력은 다음 여덟 필드만 가진다.

- `mission_id`, `session_id`, `pending_event_id`, `plan_event_id`
- `idempotency_key`
- MCP tool session으로 고정된 `producer`
- `opening_markdown`, `closing_markdown`

에이전트는 최종 artifact ID, 파일명, 제목, 보고서 모드, 파트와 섹션 순서,
공급자 provenance, 모델 설정, 전체 보고서 Markdown을 선택할 수 없다. 이 값은
서버가 binding하며 commit 전에 mission ledger와 raw artifact에 다시 대조한다.

최종 raw artifact와 기존 canonical event는 SQLite 한 트랜잭션에서 commit된다.
같은 binding과 조립 SHA는 기존 결과를 replay한다. 식별자, provenance, 파트
순서, idempotency key, 조립 내용이 다르면 재시작이나 동시 호출 뒤에도
conflict다.
이 조건부 트랜잭션은 현재 ledger 상태를 기준으로 함께 판정하므로, pending
보고서의 terminal event와 최종 canonical artifact/event 생성은 경합할 수 없다.

## 완료와 재시도

matching canonical artifact/event가 존재하고 공급자 정규화 응답 전체가 정확히
`REPORT_FINALIZED`일 때만 최종 공급자 호출이 성공한다. 최종 단계는 최대 두
번 호출할 수 있다. 두 호출은 같은 logical tool session, idempotency key,
내구 artifact binding, 보고서 공급자 세션 chain을 재사용하며 계획, 섹션,
파트 작업은 반복하지 않는다.

`resume_failed`는 실패한 시도의 조상 chain에서 검증된 계획, 섹션, 파트
artifact만 재사용한다. 실패한 시도를 다시 열거나 바꾸지 않으며, restart는
조상 출력을 재사용하지 않는다.

첫 응답이 정확히 `front_matter`와 `closing` 문자열만 가진 legacy 객체이고 루트
trailing comma가 정확히 하나일 때만 재시도 힌트를 만들 수 있다. scanner는 그
쉼표 하나만 제거한다. 정상 JSON, fence, 앞뒤 설명, 추가 값, 알 수 없거나 중복된
필드, 중첩 trailing comma, 잘린 입력은 거부한다. 복구된 글은 두 번째 공급자
호출을 위한 비내구 참고값일 뿐이며 Web 코드가 artifact나 event를 만드는 데
사용하지 않는다.

명령이 commit됐지만 exact sentinel이 없으면 재시도는 내구 replay를 수행한다.
두 번째에도 sentinel이 없으면 acknowledgment anomaly이며 canonical 보고서를
되돌리거나 모순되는 보고서 실패 event를 추가하지 않는다.

## Provenance와 관측

공개 도구의 `producer`는 기존 MCP tool-session 관례를 따른다. 최종 artifact와
canonical event producer는 서버가 binding한 실제 보고서 공급자 세션을 쓴다.
canonical payload는 기존 보고서 metadata를 보존하고 final tool session을 별도로
기록한다. 도구 호출 뒤에야 알 수 있는 공급자 usage를 canonical event에 만들어
넣거나 대화 ledger event에 기록하지 않는다. redacted 운영 로그에는 반환 세션의
존재 여부와 bound 세션 일치 여부, token 집계, duration만 남으며, 반환 세션 ID나
공급자 usage 상세를 canonical 상태에 기록하지 않는다.

공용 `mcp.tool.called` payload는 변경하지 않는다. tool name, success, created event
ID를 canonical 보고서 provenance와 결합해 경로를 검증할 수 있으며 opening,
closing, prompt, 전체 보고서 본문은 trace 요약에 기록하지 않는다.
