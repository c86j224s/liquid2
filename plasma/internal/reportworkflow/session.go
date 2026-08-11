package reportworkflow

// SessionLineage는 stage 사이에 전달되는 report session 계약의 이름 있는 모음이다.
//
// root runner는 이 값을 durable 상태로 저장하지 않고 plan 출력에서 directdraft 입력으로
// 그대로 변환한다. source of truth는 계속 ledger event payload와 raw artifact다.
type SessionLineage struct {
	ReportPlanSessionID        string
	ReportSessionID            string
	PreReportResearchSessionID string
	ForkSourceSessionID        string
	SessionChainKind           string
}
