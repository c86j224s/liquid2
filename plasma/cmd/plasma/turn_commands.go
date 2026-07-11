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
	"github.com/c86j224s/liquid2/plasma/internal/conversation"
	"github.com/c86j224s/liquid2/plasma/internal/web"
	workflowruntime "github.com/c86j224s/liquid2/plasma/internal/workflow"
)

func runTurns(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "send" {
		fmt.Fprintln(stderr, "usage: plasma turns send <mission_id> --text ... --wait")
		return 2
	}
	fs := flag.NewFlagSet("turns send", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	text := fs.String("text", "", "turn text")
	agentName := fs.String("agent", "", "agent executor")
	mcpMode := fs.String("mcp-mode", "auto", "MCP mode")
	wait := fs.Bool("wait", false, "run the agent and wait for the response")
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
	positionals, parseArgs := leadingPositionals(args[1:], 1)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 1 {
		fmt.Fprintln(stderr, "usage: plasma turns send <mission_id> --text ... --wait")
		return 2
	}
	turnText := strings.TrimSpace(*text)
	if turnText == "" {
		fmt.Fprintln(stderr, "text is required")
		return 2
	}
	if !*wait {
		fmt.Fprintln(stderr, "turns send currently requires --wait because no CLI background worker is installed")
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
	toolSessionID := cliNewID("ses")
	userEventReq := conversation.BuildTurnUserAppendRequest(conversation.TurnUserEventRequest{
		EventID:       cliNewID("evt"),
		MissionID:     missionID,
		Kind:          "user_turn",
		Text:          turnText,
		AgentExecutor: resolvedAgentName,
		MCPMode:       strings.TrimSpace(*mcpMode),
		ToolSessionID: toolSessionID,
		Producer:      app.Producer{Type: "user", ID: "plasma-cli"},
	})
	pendingEventReq := conversation.BuildTurnAgentPendingAppendRequest(conversation.TurnAgentPendingEventRequest{
		EventID:       cliNewID("evt"),
		MissionID:     missionID,
		AgentExecutor: resolvedAgentName,
		MCPMode:       strings.TrimSpace(*mcpMode),
		Text:          "CLI 에이전트 응답을 기다리는 중입니다.",
		UserEventID:   userEventReq.EventID,
		ToolSessionID: toolSessionID,
		StartedAt:     time.Now().UTC().Format(time.RFC3339Nano),
		Producer:      app.Producer{Type: "agent", ID: resolvedAgentName},
	})
	appendedEvents, err := svc.AppendEventsIfNoActiveAgentWork(ctx, missionID, []app.AppendEventRequest{userEventReq, pendingEventReq})
	if err != nil {
		fmt.Fprintf(stderr, "turns send: %v\n", err)
		return 1
	}
	userEvent := appendedEvents[0]
	pendingEvent := appendedEvents[1]
	events, _ := svc.ListEvents(ctx, missionID)
	previousSessionID := workflowruntime.LatestAgentSessionID(events, resolvedAgentName)
	result, err := executor.Run(ctx, web.AgentRequest{
		UserText:          turnText,
		Prompt:            cliTurnPrompt(missionID, turnText, toolSessionID),
		MissionID:         missionID,
		ToolSessionID:     toolSessionID,
		UserEventID:       userEvent.EventID,
		PreviousSessionID: previousSessionID,
		AgentExecutor:     resolvedAgentName,
		MCPMode:           strings.TrimSpace(*mcpMode),
	})
	if err != nil {
		_, _ = svc.AppendEvent(ctx, conversation.BuildTurnAgentResponseAppendRequest(conversation.TurnAgentResponseEventRequest{
			EventID:       cliNewID("evt"),
			MissionID:     missionID,
			Kind:          "agent_error",
			AgentExecutor: resolvedAgentName,
			Text:          "CLI 에이전트 실행이 실패했습니다.",
			Error:         err.Error(),
			IncludeError:  true,
			UserEventID:   userEvent.EventID,
			Extra: map[string]any{
				"tool_session_id": toolSessionID,
			},
			Producer: app.Producer{Type: "agent", ID: resolvedAgentName},
		}))
		if drainErr := drainCLIQueuedWorkflows(ctx, svc, missionID, executor, resolvedAgentName); drainErr != nil {
			fmt.Fprintf(stderr, "workflow drain: %v\n", drainErr)
			return 1
		}
		fmt.Fprintf(stderr, "agent run: %v\n", err)
		return 1
	}
	sessionID, err := cliValidatedSessionID(result.SessionID, previousSessionID)
	if err != nil {
		_, _ = svc.AppendEvent(ctx, conversation.BuildTurnAgentResponseAppendRequest(conversation.TurnAgentResponseEventRequest{
			EventID:       cliNewID("evt"),
			MissionID:     missionID,
			Kind:          "agent_error",
			AgentExecutor: resolvedAgentName,
			Text:          "CLI 에이전트가 이전 provider session과 다른 session을 반환했습니다.",
			Error:         err.Error(),
			IncludeError:  true,
			UserEventID:   userEvent.EventID,
			Extra: map[string]any{
				"tool_session_id":           toolSessionID,
				"previous_agent_session_id": previousSessionID,
			},
			Producer: app.Producer{Type: "agent", ID: resolvedAgentName},
		}))
		if drainErr := drainCLIQueuedWorkflows(ctx, svc, missionID, executor, resolvedAgentName); drainErr != nil {
			fmt.Fprintf(stderr, "workflow drain: %v\n", drainErr)
			return 1
		}
		fmt.Fprintf(stderr, "agent session: %v\n", err)
		return 1
	}
	result.SessionID = sessionID
	responseEvent, err := svc.AppendEvent(ctx, conversation.BuildTurnAgentResponseAppendRequest(conversation.TurnAgentResponseEventRequest{
		EventID:               cliNewID("evt"),
		MissionID:             missionID,
		Kind:                  "agent_response",
		AgentExecutor:         resolvedAgentName,
		MCPMode:               strings.TrimSpace(*mcpMode),
		IncludeMCPMode:        true,
		Text:                  strings.TrimSpace(result.Text),
		AgentSessionID:        result.SessionID,
		IncludeAgentSessionID: true,
		Resumed:               previousSessionID != "",
		IncludeResumed:        true,
		UserEventID:           userEvent.EventID,
		Extra: map[string]any{
			"previous_agent_session_id": previousSessionID,
			"tool_session_id":           toolSessionID,
		},
		Producer: app.Producer{Type: "agent", ID: resolvedAgentName},
	}))
	if err != nil {
		fmt.Fprintf(stderr, "append turn.agent.response: %v\n", err)
		return 1
	}
	if drainErr := drainCLIQueuedWorkflows(ctx, svc, missionID, executor, resolvedAgentName); drainErr != nil {
		fmt.Fprintf(stderr, "workflow drain: %v\n", drainErr)
		return 1
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{"user_event": userEvent, "pending_event": pendingEvent, "response_event": responseEvent})
	} else {
		fmt.Fprintf(stdout, "agent response %s session=%s\n", responseEvent.EventID, result.SessionID)
	}
	return 0
}

func cliTurnPrompt(missionID string, text string, toolSessionID string) string {
	return fmt.Sprintf(`You are the Plasma research agent.

Answer the user's latest turn directly and use Korean unless the user asks otherwise.
Use Plasma read tools when useful. Start with plasma.research.outline, then inspect source or ledger material with plasma.research.list, plasma.research.grep, plasma.research.read, plasma.sources.read, plasma.sources.tree, plasma.sources.grep, and plasma.research.references.

Mission ID: %s
Tool session ID: %s

Rules:
- Sources are original materials. Your answer is a result, not a source.
- Sources may be snapshot_only pinned artifacts, PDF documents, or live_reference local_path sources. PDF reads return extracted text and metadata, not raw PDF bytes. Read live local path material through plasma.sources.read, plasma.sources.tree, or plasma.sources.grep; cite source.observed metadata such as observation_event_id, observed_at, relative_path, sha256, and git state when using it.
- Do not paste original source bodies or full conversation history into the prompt or answer.
- Do not create evidence, claims, confidence updates, proposal bundles, report blocks, or report AST JSON in the default C1 path.

Latest user turn:
%s`, strings.TrimSpace(missionID), strings.TrimSpace(toolSessionID), strings.TrimSpace(text))
}
