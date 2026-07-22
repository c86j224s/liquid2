package mcp

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func (server *Server) callReportPartSectionRead(ctx context.Context, call ToolCall) ToolResult {
	var input reportPartSectionReadInput
	if err := decodeReportPlanJSON(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", "part Section read arguments are invalid", false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	sessionID := strings.TrimSpace(input.SessionID)
	if err := validateID("mis_", missionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := validateID("ses_", sessionID); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.requireBoundWriteSession(commonMutatingInput{MissionID: missionID, SessionID: sessionID}); err != nil {
		return errorResult(call.Name, missionID, "binding", err.Error(), false, nil)
	}
	binding := server.partAssemblyBinding
	if err := ValidatePartAssemblyBinding(server.binding, binding); err != nil {
		return errorResult(call.Name, missionID, "binding", "part assembly binding is incomplete", false, nil)
	}
	if err := reporting.ValidatePartAssemblySectionReadBinding(binding); err != nil {
		return errorResult(call.Name, missionID, "binding", "part Section artifacts are not bound", false, nil)
	}
	if input.SectionIndex < 1 || input.SectionIndex > len(binding.SectionArtifactIDs) {
		return errorResult(call.Name, missionID, "validation", "section_index is outside the bound Part", false, nil)
	}
	artifactID := binding.SectionArtifactIDs[input.SectionIndex-1]
	artifact, err := server.service.GetRawArtifact(ctx, artifactID)
	if err != nil {
		return errorFromErr(call.Name, missionID, err, nil)
	}
	if artifact.MissionID != missionID || !strings.HasPrefix(strings.ToLower(strings.TrimSpace(artifact.MediaType)), "text/markdown") {
		return errorResult(call.Name, missionID, "conflict", "bound Section artifact is foreign or not Markdown", false, nil)
	}
	if len(artifact.Content) == 0 || len(artifact.Content) > reportPatchMaxBytes || !utf8.Valid(artifact.Content) {
		return errorResult(call.Name, missionID, "validation", "bound Section artifact is not readable UTF-8 Markdown", false, nil)
	}
	content, offset, nextOffset, truncated, err := boundedReportPatchContent(string(artifact.Content), input.Offset, input.MaxBytes)
	if err != nil {
		return errorResult(call.Name, missionID, "validation", fmt.Sprintf("bounded Section read failed: %v", err), false, nil)
	}
	return ToolResult{ToolName: call.Name, MissionID: missionID, Content: map[string]any{
		"section_index":  input.SectionIndex,
		"content":        content,
		"offset":         offset,
		"next_offset":    nextOffset,
		"content_length": len(artifact.Content),
		"truncated":      truncated,
	}}
}
