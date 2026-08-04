// Package mermaid는 Plasma 보고서와 대화에 들어갈 Mermaid source의 서버 측
// 사전 검증 규칙을 담는다.
//
// 여기서 수행하는 검증은 알려진 parser 호환성 문제와 위험한 문법을 빠르게
// 걸러내기 위한 것이다. 브라우저 렌더링 성공을 보장하지 않으며, 다이어그램을
// 자동 수정하거나 보고서 내용을 판단하지 않는다.
package mermaid
