# GitHub 운영 워크플로우

이 문서는 Liquid Workspace에서 GitHub milestone, issue, PR, branch, tag,
release를 사용하는 기준이다. GitHub는 자동 상태기계가 아니라 작업 기록,
리뷰 기록, 사용자 판단을 남기는 장부로 쓴다.

이 워크플로우의 기본 원칙은 세 가지다.

- Issue가 작업 상태와 사용자 확인의 source of truth다.
- 작업 에이전트는 issue 하나를 맡으면 PR 생성, 리뷰 대응, main 반영 확인까지
  계속 책임진다.
- 메인 에이전트는 상시 폴링 운영자가 아니라 사용자가 필요할 때 호출하는
  HITL 리뷰와 머지 보조자다.

## 기준 구조

기본 운영 구조는 `main + tag/GitHub Release`다.

```text
issue/milestone
-> agent dev branch
-> PR to main
-> user-triggered main review
-> user-authorized main merge
-> user issue review on main
-> vX.Y.Z tag + GitHub Release
```

`main`은 최신 통합 상태이자 다음 릴리즈 후보 기준선이다. 사용자가 설치하는
안정판은 `main` 브랜치가 아니라 `vX.Y.Z` tag와 GitHub Release asset이다.

실사용자가 생기고 병렬 PR이 상시적으로 `main`에 들어와 릴리즈 지점을 잡기
어려워지면 그때 `release/x.y` 브랜치를 도입한다.

```text
dev branch -> PR -> main -> release/x.y -> vX.Y.Z tag
```

`release/x.y`는 상시 `stage` 브랜치가 아니라 특정 버전 안정화가 필요할 때만
여는 브랜치다.

## 브랜치와 태그 역할

- `main`: 최신 통합 상태이며 다음 릴리즈 후보가 쌓이는 기본 브랜치다.
- `feat/*`, `fix/*`, `docs/*`, `chore/*`, `refactor/*`: 하나의 issue와 한 명의
  작업 에이전트가 소유하는 짧은 dev 브랜치다.
- `hotfix/*`: 이미 배포된 release에 긴급 수정이 필요할 때 쓰는 짧은 브랜치다.
  기본적으로 `main`에서 처리하고, 버전 라인 유지가 생기면 해당 `release/x.y`
  에서 처리한다.
- `release/x.y`: 기본 운영에는 없다. 병렬 반영으로 안정화 지점 고정이 필요할
  때만 만든다.
- `vX.Y.Z`: 사용자가 승인한 배포 snapshot이다. 사용자는 이 tag에 연결된
  GitHub Release asset을 설치한다.

dev 브랜치는 작업 완료 후 PR 머지와 함께 삭제한다. `main`, `release/x.y`, tag에는
에이전트가 직접 push하지 않는다.

## 외부 사례에서 가져온 결론

GitHub Flow는 짧은 브랜치, PR 리뷰, 머지 후 브랜치 삭제를 기본 흐름으로 둔다.
이 저장소의 기본 운영도 dev 브랜치를 짧게 유지하고 PR을 통합 단위로 삼는다.

GitLab Flow는 feature branch와 issue tracking을 결합하면서 stable 또는
production 브랜치를 둘 수 있다. 이 저장소는 지금 당장 상시 안정화 브랜치를
두지 않고, 필요가 생기면 `release/x.y`처럼 버전 라인 기준 브랜치를 추가한다.

Gitflow의 `develop`, `release`, `hotfix` 체계는 정기 버전 릴리스에는 강하지만
브랜치 수와 머지 규칙이 무겁다. 현재는 `develop`이나 상시 `stage`를 두지 않는다.

Trunk-based development는 작은 변경과 빠른 통합을 전제로 한다. 이 저장소는 직접
push를 허용하지 않고 PR을 요구하지만, 작은 PR을 빠르게 `main`에 통합하고 release
tag로 안정 지점을 고정하는 원칙은 따른다.

## 작업 제어 단위

Milestone은 사용자가 리뷰할 issue 후보 묶음이다. 버전이 정해졌으면 `v0.4.0`
처럼 이름을 붙이고, 아직 버전이 불명확하면 `2026-07-review`처럼 리뷰 묶음
이름을 쓴다.

