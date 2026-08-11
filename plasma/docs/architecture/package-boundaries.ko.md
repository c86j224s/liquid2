# Plasma 패키지 경계 규칙

이 문서는 [아키텍처 지도](README.ko.md)가 요약한 package와 import 규칙을 정의합니다. 새 작업과 동작 보존
refactoring에 적용하는 규범입니다. 영어 [기준 문서](package-boundaries.md)와 같은 의미를 유지합니다.

## 소유권 규칙

| ID | 규칙 | 검토 질문 |
| --- | --- | --- |
| P1 | Go package는 독립 구현의 기본 단위입니다. | 책임을 서로 무관한 “그리고” 없이 설명할 수 있는가? |
| P2 | Package는 기능 하나 또는 범위가 좁은 기술 메커니즘 하나를 소유합니다. | 파일들이 하나의 contract와 변경 이유를 공유하는가? |
| P3 | 기술적 상위 분류 자체는 하나의 기능이 아닙니다. | 관련 없는 Web, MCP, CLI, reporting, SQLite 기능을 기술 기준만으로 묶었는가? |
| P4 | 제품 규칙과 상태 전이는 소유 기능에 둡니다. | 이 결정을 바꾸려면 handler, SQL query, provider adapter를 수정해야 하는가? |
| P5 | 교체 가능한 구현은 소비자 소유 port를 사용합니다. | 소비자가 구체 adapter type이나 adapter 소유 정책을 import하는가? |
| P6 | 오래 실행되는 작업은 해당 기능의 runner가 소유합니다. | 재시도, 취소, 복구, 멱등성이 transport에 숨었는가? |
| P7 | `cmd/plasma`는 생성과 연결만 소유합니다. | 재사용 동작을 command에 구현하거나 Web에서 빌려오는가? |
| P8 | 공용 핵심부는 작고 의미가 안정적이어야 합니다. | `common`, `models`, `helpers`가 정하지 못한 소유자를 숨기는가? |

Package 이름과 directory 중첩은 소유권을 따릅니다. 상위 package가 얇은 등록 또는 조립 표면을 제공할 수는
있지만, 모든 하위 model과 service를 재노출하거나 또 다른 facade가 되어서는 안 됩니다.

## 허용 의존성

| 호출자 | 의존 가능 | 의존 금지 |
| --- | --- | --- |
| `cmd/plasma` 조립부 | Transport·기능·adapter constructor | 재사용 기능 정책 |
| 기능별 transport adapter | 자신의 protocol helper와 기능 API | 이웃 transport, 구체 storage·connector·provider 구현 |
| 기능 service | 자신의 model, 소비자 소유 port, 작은 공용 핵심부, 협력 기능의 명시적 API | Web, MCP, CLI, 구체 adapter |
| 기능 runner | 기능 상태와 port, provider port, replay에 필요한 clock·ID 추상화 | Request-bound state와 transport 소유 goroutine |
| 교체 가능한 adapter | 소비자 소유 port·model, 외부 SDK 또는 system library | 제품 정책과 관련 없는 기능 상태 |
| 공용 핵심부 | 표준 library와 더 작은 primitive | Transport, 기능 workflow, 구체 adapter package |

기능 사이 호출은 실제 use case를 위한 의도적인 공개 API를 통해서만 허용합니다. Import cycle을 만들거나 다른
기능의 adapter를 노출해서는 안 됩니다.

## 경계 검토

다음 신호가 나타나면 package 경계를 검토합니다.

- 서로 무관한 동작을 “그리고”로 이어야만 책임을 설명할 수 있습니다.
- 서로 다른 소비자에게 별개 model, error, service, store 묶음을 노출합니다.
- 기능 변경이 package 안의 관련 없는 파일을 반복해서 함께 수정합니다.
- 여러 제품 기능의 test와 fixture가 한곳에 쌓입니다.
- 공용 contract 재노출 때문에 fan-in이 높거나, 관련 없는 시스템 조율 때문에 fan-out이 높습니다.
- 운영 코드가 대략 20개 파일 또는 수천 줄을 넘습니다.
- 구현 파일이나 test 파일 하나가 코드 탐색의 병목이 됩니다.

