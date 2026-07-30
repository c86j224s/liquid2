package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/sources/urlsource"
)

type Binding struct {
	MissionID          string
	AgentSessionID     string
	CurrentUserEventID string
	AgentExecutor      string
}

type ReportPatchBinding struct {
	BaseArtifactID               string
	PendingEventID               string
	AgentExecutor                string
	AgentModel                   string
	AgentReasoningEffort         string
	MCPMode                      string
	AgentSessionID               string
	PreviousAgentSessionID       string
	ReturnedAgentSessionID       string
	ReportSessionID              string
	ForkSourceAgentSessionID     string
	ReportSessionPolicy          string
	ReportSessionPolicySelection string
	SessionChainKind             string
}

type ReportPlanBinding struct {
	PendingEventID            string
	ReportMode                string
	IdempotencyKey            string
	ToolSessionID             string
	PreviousProviderSessionID string
	AgentExecutor             string
	AgentModel                string
	AgentReasoningEffort      string
	RequireWritingContract    bool
}

type idempotencyEntry struct {
	ArgumentsHash string
	Result        ToolResult
}

type Option func(*Server)

type SourceCandidateFetcher func(context.Context, string) (urlsource.Fetched, error)

type ConfluenceConnectorFactory func(context.Context, ConfluenceConnectorRequest) (app.ConfluenceSourceConnector, error)

type ConfluenceConnectorRequest struct {
	ConnectionID string
	CloudID      string
	SpaceKey     string
}

func WithLiquid2Connector(connector app.Liquid2SourceConnector) Option {
	return func(server *Server) {
		if connector != nil {
			server.connectors[app.Liquid2ConnectorID] = connector
		}
	}
}

func WithConfluenceConnectorFactory(factory ConfluenceConnectorFactory) Option {
	return func(server *Server) {
		server.confluenceConnectorFactory = factory
	}
}

func WithBinding(binding Binding) Option {
	return func(server *Server) {
		server.binding = Binding{
			MissionID:          strings.TrimSpace(binding.MissionID),
			AgentSessionID:     strings.TrimSpace(binding.AgentSessionID),
			CurrentUserEventID: strings.TrimSpace(binding.CurrentUserEventID),
			AgentExecutor:      strings.TrimSpace(strings.ToLower(binding.AgentExecutor)),
		}
	}
}

func WithLegacyResearchLoop() Option {
	return func(server *Server) {
		server.legacyResearchLoop = true
	}
}

func WithExperimentalReportComposition() Option {
	return func(server *Server) {
		server.experimentalReportComposition = true
	}
}

func WithOperatorSourceMutation() Option {
	return func(server *Server) {
		server.operatorSourceMutation = true
	}
}

func WithReportPatch() Option {
	return func(server *Server) {
		server.reportPatch = true
	}
}

func WithReportPatchBinding(binding ReportPatchBinding) Option {
	return func(server *Server) {
		server.reportPatchBinding = normalizeReportPatchBinding(binding)
	}
}

func WithReportPlanBinding(binding ReportPlanBinding) Option {
	return func(server *Server) {
		server.reportPlanBinding = normalizeReportPlanBinding(binding)
	}
}

func WithReportRequirementMapBinding(binding reporting.ReportRequirementMapBinding) Option {
	return func(server *Server) {
		binding.MissionID = strings.TrimSpace(binding.MissionID)
		binding.PendingEventID = strings.TrimSpace(binding.PendingEventID)
		binding.PlanEventID = strings.TrimSpace(binding.PlanEventID)
		binding.ToolSessionID = strings.TrimSpace(binding.ToolSessionID)
		binding.PreviousProviderSessionID = strings.TrimSpace(binding.PreviousProviderSessionID)
		binding.IdempotencyKey = strings.TrimSpace(binding.IdempotencyKey)
		binding.AgentExecutor = strings.TrimSpace(strings.ToLower(binding.AgentExecutor))
		binding.AgentModel = strings.TrimSpace(binding.AgentModel)
		binding.AgentReasoningEffort = strings.TrimSpace(binding.AgentReasoningEffort)
		binding.Producer.Type = strings.TrimSpace(binding.Producer.Type)
		binding.Producer.ID = strings.TrimSpace(binding.Producer.ID)
		server.reportRequirementMapBinding = binding
	}
}

