# 미션 폴링 측정

이슈 #96은 활성 작업 관찰 경로만 바꿉니다. 선택한 미션은 기존의 전체 detail 표현을
계속 읽으며, 브라우저는 범용 cache나 새 reconcile endpoint를 사용하지 않습니다.

## 재현 Fixture

`TestMissionPollingLargeFixtureMetrics`는 대표적인 evidence 크기의 payload를 담은
activity 비대상 ledger event 240개와 서버에 in-flight로 등록해 detail recovery가 열어 둔
turn 하나를 가진 미션을 만듭니다.
기존 전체 detail과 `/api/missions/{id}/activity` 읽기 표면의 HTTP 응답 바이트를
측정합니다. 외부 네트워크 호출은 없고 SQLite 데이터베이스는 Go test 임시 디렉터리에만
생성됩니다.

실행 방법:

```sh
cd plasma
go test ./internal/web -run '^TestMissionPollingLargeFixtureMetrics$' -count=1 -v
```

## 기록된 결과

기준치는 이 변경 전 `a9ba837`에서 기록한 역사적 측정값입니다. 해당 commit에는 이
harness가 없으므로 현재 테스트 suite가 기준치를 assertion하지는 않습니다. 아래 이후
값은 위 명령이 출력하며, 시간은 비교에서 의도적으로 제외합니다.

| 관찰 항목 | 기준치 | #96 이후 |
| --- | ---: | ---: |
| fixture event 수 | 240 | 240 |
| 전체 detail 응답 | 449,087 B | 약 449,200 B |
| activity 응답 | 1,081 B | 1,205 B |
| 변경 없는 pending poll 요청 수 | 4 | 1 |
| 전진/gap/restart pending poll 요청 수 | 4 | 최대 2 |

기존의 네 요청은 미션 detail, 미션 목록, Confluence 연결, 미션 Confluence 접근 설정입니다.
`TestSelectedMissionActivityPollUsesCursorBeforeDetailFallback`는 실제 초기 detail
상태에서 현재 브라우저 함수를 실행합니다. 변경 없는 cursor는 activity 요청 하나만
보냅니다. 정상 전진, cursor gap 또는 regression, 호환되지 않는 cursor, server instance
변경은 activity 요청 하나와 선택 미션 detail 요청 하나만 보냅니다. 미션 목록이나
Confluence 설정은 다시 읽지 않습니다.

추가된 activity 바이트는 typed cursor의 schema, sequence, server instance 식별자입니다.
전체 detail은 기존 field를 보존하고 같은 `activity_cursor`를 추가하므로, 선택 미션은
두 번째 요청 없이 폴링을 시드할 수 있습니다. 생성 시각 때문에 원시 detail 응답은 수 바이트
차이가 날 수 있으므로 테스트는 안정적인 성질인 activity payload가 전체 detail의 5%보다
작음을 검증합니다. 정적 브라우저 테스트는 변경 없음, 전진, gap, 재시작 cursor의 요청 수와
경로를 검증하며 시간에 의존하지 않습니다.
