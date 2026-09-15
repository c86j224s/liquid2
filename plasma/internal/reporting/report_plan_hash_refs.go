package reporting

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reporting/reportdocument"
)

// ReportPlanHash는 report plan의 안정 JSON 해시를 계산한다.
func ReportPlanHash(plan any) (string, json.RawMessage, error) {
	encoded, err := json.Marshal(plan)
	if err != nil {
		return "", nil, fmt.Errorf("%w: report plan cannot be encoded", producterror.ErrInvalidInput)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), encoded, nil
}

// ReportPlanRefs는 plan 안의 source reference를 dedupe된 목록으로 모은다.
func ReportPlanRefs(plan any) []ReportPlanSourceRefs {
	refs := []ReportPlanSourceRefs{}
	switch value := plan.(type) {
	case ReportPlan:
		for _, section := range value.Sections {
			refs = append(refs, section.TargetRefs)
		}
	case SectionalReportPlan:
		for _, part := range value.Parts {
			for _, section := range part.Sections {
				refs = append(refs, section.TargetRefs)
			}
		}
	}
	return refs
}

func emptyReportPlanRefs(refs reportdocument.ReportBlockSourceRefs) bool {
	return len(refs.ClaimIDs)+len(refs.EvidenceIDs)+len(refs.SnapshotIDs)+len(refs.QuestionIDs)+len(refs.OptionIDs) == 0
}
