package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/config"
	liquid2connector "github.com/c86j224s/liquid2/plasma/internal/connectors/liquid2"
	"github.com/c86j224s/liquid2/plasma/internal/mcp"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

func runMCP(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	liquid2URL := fs.String("liquid2-url", "", "optional Liquid2 base URL")
	localRoots := repeatedStringFlag{}
	fs.Var(&localRoots, "local-source-root", "allowlisted local source root root_id=path; repeatable")
	enabledTools := repeatedStringFlag{}
	fs.Var(&enabledTools, "enabled-tool", "MCP tool name to expose; repeatable; empty exposes server defaults")
	missionID := fs.String("mission-id", "", "mission id this MCP server is allowed to access")
	agentSessionID := fs.String("agent-session-id", "", "agent session id this MCP server is allowed to write as")
	currentUserEventID := fs.String("current-user-event-id", "", "current user turn event id for deferred workflow starts")
	agentExecutor := fs.String("agent-executor", "", "agent executor this MCP server is bound to")
	legacyResearchLoop := fs.Bool("legacy-research-loop", false, "developer-only: expose legacy evidence/claim/proposal mutation tools")
	experimentalReportComposition := fs.Bool("experimental-report-composition", false, "developer-only: expose experimental report composition tools")
	reportPatch := fs.Bool("report-patch", false, "report-session only: expose report artifact patch tools")
	reportPatchBaseArtifactID := fs.String("report-patch-base-artifact-id", "", "report patch base artifact id")
	reportPatchPendingEventID := fs.String("report-patch-pending-event-id", "", "report patch pending event id")
	reportPatchAgentExecutor := fs.String("report-patch-agent-executor", "", "report patch agent executor")
	reportPatchAgentModel := fs.String("report-patch-agent-model", "", "report patch agent model")
	reportPatchAgentReasoningEffort := fs.String("report-patch-agent-reasoning-effort", "", "report patch agent reasoning effort")
	reportPatchMCPMode := fs.String("report-patch-mcp-mode", "", "report patch MCP mode")
	reportPatchAgentSessionID := fs.String("report-patch-agent-session-id", "", "report patch agent session id")
	reportPatchPreviousAgentSessionID := fs.String("report-patch-previous-agent-session-id", "", "report patch previous agent session id")
	reportPatchReturnedAgentSessionID := fs.String("report-patch-returned-agent-session-id", "", "report patch returned agent session id")
	reportPatchReportSessionID := fs.String("report-patch-report-session-id", "", "report patch report session id")
	reportPatchForkSourceAgentSessionID := fs.String("report-patch-fork-source-agent-session-id", "", "report patch fork source agent session id")
	reportPatchReportSessionPolicy := fs.String("report-patch-report-session-policy", "", "report patch report session policy")
	reportPatchReportSessionPolicySelection := fs.String("report-patch-report-session-policy-selection", "", "report patch report session policy selection")
	reportPatchSessionChainKind := fs.String("report-patch-session-chain-kind", "", "report patch session chain kind")
	reportPlanPendingEventID := fs.String("report-plan-pending-event-id", "", "report plan pending event id")
	reportPlanMode := fs.String("report-plan-mode", "", "expected report plan mode")
	reportPlanIdempotencyKey := fs.String("report-plan-idempotency-key", "", "expected report plan idempotency key")
	reportPlanToolSessionID := fs.String("report-plan-tool-session-id", "", "report plan MCP tool session id")
	reportPlanPreviousProviderSessionID := fs.String("report-plan-previous-provider-session-id", "", "optional provider session resumed by planning")
	reportPlanAgentModel := fs.String("report-plan-agent-model", "", "server-bound report planning model")
	reportPlanAgentReasoningEffort := fs.String("report-plan-agent-reasoning-effort", "", "server-bound report planning reasoning effort")
	longFormFinalizeBindingJSON := fs.String("report-long-form-finalize-binding-json", "", "server-bound long-form finalization metadata")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	binding := mcp.Binding{
		MissionID:          strings.TrimSpace(*missionID),
		AgentSessionID:     strings.TrimSpace(*agentSessionID),
		CurrentUserEventID: strings.TrimSpace(*currentUserEventID),
		AgentExecutor:      strings.TrimSpace(*agentExecutor),
	}
	if strings.TrimSpace(binding.AgentExecutor) != "" {
		normalizedAgentExecutor, err := app.NormalizeAgentExecutorName(binding.AgentExecutor)
		if err != nil {
			fmt.Fprintf(stderr, "mcp binding: %v\n", err)
			return 2
		}
		binding.AgentExecutor = normalizedAgentExecutor
	}
	if err := validateMCPBinding(binding); err != nil {
		fmt.Fprintf(stderr, "mcp binding: %v\n", err)
		return 2
	}

	cfg, err := config.Load(config.Args{
		DBPath:           *dbPath,
		Liquid2URL:       *liquid2URL,
		LocalSourceRoots: []string(localRoots),
	})
	if err != nil {
		fmt.Fprintf(stderr, "config: %v\n", err)
		return 2
	}
	effectiveDBPath := cfg.EffectiveDBPath()
	store, err := sqlite.Open(ctx, effectiveDBPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer store.Close()
	svc, err := newCLIService(store, cfg, nil)
	if err != nil {
		fmt.Fprintf(stderr, "local source roots: %v\n", err)
		return 2
	}
	var options []mcp.Option
	if strings.TrimSpace(cfg.Liquid2URL) != "" {
		connector, err := liquid2connector.NewClient(cfg.Liquid2URL)
		if err != nil {
			fmt.Fprintf(stderr, "liquid2 connector: %v\n", err)
			return 2
		}
		options = append(options, mcp.WithLiquid2Connector(connector))
	}
	options = append(options, mcp.WithConfluenceConnectorFactory(func(ctx context.Context, req mcp.ConfluenceConnectorRequest) (app.ConfluenceSourceConnector, error) {
		return cliConfluenceClient(ctx, svc, req.ConnectionID, req.CloudID, "", "", false)
	}))

	options = append(options, mcp.WithBinding(binding))
	if *legacyResearchLoop {
		options = append(options, mcp.WithLegacyResearchLoop())
	}
	if *experimentalReportComposition {
		options = append(options, mcp.WithExperimentalReportComposition())
	}
	if *reportPatch {
		options = append(options, mcp.WithReportPatch())
		options = append(options, mcp.WithReportPatchBinding(mcp.ReportPatchBinding{
			BaseArtifactID:               strings.TrimSpace(*reportPatchBaseArtifactID),
			PendingEventID:               strings.TrimSpace(*reportPatchPendingEventID),
			AgentExecutor:                strings.TrimSpace(*reportPatchAgentExecutor),
			AgentModel:                   strings.TrimSpace(*reportPatchAgentModel),
			AgentReasoningEffort:         strings.TrimSpace(*reportPatchAgentReasoningEffort),
			MCPMode:                      strings.TrimSpace(*reportPatchMCPMode),
			AgentSessionID:               strings.TrimSpace(*reportPatchAgentSessionID),
			PreviousAgentSessionID:       strings.TrimSpace(*reportPatchPreviousAgentSessionID),
			ReturnedAgentSessionID:       strings.TrimSpace(*reportPatchReturnedAgentSessionID),
			ReportSessionID:              strings.TrimSpace(*reportPatchReportSessionID),
			ForkSourceAgentSessionID:     strings.TrimSpace(*reportPatchForkSourceAgentSessionID),
			ReportSessionPolicy:          strings.TrimSpace(*reportPatchReportSessionPolicy),
			ReportSessionPolicySelection: strings.TrimSpace(*reportPatchReportSessionPolicySelection),
			SessionChainKind:             strings.TrimSpace(*reportPatchSessionChainKind),
		}))
	}
	if strings.TrimSpace(*reportPlanPendingEventID) != "" || strings.TrimSpace(*reportPlanMode) != "" || strings.TrimSpace(*reportPlanIdempotencyKey) != "" || strings.TrimSpace(*reportPlanToolSessionID) != "" || strings.TrimSpace(*reportPlanPreviousProviderSessionID) != "" || strings.TrimSpace(*reportPlanAgentModel) != "" || strings.TrimSpace(*reportPlanAgentReasoningEffort) != "" {
		planBinding := mcp.ReportPlanBinding{PendingEventID: *reportPlanPendingEventID, ReportMode: *reportPlanMode, IdempotencyKey: *reportPlanIdempotencyKey, ToolSessionID: *reportPlanToolSessionID, PreviousProviderSessionID: *reportPlanPreviousProviderSessionID, AgentExecutor: binding.AgentExecutor, AgentModel: *reportPlanAgentModel, AgentReasoningEffort: *reportPlanAgentReasoningEffort}
		if err := mcp.ValidateReportPlanBinding(binding, planBinding); err != nil {
			fmt.Fprintf(stderr, "mcp report plan binding: %v\n", err)
			return 2
		}
		options = append(options, mcp.WithReportPlanBinding(planBinding))
	}
	if strings.TrimSpace(*longFormFinalizeBindingJSON) != "" {
		var finalBinding reporting.LongFormFinalizeBinding
		decoder := json.NewDecoder(strings.NewReader(*longFormFinalizeBindingJSON))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&finalBinding); err != nil {
			fmt.Fprintf(stderr, "mcp long-form finalization binding: %v\n", err)
			return 2
		}
		var extra any
		if err := decoder.Decode(&extra); err != io.EOF {
			fmt.Fprintln(stderr, "mcp long-form finalization binding: multiple JSON values")
			return 2
		}
		if err := mcp.ValidateLongFormFinalizeBinding(binding, finalBinding); err != nil {
			fmt.Fprintf(stderr, "mcp long-form finalization binding: %v\n", err)
			return 2
		}
		options = append(options, mcp.WithLongFormFinalizeBinding(finalBinding))
	}
	if len(enabledTools) > 0 {
		options = append(options, mcp.WithEnabledTools([]string(enabledTools)))
	}

	if err := mcp.ServeStdio(ctx, stdin, stdout, mcp.NewServer(svc, options...)); err != nil {
		fmt.Fprintf(stderr, "mcp: %v\n", err)
		return 1
	}
	return 0
}

func validateMCPBinding(binding mcp.Binding) error {
	if !hasIDPrefix(binding.MissionID, "mis_") {
		return fmt.Errorf("mission-id is required and must start with mis_")
	}
	if !hasIDPrefix(binding.AgentSessionID, "ses_") {
		return fmt.Errorf("agent-session-id is required and must start with ses_")
	}
	if strings.TrimSpace(binding.CurrentUserEventID) != "" && !hasIDPrefix(binding.CurrentUserEventID, "evt_") {
		return fmt.Errorf("current-user-event-id must start with evt_ when provided")
	}
	if strings.TrimSpace(binding.AgentExecutor) == "" {
		return fmt.Errorf("agent-executor is required")
	}
	if _, err := app.NormalizeAgentExecutorName(binding.AgentExecutor); err != nil {
		return err
	}
	return nil
}

func hasIDPrefix(value string, prefix string) bool {
	value = strings.TrimSpace(value)
	return strings.HasPrefix(value, prefix) && len(value) > len(prefix)
}
