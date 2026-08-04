// Package pdfdocument는 PDF 바이트에서 텍스트와 기본 문서 정보를 추출한다.
//
// 이 패키지는 PDF 판별, 전체 추출, chunk 추출을 제공하는 낮은 수준의 parser
// parser다. 업로드 source 정책, artifact 저장, 사용자 오류 메시지 구성은
// sourceingest와 app/web 계층에서 맡는다.
package pdfdocument
