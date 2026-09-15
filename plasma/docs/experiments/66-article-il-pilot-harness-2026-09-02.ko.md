# Article IL Pilot Harness

Issue: #461

상태: synthetic archive preflight 통과. 아직 실제 Article은 생성하지 않았다.

## 목적

이 wave는 Article IL 독서 pilot 전에 필요한 archive 경계를 구현한다. Frozen 3-fixture, 3-arm matrix를 정확히 한 번 실행하고, 성공과 실패를 모두 보존하며, Plasma 제품 database, ledger, API 또는 UI에 Article 상태를 쓰지 않고 검증 가능한 terminal receipt를 만들 수 있음을 확인한다.

Article 품질을 시험하는 단계는 아니다. Synthetic executor는 deterministic placeholder Markdown만 쓰며 Codex나 다른 provider를 시작하지 않는다.

## 계약

`internal/articleexperiment`은 다음을 소유한다.

- strict protocol, fixture, arm, run-cell JSON contract
- 정확히 세 fixture(`M1`, `M2`, `M3`)와 세 arm(`R`, `E`, `A`)
- fixture/arm과 일대일로 연결된 서로 다른 9개 사전 배정 run ID
- protocol, fixture, input, arm manifest, arm contract의 exact SHA-256 검증
- repository 제외, symlink 해석, regular-file 요구, 상대 manifest reference와 byte ceiling
- consumer-supplied executor에 넘기는 path-free immutable validated input bytes
- protocol 전체 64 MiB input, fixture당 16개 input, output당 16 MiB·run당 16개/64 MiB ceiling
- run pending receipt, 안전한 failed terminal, partial-artifact receipt와 atomic terminal manifest
- 기대된 단 하나의 `executor_failed` run과 9개 run-terminal hash에 binding된 최종 matrix receipt
- 변조, receipt 없는 파일, duplicate run, incomplete run, 등록되지 않은 추가 run directory를 거부하는 replay 검증

이 package는 Plasma 내부에서 stable product error class만 import한다. Architecture check는 Report policy, Web, MCP, SQLite, connector, source-reader 또는 provider 구현을 직접 import하지 못하게 한다.

Synthetic harness는 run이 열린 동안 archive parent directory를 교체하지 않는다는 local user account와 다른 local process에 대한 trust model을 가진다. Static symlink/traversal escape를 막고 `lstat`/open/`fstat` 및 executor 호출 전후 directory identity 확인으로 교체를 감지하지만, hostile concurrent local filesystem writer를 막는 directory-descriptor fencing은 제공하지 않는다. 따라서 한 protocol만 소유하고 concurrent writer가 없는 전용 0700 archive에서 실행한다. Hard-link publication 뒤 남은 same-inode terminal temp file은 crash residue로 replay할 수 있지만, 다른 temp file은 거부한다. 취소를 포함한 executor 오류는 run identity를 재사용하지 않도록 안전한 failed terminal로 보존한다. Archive를 동시에 수정할 수 있는 환경에서 real-provider pilot을 시작하기 전에는 이 trust model을 다시 검토해야 한다.

## Synthetic Command

개발 command는 의도적으로 synthetic 전용이다.

```text
go run ./cmd/plasma-article-pilot-preflight \
  --synthetic \
  --archive-root /path/to/issue-461/synthetic-preflight-v4 \
  --repository-root /path/to/liquid2 \
  --protocol /path/to/issue-461/synthetic-preflight-v4/protocol.json \
  --protocol-sha256 <sha256> \
  --fail-run preflight-M2-A
```

`--synthetic`과 사전 배정된 강제 실패 run 하나가 필수다. 이 command는 real provider 실행을 지원하지 않는다. 기존 receipt는 cell을 다시 실행하지 않고 검증할 수 있다.

```text
go run ./cmd/plasma-article-pilot-preflight \
  --verify \
  --archive-root /path/to/issue-461/synthetic-preflight-v4 \
  --repository-root /path/to/liquid2 \
  --protocol /path/to/issue-461/synthetic-preflight-v4/protocol.json \
  --protocol-sha256 <sha256> \
  --matrix-sha256 <sha256>
```

## Durable Preflight 결과

Issue #461 archive에 다음 synthetic matrix 결과를 보존했다.

- protocol SHA-256:
  `a17a906b7bb4841c7b6398752c178d248399326382a7e9a1bad6e4d624d2af5e`
- matrix SHA-256:
  `fb504b6754d75580162ffd71368770d2e6cf097b75383a510246d9ea58b2946b`
- completed cell: 8
- 의도적으로 실패한 cell: 1 (`M2/A`)
- run directory: 9
- pending manifest: 9
- terminal manifest: 9
- 성공 placeholder artifact: 8

독립 replay에서 모든 matrix terminal digest, run identity, status가 일치했다. Durable JSON, Markdown, text artifact에는 repository path, scratch path, raw forced-provider error, session identifier가 없었다.

Archive 절대 경로는 local operational state이므로 이 공개 요약에 기록하지 않는다. 기본 local archive 정책에 따라 Git 밖의 Issue #461 experiment directory에 둔다.

## 아직 증명하지 않은 것

이 결과는 다음을 증명하지 않는다.

- R, E, A의 real provider 실행
- E와 A의 matched source, author, reader, factual-audit, repair adapter
- Article Narrative의 독서 개선
- manuscript의 사실 안전성 또는 독서 가치
- Article event, DB table, route, UI 제품화 필요성

다음 wave에서 bounded real-provider adapter를 이 harness에 연결할 수 있다. 실제 Article pilot을 시작하기 전에 현재의 immutable matrix와 failure-retention contract를 보존해야 한다.
