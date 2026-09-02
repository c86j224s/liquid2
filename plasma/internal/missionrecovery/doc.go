// Package missionrecovery는 lazy full-detail mission 복구를 위한 유한 coordinator다.
// 고정 복구 순서를 검증하고, step 결과와 failure policy를 관리하며, 주입된 report
// lock의 범위를 소유한다. 실제 복구 callback은 capability와 Web 조립부가 제공하고,
// startup DB-only 복구와 provider/executor startup 작업은 이 package 밖에 둔다.
package missionrecovery
