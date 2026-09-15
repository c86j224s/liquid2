package mcp

import (
	"context"
	experimenthandler "github.com/c86j224s/liquid2/plasma/internal/mcp/reportexperiment"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/reportfinaledit"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/reportil"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/reportparts"
	patchhandler "github.com/c86j224s/liquid2/plasma/internal/mcp/reportpatch"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/reportplan"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/reportrequirements"
	"sync"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	missionadapter "github.com/c86j224s/liquid2/plasma/internal/mcp/mission"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/research"
	workflowadapter "github.com/c86j224s/liquid2/plasma/internal/mcp/workflow"
	"github.com/c86j224s/liquid2/plasma/internal/mission"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/researchrecords"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
	"github.com/c86j224s/liquid2/plasma/internal/source/confluencesource"
	"github.com/c86j224s/liquid2/plasma/internal/source/liquid2source"
	"github.com/c86j224s/liquid2/plasma/internal/workflowstate"
)

// Service는 MCP tool handler가 호출하는 application/reporting port 모음이다.
//
// MCP 패키지는 이 interface 밖의 concrete store나 web handler에 의존하지 않는다.
// 새 tool이 지속 상태를 바꿔야 하면 먼저 app/reporting 계층의 계약을 통해
// 표현되어야 한다.
type Service interface {
	GetProjection(context.Context, string) (mission.Projection, error)
	ListEvents(context.Context, string) ([]ledger.Event, error)
	ListSourceSnapshots(context.Context, string) ([]sourcecontract.Snapshot, error)
	ListSourceSnapshotsWithState(context.Context, sourcecontract.ListRequest) ([]sourcecontract.Snapshot, error)
	GetSourceSnapshot(context.Context, string) (sourcecontract.Snapshot, error)
	GetRawArtifact(context.Context, string) (artifactcontract.Raw, error)
	ListRawArtifacts(context.Context, string) ([]artifactcontract.Raw, error)
	ListLocalPathRoots(context.Context) ([]sourcecontract.LocalPathRoot, error)
	BrowseLocalPathRoot(context.Context, app.BrowseLocalPathRootRequest) (sourcecontract.LocalPathTreeResult, error)
	AttachLocalPathSource(context.Context, app.AttachLocalPathSourceRequest) (app.LocalPathSourceResult, error)
	ReadLocalPathSource(context.Context, app.ReadLocalPathSourceRequest) (app.ReadLocalPathSourceResult, error)
	TreeLocalPathSource(context.Context, app.TreeLocalPathSourceRequest) (app.TreeLocalPathSourceResult, error)
	GrepLocalPathSource(context.Context, app.GrepLocalPathSourceRequest) (app.GrepLocalPathSourceResult, error)
	RemoveSource(context.Context, app.RemoveSourceRequest) (app.SourceStateChangeResult, error)
	RestoreSource(context.Context, app.RestoreSourceRequest) (app.SourceStateChangeResult, error)
	SearchLiquid2Sources(context.Context, liquid2source.Liquid2SourceConnector, liquid2source.Liquid2SourceSearchRequest) (liquid2source.Liquid2SourceSearchResult, error)
	SearchConfluenceSources(context.Context, confluencesource.ConfluenceSourceConnector, confluencesource.ConfluenceSourceSearchRequest) (confluencesource.ConfluenceSourceSearchResult, error)
	GetMissionConnectorAccess(context.Context, string, string) (app.ConnectorAccessProjection, error)
	GetEvidenceRecord(context.Context, string) (researchrecords.EvidenceRecord, error)
	ListEvidenceRecords(context.Context, string) ([]researchrecords.EvidenceRecord, error)
	ListClaimRecords(context.Context, string) ([]researchrecords.ClaimRecord, error)
	ListQuestionRecords(context.Context, string) ([]researchrecords.QuestionRecord, error)
	RequestWorkflowRun(context.Context, workflowstate.RequestWorkflowRunRequest) (workflowstate.WorkflowRunView, error)
	GetWorkflowRun(context.Context, string, string) (workflowstate.WorkflowRunView, error)
	ListWorkflowRuns(context.Context, string) ([]workflowstate.WorkflowRunView, error)
	RequestWorkflowStop(context.Context, workflowstate.RequestWorkflowStopRequest) (workflowstate.WorkflowRunView, error)
	CreateRawArtifact(context.Context, artifactcontract.CreateRequest) (artifactcontract.Raw, error)
	CreateRawArtifactWithEvent(context.Context, artifactcontract.CreateRequest, func(artifactcontract.Raw) ledger.AppendRequest) (artifactcontract.Raw, ledger.Event, error)
	CreateRawArtifactWithEventConditionally(context.Context, artifactcontract.CreateRequest, func([]ledger.Event, artifactcontract.Raw) (ledger.AppendRequest, ledger.Event, bool, error)) (artifactcontract.Raw, ledger.Event, bool, error)
	AppendEventConditionally(context.Context, string, func([]ledger.Event) (ledger.AppendRequest, ledger.Event, bool, error)) (ledger.Event, bool, error)
	AppendEvent(context.Context, ledger.AppendRequest) (ledger.Event, error)
}

