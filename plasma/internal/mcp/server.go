package mcp

import (
	"context"
	"sync"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/research"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/sources/localpath"
)

// Service는 MCP tool handler가 호출하는 application/reporting port 모음이다.
//
// MCP 패키지는 이 interface 밖의 concrete store나 web handler에 의존하지 않는다.
// 새 tool이 지속 상태를 바꿔야 하면 먼저 app/reporting 계층의 계약을 통해
// 표현되어야 한다.
type Service interface {
	GetProjection(context.Context, string) (app.MissionProjection, error)
	ListEvents(context.Context, string) ([]app.LedgerEvent, error)
	ListSourceSnapshots(context.Context, string) ([]app.SourceSnapshot, error)
	ListSourceSnapshotsWithState(context.Context, app.ListSourceSnapshotsRequest) ([]app.SourceSnapshot, error)
	GetSourceSnapshot(context.Context, string) (app.SourceSnapshot, error)
	GetRawArtifact(context.Context, string) (app.RawArtifact, error)
	ListRawArtifacts(context.Context, string) ([]app.RawArtifact, error)
	ListLocalPathRoots(context.Context) ([]localpath.RootView, error)
	BrowseLocalPathRoot(context.Context, app.BrowseLocalPathRootRequest) (localpath.TreeResult, error)
	AttachLocalPathSource(context.Context, app.AttachLocalPathSourceRequest) (app.LocalPathSourceResult, error)
	ReadLocalPathSource(context.Context, app.ReadLocalPathSourceRequest) (app.ReadLocalPathSourceResult, error)
	TreeLocalPathSource(context.Context, app.TreeLocalPathSourceRequest) (app.TreeLocalPathSourceResult, error)
	GrepLocalPathSource(context.Context, app.GrepLocalPathSourceRequest) (app.GrepLocalPathSourceResult, error)
	RemoveSource(context.Context, app.RemoveSourceRequest) (app.SourceStateChangeResult, error)
	RestoreSource(context.Context, app.RestoreSourceRequest) (app.SourceStateChangeResult, error)
	SearchLiquid2Sources(context.Context, app.Liquid2SourceConnector, app.Liquid2SourceSearchRequest) (app.Liquid2SourceSearchResult, error)
	SearchConfluenceSources(context.Context, app.ConfluenceSourceConnector, app.ConfluenceSourceSearchRequest) (app.ConfluenceSourceSearchResult, error)
	GetMissionConnectorAccess(context.Context, string, string) (app.ConnectorAccessProjection, error)
	GetEvidenceRecord(context.Context, string) (app.EvidenceRecord, error)
	ListEvidenceRecords(context.Context, string) ([]app.EvidenceRecord, error)
	ListClaimRecords(context.Context, string) ([]app.ClaimRecord, error)
	ListQuestionRecords(context.Context, string) ([]app.QuestionRecord, error)
	RequestWorkflowRun(context.Context, app.RequestWorkflowRunRequest) (app.WorkflowRunView, error)
	GetWorkflowRun(context.Context, string, string) (app.WorkflowRunView, error)
	ListWorkflowRuns(context.Context, string) ([]app.WorkflowRunView, error)
	RequestWorkflowStop(context.Context, app.RequestWorkflowStopRequest) (app.WorkflowRunView, error)
	CreateRawArtifact(context.Context, app.CreateRawArtifactRequest) (app.RawArtifact, error)
	CreateRawArtifactWithEvent(context.Context, app.CreateRawArtifactRequest, func(app.RawArtifact) app.AppendEventRequest) (app.RawArtifact, app.LedgerEvent, error)
	CreateRawArtifactWithEventConditionally(context.Context, app.CreateRawArtifactRequest, func([]app.LedgerEvent, app.RawArtifact) (app.AppendEventRequest, app.LedgerEvent, bool, error)) (app.RawArtifact, app.LedgerEvent, bool, error)
	AppendEventConditionally(context.Context, string, func([]app.LedgerEvent) (app.AppendEventRequest, app.LedgerEvent, bool, error)) (app.LedgerEvent, bool, error)
	AppendEvent(context.Context, app.AppendEventRequest) (app.LedgerEvent, error)
}

// Server는 Plasma MCP tool registry와 tool 호출 중 필요한 bounded draft 상태를
// 보관하는 transport adapter다.
//
// Server 내부 map은 process-local in-flight draft와 idempotency cache다. 장기
// 제품 상태의 source of truth가 아니므로, 완료된 결과는 app/reporting service를
// 통해 장부나 artifact로 기록되어야 한다.
type Server struct {
	service                       Service
	research                      *research.Handler
	connectors                    map[string]app.Liquid2SourceConnector
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
	enabledTools                  map[string]struct{}
	sourceCandidateFetcher        SourceCandidateFetcher

	mu                           sync.Mutex
	idempotency                  map[string]idempotencyEntry
	reportDrafts                 map[string]*experimentReportDraft
	reportPatches                map[string]*reportPatchDraft
	partAssemblyDrafts           map[string]*partAssemblyDraft
	partEditDrafts               map[string]*partEditDraft
	longFormEditDrafts           map[string]*longFormEditDraft
	longFormStageEditDrafts      map[string]*longFormStageEditDraft
	readOnlyValidationDrafts     map[string]*readOnlyValidationDraft
	reportPlanParsedCalls        int
	reportRequirementParsedCalls int
}

// NewServer는 MCP server를 구성하고 전달된 Option을 적용한다.
//
// 구성 검증은 tool 노출 여부를 결정하기 위해 생성 시점에 수행하지만, 실제 tool
// 호출은 각 handler에서 binding과 app/reporting 계약을 다시 확인한다.
func NewServer(service Service, options ...Option) *Server {
	server := &Server{
		service:                  service,
		connectors:               map[string]app.Liquid2SourceConnector{},
		idempotency:              map[string]idempotencyEntry{},
		reportDrafts:             map[string]*experimentReportDraft{},
		reportPatches:            map[string]*reportPatchDraft{},
		partAssemblyDrafts:       map[string]*partAssemblyDraft{},
		partEditDrafts:           map[string]*partEditDraft{},
		longFormEditDrafts:       map[string]*longFormEditDraft{},
		longFormStageEditDrafts:  map[string]*longFormStageEditDraft{},
		readOnlyValidationDrafts: map[string]*readOnlyValidationDraft{},
	}
	for _, option := range options {
		option(server)
	}
	server.research = research.NewHandler(service, server.binding.MissionID, server.legacyResearchLoop)
	server.finalEditConfigErr = server.validateFinalEditConfiguration()
	return server
}
