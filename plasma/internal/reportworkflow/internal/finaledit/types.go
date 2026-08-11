package finaledit

import (
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

const (
	// StageSubmittedSentinel은 final edit stage agent가 durable 제출 뒤 정확히 반환해야 하는 응답이다.
	StageSubmittedSentinel = "FINAL_EDIT_STAGE_SUBMITTED"
	// GateSubmittedSentinel은 gate agent가 canonical finalization 뒤 정확히 반환해야 하는 응답이다.
	GateSubmittedSentinel = "REPORT_FINALIZED"
)

// IDGenerator는 runner가 새 ledger, tool session, artifact, draft id를 만들 때 쓰는 포트다.
//
// 같은 prefix 입력에 대한 구체 형식은 caller가 소유하며, 이 패키지는 prefix 의미만
// 요구한다. nil이면 Runner.ID는 기본 난수 기반 ID를 사용한다.
type IDGenerator func(prefix string) string

// Runner는 final edit stage들이 공유하는 durable replay와 session fork helper다.
//
// Store는 reporting의 durable stage/finalization 계약을 구현해야 한다. 이 내부
// helper는 stage 목록이나 prompt 정책을 받지 않으며, 각 public stage package가 고정한
// 단일 stage 계약을 실행할 때만 사용된다.
type Runner struct {
	Store reporting.FinalEditStageStore
	NewID IDGenerator
}

// Rigor는 gate prompt에 들어가는 검증 강도 표시값이다.
//
// Web의 private profile에서 최종화에 필요한 안정 필드만 복사한다.
type Rigor struct {
	Level string
	Label string
}

// Input은 최종화에 필요한 durable identity, writing contract, artifact lineage,
// provider metadata를 담는다.
//
// Pipeline, stage 목록, prompt, tool allowlist, retry 횟수, session strategy는 의도적으로
// 노출하지 않는다. 파이프라인 선택과 humanize 계약은 저장된 PlanEvent와 reporting
// 검증에서 나온다.
type Input struct {
	MissionID                    string
	Title                        string
	ExecutorName                 string
	AgentModel                   string
	AgentReasoningEffort         string
	AgentSelectionSource         string
	MCPMode                      string
	Rigor                        Rigor
	ReportSessionPolicy          string
	ReportSessionPolicySelection string
	PostReportHumanize           string
	GenerationGuidanceProfile    string
	GenerationGuidanceSHA256     string
	PendingEventID               string
	DirectionHint                string
	ArtifactID                   string
	PlanEvent                    ledger.Event
	Plan                         reporting.SectionalReportPlan
	RequirementMap               reporting.ReportRequirementMap
	PartArtifactIDs              []string
	SectionArtifactIDs           []string
	SectionWordTotal             int
	SessionChainKind             string
	PreReportResearchSessionID   string
	ReportPlanSessionID          string
	ForkSourceAgentSessionID     string
	Started                      time.Time
}

// Result는 terminal final edit gate가 만든 canonical report artifact와 event다.
//
// Markdown은 Artifact.Content의 문자열 view이며, Web adapter가 기존 응답 map으로
// 되돌릴 때 별도 재계산 없이 사용한다.
type Result struct {
	Artifact artifact.Raw
	Event    ledger.Event
	Markdown string
}

// StageRun은 하나의 final edit stage가 제출했거나 gate canonicalization을 완료한
// durable 결과다.
//
// Binding은 provider session lineage의 원천이며, Stage는 다음 stage의
// SourceArtifactID로만 소비된다. Final은 terminal gate stage에서만 채워진다.
type StageRun struct {
	Binding reporting.FinalEditStageBinding
	Stage   reporting.FinalEditStageResult
	Final   reporting.LongFormFinalizeResult
}
