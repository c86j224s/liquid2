// Package sourceevents는 source snapshot 관련 장부 payload를 안정적으로 만드는
// 작은 builder 경계다.
//
// connector, 업로드 파일, Confluence update 등 source 종류별 payload 모양을
// 통일하지만, snapshot 생성 자체나 source 상태 전이는 app service가 수행한다.
package sourceevents
