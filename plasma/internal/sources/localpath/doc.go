// Package localpath는 allowlist된 로컬 경로를 bounded source로 관찰하는 engine을
// 제공한다.
//
// 이 패키지는 root ID와 상대 경로만 받아 read, tree, grep을 수행하고 절대 경로와
// 파일 본문이 설정 경계 밖으로 나가지 않게 제한한다. 거부 결과에는 적용된 deny
// pattern이 포함될 수 있으므로, pattern 비공개가 필요하면 별도 동작 변경이 필요하다.
// 파일 내용을 저장 artifact로 저장할지 여부는 상위 source service가 결정한다.
package localpath
