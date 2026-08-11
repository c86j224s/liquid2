package reportexperiment

import (
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow"
	workflowplan "github.com/c86j224s/liquid2/plasma/internal/reportworkflow/plan"
)

// preparedSeed는 provider 호출 전에 끝낼 수 있는 fixture-to-prefix 순수 검증 결과다.
//
// 이 값은 DB, provider session, run directory에 의존하지 않으며 seed 단계는 여기서
// 검증한 plan, requirements, guidance receipt를 그대로 장부에 기록해야 한다.
type preparedSeed struct {
	Plan               reporting.SectionalReportPlan
	PlanSHA256         string
	RequirementMap     reporting.ReportRequirementMap
	GuidanceProfile    string
	GuidanceSHA256     string
	FinalEditPipeline  string
	FinalTail          reportworkflow.FinalTail
	PostReportHumanize string
	PartCount          int
	SectionCount       int
}

func preflightFixture(loaded LoadedFixture, runID string) (preparedSeed, error) {
	guidanceProfile, guidanceSHA, err := reportprompt.SelectReportGenerationGuidanceForMode(reportexecution.ModeLongForm, loaded.Spec.GenerationGuidanceProfile)
	if err != nil {
		return preparedSeed{}, err
	}
	if expected := strings.TrimSpace(loaded.Spec.GenerationGuidanceSHA256); expected != "" && !strings.EqualFold(expected, guidanceSHA) {
		return preparedSeed{}, fmt.Errorf("%w: fixture generation guidance SHA-256 differs from current product prompt", producterror.ErrConflict)
	}
	pipeline := workflowplan.LongFormFinalEditPipelineForPlan(guidanceProfile)
	if pipeline != reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3 {
		return preparedSeed{}, fmt.Errorf("%w: fixture guidance profile does not select the current V3 final tail", producterror.ErrConflict)
	}
	plan, err := planFromFixture(loaded)
	if err != nil {
		return preparedSeed{}, err
	}
	if err := reporting.RequireReportWritingContract(plan); err != nil {
		return preparedSeed{}, err
	}
	planSHA, _, err := reporting.ReportPlanHash(plan)
	if err != nil {
		return preparedSeed{}, err
	}
	stem := safeIDStem(runID)
	pendingID := "evt_reportexperiment_" + stem + "_pending"
	requirements, err := requirementsFromFixture(loaded, plan, pendingID, stem)
	if err != nil {
		return preparedSeed{}, err
	}
	return preparedSeed{
		Plan: plan, PlanSHA256: planSHA, RequirementMap: requirements,
		GuidanceProfile: guidanceProfile, GuidanceSHA256: guidanceSHA,
		FinalEditPipeline: pipeline, FinalTail: reportworkflow.FinalTailV3,
		PostReportHumanize: loaded.Spec.PostReportHumanize,
		PartCount:          len(loaded.Parts), SectionCount: len(loaded.Parts),
	}, nil
}