func WithLongFormFinalizeBinding(binding reporting.LongFormFinalizeBinding) Option {
	return func(server *Server) {
		server.longFormFinalizeBinding = binding
		server.longFormFinalizeBindingSet = true
	}
}

func WithFinalEditStageBinding(binding reporting.FinalEditStageBinding) Option {
	return func(server *Server) {
		server.finalEditStageBinding = binding
		server.finalEditStageBindingSet = true
	}
}

func WithPartAssemblyBinding(binding reporting.PartAssemblyBinding) Option {
	return func(server *Server) { server.partAssemblyBinding = binding }
}

func WithPartEditBinding(binding reporting.PartEditBinding) Option {
	return func(server *Server) { server.partEditBinding = binding }
}

func ValidatePartAssemblyBinding(binding Binding, part reporting.PartAssemblyBinding) error {
	if err := reporting.ValidatePartAssemblyBinding(part); err != nil {
		return fmt.Errorf("part assembly binding is incomplete: %w", err)
	}
	if strings.TrimSpace(part.MissionID) != strings.TrimSpace(binding.MissionID) || strings.TrimSpace(part.ToolSessionID) != strings.TrimSpace(binding.AgentSessionID) || strings.TrimSpace(strings.ToLower(part.AgentExecutor)) != strings.TrimSpace(strings.ToLower(binding.AgentExecutor)) {
		return fmt.Errorf("part assembly binding conflicts with MCP binding")
	}
	return nil
}

func ValidatePartEditBinding(binding Binding, part reporting.PartEditBinding) error {
	if err := reporting.ValidatePartEditBinding(part); err != nil {
		return fmt.Errorf("part edit binding is incomplete: %w", err)
	}
	if strings.TrimSpace(part.MissionID) != strings.TrimSpace(binding.MissionID) || strings.TrimSpace(part.ToolSessionID) != strings.TrimSpace(binding.AgentSessionID) || strings.TrimSpace(strings.ToLower(part.AgentExecutor)) != strings.TrimSpace(strings.ToLower(binding.AgentExecutor)) {
		return fmt.Errorf("part edit binding conflicts with MCP binding")
	}
	return nil
}

func ValidateLongFormFinalizeBinding(binding Binding, final reporting.LongFormFinalizeBinding) error {
	if err := reporting.ValidateLongFormFinalizeBinding(final); err != nil {
		return fmt.Errorf("long-form finalization binding is incomplete: %w", err)
	}
	if strings.TrimSpace(final.MissionID) != strings.TrimSpace(binding.MissionID) || strings.TrimSpace(final.ToolSessionID) != strings.TrimSpace(binding.AgentSessionID) || strings.TrimSpace(strings.ToLower(final.AgentExecutor)) != strings.TrimSpace(strings.ToLower(binding.AgentExecutor)) {
		return fmt.Errorf("long-form finalization binding conflicts with MCP binding")
	}
	return nil
}

func ValidateFinalEditStageBinding(binding Binding, stage reporting.FinalEditStageBinding) error {
	if err := reporting.ValidateFinalEditStageBinding(stage); err != nil {
		return fmt.Errorf("final edit stage binding is incomplete: %w", err)
	}
	if strings.TrimSpace(stage.MissionID) != strings.TrimSpace(binding.MissionID) || strings.TrimSpace(stage.ToolSessionID) != strings.TrimSpace(binding.AgentSessionID) || strings.TrimSpace(strings.ToLower(stage.AgentExecutor)) != strings.TrimSpace(strings.ToLower(binding.AgentExecutor)) {
		return fmt.Errorf("final edit stage binding conflicts with MCP binding")
	}
	return nil
}

