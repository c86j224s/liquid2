# 최소 에이전트 capability profile 제품 기록

상태: 이슈 #349 worktree에서 구현 및 offline test 완료. commit, merge, deploy,
release는 하지 않았다.
날짜: 2026-08-18.
소유 맥락: 이슈 #349.
리뷰 맥락: 이전의 넓은 계획은 Luna 적대적 리뷰를 거쳤고, 이후 코드 정찰 결과 이번
제품 변경은 이 문서의 최소 범위로 축소했다.

## 목표

추가 provider 실험 없이 Plasma의 기존 capability control을 재사용해 caller가 선택하는
작고 versioned된 capability profile을 일반 제품 경로에 적용한다.

이 기록은 이 파일에 있던 과대 설계를 대체한다. 이번 구현은 별도 operation-ID 계층을
추가하지 않고, 일반 Codex inference를 app-server로 옮기지 않으며, 호출 전 complete
runtime catalog assertion을 요구하지 않는다.

## 근거 경계

### 기존 메커니즘 근거

이전의 격리된 Luna 측정은 한 가지 좁은 메커니즘만 확인했다. 실제 provider-facing
capability bundle이 줄면 provider가 보고한 input usage도 줄었다.

균형을 맞춘 fresh-session 11쌍에서:

- 모든 candidate 호출은 대응 current보다 input token을 4,206개 적게 보고했다.
- 단순 synthetic task 11개 모두 semantic correctness가 양쪽 arm에서 통과했다.
- 대표 1쌍은 direct tool 3개 대 2개, deferred tool 164개 대 13개를 노출했다.
- 같은 쌍은 model-input 59,112바이트 대 39,947바이트, wire-request 101,159바이트 대
  44,622바이트를 직렬화했다.

측정 candidate는 broad experimental bundle이었고 제품 필수 Plasma 도구를 잘못
제거했다. 따라서 4,206-token 차이는 특정 도구의 marginal effect도 아니고 제품
profile의 예상 절감량도 아니다. Adoption threshold로도 사용하지 않는다.

### 추가 provider 실험 없음

이번 구현은 unit, fake-provider, adapter, package, 전체 offline Go test만 사용했다. Luna
또는 다른 provider를 새로 호출하지 않았다. 구현 profile의 제품 절감량을 주장하지 않는다.

### Artifact와 privacy 경계

이 기록에는 raw prompt, provider request/response, provider JSONL, provider session ID,
credential, private mission content, 실제 source content, private runtime path가 없다. Raw
experiment material은 저장소 밖에 유지한다.

## 제품 capability 계약

일반 conversation과 자율 workflow step은 Plasma research/control 도구 16개를 모두
유지한다.

| Capability 그룹 | 도구 | 판단 |
| --- | --- | --- |
| Mission orientation | `plasma.research.outline`, `plasma.research.changes` | 유지 |
| Mission discovery | `plasma.research.list`, `plasma.research.grep` | 유지 |
| Mission grounding | `plasma.research.read`, `plasma.research.references` | 유지 |
| 승인 source 접근 | `plasma.sources.list`, `plasma.sources.read` | 유지 |
| Live local source 접근 | `plasma.sources.tree`, `plasma.sources.grep` | 유지 |
| Source discovery | `plasma.sources.search` | 유지 |
| Candidate review | `plasma.sources.candidates.propose`, `plasma.sources.candidates.read` | 유지 |
| Presentation 검증 | `plasma.mermaid.validate` | 유지 |
| Workflow control | `plasma.workflow.status`, `plasma.workflow.stop` | 유지 |

Source discovery, candidate review, workflow control 도구를 제거했던 이전 experimental
candidate는 제품 profile이 아니다.

승인된 live local-path source는 계속 Plasma source 도구를 사용한다. 그래야 승인된 source
경계 안에서 읽고 `source.observed` provenance를 만들 수 있다. Provider의 work-directory
reader는 mission-bound source read를 대체하지 않는다.

Source search는 승인 source가 아니라 review candidate를 반환한다. Candidate propose와
staged-candidate read를 유지하며, staged material은 사용자가 승인하기 전까지 unapproved이고
기본 report source set 밖에 있다.

## 구현한 최소 profile

