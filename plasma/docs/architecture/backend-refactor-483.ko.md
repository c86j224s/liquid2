# Backend Refactor #483 아키텍처 기록

> 복구 재구성 기록: 원래의 미커밋 문서가 소실된 뒤 성공한 source, artifact, mission closeout transcript를 바탕으로 재구성했다. 복구 범위에는 충실하지만 byte-exact 동일성을 주장하지 않으며, 복구 사고와 문서 재구성 상태도 이 기록에 포함한다.

## 상태와 경계

복구된 작업과 이번 후속 slice는 #483의 source, artifact, mission, research catalog, evidence record, question/option record, claim confidence와 proposal 생성/decision 소유권 부분을 완료한다. **전체 #483 application 경계 refactoring을 완료한 것은 아니다.** `internal/app`에는 research application orchestration, visibility, materialization, workflow, report legacy model, connector facade, lookup과 persistence/atomic I/O 조율이 여전히 남아 있으며, proposal storage와 atomic commit 조율도 남아 있다.

이번 recovery-lineage slice는 Web의 `ReportRecoveryLineage`와 직접적인 단위 테스트만 `internal/reportexecution`으로 옮겼습니다. Web은 canonical 함수를 직접 호출하며, reporting과 reportrun의 lineage 의미는 통합하지 않았습니다.

Research record 이관은 evidence, claim, question, option record 생성과 claim confidence를 `internal/researchrecords`로 옮겼고 proposal 생성과 decision을 `internal/researchproposal`로 옮겼습니다. App에는 lookup, visibility, materialization과 persistence/atomic I/O 조율이 남았습니다.

| 기능 | #483 slice 이전 | 이번 closeout 이후 | `internal/app`에 남은 책임 |
| --- | --- | --- | --- |
| Evidence record | Evidence identity, source-ref validation, confidence normalization, creation lifecycle을 app이 소유 | `internal/researchrecords`가 evidence model, 좁은 source-snapshot reader port, validation/normalization, builder test를 소유 | Record/source lookup, visibility, materialization과 persistence/atomic I/O 조율 |
| Claim confidence update | Claim confidence request, event payload, append validation, origin normalization, projection을 app이 소유 | `internal/researchrecords`가 confidence request/payload/projection type, validation builder, origin normalization과 event projection을 소유하며 app callback을 사용 | Record/evidence lookup과 persistence/atomic append I/O는 app에 남음 |
| Question 및 option record | Question/option identity, 생성 validation, normalization, builder를 app이 소유 | `internal/researchrecords`가 question/option model, schema/object constant, requirement port, builder와 builder test를 소유 | Mission-event/record lookup, visibility, materialization과 persistence/atomic creation I/O는 app에 남음 |
| Research proposal surface | Proposal와 claim/question 정책이 app 및 proposal/MCP adapter에 분산 | `internal/researchproposal`가 proposal 생성, 제출 allowlist, decision payload 검증과 terminal transition을 소유하고 `internal/mcp/research`는 transport adaptation을 소유 | Ledger/record lookup, storage, visibility, materialization과 persistence/atomic commit I/O는 app에 남고 evidence 생성은 `internal/researchrecords`에 위임 |

이 research-record slice는 architecture baseline debt를 추가하지 않으며, app의 evidence allowlist와 confidence allowlist map은 실제 app consumer가 없음을 확인한 뒤 제거했습니다.

## 전 → 후 기능 지도


