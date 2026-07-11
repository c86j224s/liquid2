# Plasma Automatic Investigation

이 문서는 Plasma의 자동조사 흐름을 정의한다. 자동조사는 사용자가 대화로 진행하던
같은 미션 안에서 에이전트에게 조사를 맡기는 기능이다. 별도 미션이나 별도 제품
모드가 아니다.

## C1 Cutover Note

C1 기본 루프에서는 자동조사가 evidence, claim, confidence update, proposal bundle을
새로 만드는 경로가 아니다. 다만 일반 대화와 bounded workflow run 모두 에이전트가
새 원자료의 URL과 채택 의견을 명시했을 때 source candidate review record를 남길 수
있다. 이 후보는 소스가 아니며 사용자가 승인하기 전에는 스냅샷을 만들지 않는다.
controller가 있다면 최신 result를 읽고 다음 사용자식 steering turn을 만드는 역할을
한다. controller 출력도 result로 취급하며 source/evidence/claim으로 자동 저장하지
않는다.

2026-06-26 C0/PAL2/NAV 실험 이후 controller는 자동조사의 기본 엔진으로 보지 않는다.
상시 구조화 steering은 조사 품질을 보장하지 않았고, NAV는 baseline보다 나빴다.
자동조사의 기본은 같은 provider session이 MCP/source read 도구를 사용해 진행하는
bounded workflow run이다. controller는 향후 정체 감지나 방향 회복 같은 구체 실패
조건에서만 약한 steering 후보로 다시 실험한다.

현재 기본 구현에서 자동조사는 bounded workflow run이다. run은 같은 미션 장부에
요청, 시작, step, 중지 요청, 완료, 중지, 실패, 중단 이벤트를 남기고, 각 step은
일반 대화와 같은 provider session을 재개한다. 아래에서 후보/승인 중심으로 남은
설명 중 evidence, claim, confidence, proposal 중심 흐름은 legacy ledger loop와 향후
실험의 배경으로만 취급한다. source candidate 검토는 원자료 승인 표면으로만 남기며,
기본 제품 UI나 기본 MCP tools/list가 근거/주장 중심 legacy 경로를 사용자 선택지로
노출해서는 안 된다.

## Purpose

자동조사의 목적은 사용자가 명시적으로 조사를 맡겼을 때 미션을 앞으로 밀어주는
것이다. C1 기본 루프에서 에이전트는 연결된 소스를 읽고 필요한 외부 자료 경로를
찾아 result로 설명하지만, 그 답변을 source/evidence/claim/proposal로 자동 저장하지
않는다.

자동조사는 최종 판단을 대신하지 않는다. 사용자는 돌아와서 result를 읽고, 필요한
원본 자료를 소스로 직접 붙이거나 다음 steering turn으로 같은 미션에서 대화를
이어간다.

첫 구현은 무한 실행이 아니다. 모든 run은 최대 step 수, 최대 실행 시간, stop
condition을 가진다. 기본 예산은 최대 10 step과 최대 25분이다. Web, CLI, MCP는
같은 run ID와 같은 projection으로 상태를 읽고 중지를 요청한다.

## Start Conditions

현재 제품 기준에서 자동조사는 사용자가 명시적으로 요청했을 때 시작한다.

예:

- "이 미션 자동으로 조사해줘."
- "내가 돌아올 때까지 관련 원본 자료를 확인해 둬."
- "이 소스들 안에서 반대 근거를 더 찾아봐."

후속 버전에서는 옵션에 따라 idle 상태에서 자동조사를 시작할 수 있다. 이 옵션은
기본값이 아니며, 사용자가 켜야 한다.

agent/MCP turn 안에서 시작 요청이 들어오면 Plasma는 provider를 즉시 재귀 실행하지
않는다. `workflow.run.requested`를 장부에 남기고, 현재 `turn.agent.response`,
`turn.agent.error`, 또는 `turn.agent.canceled`가 생긴 뒤 runner가 실행한다.

## Mission Continuity

자동조사는 대화형 연구와 같은 미션 장부를 사용한다.

- 사용자가 대화로 미션을 만들고 조사한다.
- 사용자가 bounded workflow run을 시작한다.
- 에이전트가 같은 미션 목표, 범위, 도구로 조회 가능한 소스를 기준으로 한 step씩
  조사한다.
- 각 step 결과와 control decision이 장부 이벤트로 남는다.
- 사용자는 돌아와 run status와 result를 확인하고 다음 steering을 결정한다.
- 새 원본 자료가 필요하면 사용자가 직접 source로 붙인다.
- 이후 사용자는 같은 미션에서 다시 대화를 이어간다.

이 흐름 때문에 자동조사는 별도 미션 ID나 별도 연구 상태를 만들지 않는다.

## Source Search Authority