Provider-neutral 계약은 작은 typed pair 하나다.

```go
type ProfileID string

type Profile struct {
    ID       ProfileID
    Revision string
}
```

Revision `1`은 세 profile을 정의한다.

| Profile | 소유자와 용도 | 호환 동작 |
| --- | --- | --- |
| `legacy.v1` | Historical session과 profile을 선택하지 않은 caller | Profile 도입 전 동작 유지 |
| `research.v1` | 새 conversation과 workflow session | Plasma research 계약을 유지하고 지원되는 불필요 ambient capability를 제거 |
| `workflow_goal_draft.v1` | Request-local workflow goal draft 한 번 | Helper 호출에서 제품 도구와 session persistence를 비활성화 |

Zero-value request는 immutable `legacy.v1`로 resolve한다. 새 field가 존재한다는 이유만으로
기존 report와 compatibility caller의 동작이 조용히 바뀌지 않는다.

Profile은 제품 caller가 선택한다. Prompt, user text, `mcp_mode`, heuristic은 profile을
선택하지 않는다.

## Caller 연결

### Conversation과 workflow

- 새 browser 또는 CLI conversation은 `research.v1`을 선택한다.
- 새 browser 또는 CLI workflow는 `research.v1`을 선택한다.
- Resumed conversation/workflow는 provider session과 함께 persist된 profile을 resolve한다.
- Workflow step, automatic compaction, manual compaction, compaction 뒤 retry는 같은 profile
  ID와 revision을 전달한다.
- 성공 response와 compaction event는 profile identity를 기록한다.
- 알 수 없는 profile ID 또는 지원하지 않는 revision은 provider 실행 전에 실패한다.

### Historical session

Provider session ID는 있지만 profile field가 없는 historical success response는
`legacy.v1` revision `1`로 deterministic mapping한다. Resume 중 `research.v1`로
upgrade하지 않는다.

### Report

기존 report stage별 MCP allowlist와 report binding이 계속 stage 계약이다. 이를 중복하는
operation-ID 계층은 추가하지 않았다.

Report가 기존 research session 또는 그 isolated fork를 사용하면 작은 executor wrapper가
모든 report-stage request에 source session profile을 적용한다. Wrapper는 optional fork와
fork-readiness interface를 보존한다. Report artifact lineage를 통해 report session과
pre-report research session은 이후 resume, humanize, patch에서도 같은 profile을 상속한다.

기록된 research lineage가 없는 report-only path는 compatibility를 위해 `legacy.v1`을
유지한다. 더 좁은 report-only profile은 만들지 않았다.

### Workflow goal draft

Goal-draft route는 `workflow_goal_draft.v1`을 명시적으로 선택한다. Prompt에서 profile을
추론하지 않는다.

## Provider mapping

### Claude

`research.v1`은 기존 exact Plasma MCP configuration, built-in allowlist, deny set,
slash-command disablement, safe-mode isolation을 재사용한다. Plasma research/control 도구
16개는 유지한다.

`workflow_goal_draft.v1`에서 adapter는:

- `--tools ""`를 전달한다.
- Plasma MCP config를 생략한다.
- Safe mode를 사용한다.
- Slash command를 비활성화한다.
- Non-persistent helper session을 사용한다.

이는 CLI argument와 생성 MCP config 경계에서 구현된 Claude goal-draft zero-tool mapping이다.

### Codex

일반 inference는 계속 `codex exec`와 `codex exec resume`을 사용한다. Native Codex
compaction은 기존 app-server compaction path를 유지한다. 이번 변경은 app-server
inference를 추가하지 않는다.

`research.v1`은 request-scoped Codex config로 Plasma가 사용하지 않는 지원 가능한 ambient
feature를 비활성화한다.

```text
features.tool_suggest=false
features.recommended_plugins=false
tools.experimental_request_user_input.enabled=false
features.apps=false
features.plugins=false
features.multi_agent=false
features.shell_tool=false
features.view_image=false
tools.update_plan.enabled=false
skills.include_instructions=false
project_doc_max_bytes=0
```

이 profile은 Codex `--ignore-user-config`를 의도적으로 사용하지 않는다. Configured
service에 도달하는 데 필요한 provider connection, authentication, model configuration까지
제거할 수 있기 때문이다.