Issue는 에이전트에게 줄 작업 명세이자 사용자가 최종 확인할 단위다. 하나의 issue는
한 명의 작업 에이전트가 소유하는 것이 기본이다. 구현 위험을 줄이기 위해 여러 dev
브랜치와 여러 PR로 나눌 수 있지만, 사용자는 PR 각각이 아니라 issue 단위로 실사용
확인과 완료 판단을 한다.

PR은 코드 변경 운반, 리뷰, 검증 기록의 단위다. PR은 작업 상태의 source of truth가
아니다. PR이 많아져도 사용자는 issue와 milestone을 보고 통합과 release 판단을 한다.

## 역할과 책임

모든 에이전트 세션은 기본적으로 일반 대화와 스티어링 모드에서 시작한다. GitHub
상태를 바꾸는 역할은 사용자가 명시적으로 발동해야 한다.

### 사용자

사용자는 다음 판단을 직접 한다.

- issue 방향, 우선순위, 수용 기준 확정.
- `issue:ready` 전환 승인.
- 메인 에이전트에게 PR 리뷰, 추가 검증, 머지를 요청할지 판단.
- `main` 반영 후 issue 기준 실사용 확인.
- milestone release 가능 여부와 tag/GitHub Release 생성.

사용자 판단은 PR formal approval이 아니라 명령과 issue comment로 남긴다. 예시는
`PR #37 리뷰해줘`, `문제 없으면 PR #37 머지해`, `이슈 #33 테스트 통과`,
`릴리즈 후보 승인`, `수정 필요`, `보류`다. 침묵은 승인으로 해석하지 않는다.

### 작업 에이전트

작업 에이전트는 사용자가 `작업 에이전트 A로 작동 시작: issue #123`처럼 발동했을
때 해당 issue만 다룬다. 작업 에이전트는 다음을 책임진다.

- issue 재조회와 claim 가능 여부 확인.
- dev branch 생성.
- draft PR 생성.
- 구현, 문서, 테스트, 자체 검증.
- PR ready 전환과 작업 요약 comment.
- 리뷰 피드백 반영.
- PR이 merge 또는 close될 때까지 리뷰 결과와 사용자 추가 요청 확인.

작업 에이전트는 리뷰 요청 뒤 "완료"라고 보고하고 세션을 끝내면 안 된다. 종료
조건은 다음 중 하나다.

- 연결 PR이 `main`에 머지되고 issue가 사용자 확인 대기로 넘어갔다.
- PR 또는 issue가 닫혔다.
- issue가 `issue:completed`, `issue:hold`, `issue:declined`가 됐다.
- 사용자가 역할 해제를 지시했다.
- 더 진행할 수 없는 blocker를 issue에 명확히 남기고 사용자에게 보고했다.

작업 에이전트가 대화 세션, 도구 제한, 외부 장애 때문에 더 유지될 수 없으면 GitHub
comment에 현재 head, 기다리는 reviewer, 다음 owner, 다음 확인 조건을 남긴다. 이유
없는 자의적 중단은 워크플로우 위반이다.

### 메인 에이전트

메인 에이전트는 상시 폴링 운영자가 아니다. 사용자가 특정 명령으로 호출할 때만
해당 범위를 처리하고, 요청한 작업이 끝나면 일반 모드로 돌아간다.

권장 호출 예시는 다음과 같다.

```text
PR #123 리뷰해줘
PR #123 Sentinel까지 보고 머지 가능 여부 알려줘
PR #123 문제 없으면 main에 머지해
이슈 #123 main 기준 테스트 항목 정리해줘
v0.0 릴리즈 가능 상태 점검해줘
```

메인 에이전트는 다음을 수행할 수 있다.

- PR 변경 범위, 연결 issue, milestone, 위험도를 확인한다.
- 필요한 로컬 테스트와 리뷰를 수행한다.
- 필요하면 Sentinel 같은 QA carrier를 붙인다.
- blocker가 있으면 PR comment로 수정 요청을 남긴다.
- blocker가 없으면 LGTM 또는 merge-ready comment를 남긴다.
- 사용자가 머지까지 위임한 명령을 줬다면 최종 head를 다시 확인한 뒤 `main`에
  머지한다.
- merge 후 issue를 닫지 않고 `issue:testing` 또는 `issue:completed`로 정리한다.