에이전트가 조사할 수 있는 소스 범위는 사용자의 지시로 정한다.

이미 미션에 붙은 소스 안에서 조사하라는 지시라면, 에이전트는 그 소스 안에서만
읽고 답한다.

사용자가 조사를 요청했거나 에이전트가 답변에 새 자료가 필요하다고 판단하면,
에이전트는 붙어 있는 소스를 먼저 확인하고, 필요하면 Liquid2 같은 연결된 소스
커넥터나 웹 검색처럼 사용 가능한 경로를 시도해 새 자료를 찾을 수 있다. 별도의
검색 사전 승인은 요구하지 않는다. 다만 검색 결과나 URL 언급은 자동 source candidate가
아니며, 에이전트가 원자료의 쓰임을 설명하는 명시적 후보와 채택 의견을 남긴 경우에만
검토 대상으로 보여준다. 미션 소스가 되려면 사용자가 직접 source로 붙여야 한다.

사용자가 새 자료 탐색을 명시적으로 제한했는데 에이전트가 새 자료가 필요하다고
판단하면, 에이전트는 먼저 사용자에게 조사 범위 변경을 요청해야 한다. 제한이 없는
일반 조사에서는 검색 자체가 검토 지점이 아니라, 결과를 보고 다음 source 추가나
steering을 결정하는 사용자의 판단이 검토 지점이다.

이미 미션에 붙은 소스의 목록과 원문 일부를 읽는 작업은 새 소스 탐색이 아니다.
에이전트는 `plasma.research.list`, `plasma.research.grep`,
`plasma.research.read`, `plasma.research.references` 같은 읽기 전용 도구로 저장
소스와 장부를 확인할 수 있다. 반대로 `plasma.sources.search`는 Liquid2 같은 외부
소스 커넥터에서 새 원본 자료 경로를 찾는 작업이다. 검색 결과는 아직 수락된
소스가 아니며, 사용자가 원본 자료를 붙여야 Plasma가 스냅샷을 만든다. 특정 경로가 실패하면 전체
조사를 실패로 끝내지 않고, 실패한 경로를 보고한 뒤 사용 가능한 다른 경로를
시도해야 한다.

local path source는 서버가 설정한 allowlisted root 안에서만 허용된다. 자동조사와
MCP 도구는 `root_id`와 `relative_path`만 사용하며 absolute path를 입력하거나
출력하지 않는다. live local path source를 읽거나 grep하면 `source.observed`
이벤트가 남고, 이 관찰 이벤트가 그 시점의 mutable source 상태를 설명한다. grep
스니펫은 후보이며, 최종 답변이나 리포트 문장은 필요한 경우 명시적 read 관찰로
확인해야 한다.

MCP 조회면은 `plasma.research.outline`, `plasma.research.list`,
`plasma.research.read`, `plasma.research.grep`,
`plasma.research.references`의 다섯 도구 모델을 따른다. 자동조사 에이전트는
`plasma.research.outline`으로 미션 전체를 확인하고,
`plasma.research.list`와 `plasma.research.grep`으로 읽을 대상을 찾고,
`plasma.research.read`로 source snapshot, raw artifact, ledger result event의
필요한 부분만 임의 위치에서 확인한 뒤, `plasma.research.references`로 C1 객체의
관계를 따라가야 한다. 검색 결과와 스니펫은 후보일 뿐이며 source/evidence가 아니다.

브라우저에서 실행되든 UI 없는 클라이언트에서 실행되든 MCP 서버는 매 턴 특정
`mission_id`와 `ses_...` 도구 세션에 묶인다. 따라서 에이전트가 다른 미션 ID나
다른 세션 ID로 도구를 호출하면 백엔드가 거절해야 한다.

## Source Candidate Review Flow

현재 C1 일반 대화와 bounded workflow run은 새 source candidate review records를
만들 수 있다. 에이전트가 찾은 새 자료는 곧바로 미션 소스가 되지 않는다.

흐름은 다음과 같다.

1. 에이전트가 새 자료를 발견한다.
2. Plasma가 그 자료를 소스 후보로 검토 영역에 보여준다.
3. URL 후보는 가능한 경우 백그라운드에서 원문 가져오기를 시작하고,
   `source.candidate.staging_started`를 남긴다.
4. 가져오기가 성공하면 `source.candidate.staged`, 실패하면
   `source.candidate.staging_failed`를 남긴다.
5. 에이전트는 staged 후보를 전용 MCP 도구로 읽을 수 있지만, 응답은 항상
   미승인 후보임을 표시한다.
6. 사용자가 소스 후보를 승인하거나 거절한다.
7. 승인된 소스 후보만 Plasma가 스냅샷을 만든다.
8. 스냅샷이 만들어진 뒤에만 그 자료는 미션 소스가 된다.

