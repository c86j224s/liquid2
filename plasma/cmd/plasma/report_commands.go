package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentmodels"
	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/config"
	"github.com/c86j224s/liquid2/plasma/internal/conversation"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/web"
	workflowruntime "github.com/c86j224s/liquid2/plasma/internal/workflow"
)

func runReports(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: plasma reports <draft|patch> ...")
		return 2
	}
	switch args[0] {
	case "draft":
		return runReportsDraft(ctx, args[1:], stdout, stderr)
	case "patch":
		return runReportsPatch(ctx, args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown reports command %q\n", args[0])
		return 2
	}
}

func runReportsDraft(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("reports draft", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	title := fs.String("title", "Mission report", "report title")
	directionHint := fs.String("direction-hint", "", "request-specific weak editorial direction")
	mode := fs.String("mode", reporting.DefaultMode, "report mode: planned or one_take; long_form is browser/API-only until the CLI section runner is added")
	agentName := fs.String("agent", "", "agent executor")
	agentModel := fs.String("agent-model", "", "report agent model for this request")
	agentReasoningEffort := fs.String("agent-reasoning-effort", "", "report reasoning effort for this request")
	mcpMode := fs.String("mcp-mode", "auto", "MCP mode")
	wait := fs.Bool("wait", false, "run the report agent and wait for the artifact")
	jsonOut := fs.Bool("json", false, "write JSON")
	humanize := fs.Bool("humanize", false, "run the optional post-report H5 humanize pass")
	generationGuidance := fs.String("generation-guidance", "g2", "report generation guidance profile: g2 or none")
	experimentalGenerationGuidance := fs.String("experimental-generation-guidance", "", "deprecated alias for -generation-guidance")
	reportSessionPolicyFlag := fs.String("report-session-policy", "", "report session policy: auto, same_session, or isolated_fork")
	liquid2URL := fs.String("liquid2-url", "", "optional Liquid2 base URL")
	codexCommand := fs.String("codex-command", "", "Codex CLI command")
	claudeCommand := fs.String("claude-command", "", "Claude Code CLI command")
	claudeModel := fs.String("claude-model", "", "Claude model alias")
	claudeMaxBudgetUSD := fs.String("claude-max-budget-usd", "", "optional Claude max budget per turn")
	agentWorkDir := fs.String("agent-workdir", "", "agent working directory")
	agentTimeout := fs.Duration("agent-timeout", 0, "agent response timeout; 0 disables the limit")
	localRoots := repeatedStringFlag{}
	fs.Var(&localRoots, "local-source-root", "allowlisted local source root root_id=path; repeatable")
	positionals, parseArgs := leadingPositionals(args, 1)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 1 {
		fmt.Fprintln(stderr, "usage: plasma reports draft <mission_id> --title ... --wait")
		return 2
	}
	if !*wait {
		fmt.Fprintln(stderr, "reports draft currently requires --wait because no CLI background worker is installed")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath, []string(localRoots)...)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	missionID := positionals[0]
	reportTitle := strings.TrimSpace(*title)
	if reportTitle == "" {
		reportTitle = "Mission report"
	}
	reportMode, err := reporting.NormalizeMode(*mode)
	if err != nil {
		fmt.Fprintf(stderr, "reports draft: %v\n", err)
		return 2
	}
	if reportMode == reporting.ModeLongForm {
		fmt.Fprintln(stderr, "reports draft: --mode long_form requires the browser/report API section runner; CLI reports currently support planned or one_take")
		return 2
	}
	guidanceSelection := *generationGuidance
	if flagWasSet(fs, "experimental-generation-guidance") {
		guidanceSelection = *experimentalGenerationGuidance
	}
	guidanceProfile, guidanceSHA, err := cliReportGenerationGuidanceSelection(guidanceSelection)
	if err != nil {
		fmt.Fprintf(stderr, "reports draft: %v\n", err)
		return 2
	}
	agentCfg, effectiveAgentTimeout, err := loadAgentConfig(config.Args{
		DBPath:             *dbPath,
		Liquid2URL:         stringFlagArg(fs, "liquid2-url", *liquid2URL),
		Agent:              stringFlagArg(fs, "agent", *agentName),
		CodexCommand:       stringFlagArg(fs, "codex-command", *codexCommand),
		ClaudeCommand:      stringFlagArg(fs, "claude-command", *claudeCommand),
		ClaudeModel:        stringFlagArg(fs, "claude-model", *claudeModel),
		ClaudeMaxBudgetUSD: stringFlagArg(fs, "claude-max-budget-usd", *claudeMaxBudgetUSD),
		AgentWorkDir:       *agentWorkDir,
		AgentTimeout:       durationFlagArg(fs, "agent-timeout", *agentTimeout),
		LocalSourceRoots:   listFlagArg(fs, "local-source-root", []string(localRoots)),
	})
	if err != nil {
		fmt.Fprintf(stderr, "config: %v\n", err)
		return 2
	}
	resolvedAgentName := firstNonEmptyString(strings.TrimSpace(agentCfg.Agent), "codex")
	executor, err := newCLIAgentExecutor(ctx, cliAgentConfig{
		AgentName:          resolvedAgentName,
		DBPath:             agentCfg.EffectiveDBPath(),
		Liquid2URL:         strings.TrimSpace(agentCfg.Liquid2URL),
		CodexCommand:       strings.TrimSpace(agentCfg.CodexCommand),
		ClaudeCommand:      strings.TrimSpace(agentCfg.ClaudeCommand),
		ClaudeModel:        strings.TrimSpace(agentCfg.ClaudeModel),
		ClaudeMaxBudgetUSD: strings.TrimSpace(agentCfg.ClaudeMaxBudgetUSD),
		AgentWorkDir:       strings.TrimSpace(agentCfg.AgentWorkDir),
		AgentTimeout:       effectiveAgentTimeout,
		LocalRoots:         agentCfg.LocalSourceRoots,
	})
	if err != nil {
		fmt.Fprintf(stderr, "agent: %v\n", err)
		return 2
	}
	events, err := svc.ListEvents(ctx, missionID)
	if err != nil {
		fmt.Fprintf(stderr, "reports draft: %v\n", err)
		return 1
	}
	preReportSessionID := workflowruntime.LatestAgentSessionID(events, resolvedAgentName)
	providerModel := strings.TrimSpace(agentCfg.ClaudeModel)
	if resolvedAgentName == "claude" && providerModel == "" {
		providerModel = "haiku"
	}
	providerEffort := ""
	reasoningSupported := false
	if resolvedAgentName == "codex" {
		providerModel = agentmodels.DefaultModel
		providerEffort = agentmodels.DefaultReasoningEffort
		reasoningSupported = true
	}
	selection, err := reporting.ResolveModelSelection(reporting.ModelSelectionInput{
		Executor: resolvedAgentName, RequestModel: *agentModel, RequestReasoningEffort: *agentReasoningEffort,
		SessionModel: conversation.LatestAgentModel(events, resolvedAgentName), SessionReasoningEffort: conversation.LatestAgentReasoningEffort(events, resolvedAgentName),
		ProviderModel: providerModel, ProviderReasoningEffort: providerEffort, ReasoningEffortSupported: reasoningSupported,
	})
	if err != nil {
		fmt.Fprintf(stderr, "reports draft: %v\n", err)
		return 2
	}
	_, canFork := executor.(web.AgentSessionForker)
	_, canCheckFork := executor.(web.AgentSessionForkReadiness)
	reportSessionPolicy, reportSessionPolicySelection, err := reporting.SelectSessionPolicy(reporting.SessionPolicySelectionInput{
		RequestedPolicy:             cliRequestedReportSessionPolicy(*reportSessionPolicyFlag),
		ReportMode:                  reportMode,
		CanForkSession:              canFork,
		HasPreReportResearchSession: strings.TrimSpace(preReportSessionID) != "",
		ForkReady:                   canFork && canCheckFork && web.AgentSessionForkReady(ctx, executor, preReportSessionID),
	})
	if err != nil {
		fmt.Fprintf(stderr, "reports draft: %v\n", err)
		return 1
	}
	inFlight := &reporting.InFlight{}
	inFlight.SetNewID(cliNewID)
	resultCh := make(chan cliReportDraftRunResult, 1)
	var runner reporting.Runner
	runner = reporting.Runner{
		Service:  svc,
		InFlight: inFlight,
		NewID:    cliNewID,
		GenerateDraft: func(runCtx context.Context, runMissionID string, req reporting.DraftRequest, pendingEventID string) error {
			result := createCLIReportDraftArtifact(runCtx, svc, executor, runMissionID, pendingEventID, req)
			if result.Err != nil {
				_, _ = runner.AppendDraftFailed(runCtx, runMissionID, pendingEventID, req.AgentExecutor, req.ReportMode, result.Err)
			}
			resultCh <- result
			return nil
		},
	}
	pendingEvent, err := runner.StartDraft(ctx, missionID, reporting.DraftRequest{
		Title:                        reportTitle,
		DirectionHint:                *directionHint,
		AgentExecutor:                resolvedAgentName,
		AgentModel:                   selection.Model,
		AgentReasoningEffort:         selection.ReasoningEffort,
		AgentSelectionSource:         selection.Source,
		MCPMode:                      strings.TrimSpace(*mcpMode),
		ReportMode:                   reportMode,
		ReportSessionPolicy:          reportSessionPolicy,
		ReportSessionPolicySelection: reportSessionPolicySelection,
		PostReportHumanize:           cliPostReportHumanizeFlag(*humanize),
		GenerationGuidanceProfile:    guidanceProfile,
		GenerationGuidanceSHA256:     guidanceSHA,
	}, app.Producer{Type: "user", ID: "plasma-cli"})
	if err != nil {
		fmt.Fprintf(stderr, "reports draft: %v\n", err)
		return 1
	}
	runResult := <-resultCh
	if runResult.Err != nil {
		fmt.Fprintf(stderr, "reports draft: %v\n", runResult.Err)
		return 1
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{"pending_event": pendingEvent, "artifact": runResult.Artifact, "event": runResult.Event, "humanized": runResult.Humanized})
	} else {
		humanized := ""
		if runResult.Humanized.Applied {
			humanized = fmt.Sprintf(" humanized=%s", runResult.Humanized.Artifact.ArtifactID)
		}
		fmt.Fprintf(stdout, "report artifact %s event=%s session=%s%s\n", runResult.Artifact.ArtifactID, runResult.Event.EventID, runResult.SessionID, humanized)
	}
	return 0
}