func (server *Server) validateFinalEditConfiguration() error {
	stageProvided := server.finalEditStageBindingSet
	finalProvided := server.longFormFinalizeBindingSet
	if !stageProvided {
		return nil
	}
	if err := ValidateFinalEditStageBinding(server.binding, server.finalEditStageBinding); err != nil {
		return err
	}
	switch strings.TrimSpace(server.finalEditStageBinding.Stage) {
	case reporting.FinalEditStageWriter:
		if finalProvided {
			return fmt.Errorf("%w: final writer final edit stage MCP server must not carry a final binding", app.ErrInvalidInput)
		}
	case reporting.FinalEditStageReader, reporting.FinalEditStageStyle:
		if finalProvided {
			return fmt.Errorf("%w: reader/style final edit stage MCP servers must not carry a final binding", app.ErrInvalidInput)
		}
	case reporting.FinalEditStageGate:
		if !finalProvided {
			return fmt.Errorf("%w: corrective gate final edit MCP server requires a final binding", app.ErrInvalidInput)
		}
		if err := ValidateLongFormFinalizeBinding(server.binding, server.longFormFinalizeBinding); err != nil {
			return err
		}
		if err := reporting.ValidateFinalEditGateBindingsCompatible(server.finalEditStageBinding, server.longFormFinalizeBinding); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%w: unsupported final edit stage", app.ErrInvalidInput)
	}
	return nil
}

func normalizeReportPlanBinding(binding ReportPlanBinding) ReportPlanBinding {
	return ReportPlanBinding{
		PendingEventID: strings.TrimSpace(binding.PendingEventID), ReportMode: strings.TrimSpace(binding.ReportMode),
		IdempotencyKey: strings.TrimSpace(binding.IdempotencyKey), ToolSessionID: strings.TrimSpace(binding.ToolSessionID),
		PreviousProviderSessionID: strings.TrimSpace(binding.PreviousProviderSessionID), AgentExecutor: strings.TrimSpace(strings.ToLower(binding.AgentExecutor)),
		AgentModel: strings.TrimSpace(binding.AgentModel), AgentReasoningEffort: strings.TrimSpace(binding.AgentReasoningEffort),
		RequireWritingContract: binding.RequireWritingContract,
	}
}

func (binding ReportPlanBinding) complete() bool {
	return binding.PendingEventID != "" && (binding.ReportMode == "planned" || binding.ReportMode == "long_form") && binding.IdempotencyKey != "" && binding.ToolSessionID != "" && binding.AgentExecutor != ""
}

func ValidateReportPlanBinding(binding Binding, plan ReportPlanBinding) error {
	plan = normalizeReportPlanBinding(plan)
	if !plan.complete() {
		return fmt.Errorf("report plan binding is incomplete")
	}
	if plan.ToolSessionID != strings.TrimSpace(binding.AgentSessionID) || plan.AgentExecutor != strings.TrimSpace(strings.ToLower(binding.AgentExecutor)) {
		return fmt.Errorf("report plan binding conflicts with MCP binding")
	}
	return nil
}

func ValidateReportRequirementMapBinding(binding Binding, requirements reporting.ReportRequirementMapBinding) error {
	if err := reporting.ValidateReportRequirementMapBinding(requirements); err != nil {
		return fmt.Errorf("report requirement map binding is incomplete: %w", err)
	}
	if strings.TrimSpace(requirements.MissionID) != strings.TrimSpace(binding.MissionID) || strings.TrimSpace(requirements.ToolSessionID) != strings.TrimSpace(binding.AgentSessionID) || strings.TrimSpace(strings.ToLower(requirements.AgentExecutor)) != strings.TrimSpace(strings.ToLower(binding.AgentExecutor)) {
		return fmt.Errorf("report requirement map binding conflicts with MCP binding")
	}
	return nil
}

func WithEnabledTools(tools []string) Option {
	return func(server *Server) {
		enabled := map[string]struct{}{}
		for _, tool := range tools {
			tool = strings.TrimSpace(tool)
			if tool != "" {
				enabled[tool] = struct{}{}
			}
		}
		if len(enabled) > 0 {
			server.enabledTools = enabled
		}
	}
}

func WithSourceCandidateFetcher(fetcher SourceCandidateFetcher) Option {
	return func(server *Server) {
		server.sourceCandidateFetcher = fetcher
	}
}
