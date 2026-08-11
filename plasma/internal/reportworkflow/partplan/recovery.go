package partplan

import (
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

// RecoverInput은 durable Part-plan event가 현재 canonical plan lineage에 속하는지
// 검증하기 위한 불변 기대값이다.
type RecoverInput struct {
	Event                        ledger.Event
	MissionID                    string
	PendingEventID               string
	PlanEventID                  string
	PartCount                    int
	AgentExecutor                string
	AgentModel                   string
	AgentReasoningEffort         string
	AgentSelectionSource         string
	ReportSessionPolicy          string
	ReportSessionPolicySelection string
	GenerationGuidanceProfile    string
	GenerationGuidanceSHA256     string
	SessionChainKind             string
	ReportPlanSessionID          string
}

// Recover는 저장된 Part-plan event를 stage의 typed Output으로 복구한다.
// root는 반환된 zero-based PartIndex만 집계하며 payload decoding 정책을 갖지 않는다.
func Recover(input RecoverInput) (Output, bool, error) {
	stored, ok, err := reporting.DecodeStoredPartPlan(input.Event, reporting.StoredPartPlanExpectation{
		MissionID: input.MissionID, PendingEventID: input.PendingEventID, PlanEventID: input.PlanEventID,
		PartCount: input.PartCount, AgentExecutor: input.AgentExecutor, AgentModel: input.AgentModel,
		AgentReasoningEffort: input.AgentReasoningEffort, AgentSelectionSource: input.AgentSelectionSource,
		ReportMode: reportexecution.ModeLongForm, ReportSessionPolicy: input.ReportSessionPolicy,
		ReportSessionPolicySelection: input.ReportSessionPolicySelection,
		GenerationGuidanceProfile:    input.GenerationGuidanceProfile,
		GenerationGuidanceSHA256:     input.GenerationGuidanceSHA256,
		SessionChainKind:             input.SessionChainKind,
		ReportPlanSessionID:          input.ReportPlanSessionID,
	})
	if err != nil || !ok {
		return Output{}, ok, err
	}
	return Output{
		PartIndex: stored.PartIndex - 1, Brief: stored.Brief,
		ProviderSessionID: stored.ProviderSessionID, Event: stored.Event, Recovered: true,
	}, true, nil
}