사용자가 직접 추가한 자료는 사용자가 이미 선택한 것이므로 승인된 소스로 본다.
그래도 Plasma는 이후 인용과 리포트를 안정적으로 만들기 위해 스냅샷을 남겨야 한다.
현재 브라우저 워크스페이스에서는 URL 소스 후보를 "소스로 추가"하면 Plasma가
이미 staged된 본문이 있으면 그 artifact를 재사용해 URL 소스 스냅샷을 만들고,
없으면 해당 HTTP/HTTPS 원문을 가져와 스냅샷을 만든다. "기각"하면
`source.candidate.rejected` 이벤트를 남겨 같은 미션의 후보 목록에서 숨긴다. 기각은
그 URL을 원본 자료로 쓰지 않겠다는 미션 안의 결정이지, 에이전트 결과를 삭제하거나
외부 자료를 변경하는 작업이 아니다.

URL 후보를 소스로 추가할 때 fetch 보안 정책을 통과하지 못하면 소스 스냅샷은
만들어지지 않는다. 현재 fetcher는 DNS 해석 뒤 loopback, private, link-local,
multicast, unspecified, `100.64.0.0/10` CGNAT 주소를 거절하고,
리다이렉트된 요청에도 같은 검사를 적용한다. 프록시 설정은 사용하지 않고, 전체
60초 타임아웃, 45초 응답 헤더 타임아웃, 최대 5회 리다이렉트, 64 KiB 응답 헤더,
20 MiB 본문, 텍스트성 미디어 타입 제한도 적용한다. 같은 미션에 이미 같은 정규화
URL 스냅샷이 있으면 Plasma는 새로 fetch하지 않고 기존 소스 스냅샷을 돌려준다.
credential-bearing URL은 후보 파싱과 MCP 제안 양쪽에서 거부한다.

## Bounded Workflow Run Loop

새 workflow run은 3층 지시 구조만 사용한다. 즉 agent step prompt에는 사용자의 원문
자율 진행 요청, Plasma가 도출한 자율 진행 목표, 이번 step에서 수행할 구체 지시가
함께 들어간다. 이전 `current` step instruction mode는 과거 장부 호환 입력으로만
남아 있으며, 새 요청에서는 `layered`로 정규화된다.

현재 기본 반복은 다음 순서다.

1. `plasma.research.outline`으로 현재 미션 목표와 범위를 읽는다.
2. 열린 질문과 이미 연결된 소스를 확인한다.
3. 다음에 조사할 작은 질문을 만든다.
4. `plasma.research.list`와 `plasma.research.grep`으로 허락된 소스 범위 안의
   읽기 대상을 찾는다.
5. `plasma.research.read`로 원문, raw artifact, ledger result event의 필요한
   부분만 확인한다.
6. `plasma.research.references`로 소스, 근거, 저장 지식, report artifact 관계를
   확인한다.
7. 사용자가 읽을 수 있는 result를 남긴다.
8. runner가 읽을 수 있는 작은 control decision을 남긴다. decision은 `continue`,
   `stop`, 다음 지시, 사유를 포함한다.
9. runner가 control decision, stop 요청, 최대 step 수, 최대 실행 시간, provider
   오류를 확인해 계속할지 terminal 이벤트를 남길지 결정한다.
10. 최대 step 수나 최대 실행 시간에 도달했지만 마지막 decision이 `continue`이면
    runner는 완료가 아니라 `paused` 상태로 종료하고 다음 지시를 남긴다.

각 step이 시작되기 전에 runner는 active source projection을 다시 확인한다. workflow
시작 이후 source가 soft removed 상태가 되었고 아직 같은 removal에 대한 skip 기록이
없다면 `workflow.source.skipped`를 남긴다. 제거된 source는 다음 step의 기본 read,
reporting, source planning 대상에서 제외된다. 이미 안전하게 시작된 read를 억지로
중단하는 것이 첫 구현 목표는 아니며, 완료된 read는 그 시점의 `source.observed`
이벤트로 감사 가능하게 남는다.

이 반복은 무한히 돌면 안 된다. 사용자 요청마다 제한된 시간, 제한된 읽기 대상 수,
제한된 도구 호출 수를 둔다.

## Workflow Status Surface

자동조사 진행 상태는 채팅 답변에 프로토콜 JSON을 섞지 않고 별도 상태로 보여준다.
각 step의 agent answer는 result로 남고, workflow event는 진행 상태와 중지 사유를
설명한다.

상태 영역은 최소한 다음을 보여줘야 한다.

