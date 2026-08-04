// Package workflow는 미션의 자율 진행 workflow run 실행과 process-local
// supervisor를 소유한다. 단계별 원장 이벤트, 에이전트 호출, control decision,
// 취소와 durable 재조정을 조율하지만 provider와 저장소 구현은 consumer-side
// 포트를 통해서만 사용한다.
package workflow
