package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/config"
	workflowruntime "github.com/c86j224s/liquid2/plasma/internal/workflow"
)

func runWorkflow(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: plasma workflow <start|status|stop> ...")
		return 2
	}
	switch args[0] {
	case "start":
		return runWorkflowStart(ctx, args[1:], stdout, stderr)
	case "status":
		return runWorkflowStatus(ctx, args[1:], stdout, stderr)
	case "stop":
		return runWorkflowStop(ctx, args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown workflow command %q\n", args[0])
		return 2
	}
}

func runWorkflowStart(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("workflow start", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	instruction := fs.String("instruction", "", "workflow instruction")
	stepInstructionMode := fs.String("step-instruction-mode", app.WorkflowStepInstructionModeLayered, "workflow step instruction mode: layered; legacy current input is accepted and normalized to layered")
	userInstructionRaw := fs.String("user-instruction-raw", "", "original autonomous-run request for layered mode")
	runGoal := fs.String("run-goal", "", "derived autonomous-run goal for layered mode")
	agentName := fs.String("agent", "", "agent executor")
	mcpMode := fs.String("mcp-mode", "auto", "MCP mode")
	maxSteps := fs.Int("max-steps", 10, "maximum workflow steps")
	maxDurationMS := fs.Int64("max-duration-ms", int64((25*time.Minute)/time.Millisecond), "maximum workflow duration in milliseconds")
	wait := fs.Bool("wait", false, "run immediately and wait for terminal status")
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
	if len(positionals) != 1 || strings.TrimSpace(*instruction) == "" {
		fmt.Fprintln(stderr, "usage: plasma workflow start <mission_id> --instruction ... --wait")
		return 2
	}
	if !*wait {
		fmt.Fprintln(stderr, "workflow start currently requires --wait because no CLI background worker is installed")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath, []string(localRoots)...)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	missionID := positionals[0]
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
	view, err := svc.RequestWorkflowRun(ctx, app.RequestWorkflowRunRequest{
		MissionID:           missionID,
		RequestedBySurface:  app.WorkflowSurfaceCLI,
		AgentExecutor:       resolvedAgentName,
		MCPMode:             strings.TrimSpace(*mcpMode),
		StepInstructionMode: strings.TrimSpace(*stepInstructionMode),
		UserInstructionRaw:  strings.TrimSpace(*userInstructionRaw),
		RunGoal:             strings.TrimSpace(*runGoal),
		Instruction:         strings.TrimSpace(*instruction),
		MaxSteps:            *maxSteps,
		MaxDurationMS:       *maxDurationMS,
		StopCondition:       "CLI bounded workflow run",
		ArgumentSummary:     strings.TrimSpace(*instruction),
	})
	if err != nil {
		fmt.Fprintf(stderr, "workflow start: %v\n", err)
		return 1
	}
	if *wait {
		runner := workflowruntime.Runner{
			Service:               svc,
			Agent:                 cliWorkflowAgentAdapter{executor: executor},
			NewID:                 cliNewID,
			SourceCandidateStager: cliSourceCandidateStager(svc),
		}
		view, err = runner.Run(ctx, missionID, view.WorkflowRunID)
		if err != nil {
			fmt.Fprintf(stderr, "workflow run: %v\n", err)
			return 1
		}
	}
	writeWorkflowView(stdout, view, *jsonOut)
	return 0
}

func runWorkflowStatus(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("workflow status", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	jsonOut := fs.Bool("json", false, "write JSON")
	positionals, parseArgs := leadingPositionals(args, 2)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) < 1 || len(positionals) > 2 {
		fmt.Fprintln(stderr, "usage: plasma workflow status <mission_id> [workflow_run_id]")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	missionID := positionals[0]
	if len(positionals) == 2 {
		view, err := svc.GetWorkflowRun(ctx, missionID, positionals[1])
		if err != nil {
			fmt.Fprintf(stderr, "workflow status: %v\n", err)
			return 1
		}
		writeWorkflowView(stdout, view, *jsonOut)
		return 0
	}
	runs, err := svc.ListWorkflowRuns(ctx, missionID)
	if err != nil {
		fmt.Fprintf(stderr, "workflow status: %v\n", err)
		return 1
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{"workflow_runs": runs})
		return 0
	}
	for _, run := range runs {
		fmt.Fprintf(stdout, "%s\t%s\tlatest=%s\treason=%s\n", run.WorkflowRunID, run.Status, run.LatestEventID, run.StopReason)
	}
	return 0
}

func runWorkflowStop(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("workflow stop", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	reason := fs.String("reason", "CLI stop requested", "stop reason")
	jsonOut := fs.Bool("json", false, "write JSON")
	positionals, parseArgs := leadingPositionals(args, 2)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 2 {
		fmt.Fprintln(stderr, "usage: plasma workflow stop <mission_id> <workflow_run_id>")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	view, err := svc.RequestWorkflowStop(ctx, app.RequestWorkflowStopRequest{
		MissionID:          positionals[0],
		WorkflowRunID:      positionals[1],
		RequestedBySurface: app.WorkflowSurfaceCLI,
		Reason:             *reason,
	})
	if err != nil {
		fmt.Fprintf(stderr, "workflow stop: %v\n", err)
		return 1
	}
	writeWorkflowView(stdout, view, *jsonOut)
	return 0
}