- workflow run ID
- 현재 상태: queued, running, stopping, paused, completed, stopped, failed, interrupted
- 요청한 지시와 제한된 실행 예산
- 현재 step 또는 마지막 완료 step
- 사용자가 읽을 수 있는 step result 요약
- stop reason, failure reason, 또는 이어서 진행할 다음 조사 지시
- 사용자가 할 수 있는 행동: 중지, 완료 대기, 일반 대화 재개, 새 run 시작, 리포트 요청

소스 후보 검토 영역은 기본 workflow status와 분리해서 보여준다. 후보별 승인,
거절, 보류, 수정 요청은 workflow run의 완료 여부와 별개의 사용자 결정이다.

## Approval Rules

자동조사는 제안할 수 있지만 확정하지 않는다.

사용자 승인 없이 확정하면 안 되는 것:

- 새 원본 자료를 미션 소스로 격상
- 소스 스냅샷 생성
- 주장을 저장 지식으로 확정
- 리포트 초안 확정
- 외부 서비스 게시
- Liquid2 원본 자료 변경

승인 없이 남길 수 있는 것:

- 자동조사 시작, 진행, 종료 같은 작업 기록
- workflow step result와 bounded summary
- 필요한 후속 source 추가나 조사 범위 변경 제안
- 오류와 중단 사유

신뢰도 갱신은 C1 기본 workflow run이 새로 만드는 경로가 아니다. legacy 실험에서
자동조사가 새 근거 때문에 기존 주장을 더 강하거나 약하게 봐야 한다고 판단하면
장부 이벤트로 남길 수 있지만, 그 자체가 주장의 승인, 기각, 재승인 요구로 이어지면
안 된다.

## Conversation Logging

provider transcript 전체를 미션 장부에 복사하지 않는다.

미션 장부는 사용자 turn, agent result, workflow status event, source snapshot,
저장 지식, 리포트 산출물을 중심으로 유지한다. provider 내부 로그, 전체 도구
transcript, 소스 본문 묶음, 대형 recall JSON은 매 step 이벤트에 넣지 않는다.
나중에 사용자가 "그때 왜 이렇게 판단했는지"를 더 자주 확인해야 한다면, 전체
transcript 복사가 아니라 턴별 요약이나 결정 근거만 저장하는 방식을 검토한다.

## Runtime Controls

자동조사에는 다음 제어가 필요하다.

- 시작
- 중지
- 현재 진행상태 확인
- 실패 이유 확인
- step result 확인
- 완료 뒤 일반 대화 재개
- 같은 세션 기반 리포트 요청

이 제어는 브라우저 검토 영역에만 묶이면 안 된다. Web, CLI, mission-bound MCP
도구는 같은 미션 장부 projection으로 workflow status를 읽어야 한다. MCP
`plasma.workflow.start`는 provider를 직접 실행하지 않고 요청을 queued 상태로
남긴다. 이 요청은 현재 `turn.user`와 현재 `agent_executor` binding에 묶이며,
다른 executor를 지정하면 queued run을 만들지 않고 거절된다.

실행 중 프로세스가 사라지거나 terminal 이벤트 없이 오래 멈춘 run은 projection에서
`interrupted`로 보여준다. 첫 구현은 새 durable queue/lease table을 만들지 않는다.
사용자는 보이는 상태를 보고 stop을 요청하거나 새 bounded run을 시작해 회복한다.
아직 runner가 시작하지 않은 queued run은 stop 요청 시 즉시 `stopped` terminal
상태로 닫힌다.

source removal은 자동조사 제어면에서도 soft archive다. Web, CLI, MCP는 제거된 source를
기본적으로 숨기고 사용하지 않지만, `include_removed` 같은 명시적 audit 보기로 확인할
수 있다. 이 기능은 로컬 파일 삭제, raw artifact 삭제, 물리 purge/redaction을 수행하지
않는다.

후속 버전에서는 다음을 추가한다.

- idle 자동 실행 옵션
- 실행 예산 설정
- 특정 소스 집합만 사용하도록 제한
- 특정 질문만 조사하도록 제한
- 실패한 step만 다시 조사

## Future Expansion

클로드 코드식 대규모 병렬 조사나 스크립트 기반 연구는 후속 확장 방향으로 남긴다.
이 확장은 하나의 미션 안에서 많은 하위 질문을 병렬로 조사하고, 결과를 후보로
모아 사용자가 검토하는 방식이어야 한다.

복잡한 워크플로우 엔진은 현재 제품 흐름이 안정적으로 동작한 뒤 확장한다. 먼저
다음 범위에 머문다.

- 하위 질문 생성
- 허락된 범위 안의 소스 탐색
- result로 새 원본 자료 필요성 설명
- read-first MCP 도구로 소스와 장부 확인
- 사용자가 판단할 수 있는 중간 결론 작성
- bounded step 반복

대규모 병렬 조사는 이 흐름이 실제 사용에서 안정적으로 작동한 뒤에 추가한다.
