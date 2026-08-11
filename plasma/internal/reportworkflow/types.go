package reportworkflow

import (
	"context"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reporthumanize"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/directdraft"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/evidencecheck"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/finalstore"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/finalwrite"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/legacyfinalize"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/partassembly"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/partedit"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/partplan"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/plan"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/readeredit"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/reportassembly"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/requirements"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/sectiondraft"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/semanticcheck"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow/styleedit"
)

// Service는 보고서 계획부터 최종 저장까지 각 단계가 소비하는 저장소 기능의 합집합이다.
// 단계 패키지는 이 인터페이스의 필요한 부분만 좁은 소비자 계약으로 다시 받는다.
type Service interface {
	plan.Service
	plan.LongFormSectionPlanRepairService
	finalstore.Service
	reporting.FinalEditStageStore
	reporthumanize.Service
	requirements.Service
	partplan.Service
	sectiondraft.Service
	partassembly.Service
	partedit.Service
}

// RunnerConfig는 제품 고정 workflow runner를 만들 때 필요한 adapter wiring이다.
//
// Service는 one_take/planned 저장뿐 아니라 장문 final-edit durable replay와 H5
// terminal 기록 포트를 compile-time 계약으로 제공해야 한다. NewRunner는 capability를
// 추측하지 않으며, nil service는 구성 오류로 즉시 거부한다.
type RunnerConfig struct {
	Service         Service
	Lifecycle       reporting.Runner
	Executor        agentexec.AgentExecutor
	NewID           func(string) string
	LatestSessionID func(context.Context, string, string) string
}

// DraftInput은 reportexecution pending payload에서 복원된 typed 실행 입력이다.
type DraftInput struct {
	MissionID, PendingEventID        string
	Title, DirectionHint             string
	ExecutionStrategy                string
	AgentExecutor                    string
	AgentModel, AgentReasoningEffort string
	AgentSelectionSource, MCPMode    string
	Rigor                            reportprompt.RigorProfile
	ReportMode                       string
	ReportSessionPolicy              string
	ReportSessionPolicySelection     string
	PostReportHumanize               string
	GenerationGuidanceProfile        string
	GenerationGuidanceSHA256         string
}

// DraftOutput은 workflow가 생성한 Markdown report artifact와 report session이다.
type DraftOutput struct {
	Artifact        artifact.Raw
	Event           ledger.Event
	Markdown        string
	ReportSessionID string
	Humanized       *reporthumanize.Result
}

// PrefixPart는 finalization tail에 넘길 ordered Part artifact 계약이다.
type PrefixPart struct {
	Title      string
	Markdown   string
	ArtifactID string
	WordCount  int
}

// PrefixSection은 prefix가 생성/복구한 ordered Section artifact 계약이다.
type PrefixSection struct {
	Title      string
	Markdown   string
	ArtifactID string
	WordCount  int
}

// PrefixOutput은 long-form finalization tail이 소비하는 typed handoff다.
//
// prompt나 callback을 노출하지 않고, canonical plan/event와 durable prefix artifact
// inventories, session lineage, frozen request metadata만 담는다.
type PrefixOutput struct {
	MissionID                    string
	PendingEventID               string
	Title                        string
	DirectionHint                string
	ExecutionStrategy            string
	AgentExecutor                string
	AgentModel                   string
	AgentReasoningEffort         string
	AgentSelectionSource         string
	MCPMode                      string
	Rigor                        reportprompt.RigorProfile
	ReportSessionPolicy          string
	ReportSessionPolicySelection string
	PostReportHumanize           string
	GenerationGuidanceProfile    string
	GenerationGuidanceSHA256     string
	ArtifactID                   string
	PlanEvent                    ledger.Event
	Plan                         reporting.SectionalReportPlan
	RequirementMap               reporting.ReportRequirementMap
	RequirementMapEvent          ledger.Event
	Parts                        []PrefixPart
	Sections                     [][]PrefixSection
	PartArtifactIDs              []string
	SectionArtifactIDs           []string
	SectionWordTotal             int
	SessionChainKind             string
	PreReportResearchSessionID   string
	ReportPlanSessionID          string
	ForkSourceAgentSessionID     string
	ReportSessionID              string
	PartEditEnabled              bool
	PartPlanningEnabled          bool
	FinalEditPipeline            string
	FinalTail                    FinalTail
	StartedAt                    time.Time
}

// Runner는 Plasma 보고서 생성 graph family를 정적으로 선택하고 stage를 순서대로 호출한다.
type Runner struct {
	service              Service
	finalEditStore       reporting.FinalEditStageStore
	humanizeService      reporthumanize.Service
	executor             agentexec.AgentExecutor
	newID                func(string) string
	planRunner           plan.Runner
	directDraftRunner    directdraft.Runner
	finalStoreRunner     finalstore.Runner
	requirementsRunner   requirements.Runner
	partPlanRunner       partplan.Runner
	sectionDraftRunner   sectiondraft.Runner
	partAssemblyRunner   partassembly.Runner
	partEditRunner       partedit.Runner
	reportAssemblyRunner reportassembly.Runner
	finalWriteRunner     finalwrite.Runner
	readerEditRunner     readeredit.Runner
	styleEditRunner      styleedit.Runner
	semanticCheckRunner  semanticcheck.Runner
	evidenceCheckRunner  evidencecheck.Runner
	legacyFinalizeRunner legacyfinalize.Runner
	observer             Observer
}