| 기능 | #483 slice 이전 | 복구 closeout 이후 | `internal/app`에 남은 책임 |
| --- | --- | --- | --- |
| Source | App이 source snapshot builder와 정책성 model을 소유 | Snapshot 생성 계약, 검증, locator parsing, retrieval policy, 순서가 있는 hashing, 비영속 조립을 `internal/source`가 소유 | Artifact/snapshot/event 원자적 commit 조율과 application service 연결 |
| Artifact | App이 raw-artifact 생성과 identity/content 정책을 소유 | 생성, storage URI, identity 검증, hash 확인, 정규화, timestamp, 방어적 복사를 `internal/artifact`가 소유 | Persistence와 atomic transaction 조율 |
| Mission | App이 mission metadata/lifecycle 계약과 append 정책을 소유 | Metadata, lifecycle 변경 계약, append-request 조립, 정규화, idempotency 정책을 `internal/mission`이 소유 | Conditional append, projection rebuild, active-work 검증과 관련 I/O 조율 |
| Research catalog | Research IDE reference, summary, page, kind constant와 순수 list/reference algorithm을 app이 소유 | `internal/researchcatalog`가 service, `ObjectRef`, `ObjectSummary`, `Page`, `References`, 정확한 kind constant, outline/list orchestration, source/artifact/ledger/research-record summary, report block read, locator metadata projection, reference deduplication과 normalization/gate/limit/cursor/pagination/reference algorithm을 소유하며 canonical root mission/source/artifact/ledger/researchrecords contract와 product error만 협력합니다. Proposal이나 app reverse dependency는 없습니다. | DTO, visibility, persistence callback/adapter, upload read policy, payload materialization, transport orchestration은 app에 남음 |
| Research inspection read와 grep | Bounded object-read validation, UTF-8 chunking과 literal grep sequencing이 materialization과 함께 app에 있음 | `internal/researchinspection`가 exact `ReadRequest`/`ObjectRead`와 `GrepMatch`/`GrepResult` contract, validation, callback 순서, byte-preserving `ClampBytes`/`ChunkBytes`, case-insensitive non-overlapping literal matching, snippet과 pagination을 소유합니다. App은 callback과 candidate construction으로 연결하고 PDF/local-path observation, security/visibility, payload materialization을 유지합니다. | Callback 구현과 candidate construction은 app에 남고 generic source I/O layer와 query engine은 도입하지 않음 |
| Evidence record | Evidence identity, source-ref validation, confidence normalization, creation lifecycle을 app이 소유 | `internal/researchrecords`가 evidence model, 좁은 source-snapshot reader port, validation/normalization, builder test를 소유 | Record/source lookup, visibility, materialization과 persistence/atomic I/O 조율 |
| Claim confidence update | Claim confidence request, event payload, append validation, origin normalization, projection을 app이 소유 | `internal/researchrecords`가 confidence request/payload/projection type, validation builder, origin normalization과 event projection을 소유하며 app callback을 사용 | Record/evidence lookup과 persistence/atomic append I/O는 app에 남음 |
| Question 및 option record | Question/option identity, 생성 validation, normalization, builder를 app이 소유 | `internal/researchrecords`가 question/option model, schema/object constant, requirement port, builder와 builder test를 소유 | Mission-event/record lookup, visibility, materialization과 persistence/atomic creation I/O는 app에 남음 |
| Research proposal surface | Proposal와 claim/question 정책이 app 및 proposal/MCP adapter에 분산 | `internal/researchproposal`가 proposal 생성, 제출 allowlist, decision payload 검증과 terminal transition을 소유하고 `internal/mcp/research`는 transport adaptation을 소유 | Ledger/record lookup, storage, visibility, materialization과 persistence/atomic commit I/O는 app에 남고 evidence 생성은 `internal/researchrecords`에 위임 |
| Application facade | 기능 model 재노출을 포함한 넓은 호환 hub | 완료된 source/artifact/mission model과 정책 본문 및 catalog, research-record, confidence, proposal 생성/decision contract와 algorithm은 더 이상 app이 소유하지 않음 | Workflow, research application orchestration, lookup/materialization, report legacy model, persistence/atomic I/O, connector service와 source/artifact/mission 조율 |

기존 app proposal 정책 본문과 source builder wrapper는 재export하지 않고 제거했다. Research catalog 이관은 의도적으로 부분 범위이며 app의 visibility, lookup, materialization, report legacy model과 persistence/atomic I/O는 다음 독립 검증 slice까지 남긴다. `internal/researchrecords`는 evidence, claim, question, option record 생성과 claim confidence를 소유하고 `internal/researchproposal`는 proposal 생성과 decision을 소유한다. 여섯 source 계약은 구체적인 source 소유 타입이다. 이 closeout 범위 밖의 legacy alias는 해당 이관이 포함되지 않은 곳에서만 유지한다.

## 측정된 변화

수치는 각 package 아래의 직접적인 non-test, non-generated Go file와 line 수입니다. 다음 전후 표는
`d027ab6` checkpoint에 기록된 역사적 stage 비교이며, `d027ab6`만의 전체 snapshot도 현재 tree도
뜻하지 않습니다. 복구 closeout 기록을 보존하기 위해 남깁니다.

