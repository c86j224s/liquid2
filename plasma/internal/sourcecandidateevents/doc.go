// Package sourcecandidateevents는 source 후보 staging 이벤트를 읽고 해석한다.
//
// 이 패키지는 승인 전 후보와 snapshot artifact의 현재 열림/닫힘 상태를 계산한다.
// 후보를 실제 mission source로 승격하거나 새 snapshot을 만드는 일은 하지 않는다.
package sourcecandidateevents