// Server는 Plasma MCP tool registry와 tool 호출 중 필요한 bounded draft 상태를
// 보관하는 transport adapter다.
//
// Server 내부 map은 process-local in-flight draft와 idempotency cache다. 장기
// 제품 상태의 source of truth가 아니므로, 완료된 결과는 app/reporting service를
// 통해 장부나 artifact로 기록되어야 한다.
type Server struct {
	reportFinalEditState          *reportfinaledit.State
	service                       Service
	research                      *research.Handler
	mission                       *missionadapter.Handler
	workflow                      *workflowadapter.Handler
	connectors                    map[string]liquid2source.Liquid2SourceConnector
	confluenceConnectorFactory    ConfluenceConnectorFactory
	binding                       Binding
	legacyResearchLoop            bool
	experimentalReportComposition bool
	operatorSourceMutation        bool
	reportPatch                   bool
	reportPatchBinding            ReportPatchBinding
	reportPlanBinding             ReportPlanBinding
	reportRequirementMapBinding   reporting.ReportRequirementMapBinding
	partAssemblyBinding           reporting.PartAssemblyBinding
	partEditBinding               reporting.PartEditBinding
	longFormFinalizeBinding       reporting.LongFormFinalizeBinding
	longFormFinalizeBindingSet    bool
	finalEditStageBinding         reporting.FinalEditStageBinding
	finalEditStageBindingSet      bool
	finalEditConfigErr            error
	reportILSourceBinding         reportilcontract.SourceAccessBinding
	reportILSourceBindingSet      bool
	reportILState                 *reportil.State
	enabledTools                  map[string]struct{}
	enabledToolsSet               bool
	sourceCandidateFetcher        SourceCandidateFetcher

	mu                        sync.Mutex
	idempotency               map[string]idempotencyEntry
	reportExperimentState     *experimenthandler.State
	reportPatchState          *patchhandler.State
	reportPartsState          *reportparts.State
	reportPlanHandler         *reportplan.Handler
	reportRequirementsHandler *reportrequirements.Handler
}

