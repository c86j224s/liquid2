// Package workflowruns는 workflow 실행 요청, 중지 요청, terminal event 기록을
// 조건부 장부 append로 처리하는 application service 경계다.
//
// 이 패키지는 동시 실행 금지, stale/pending 상태, idempotent terminal 기록 같은
// 실행 생명주기 계약을 소유한다. 실제 agent process 실행과 화면 표시 방식은 각각
// workflow runner와 web adapter의 책임이다.
package workflowruns