메인 에이전트는 milestone 전체를 자동 순찰하지 않는다. 사용자가 별도로 "계속
모니터링"을 요청하더라도 이는 현재 대화 세션이 살아 있는 동안의 best-effort
반복일 뿐이며 durable automation이 아니다. 지속 자동 운영이 필요하면 GitHub App,
GitHub Actions, 외부 봇처럼 별도 실행 주체를 설계한다.

### 트리아지 에이전트

트리아지 에이전트는 issue를 구체화하고, 질문을 남기고, `issue:ready` 후보를
제안한다. 사용자가 명시 승인했거나 사전 위임 scope가 있을 때만 `issue:ready`로
전환한다. 구현 claim, PR approval, merge, release 판단은 하지 않는다.

## Issue 규칙

Issue에는 다음 항목을 둔다.

- 목표: 무엇을 바꾸는가.
- 이유: 왜 필요한가.
- 수용 기준: 완료로 판단할 조건.
- 금지 범위: 이번 작업에서 하지 않을 것.
- 대상 영역: `liquid2`, `plasma`, `root/docs`, `.github` 등.
- 검증 방법: 실행할 테스트나 확인 절차.
- 릴리즈 영향: 사용자에게 보이는 변화인지, 내부 정리인지.

### Issue 댓글 작성

Issue 댓글은 시간순 운영 로그이면서 나중에 다시 읽는 결정 기록이다. 독자가 이전
댓글, PR, 커밋을 모두 따라가지 않아도 댓글 하나에서 해당 단계의 배경, 실행 내용,
결과, 다음 상태를 이해할 수 있게 쓴다. 이슈 본문 전체를 반복하지 말고 그 댓글을
이해하는 데 필요한 맥락만 포함한다.

- 에이전트가 작성한 댓글은 제목 바로 아래에 `작성: <agent>`처럼 작성 주체를
  남긴다. 사용자 판단을 대신 기록할 때는 `판단: 사용자 · 작성: <agent>`, Carrier
  조사 결과를 정리할 때는 `조사: <Carrier> · 작성 및 검증: <agent>`처럼 책임을
  구분한다. 기록한 에이전트를 판단 주체로 표현하지 않는다.
- 실험 코드값, 변형 ID, 내부 상태명은 처음 등장할 때 실제 의미를 함께 설명한다.
  코드값만 나열해 이전 문서나 댓글을 찾아보게 만들지 않는다.
- 승패, 점수, 자동 지표는 무엇을 몇 번 비교했는지와 숫자의 의미를 함께 쓴다.
- 자동 평가의 중간 결론, 에이전트의 해석, 사용자의 최종 판단을 명확히 구분한다.
- 링크와 커밋 SHA는 근거와 상세 내용을 보충하는 수단으로 사용한다. 댓글의 핵심
  의미를 링크 안에만 두지 않는다.
- `Summary`, `Role`, `Agent`, `State` 같은 운영 필드 나열로 댓글을 시작하지 않는다.
  사람이 읽는 설명을 먼저 쓰고, action briefing이나 handoff에 필요한 역할, 모델,
  branch, 예정 PR, 다음 행동은 그 뒤에 보조 정보로 남긴다.

Issue가 크면 구현 전에 여러 PR 단계나 하위 issue로 나눌지 결정한다. PR을 여러 개로
나누더라도 사용자의 확인 단위는 기본적으로 원래 issue다.

Issue 상태 label은 한 issue에 하나만 둔다.

- `issue:backlog`: 아이디어, 미정리 요구사항, 구체화 대기 상태다. 작업 에이전트가
  claim할 수 없다.
- `issue:ready`: 목표, 수용 기준, 금지 범위, 대상 영역, 검증 방법, 사용자 gate,
  release 영향이 충분히 정리되어 작업 에이전트가 claim할 수 있다.
- `issue:working`: 특정 작업 에이전트가 claim했고 dev branch나 PR 작업이 진행
  중이다.
- `issue:testing`: PR이 `main`에 머지됐지만 release 전 사용자 실사용 확인 또는
  release 후보 확인이 남아 있다.
- `issue:completed`: 사용자가 확인했고 release 후보로 검토할 수 있다. Issue는 아직
  닫지 않는다.
- `issue:hold`: 진행을 일시 중단했다.
- `issue:declined`: 진행하지 않기로 결정했다.

