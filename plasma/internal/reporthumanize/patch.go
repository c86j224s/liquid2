package reporthumanize

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reportpatch"
)

func patchRequest(input Input) reportexecution.PatchRequest {
	return reportexecution.PatchRequest{
		BaseArtifactID:               strings.TrimSpace(input.SourceArtifact.ArtifactID),
		Instruction:                  PatchInstruction(),
		Title:                        firstNonEmpty(strings.TrimSpace(input.Title)+" humanized", "Humanized report"),
		AgentExecutor:                strings.TrimSpace(input.ExecutorName),
		AgentModel:                   strings.TrimSpace(input.AgentModel),
		AgentReasoningEffort:         strings.TrimSpace(input.ReasoningEffort),
		MCPMode:                      strings.TrimSpace(input.MCPMode),
		ReportSessionID:              strings.TrimSpace(input.PreviousSessionID),
		PreviousAgentSessionID:       strings.TrimSpace(input.PreviousSessionID),
		ReportSessionPolicy:          reportexecution.SessionPolicySameSession,
		ReportSessionPolicySelection: "auto_same_report_session_h5",
		SessionChainKind:             "same_report_session_h5_humanize_patch",
	}
}

// PatchInstruction is the exact H5 MCP patch instruction embedded in the agent
// prompt and patch context.
func PatchInstruction() string {
	return "Apply the H5 Korean tone pass to this Markdown report. Smooth stiff AI-like Korean phrasing, repetitive transitions, and unnatural sentence endings, but preserve the report structure, claims, citations, numbers, tables, code, links, headings, paragraph boundaries, and useful detail. Use bounded MCP reads and small targeted patch operations. Do not rewrite or summarize the whole report."
}

func noChangesResult(text string) bool {
	return strings.Contains(strings.TrimSpace(text), "NO_H5_CHANGES")
}

type patchActivity struct {
	Started       bool
	ApplyCount    int
	FinalizeCount int
	LastError     string
}

func patchToolActivity(ctx context.Context, service Service, missionID string, toolSessionID string) (patchActivity, error) {
	lister, ok := service.(eventLister)
	if !ok {
		return patchActivity{}, fmt.Errorf("%w: H5 MCP patch requires event listing", producterror.ErrInvalidInput)
	}
	events, err := lister.ListEvents(ctx, missionID)
	if err != nil {
		return patchActivity{}, err
	}
	toolSessionID = strings.TrimSpace(toolSessionID)
	tools := reportpatch.MCPTools()
	var activity patchActivity
	for _, event := range events {
		if event.EventType != "mcp.tool.called" {
			continue
		}
		var payload struct {
			ToolName      string `json:"tool_name"`
			ToolSessionID string `json:"tool_session_id"`
			Success       bool   `json:"success"`
			Result        struct {
				Error struct {
					Message string `json:"message"`
				} `json:"error"`
			} `json:"result"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return patchActivity{}, fmt.Errorf("%w: invalid MCP tool call payload", producterror.ErrInvalidInput)
		}
		if strings.TrimSpace(payload.ToolSessionID) != toolSessionID {
			continue
		}
		switch payload.ToolName {
		case tools[0]:
			activity.Started = true
		case tools[2]:
			activity.ApplyCount++
		case tools[3]:
			activity.FinalizeCount++
		}
		if !payload.Success {
			if message := strings.TrimSpace(payload.Result.Error.Message); message != "" {
				activity.LastError = message
			}
		}
	}
	return activity, nil
}

func finalizedPatchEvent(ctx context.Context, service Service, missionID string, pendingEventID string) (ledger.Event, bool, error) {
	lister, ok := service.(eventLister)
	if !ok {
		return ledger.Event{}, false, fmt.Errorf("%w: H5 MCP patch requires event listing", producterror.ErrInvalidInput)
	}
	events, err := lister.ListEvents(ctx, missionID)
	if err != nil {
		return ledger.Event{}, false, err
	}
	pendingEventID = strings.TrimSpace(pendingEventID)
	for index := len(events) - 1; index >= 0; index-- {
		event := events[index]
		if event.EventType != "report.patch.finalized" {
			continue
		}
		var payload struct {
			PendingEventID string `json:"pending_event_id"`
			ArtifactID     string `json:"artifact_id"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		if strings.TrimSpace(payload.PendingEventID) == pendingEventID && strings.TrimSpace(payload.ArtifactID) != "" {
			return event, true, nil
		}
	}
	return ledger.Event{}, false, nil
}