func runReportsPatch(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("reports patch", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	baseArtifactID := fs.String("base-artifact", "", "base Markdown report artifact id")
	instruction := fs.String("instruction", "", "patch instruction")
	title := fs.String("title", "", "patched report title")
	agentName := fs.String("agent", "", "agent executor; defaults to the base report executor")
	mcpMode := fs.String("mcp-mode", "auto", "MCP mode")
	reportSessionPolicy := fs.String("report-session-policy", "", "report patch session policy: same_session or isolated_fork")
	wait := fs.Bool("wait", false, "run the report patch agent and wait for the artifact")
	jsonOut := fs.Bool("json", false, "write JSON")
	liquid2URL := fs.String("liquid2-url", "", "optional Liquid2 base URL")
	codexCommand := fs.String("codex-command", "", "Codex CLI command")
	claudeCommand := fs.String("claude-command", "", "Claude Code CLI command")
	claudeModel := fs.String("claude-model", "", "Claude model alias")
	claudeMaxBudgetUSD := fs.String("claude-max-budget-usd", "", "optional Claude max budget per turn")
	agentWorkDir := fs.String("agent-workdir", "", "agent working directory")
	agentTimeout := fs.Duration("agent-timeout", 0, "agent response timeout; 0 disables the limit")
	localRoots := repeatedStringFlag{}
	fs.Var(&localRoots, "local-source-root", "allowlisted local source root root_id=path; repeatable")
	positionals, parseArgs := leadingPositionals(args, 1)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 1 || strings.TrimSpace(*baseArtifactID) == "" || strings.TrimSpace(*instruction) == "" {
		fmt.Fprintln(stderr, "usage: plasma reports patch <mission_id> --base-artifact art_... --instruction ... --wait")
		return 2
	}
	if !*wait {
		fmt.Fprintln(stderr, "reports patch currently requires --wait because no CLI background worker is installed")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath, []string(localRoots)...)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	missionID := positionals[0]
	info, err := cliReportArtifactSessionInfo(ctx, svc, missionID, strings.TrimSpace(*baseArtifactID))
	if err != nil {
		fmt.Fprintf(stderr, "reports patch: %v\n", err)
		return 1
	}
	agentArg := ""
	if flagWasSet(fs, "agent") {
		agentArg = strings.TrimSpace(*agentName)
	} else if strings.TrimSpace(info.AgentExecutor) != "" {
		agentArg = strings.TrimSpace(info.AgentExecutor)
	}
	agentCfg, effectiveAgentTimeout, err := loadAgentConfig(config.Args{
		DBPath:             *dbPath,
		Liquid2URL:         stringFlagArg(fs, "liquid2-url", *liquid2URL),
		Agent:              agentArg,
		CodexCommand:       stringFlagArg(fs, "codex-command", *codexCommand),
		ClaudeCommand:      stringFlagArg(fs, "claude-command", *claudeCommand),
		ClaudeModel:        stringFlagArg(fs, "claude-model", *claudeModel),
		ClaudeMaxBudgetUSD: stringFlagArg(fs, "claude-max-budget-usd", *claudeMaxBudgetUSD),
		AgentWorkDir:       *agentWorkDir,
		AgentTimeout:       durationFlagArg(fs, "agent-timeout", *agentTimeout),
		LocalSourceRoots:   listFlagArg(fs, "local-source-root", []string(localRoots)),
	})
	if err != nil {
		fmt.Fprintf(stderr, "config: %v\n", err)
		return 2
	}
	executorName := firstNonEmptyString(strings.TrimSpace(agentCfg.Agent), "codex")
	if info.AgentExecutor != "" && info.AgentExecutor != executorName {
		fmt.Fprintf(stderr, "reports patch: base artifact uses agent %q\n", info.AgentExecutor)
		return 2
	}
	executor, err := newCLIAgentExecutor(ctx, cliAgentConfig{
		AgentName:          strings.TrimSpace(agentCfg.Agent),
		DBPath:             agentCfg.EffectiveDBPath(),
		Liquid2URL:         strings.TrimSpace(agentCfg.Liquid2URL),
		CodexCommand:       strings.TrimSpace(agentCfg.CodexCommand),
		ClaudeCommand:      strings.TrimSpace(agentCfg.ClaudeCommand),
		ClaudeModel:        strings.TrimSpace(agentCfg.ClaudeModel),
		ClaudeMaxBudgetUSD: strings.TrimSpace(agentCfg.ClaudeMaxBudgetUSD),
		AgentWorkDir:       strings.TrimSpace(agentCfg.AgentWorkDir),
		AgentTimeout:       effectiveAgentTimeout,
		LocalRoots:         agentCfg.LocalSourceRoots,
	})
	if err != nil {
		fmt.Fprintf(stderr, "agent: %v\n", err)
		return 2
	}
	selection, err := web.SelectReportPatchSession(ctx, executor, info.ReportSessionID, *reportSessionPolicy)
	if err != nil {
		fmt.Fprintf(stderr, "reports patch: %v\n", err)
		return 1
	}
	patchTitle := strings.TrimSpace(*title)
	if patchTitle == "" {
		patchTitle = firstNonEmptyString(info.Title+" 수정본", "Patched report")
	}
	inFlight := &reporting.InFlight{}
	inFlight.SetNewID(cliNewID)
	resultCh := make(chan cliReportPatchRunResult, 1)
	var runner reporting.Runner
	runner = reporting.Runner{
		Service:  svc,
		InFlight: inFlight,
		NewID:    cliNewID,
		GeneratePatch: func(runCtx context.Context, runMissionID string, req reporting.PatchRequest, pendingEventID string) error {
			result := createCLIReportPatchArtifact(runCtx, svc, executor, runMissionID, pendingEventID, req)
			if result.Err != nil {
				_, _ = runner.AppendPatchFailed(runCtx, runMissionID, pendingEventID, req.AgentExecutor, req.BaseArtifactID, result.Err)
			}
			resultCh <- result
			return nil
		},
	}
	pendingEvent, err := runner.StartPatch(ctx, missionID, reporting.PatchRequest{
		BaseArtifactID:               strings.TrimSpace(*baseArtifactID),
		Instruction:                  strings.TrimSpace(*instruction),
		Title:                        patchTitle,
		AgentExecutor:                executorName,
		AgentModel:                   info.AgentModel,
		AgentReasoningEffort:         info.AgentReasoningEffort,
		MCPMode:                      strings.TrimSpace(*mcpMode),
		ReportSessionID:              selection.SessionID,
		PreviousAgentSessionID:       selection.PreviousAgentSessionID,
		ForkSourceAgentSessionID:     selection.ForkSourceAgentSessionID,
		ReportSessionPolicy:          selection.ReportSessionPolicy,
		ReportSessionPolicySelection: selection.ReportSessionPolicySelection,
		SessionChainKind:             selection.SessionChainKind,
	}, app.Producer{Type: "user", ID: "plasma-cli"})
	if err != nil {
		fmt.Fprintf(stderr, "reports patch: %v\n", err)
		return 1
	}
	runResult := <-resultCh
	if runResult.Err != nil {
		fmt.Fprintf(stderr, "reports patch: %v\n", runResult.Err)
		return 1
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{"pending_event": pendingEvent, "artifact": runResult.Artifact, "event": runResult.Event, "session_id": runResult.SessionID})
	} else {
		fmt.Fprintf(stdout, "patched report artifact %s event=%s session=%s\n", runResult.Artifact.ArtifactID, runResult.Event.EventID, runResult.SessionID)
	}
	return 0
}