작업 에이전트가 claim할 수 있는 상태는 `issue:ready`뿐이다. `needs:decision`이
있으면 `issue:ready`보다 우선하므로 claim하지 않는다.

Claim 전에는 issue를 다시 조회해 `issue:ready`가 아직 유효하고 다른 claim이 없는지
확인한다. Claim할 때는 `issue:ready`를 `issue:working`으로 바꾸고 action briefing
comment에 역할, 모델, branch, 예정 PR, 다음 행동을 남긴다. 이미 claim된 issue를
발견한 후발 에이전트는 상태를 바꾸지 않고 물러난다.

오래 멈춘 claim은 후발 작업 에이전트가 직접 빼앗지 않는다. 사용자 또는 메인
에이전트가 stale claim 회수 comment를 남기고 `issue:working`을 해제한 뒤에만 다시
`issue:ready`로 돌릴 수 있다.

Issue는 PR이 `main`에 머지되어도 닫지 않는다. 사용자 확인이 남아 있으면
`issue:testing`과 `needs:user-review`를 둔다. 확인할 사용자 surface가 없거나 확인이
끝났으면 `issue:completed`와 `release:ready`를 붙일 수 있다. Issue 닫기는 사용자가
직접 수행한다.

사용자가 `수정 필요`를 남기면 기존 issue를 `issue:testing`에 남긴 채 수정 PR을
만들지, 새 follow-up issue로 뺄지, revert PR이 필요한지 판단한다.

## PR 규칙

PR은 항상 `main`을 대상으로 연다. PR은 하나 이상의 issue와 연결한다. Issue는 release
처리 전까지 열어 두므로 PR 본문은 기본적으로 `Refs`를 사용한다.

```text
Refs #123
```

`Fixes`, `Closes`, `Resolves`는 사용자가 PR merge와 동시에 issue를 닫기로 명시한
경우에만 쓴다.

PR에는 다음 항목을 적는다.

- 변경 요약.
- 연결 issue.
- 검증 결과.
- 사용자에게 보이는 변화.
- 사용자 확인 종류: `none`, `docs`, `runtime`, `release` 중 하나.
- 릴리즈 노트 필요 여부.
- 위험 또는 남은 작업.

PR 제목은 Conventional Commits 형식을 따른다. 예시는 `feat(plasma): add source review
queue`, `fix(liquid2): handle import error`다.

Draft PR은 작업 공유 상태이지 review-ready가 아니다. 작업 에이전트가 구현, 문서,
검증 결과, 남은 blocker를 정리한 뒤 draft를 해제한다. 별도 PR 상태 label로 같은
의미를 중복 표현하지 않는다.

PR 상태 label은 정상 워크플로우의 source of truth가 아니다. `pr:working`,
`pr:ready-for-review`, `pr:reviewing`, `pr:changes-requested`, `pr:awaiting-user`,
`pr:review-approved` 같은 label이 남아 있으면 deprecated 운영 흔적으로 보고 새 판단에
의존하지 않는다. 필요하면 정리하되, 새 상태 전이를 만들기 위해 PR label을 추가하지
않는다.

PR head가 바뀌면 이전 LGTM과 merge-ready 판단은 무효가 된다. PR이 닫히거나
merge되면 PR 상태 label은 더 이상 다음 행동 판단에 쓰지 않는다.

## 리뷰와 머지

자동 머지 운영은 기본값이 아니다. 사용자가 메인 에이전트에게 리뷰 또는 머지를
명시적으로 요청한다.

dev PR은 다음 조건을 만족해야 `main`에 머지될 수 있다.

- PR base가 `main`이다.
- 연결 issue가 있다.
- release 후보에 들어갈 작업이면 현재 milestone에 포함되어 있다.
- PR 제목이 Conventional Commits 형식을 따른다.
- 작업 에이전트가 실행한 검증 결과가 PR이나 briefing comment에 남아 있다.
- 필요한 로컬 리뷰, 테스트, QA가 끝났다.
- blocking finding이 없다.
- 모든 대화가 해결됐다.
- 충돌이 없다.
- merge 직전 head SHA가 리뷰한 head SHA와 같다.

메인 에이전트가 리뷰 또는 merge-ready를 남길 때는 comment에 head SHA, base branch,
base SHA, 검증 결과, unresolved thread 여부, 충돌 여부, 연결 issue와 사용자 확인
종류를 남긴다. 이 comment는 지속 상태가 아니라 특정 snapshot 판정이다.