// NewServer는 MCP server를 구성하고 전달된 Option을 적용한다.
//
// 구성 검증은 tool 노출 여부를 결정하기 위해 생성 시점에 수행하지만, 실제 tool
// 호출은 각 handler에서 binding과 app/reporting 계약을 다시 확인한다.
func NewServer(service Service, options ...Option) *Server {
	server := &Server{
		reportFinalEditState:  &reportfinaledit.State{LegacyDrafts: map[string]*reportfinaledit.LongFormEditDraft{}, StageDrafts: map[string]*reportfinaledit.LongFormStageEditDraft{}, ValidationDrafts: map[string]*reportfinaledit.ReadOnlyValidationDraft{}},
		reportILState:         reportil.NewState(),
		service:               service,
		connectors:            map[string]liquid2source.Liquid2SourceConnector{},
		idempotency:           map[string]idempotencyEntry{},
		reportExperimentState: &experimenthandler.State{Drafts: map[string]*experimenthandler.Draft{}},
		reportPatchState:      &patchhandler.State{Drafts: map[string]*patchhandler.Draft{}},
		reportPartsState:      &reportparts.State{EditDrafts: map[string]*reportparts.PartEditDraft{}, AssemblyDrafts: map[string]*reportparts.PartAssemblyDraft{}},
	}
	for _, option := range options {
		option(server)
	}
	if server.reportILSourceBindingSet {
		if reportilcontract.ValidateSourceAccessBinding(server.reportILSourceBinding) != nil {
			server.enabledTools = map[string]struct{}{}
			server.enabledToolsSet = true
			server.research = research.NewHandler(service, server.binding.MissionID, server.legacyResearchLoop)
			server.mission = newMissionHandler(server)
			server.workflow = newWorkflowHandler(server)
			server.finalEditConfigErr = server.validateFinalEditConfiguration()
			return server
		}
		server.enabledTools = map[string]struct{}{}
		switch server.reportILSourceBinding.Stage {
		case "il_source_selection", "il_editorial_memory", "il_document", "il_flow":
			server.enabledTools[reportilcontract.SourceListTool] = struct{}{}
			server.enabledTools[reportilcontract.SourceReadTool] = struct{}{}
		}
		switch server.reportILSourceBinding.Stage {
		case "il_editorial_memory":
			server.enabledTools[reportilcontract.SourceQuoteRegisterTool] = struct{}{}
			server.enabledTools[reportilcontract.EditorialMemoryStartTool] = struct{}{}
			server.enabledTools[reportilcontract.EditorialMemoryAppendTool] = struct{}{}
			server.enabledTools[reportilcontract.EditorialMemoryReadTool] = struct{}{}
			server.enabledTools[reportilcontract.EditorialMemoryFinalizeTool] = struct{}{}
		case "il_narrative":
			if server.reportILSourceBinding.MaxReadBytes > 0 {
				server.enabledTools[reportilcontract.SourceListTool] = struct{}{}
				server.enabledTools[reportilcontract.SourceReadTool] = struct{}{}
				server.enabledTools[reportilcontract.AuthorDocumentAppendSourceTool] = struct{}{}
			} else {
				server.enabledTools[reportilcontract.EditorialMemoryReadTool] = struct{}{}
				server.enabledTools[reportilcontract.AuthorDocumentAppendTool] = struct{}{}
			}
			server.enabledTools[reportilcontract.AuthorDocumentStartTool] = struct{}{}
			server.enabledTools[reportilcontract.AuthorDocumentReadTool] = struct{}{}
			server.enabledTools[reportilcontract.AuthorDocumentReplaceTool] = struct{}{}
			server.enabledTools[reportilcontract.AuthorDocumentFinalizeTool] = struct{}{}
		case "il_continuity":
			server.enabledTools[reportilcontract.EditorialMemoryReadTool] = struct{}{}
			server.enabledTools[reportilcontract.AuthorDocumentOpenTool] = struct{}{}
			server.enabledTools[reportilcontract.AuthorDocumentReadTool] = struct{}{}
			server.enabledTools[reportilcontract.AuthorDocumentReviseBlockTool] = struct{}{}
			server.enabledTools[reportilcontract.AuthorDocumentFinalizeTool] = struct{}{}
		case "il_reader":
			server.enabledTools[reportilcontract.EditorialMemoryReadTool] = struct{}{}
			server.enabledTools[reportilcontract.AuthorDocumentOpenTool] = struct{}{}
			server.enabledTools[reportilcontract.AuthorDocumentReadTool] = struct{}{}
			server.enabledTools[reportilcontract.AuthorDocumentEditTextTool] = struct{}{}
			server.enabledTools[reportilcontract.AuthorDocumentFinalizeTool] = struct{}{}
		case "il_long_form_plan":
			if server.reportILSourceBinding.MaxReadBytes > 0 {
				server.enabledTools[reportilcontract.SourceListTool] = struct{}{}
				server.enabledTools[reportilcontract.SourceReadTool] = struct{}{}
			} else {
				server.enabledTools[reportilcontract.EditorialMemoryReadTool] = struct{}{}
			}
			server.enabledTools[reportilcontract.LongFormPlanSubmitTool] = struct{}{}
		case "il_long_form_section":
			if server.reportILSourceBinding.MaxReadBytes > 0 {
				server.enabledTools[reportilcontract.SourceReadTool] = struct{}{}
			} else {
				server.enabledTools[reportilcontract.EditorialMemoryReadTool] = struct{}{}
			}
			server.enabledTools[reportilcontract.LongFormPlanReadTool] = struct{}{}
			server.enabledTools[reportilcontract.LongFormDocumentStartTool] = struct{}{}
			server.enabledTools[reportilcontract.LongFormDocumentAppendTool] = struct{}{}
			server.enabledTools[reportilcontract.LongFormDocumentReadTool] = struct{}{}
			server.enabledTools[reportilcontract.LongFormDocumentReplaceTool] = struct{}{}
			server.enabledTools[reportilcontract.LongFormDocumentCorrectBlockTool] = struct{}{}
			server.enabledTools[reportilcontract.LongFormDocumentFinalizeTool] = struct{}{}
		case "il_long_form_part", "il_long_form_final":
			if server.reportILSourceBinding.EditorialMemoryArtifactID != "" {
				server.enabledTools[reportilcontract.EditorialMemoryReadTool] = struct{}{}
			}
			server.enabledTools[reportilcontract.LongFormPlanReadTool] = struct{}{}
			server.enabledTools[reportilcontract.LongFormDocumentStartTool] = struct{}{}
			server.enabledTools[reportilcontract.LongFormDocumentReadTool] = struct{}{}
			server.enabledTools[reportilcontract.LongFormDocumentReplaceTool] = struct{}{}
			server.enabledTools[reportilcontract.LongFormDocumentFinalizeTool] = struct{}{}
		}
		server.enabledToolsSet = true
	}
	server.research = research.NewHandler(service, server.binding.MissionID, server.legacyResearchLoop)
	server.mission = newMissionHandler(server)
	server.workflow = newWorkflowHandler(server)
	server.finalEditConfigErr = server.validateFinalEditConfiguration()
	return server
}
