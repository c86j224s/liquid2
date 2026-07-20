package mcp

import (
	"context"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/mermaid"
)

func (server *Server) callMermaidValidate(_ context.Context, call ToolCall) ToolResult {
	var input mermaidValidateInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.enforceBoundMission(missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	return ToolResult{
		ToolName:  call.Name,
		MissionID: missionID,
		Content:   mermaid.Validate(input.Source),
	}
}
