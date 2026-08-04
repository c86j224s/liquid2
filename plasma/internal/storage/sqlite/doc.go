// Package sqlite는 Plasma application port를 SQLite 저장소에 연결하는 adapter다.
//
// 이 패키지는 SQL schema, transaction, projection query, maintenance 작업의
// persistence shape을 소유한다. 제품 상태 전이의 의미와 정책은 app/reporting/
// workflow 계층의 요청 계약을 따른다.
package sqlite
