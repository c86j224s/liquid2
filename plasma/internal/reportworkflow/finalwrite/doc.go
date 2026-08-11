// Package finalwrite는 V2/V3 장문 final edit writer stage의 typed 계약을 소유한다.
//
// 이 패키지는 final writer prompt, MCP allowlist, durable stage replay, 단일 provider
// 실행만 맡는다. graph 순서와 다음 stage 선택은 reportworkflow root가 소유하며,
// 다른 stage package를 import하지 않는다.
package finalwrite
