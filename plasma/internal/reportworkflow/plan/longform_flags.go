package plan

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
)

// LongFormPartEditEnabled는 canonical plan payload에 고정되는 Part edit 활성 규칙이다.
func LongFormPartEditEnabled(profile string) bool {
	return reportprompt.IsNarrativeContract(profile)
}

// LongFormPartPlanningEnabled는 section_fanout에서만 실행 가능한 Part planning 활성 규칙이다.
func LongFormPartPlanningEnabled(profile string) bool {
	return reportprompt.IsPartConnectiveEconomyVoice(profile) ||
		reportprompt.IsPartConnectiveSubjectDirectSynthesis(profile)
}

// LongFormFinalEditPipelineForPlan은 finalization tail 선택을 canonical plan event에 얼리는 규칙이다.
func LongFormFinalEditPipelineForPlan(profile string) string {
	if LongFormPartEditEnabled(profile) {
		return reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3
	}
	return ""
}

func longFormActivationFlags(event ledger.Event) (bool, bool, error) {
	var payload struct {
		PartEditEnabled     bool   `json:"part_edit_enabled"`
		PartPlanningEnabled bool   `json:"part_planning_enabled"`
		SessionChainKind    string `json:"session_chain_kind"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return false, false, fmt.Errorf("%w: report plan payload is invalid", producterror.ErrConflict)
	}
	if payload.PartPlanningEnabled && !payload.PartEditEnabled {
		return false, false, fmt.Errorf("%w: Part planning requires Part edit", producterror.ErrConflict)
	}
	if payload.PartPlanningEnabled && strings.TrimSpace(payload.SessionChainKind) != "section_fanout_report" {
		return false, false, fmt.Errorf("%w: Part planning requires section_fanout lineage", producterror.ErrConflict)
	}
	return payload.PartEditEnabled, payload.PartPlanningEnabled, nil
}

// LongFormActivationFlags는 canonical plan event의 optional node activation만 검증한다.
// Web 호환 wrapper는 값을 재해석하지 않고 이 결과만 표시용 state로 변환한다.
func LongFormActivationFlags(event ledger.Event) (bool, bool, error) {
	return longFormActivationFlags(event)
}

func partPlanningParent(event ledger.Event, pendingEventID string) (reporting.PartPlanParentState, error) {
	parent, ok, err := reporting.DecodePartPlanParent(event, pendingEventID, event.EventID)
	if err != nil {
		return reporting.PartPlanParentState{}, err
	}
	if !ok {
		return reporting.PartPlanParentState{}, fmt.Errorf("%w: Part planning parent is missing", producterror.ErrConflict)
	}
	return parent, nil
}