| Package | 이전 files / lines | 이후 files / lines | 변화 |
| --- | ---: | ---: | ---: |
| `internal/app` | 78 / 15,961 | 75 / 15,402 | -3 / -559 |
| `internal/source` | 3 / 180 | 9 / 679 | +6 / +499 |
| `internal/artifact` | 3 / 51 | 4 / 123 | +1 / +72 |
| `internal/mission` | 7 / 627 | 8 / 760 | +1 / +133 |

역사적 architecture import-debt 기준선은 145개에서 118개로 감소했고 catalog와 evidence 경계로
baseline debt를 추가하지 않았습니다. 삭제한 세 app model file은 `package app`만 포함했고
(`projection_models.go`에는 deprecated 주석 하나 포함), 선언은 없었습니다.

### `6d89457` 현재 측정 범위 (2026-09-08)

현재 직접 non-test, non-generated Go file 및 line 수는 다음과 같습니다.

| Package | 현재 files / lines |
| --- | ---: |
| `internal/app` | 77 / 12,639 |
| `internal/researchcatalog` | 6 / 921 |
| `internal/researchinspection` | 4 / 271 |
| `internal/researchrecords` | 10 / 1,117 |
| `internal/researchproposal` | 5 / 786 |

Architecture import-debt 기준선은 원래 145개에서 현재 105개로 감소했습니다. 현재 105개는
`app-hub` 101개와 `transport-to-adapter` 4개이며, `capability-to-adapter`는 0개입니다. 이는
현재 측정값이며 전체 application-boundary refactoring이 완료되었다는 뜻이 아닙니다.

역사적 표는 dated step으로 보존하며 현재 값으로 읽지 않습니다. 현재 표가 `6d89457` 기준의
측정값입니다.

## Checkpoint와 handoff

이번 CLI source file partition은 mechanical same-package 변경입니다. Confluence dispatch, auth, connections, browse, snapshot, update, client, local-source, read, output 책임을 파일로 나누어 가독성을 높였지만 package ownership은 바꾸지 않았습니다. 기존 `source_commands.go`에는 `runSources`, shared repeated-string flag, read-size constant, upload을 유지했습니다.

이관 선언은 다음과 같습니다. `runSourcesConfluence`와 `printSourcesConfluenceUsage` → `source_commands_confluence_dispatch.go`; OAuth command/config helper → `source_commands_auth.go`; connection/site command → `source_commands_connections.go`; spaces/pages/children/search와 page print → `source_commands_browse.go`; preview/snapshot → `source_commands_snapshot.go`; check-update/update-preview/update → `source_commands_update.go`; client/browser/site/cloud discovery helper → `source_commands_client.go`; local roots/tree/attach/list/show/remove/restore 및 local config helper → `source_commands_local.go`; read/live/snapshot/grep 및 artifact content read → `source_commands_read.go`; raw-artifact metadata/response와 공통 output/error helper → `source_commands_output.go`입니다.

Architecture baseline에서 바뀐 것은 기존 `cmd/plasma -> internal/app` app-hub debt의 위치뿐입니다. CLI same-package 분할은 열 개 파일과 app-import baseline row 일곱 개를 추가했지만 기존 `source_commands.go` edge는 남아 있고, `internal/app`을 계속 import하는 일곱 파일은 `source_commands_auth.go`, `source_commands_client.go`, `source_commands_local.go`, `source_commands_output.go`, `source_commands_read.go`, `source_commands_snapshot.go`, `source_commands_update.go`입니다. 새 package edge나 package ownership 경계는 추가하지 않았습니다.

알려진 CLI vet 진단은 `source_commands_auth.go` 36행과 90행의 unreachable-code 진단입니다. 이는 알려진 검증 부채이며 vet 전체가 clean하다고 기록하지 않습니다.

`internal/reporthumanize`는 legacy/manual post-canonical H5 compatibility contract를 기록합니다. 여기에는 same-session binding, validation, terminal event 적용, 실패/no-op 원본 보존과 restart recovery가 포함되지만 active long-form 경로는 아닙니다. Active long-form pre-canonical style-edit stage는 별도의 `internal/reportworkflow` reporting pipeline에 남습니다. Web은 관련 경로의 HTTP request normalization, executor/model 선택, route lock과 orchestration을 유지하고 CLI는 Web을 import하지 않는 transport-neutral 경로를 호출합니다.

