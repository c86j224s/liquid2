// Package reportworkflow는 Plasma 보고서 생성 graph의 제품 전용 typed entrypoint다.
//
// root 패키지는 mode/strategy/final-tail topology 선택과 stage 간 typed 값 변환만 맡는다.
// prompt, MCP allowlist, provider 호출, durable replay 세부 정책은 plan/directdraft 같은
// 단계 패키지가 소유한다. caller가 stage 목록, prompt, tool allowlist, retry budget을 주입하는
// generic graph runner는 이 경계에 두지 않는다.
package reportworkflow
