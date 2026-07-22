package mcp

import (
	"context"
	"sync"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/sources/localpath"
)

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
	ListEvidenceRecords(context.Context, string) ([]app.EvidenceRecord, error)
	ListClaimRecords(context.Context, string) ([]app.ClaimRecord, error)
	ListQuestionRecords(context.Context, string) ([]app.QuestionRecord, error)
	OutlineMission(context.Context, string) (app.ResearchIDEOutline, error)
	ListMissionObjects(context.Context, string, string, int, string) (app.ResearchIDEPage, error)
	ReadMissionObject(context.Context, app.ResearchIDEReadRequest) (app.ResearchIDEObjectRead, error)
	GrepMissionObjects(context.Context, string, string, int, string) (app.ResearchIDEGrepResult, error)
	ListObjectReferences(context.Context, string, string, string, int, string) (app.ResearchIDEReferences, error)
	RequestWorkflowRun(context.Context, app.RequestWorkflowRunRequest) (app.WorkflowRunView, error)
	GetWorkflowRun(context.Context, string, string) (app.WorkflowRunView, error)
	ListWorkflowRuns(context.Context, string) ([]app.WorkflowRunView, error)
	RequestWorkflowStop(context.Context, app.RequestWorkflowStopRequest) (app.WorkflowRunView, error)
	CreateRawArtifact(context.Context, app.CreateRawArtifactRequest) (app.RawArtifact, error)
	CreateRawArtifactWithEvent(context.Context, app.CreateRawArtifactRequest, func(app.RawArtifact) app.AppendEventRequest) (app.RawArtifact, app.LedgerEvent, error)
	CreateRawArtifactWithEventConditionally(context.Context, app.CreateRawArtifactRequest, func([]app.LedgerEvent, app.RawArtifact) (app.AppendEventRequest, app.LedgerEvent, bool, error)) (app.RawArtifact, app.LedgerEvent, bool, error)
	AppendEvent(context.Context, app.AppendEventRequest) (app.LedgerEvent, error)
	CreateEvidenceProposal(context.Context, app.CreateEvidenceProposalRequest) (app.EvidenceProposalResult, error)
	CreateQuestionProposal(context.Context, app.CreateQuestionProposalRequest) (app.QuestionProposalResult, error)
	CreateClaimProposal(context.Context, app.CreateClaimProposalRequest) (app.ClaimProposalResult, error)
	UpdateClaimConfidence(context.Context, app.UpdateClaimConfidenceRequest) (app.LedgerEvent, error)
	SubmitProposal(context.Context, app.SubmitProposalRequest) (app.SubmitProposalResult, error)
}

type legacyResearchReader interface {
	OutlineMissionLegacy(context.Context, string) (app.ResearchIDEOutline, error)
	ListMissionObjectsLegacy(context.Context, string, string, int, string) (app.ResearchIDEPage, error)
	GrepMissionObjectsLegacy(context.Context, string, string, int, string) (app.ResearchIDEGrepResult, error)
	ListObjectReferencesLegacy(context.Context, string, string, string, int, string) (app.ResearchIDEReferences, error)
}

type Server struct {
	service                       Service
	connectors                    map[string]app.Liquid2SourceConnector
	confluenceConnectorFactory    ConfluenceConnectorFactory
	binding                       Binding
	legacyResearchLoop            bool
	experimentalReportComposition bool
	operatorSourceMutation        bool
	reportPatch                   bool
	reportPatchBinding            ReportPatchBinding
	reportPlanBinding             ReportPlanBinding
	partAssemblyBinding           reporting.PartAssemblyBinding
	longFormFinalizeBinding       reporting.LongFormFinalizeBinding
	enabledTools                  map[string]struct{}
	sourceCandidateFetcher        SourceCandidateFetcher

	mu                    sync.Mutex
	idempotency           map[string]idempotencyEntry
	reportDrafts          map[string]*experimentReportDraft
	reportPatches         map[string]*reportPatchDraft
	partAssemblyDrafts    map[string]*partAssemblyDraft
	longFormEditDrafts    map[string]*longFormEditDraft
	reportPlanParsedCalls int
}

func NewServer(service Service, options ...Option) *Server {
	server := &Server{
		service:            service,
		connectors:         map[string]app.Liquid2SourceConnector{},
		idempotency:        map[string]idempotencyEntry{},
		reportDrafts:       map[string]*experimentReportDraft{},
		reportPatches:      map[string]*reportPatchDraft{},
		partAssemblyDrafts: map[string]*partAssemblyDraft{},
		longFormEditDrafts: map[string]*longFormEditDraft{},
	}
	for _, option := range options {
		option(server)
	}
	return server
}