사용자와 에이전트가 같은 GitHub 계정을 쓰는 동안에는 GitHub formal approval로
사용자 판단과 에이전트 판단을 구분할 수 없다. 따라서 PR approval은 사용자 gate가
아니다. 사용자 판단은 명령과 issue comment로 남긴다.

에이전트용 GitHub App 또는 bot 계정을 도입하더라도 "마지막 push 사용자와 다른 승인"
규칙은 메인 리뷰 강제를 위한 보호 장치로만 쓴다. 사용자 PR approval 요구는 기본
branch ruleset에 넣지 않는다.

## User gate 규칙

`User gate`는 사용자가 어떤 표면을 확인해야 하는지 나타낸다. PR 본문과 action
briefing comment에는 아래 값 중 하나만 쓴다.

| 값 | 의미 | 필요한 증거 | 다음 상태 |
|---|---|---|---|
| `none` | 사용자가 직접 확인할 UI, 문서 판단, release 후보 확인이 없다. | 작업 에이전트 검증과 리뷰 결과. | merge 후 `issue:completed`와 `release:ready`로 갈 수 있다. |
| `docs` | 문서, 설계, 운영 규칙을 사용자가 읽고 판단해야 한다. | main 반영 후 사용자 확인 또는 수정 요청 댓글. | merge 후 `issue:testing`으로 둔다. |
| `runtime` | UI, CLI, API, MCP 등 사용자가 실행해 볼 동작이 있다. | main 반영 후 사용자 확인 댓글. | merge 후 `issue:testing`으로 둔다. |
| `release` | `main` 반영 뒤 release 후보를 사용자가 확인해야 한다. | release 후보 확인 또는 release 승인 댓글. | 확인 전에는 `release:ready`를 붙이지 않는다. |

`needs:user-review`는 `User gate`가 `docs`, `runtime`, `release`인 경우 붙인다.
사용자 확인이 끝나면 제거한다. `User gate: none`이면 `needs:user-review`를 붙이지
않는다.

## Action briefing comment

상태 전환이나 역할 인계가 있는 운영 이벤트에는 action briefing comment를 남긴다.
일반 토론 댓글에는 강제하지 않는다.

필수 이벤트:

- 역할 발동 또는 해제.
- issue를 `issue:ready`로 전환.
- issue claim 또는 stale claim 회수.
- draft PR 생성.
- PR ready 전환.
- 비동기 리뷰나 QA job dispatch.
- 수정 요청 또는 수정 완료.
- LGTM 또는 merge-ready 판정.
- user issue/release approval 확인.
- `main` merge.
- `issue:testing`, `issue:completed`, `release:ready`, `issue:hold`,
  `issue:declined` 전환.

Comment는 사람이 먼저 읽는 `Summary`로 시작하고, 그 뒤 구조화 필드를 둔다.

```text
Summary:
Role:
Agent:
Model:
Scope:
Action:
Intent:
Result:
State:
Next:
User gate:
Verification:
Blockers:
```

`User gate`는 `none`, `docs`, `runtime`, `release` 중 하나로 쓴다. 검증을 실행하지
못했으면 `Verification`에 `not run`과 이유를 적는다. 남은 blocker가 없으면
`Blockers: none`이라고 명시한다.

PR 리뷰, LGTM, merge-ready, merge comment에는 아래 필드를 추가한다.

```text
Head/Base:
Status checks:
Invalidates when:
```

`Head/Base`에는 head SHA, base branch, base SHA를 적는다. `Status checks`에는 required
status/check 이름과 결론을 적는다. `Invalidates when`에는 `main` 전진, PR head 변경,
status/check set 변경, status/check conclusion 변경, 충돌 발생, unresolved thread
추가처럼 이전 판정을 다시 확인해야 하는 조건을 적는다.

## Milestone 규칙

Milestone은 작업 보관함이 아니라 release 후보 묶음이다. 이번 release나 이번 리뷰
후보에 넣을 생각이 없는 issue는 milestone에 넣지 않는다.

Milestone에는 다음 내용을 둔다.

