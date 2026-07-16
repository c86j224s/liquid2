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

func WithLongFormFinalizeBinding(binding reporting.LongFormFinalizeBinding) Option {
	return func(server *Server) { server.longFormFinalizeBinding = binding }
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

func normalizeReportPlanBinding(binding ReportPlanBinding) ReportPlanBinding {
	return ReportPlanBinding{
		PendingEventID: strings.TrimSpace(binding.PendingEventID), ReportMode: strings.TrimSpace(binding.ReportMode),
		IdempotencyKey: strings.TrimSpace(binding.IdempotencyKey), ToolSessionID: strings.TrimSpace(binding.ToolSessionID),
		PreviousProviderSessionID: strings.TrimSpace(binding.PreviousProviderSessionID), AgentExecutor: strings.TrimSpace(strings.ToLower(binding.AgentExecutor)),
		AgentModel: strings.TrimSpace(binding.AgentModel), AgentReasoningEffort: strings.TrimSpace(binding.AgentReasoningEffort),
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