수치는 자동 위반 판정이 아니라 검토 신호입니다. 큰 package를 유지하려면 파일들이 여전히 하나의 contract와
변경 이유를 공유한다는 근거를 남겨야 합니다.

다음 작업만으로는 경계를 해결한 것이 아닙니다.

- 모든 책임을 같은 package에 둔 채 큰 파일만 나눕니다.
- 하위 package를 만들고 상위 package가 전체 API를 다시 노출합니다.
- 관련 없는 type을 `common`, `models`, `helpers`, service locator로 옮깁니다.
- 교체 구현이나 소비자 경계가 없는 무의미한 interface를 만듭니다.

## Refactoring 순서

1. 유지해야 할 공개 Web, MCP, CLI, event, persistence, error, recovery 동작을 특성화합니다.
2. 기능 소유자와 소비자 관점 port를 정합니다.
3. Transport와 adapter보다 model과 policy를 먼저 소유자에게 옮깁니다.
4. 구체 구현을 새 port 뒤로 옮깁니다.
5. 제품 표면을 한 번에 하나씩 전환하고 각 단계를 독립적으로 test하고 revert할 수 있게 합니다.
6. 임시 alias, 재노출, compatibility facade를 같은 refactoring 계열 안에서 제거합니다.
7. 고친 의존 방향을 지키는 architecture check를 추가하거나 갱신합니다.

명시적 승인이 없다면 경계 refactoring에 DB schema 변경, 새 제품 동작, 관련 없는 UX 작업을 섞지 않습니다.

## 주석, Test, 강제

- 단순하지 않은 package의 comment에는 소유 기능, 이웃 경계, 이 package에 속하지 않는 것을 적습니다.
- 기능 test는 제품 규칙과 함께 둡니다. Transport test는 protocol 매핑을, adapter test는 infrastructure 동작을
  검증합니다. 조립된 경로는 작은 integration suite로 검증합니다.
- Architecture check는 Go import graph를 사용해 금지 의존을 차단해야 합니다. 크기 신호는 build를 기계적으로
  실패시키지 않고 경계 검토를 요구합니다.
- 구현이 시작되면 문서와 코드를 함께 옮깁니다. 추적 중인 refactoring이 끝나기 전에는 아키텍처 지도에 현재
  예외를 명시합니다.

현재 import 경계 검사는 `plasma/`에서 실행합니다.

```sh
go test ./internal/architecturecheck
```

저장소에 포함된 기준선은 알려진 부채의 정확한 file/import 조합을 기록합니다. 의도적인 refactoring으로 항목을
제거한 뒤 기준선을 다시 만들고 diff를 검토합니다. 관련 없는 새 부채를 받아들이는 용도로 갱신하면 안 됩니다.

```sh
go test ./internal/architecturecheck -args -update
```

## 현재 예외

Issue #66은 넓은 `internal/app`, `internal/web`, `internal/mcp`, `internal/reporting` 경계를 추적합니다.
현재 이 package들이 존재한다는 사실은 같은 형태의 새 의존을 허용하지 않습니다.

첫 이관 단계에서는 ledger, mission, artifact, source, 안정된 오류 model, 일부 report 실행 경계를 좁은
package로 옮겼습니다. #111부터 `internal/reportworkflow`가 제품 고정 report topology, typed stage 연결,
장문 prefix stage, final edit stage, legacy finalization stage를 소유합니다. 각 stage package는 자기 prompt,
MCP allowlist, provider 실행, 검증, durable replay 경계를 맡고, `internal/reporting`은 durable final-edit
계약과 artifact lineage를 유지합니다. `internal/app`의 호환 alias는 이행용 표면이며 새 소유 경계가 아닙니다.

SQLite 이관은 connection, migration, maintenance와 기능 간 transaction을 안정된 루트 facade에
유지합니다. 기능별 repository package는 이 facade의 구현 세부사항입니다. 루트 package만 이를 import할
수 있고, 이웃 repository끼리는 서로 의존할 수 없습니다.
