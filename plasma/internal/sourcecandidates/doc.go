// Package sourcecandidates는 에이전트가 제안한 소스 후보의 정규화와 승인 전
// 원문 staging lifecycle을 소유한다. 후보와 staged artifact는 승인된 source
// snapshot이 아니며, 승인·기각 결정과 정식 source 생성은 이 경계 밖에 둔다.
package sourcecandidates