- 목표: 어떤 사용자 가치나 안정화 목표를 확인할 것인가.
- 포함 issue: 이번 후보에 들어갈 작업.
- 제외 범위: 이번 후보에서 의도적으로 뺄 작업.
- 실사용 리뷰 기준: 사용자가 어떤 흐름을 직접 확인할 것인가.
- release 판단 기준: tag를 찍어도 되는 조건.

Milestone의 모든 issue가 닫혔다고 자동으로 release하지 않는다. 사용자가 `main`
상태를 확인하고 release 여부를 결정한다.

Milestone을 여러 개 미리 만들어 두더라도 `main`은 현재 release 후보 milestone의 통합
지점으로 유지한다. 현재 milestone에 속하지 않은 issue의 PR은 작업을 시작하거나 draft
PR로 열 수 있지만, 사용자가 milestone을 옮기기로 결정하기 전에는 `main`에 머지하지
않는다. 다음 milestone 작업이 현재 release에 꼭 필요해지면 먼저 issue의 milestone을
현재 후보로 옮긴 뒤 PR을 머지한다.

## Label 규칙

GitHub label은 처음부터 많이 만들지 않고, 에이전트 투입 가능 여부와 릴리즈 위험
판단에 필요한 최소 세트만 둔다.

Issue 상태 label:

- `issue:backlog`: 구체화 전, claim 불가.
- `issue:ready`: 구체화 완료, claim 가능.
- `issue:working`: 작업 에이전트가 claim하고 작업 중.
- `issue:testing`: `main` 반영 후 release 전 사용자 확인 중.
- `issue:completed`: release 후보 검토 완료.
- `issue:hold`: 일시 중단.
- `issue:declined`: 진행하지 않음.

분류와 gate label:

- `type:feat`: 사용자 기능 추가.
- `type:fix`: 버그 수정.
- `type:docs`: 문서 변경.
- `type:refactor`: 동작 변경을 의도하지 않은 구조 정리.
- `type:chore`: 설정과 운영 보조 작업.
- `area:liquid2`: Liquid2 제품 영역.
- `area:plasma`: Plasma 제품 영역.
- `area:github`: GitHub Actions, issue, PR, release 운영 영역.
- `area:docs`: 루트 또는 제품 문서 영역.
- `risk:low`: 영향 범위가 작고 되돌리기 쉬움.
- `risk:medium`: 사용자 흐름이나 공유 경계에 영향 가능.
- `risk:high`: 저장소 구조, release, 데이터, 보안, 공개 API 영향.
- `release:blocker`: release 전에 반드시 해결해야 함.
- `release:ready`: `main`에 머지됐고 release 후보로 검토 완료됨.
- `needs:decision`: 구현 전에 사용자 결정이 필요함.
- `needs:user-review`: 사용자가 직접 동작, 문서, 또는 release 후보를 확인해야 함.

PR 상태 label은 새로 쓰지 않는다. 기존 `pr:*` label은 원격에 남아 있어도 새
워크플로우의 판단 근거가 아니다. 정리가 필요하면 별도 라벨 정리 작업으로 삭제하거나
deprecated 설명을 붙인다.

상태 label을 바꿀 때는 같은 축의 기존 상태 label을 먼저 제거하고 새 상태 label 하나만
남긴다. 알 수 없는 `issue:*` label은 상태 판단에 사용하지 않고 사용자에게 확인한다.

## Release 규칙

`main`에 머지된 변경은 아직 release가 아니다. 사용자는 milestone 또는 현재 `main`
상태를 직접 확인하고 release할 commit을 고른다.

기본 release 흐름:

1. milestone에 포함된 issue가 `release:ready`인지 확인한다.
2. `main` 최신 상태를 로컬 release/dev 서버로 실행해 본다.
3. 문제가 있으면 새 fix PR 또는 revert PR을 `main`에 넣는다.
4. release 가능하다고 판단한 commit에 `vX.Y.Z` tag를 만든다.
5. GitHub Release를 만들고 앱 또는 바이너리 asset을 첨부한다.
6. 사용자는 GitHub Release asset을 받아 설치한다.
7. release 후 사용자가 issue를 닫는다.

이미 release된 버전에 긴급 패치가 필요하고 `main`이 다음 버전 작업으로 많이 앞서간
상태라면, 해당 tag에서 `release/x.y` 브랜치를 만들고 패치한 뒤 `vX.Y.Z+1` tag를
만든다. 같은 수정은 `main`에도 반영한다.

## Branch ruleset 권장값

