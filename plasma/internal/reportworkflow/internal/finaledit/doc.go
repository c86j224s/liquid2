// Package finaledit은 V1/V2/V3 final edit stage들이 공유하는 반복 실행,
// provider session fork, durable replay, gate resume helper만 담는다.
//
// prompt, MCP allowlist, stage identity, graph 순서는 이 내부 패키지의 책임이 아니다.
// 각 public stage package가 고정 계약을 소유하고 root reportworkflow runner가 typed
// 출력과 입력을 연결한다.
package finaledit
