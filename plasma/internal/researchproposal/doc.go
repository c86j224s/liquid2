// Package researchproposal owns proposal bundle identity, submission validation,
// decision payload rules, and terminal state transitions. Application services
// retain ledger lookup, storage, and atomic commit orchestration; transport and
// persistence adapters do not define proposal policy.
//
// 이 패키지는 proposal bundle identity, 제출 검증, decision payload 규칙과
// terminal state transition을 소유한다. Application service에는 ledger 조회,
// 저장과 atomic commit 조율만 남고 transport와 persistence adapter는 proposal
// 정책을 정의하지 않는다.
package researchproposal