`main`:

- direct push 금지.
- force push와 삭제 금지.
- PR 필수.
- 대화 해결 필수.
- PR branch 최신화 권장.
- formal reviewer approval은 bot 계정 분리가 가능해진 뒤 다시 판단한다.

dev branch:

- 에이전트가 자유롭게 push할 수 있다.
- 작업 완료 후 PR 머지와 함께 삭제한다.
- 오래 열린 브랜치는 milestone 리뷰 전에 다시 쪼개거나 닫는다.

`release/x.y`:

- 기본 운영에는 만들지 않는다.
- 생성 시 `main`과 같은 보호 규칙을 적용한다.
- 해당 버전 안정화와 긴급 패치만 받는다.
- release 후 유지 필요가 사라지면 보관하거나 삭제한다. tag는 삭제하지 않는다.

## 운영 체크리스트

작업 시작 전:

- issue가 있다.
- issue에 수용 기준과 금지 범위가 있다.
- 병렬 작업 묶음이면 milestone이 있다.
- `needs:decision`이 없는 상태다.
- 작업 에이전트는 `issue:ready` 상태만 claim한다.
- 역할 발동 명령에 역할과 scope가 모두 있다.

작업 PR ready 전:

- PR base가 `main`이다.
- PR이 issue를 `Refs #...`로 참조한다.
- PR의 issue가 현재 release 후보 milestone에 속해 있거나, 사용자가 현재 후보에
  포함하기로 명시적으로 결정했다.
- draft가 해제됐다.
- 작업 에이전트가 실행한 검증과 남은 blocker를 남겼다.
- 사용자 확인 종류가 `none`, `docs`, `runtime`, `release` 중 하나로 적혀 있다.

메인 리뷰 전:

- 사용자가 리뷰 대상 PR이나 milestone을 명시했다.
- 메인 에이전트가 수행할 범위가 리뷰만인지, merge 가능 시 머지까지인지 명확하다.
- 기존 LGTM이나 merge-ready comment가 있더라도 head/base가 바뀌었는지 다시 확인한다.

merge 전:

- 사용자가 머지를 명시적으로 요청했거나, 최초 명령에 "문제 없으면 머지"가 포함되어
  있다.
- pending reviewer/QA job이 없다.
- head SHA와 base 최신성을 다시 확인했다.
- 미해결 대화, 충돌, 필수 검증 실패가 없다.
- 이전 merge-ready 판정 뒤 `main`이나 PR head가 바뀌었다면 다시 판정했다.

release 전:

- milestone에 포함된 issue가 모두 merge, 제외, 또는 후속 작업으로 정리되었고 release
  대상 issue에는 `release:ready`가 붙어 있다.
- `main` 상태에서 실사용 리뷰가 끝났다.
- 거절된 변경은 revert PR 또는 fix PR로 처리되었다.
- release tag 이름과 release note 범위가 명확하다.

`release/x.y` 도입 전:

- `main`에 병렬 PR이 상시적으로 들어와 release 지점을 잡기 어렵다.
- 이미 release된 사용자 버전의 패치 유지가 필요하다.
- 해당 버전 라인을 언제 닫을지 기준이 있다.

## 참고 자료

- GitHub Flow: https://docs.github.com/en/get-started/using-github/github-flow
- GitHub Pull Requests: https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/proposing-changes-to-your-work-with-pull-requests/about-pull-requests
- GitHub Milestones: https://docs.github.com/en/issues/using-labels-and-milestones-to-track-work/about-milestones
- GitHub linked issues and PRs: https://docs.github.com/en/issues/tracking-your-work-with-issues/using-issues/linking-a-pull-request-to-an-issue
- GitHub protected branches: https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches
- GitHub rulesets: https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/about-rulesets
- GitHub merge queue: https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/configuring-pull-request-merges/managing-a-merge-queue
- GitLab Flow: https://about.gitlab.com/topics/version-control/what-is-gitlab-flow/
- Atlassian feature branch workflow: https://www.atlassian.com/git/tutorials/comparing-workflows/feature-branch-workflow
- Atlassian Gitflow workflow: https://www.atlassian.com/git/tutorials/comparing-workflows/gitflow-workflow
- Atlassian trunk-based development: https://www.atlassian.com/continuous-delivery/continuous-integration/trunk-based-development
