// Package partplan은 section_fanout 장문 report의 Part별 읽기 흐름 계획 단계를 소유한다.
//
// 이 패키지는 Part planning prompt, provider 요청, canonical part_plan event 저장/재생
// 계약을 맡는다. fan-out worker scheduling과 Part 간 순서는 root runner가 결정한다.
package partplan