`workflow_goal_draft.v1`은 Plasma MCP tool도 생략하고 ephemeral session을 사용한다. 이는
**tool-minimized Codex goal-draft mapping**이지 검증된 zero-tool catalog가 아니다. 현재 공개
control로는 모든 direct, deferred, hidden, Code Mode, apply-patch surface 부재를 입증할 수
없다. 제품과 이 문서는 Codex mapping을 완전한 tool-free라고 부르지 않는다.

## Session lineage 계약

이번 변경의 durable identity는 다음과 같다.

```text
executor + provider session ID + capability profile ID + profile revision
```

Profile은 성공한 conversation/workflow response와 compaction event에 저장한다. Resume은
그 pair를 resolve하고 검증한다. 알 수 없는 profile/revision은 현재 default로 fallback하지
않고 명확하게 실패한다.

Report artifact event마다 profile field를 모든 report pipeline type에 중복 관통시키지
않았다. 대신 기록된 same-session 또는 pre-report session lineage를 따라 이미 persist된
profile을 상속한다. 작은 변경으로 provider-session 불변식을 보존한다.

이번 구현은 다음을 추가하지 않는다.

- 별도 capability operation ID
- Adapter version 또는 catalog fingerprint
- Complete direct/deferred/hidden catalog manifest
- 새 usage telemetry
- Automatic session migration

## Offline acceptance

Test 범위:

- Zero value가 `legacy.v1`로 resolve됨
- 알 수 없는 profile과 revision 거부
- Codex/Claude research-profile mapping
- Goal-draft tool disable과 ephemeral 동작
- Claude `--tools ""`와 MCP config 생략
- 새 browser/CLI conversation과 workflow profile 선택
- Normal step과 compaction을 통한 workflow adapter propagation
- Historical session의 `legacy.v1` 복원
- Isolated/fresh report의 pre-report research profile 복원
- Report executor wrapper의 fork capability 보존
- Workflow goal-draft request 선택
- 기존 report allowlist와 provider-MCP acceptance suite
- Package dependency boundary

최종 gate는 저장소 전체 offline Go suite다.

```text
go test ./...
```

이 acceptance record에는 live provider call이 없다.

## 알려진 제한

- Codex goal-draft profile은 provider tool 0개를 입증하지 못했다.
- Configured provider connection을 손상시킬 위험 없이 Codex ambient MCP server 전체를
  제거할 수 없으며, 이번 변경은 완전 격리를 주장하지 않는다.
- Complete provider-facing catalog assertion은 없다.
- 올바른 16-tool profile의 production token 절감량은 측정하지 않았다.
- Historical session은 의도적으로 새 ambient reduction 대신 `legacy.v1`을 유지한다.
- Research lineage가 없는 report-only session은 의도적으로 legacy를 유지한다.
- Production caller가 없는 dormant proposal-extraction helper는 변경하지 않았다.

이 항목들은 통과 assertion이 아니라 명시적인 제한이다.

## 보류한 변경

이슈 #349 최소 구현 범위 밖:

- Plasma research/control 도구 16개 중 하나라도 축소
- Operation ID 추가
- 일반 Codex inference를 app-server로 전환
- Full provider catalog manifest 또는 fail-before-inference catalog check 구축
- 더 좁은 report-only profile 생성
- 일반 user turn의 heuristic tool-free 분류
- Development/release server 또는 DB 변경
- 추가 provider 실험
- 별도 사용자 gate 없는 commit, pull request, merge, deploy, release

## 채택 판단

이 문서의 최소 typed profile, caller wiring, session lineage persistence, 기존 report 계약
재사용, Claude goal-draft tool disable, 지원되는 Codex ambient reduction만 채택한다.

이전 experimental tool list와 과대 설계된 operation-ID, app-server-inference,
complete-catalog architecture는 채택하지 않는다. 감수하는 비용은 작은 profile type,
provider mapping, lineage field, caller propagation이다. 감수하는 제한은 Codex가 지원되는
request-scoped control만으로 축소되며 완전한 tool-free 또는 완전 관측 가능 상태로
표현되지 않는다는 점이다.