Phase 0 PDF test 기록은 `internal/reportilpdf`의 원래 10개 중 9개와 외부에서 작성된 Phase 0 test 1개로 복원된 것입니다. bridge file 자체는 test가 아닙니다.

Legacy/manual H5 humanization duplicate rejection failure/recovery test는 아직 해결되지 않았습니다. 이 문제는 auto-compaction refactor 중 surfaced되었고, 테스트한 race condition에서 baseline과 변경 버전 모두 20/20으로 failure가 재현되었습니다. 이후 targeted normal test는 통과했지만 globally deterministic 또는 all-race-green이라고 주장하지 않습니다. 최근 full-module test pass와 selected PDF race pass는 서로 다른 검증 결과입니다.

`6d89457`까지의 checkpoint 계보와 이전 복구 사고를 이 기록에 보존합니다. 앞의 측정 표는 dated historical step이며 현재 측정값이 아닙니다.

복구된 구현 계보는 다음 순서다.

`f359b47` → `a93a1ef` → `260e6c4` → `4e59d0e` → `63c143c` → `5da2e3a` → `630018a` → `ddb56e2`

이번 slice의 필수 시작 checkpoint는 `a1106b0`이다. Source/artifact/mission closeout 계보는 `ddb56e2`로 유지하며, 이번 작업은 그 계보를 다시 쓰지 않는다. Main merge, commit, GitHub 작업, issue 종료 작업은 포함하지 않았다. 다음 담당자는 완료된 source/artifact/mission 소유권을 안정된 기준으로 취급하고, 남은 app facade 이관은 별도의 독립 검증 작업으로 이어가야 한다. 따라서 #483 전체 완료로 기록해서는 안 된다.

## 검증 기록

이번 slice에서 catalog, app, MCP, SQLite, Web, architecture 집중 검증이 통과했다. Catalog package는 표준 library와 `internal/producterror`만 의존하며 source adapter나 transport import를 추가하지 않았다. `go vet ./...`에는 CLI source command file의 알려진 unreachable-code 진단 두 건이 여전히 남는다. 실제 provider 실행과 browser 검증은 수행하지 않았다.

사용자 장비 경로, local runtime state, provider session, raw recovery artifact는 이 공개 기록에 포함하지 않았다.

## Liquid2 source capability boundary

Liquid2 소비자 contract는 이제 `internal/source/liquid2source`가 소유합니다. 검색·문서·snapshot 중첩 contract, connector identity 상수와 외부에서 사용하는 normalization 함수를 이 package에 두었습니다. 이 package는 정확히 `internal/source`, `internal/artifact`, `internal/ledger`, `internal/producterror` contract에만 의존하며, 부모 `internal/source`는 child를 import하지 않습니다. HTTP 동작은 `internal/connectors/liquid2`에 남고 child contract와 product error sentinel만 사용하며 `internal/app`에는 의존하지 않습니다. Application의 snapshot content-range, producer, artifact, event, atomic 조율은 그대로 `internal/app`에 남습니다.

## Confluence source capability 경계

`internal/source/confluencesource`는 Confluence source connector port, request/result contract, normalization 규칙, 외부 identity helper와 구조화된 failure를 소유합니다. 안정적인 `ConfluenceError` 타입, category/code 상수, 안전한 status/message helper, HTTP/transport 생성자와 private cause closure도 포함합니다. 정확히 `internal/source`와 `internal/producterror` contract에 의존하며 부모 `internal/source`는 이 child를 import하지 않습니다. 저장 credential과 접근 가능한 site projection은 `internal/confluenceaccess`에, snapshot/artifact/event와 update 조율은 `internal/app`에 남깁니다.


## Phase 0 PDF renderer 경계

Phase 0 소비자 package는 `PDFRenderer` port와 `PDFResult` contract를 소유합니다. Chrome 실행은 `internal/reportilpdf.Chrome`에 남고 executable path, 45초 timeout, 임시 profile, readiness 확인과 renderer identity를 담당합니다. CLI와 Web composition point는 기존 Chrome 경로 해석 시점을 유지하면서 adapter를 주입합니다. renderer가 nil이면 provider를 건너뛰거나 자동 Chrome fallback을 추가하지 않고 render 단계에서 명시적으로 실패합니다.
