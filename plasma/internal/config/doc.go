// Package config는 Plasma 실행 설정을 파일, 환경 변수, 명령행 인자에서 읽어
// 하나의 Config로 합친다.
//
// 우선순위와 기본값은 이 패키지의 계약이다. 호출자는 Config를 읽은 뒤 product
// service나 transport adapter에 주입하며, 이 패키지는 서버를 시작하거나 runtime
// state를 직접 만들지 않는다.
package config
