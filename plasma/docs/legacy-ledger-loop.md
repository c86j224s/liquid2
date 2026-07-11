# Legacy ledger loop

Plasma C1의 기본 제품 루프는 mission, source snapshot/raw artifact, same agent
session, user/controller steering, conversation result, Markdown report artifact를
중심으로 동작한다.

이 문서는 C1 이전 ledger 기능을 삭제하지 않고 보존하는 경계를 기록한다.

## 보존 범위

- `evidence_record`, `claim_record`, `question_record`, `option_record`,
  `proposal_bundle`, claim confidence history는 historic mission을 읽기 위해 남긴다.
- AST report, report version, report block 저장소와 export 코드는 과거 리포트 조회와
  rollback safety를 위해 남긴다.
- MCP mutation tools는 기본 `tools/list`에 나오지 않는다. 개발자/실험용
  `--legacy-research-loop` 경로에서만 노출된다.
- Research IDE의 legacy object kind는 기본 outline/list/read/grep/references에
  나오지 않는다. 명시적 legacy read 경로에서만 조회한다.

## 기본 경로에서 하지 않는 일

- 에이전트 답변을 evidence, claim, confidence update, proposal bundle로 자동
  변환하지 않는다.
- 에이전트 답변 안의 일반 URL을 source candidate로 자동 변환하지 않는다. 명시적인
  `소스 후보`와 `채택 의견`이 있는 경우만 C1의 원자료 검토 후보로 남길 수 있다.
- 리포트 생성 시 approved evidence/claim을 요구하지 않는다.
- 기본 리포트 생성은 AST plan, AST JSON, repair prompt, report version/block을 만들지
  않는다.
- 브라우저 기본 화면에 legacy proposal approval flow, claim confidence panel,
  AST report action을 제품 선택지로 노출하지 않는다. source candidate 검토 표면은
  원자료 승인 흐름으로만 유지한다.

## 삭제 후보

- agent answer URL만 보고 source candidate를 생성하는 기본 제품 흐름
- default proposal extraction prompt와 자동 proposal pass
- evidence/proposal approval 중심의 기본 UI 탭
- claim confidence mutation UI
- AST-first report plan, repair, report block rendering 중심 기본 생성 경로

DB table drop이나 destructive migration은 이 문서의 범위가 아니다. 과거 데이터 삭제는
별도 migration 계획과 복구 기준이 있을 때만 다룬다.
