package reportparts

import (
	"context"
	"fmt"
	patchhandler "github.com/c86j224s/liquid2/plasma/internal/mcp/reportpatch"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"strings"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func (server *Handler) CallReportPartSectionRead(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input ReportPartSectionReadInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, input.MissionID, "validation", "part Section read arguments are invalid", false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	sessionID := strings.TrimSpace(input.SessionID)
	if err := server.ValidateID("mis_", missionID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.ValidateID("ses_", sessionID); err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	if err := server.RequireSession(wire.CommonMutatingInput{MissionID: missionID, SessionID: sessionID}); err != nil {
		return server.ErrorResult(call.Name, missionID, "binding", err.Error(), false, nil)
	}
	binding := server.CurrentAssemblyBinding()
	if err := server.ValidateAssemblyBinding(binding); err != nil {
		return server.ErrorResult(call.Name, missionID, "binding", "part assembly binding is incomplete", false, nil)
	}
	if err := reporting.ValidatePartAssemblySectionReadBinding(binding); err != nil {
		return server.ErrorResult(call.Name, missionID, "binding", "part Section artifacts are not bound", false, nil)
	}
	if input.SectionIndex < 1 || input.SectionIndex > len(binding.SectionArtifactIDs) {
		return server.ErrorResult(call.Name, missionID, "validation", "section_index is outside the bound Part", false, nil)
	}
	artifactID := binding.SectionArtifactIDs[input.SectionIndex-1]
	artifact, err := server.Service.GetRawArtifact(ctx, artifactID)
	if err != nil {
		return server.ErrorFromErr(call.Name, missionID, err, nil)
	}
	if artifact.MissionID != missionID || !strings.HasPrefix(strings.ToLower(strings.TrimSpace(artifact.MediaType)), "text/markdown") {
		return server.ErrorResult(call.Name, missionID, "conflict", "bound Section artifact is foreign or not Markdown", false, nil)
	}
	if len(artifact.Content) == 0 || len(artifact.Content) > patchhandler.ReportPatchMaxBytes || !utf8.Valid(artifact.Content) {
		return server.ErrorResult(call.Name, missionID, "validation", "bound Section artifact is not readable UTF-8 Markdown", false, nil)
	}
	content, offset, nextOffset, truncated, err := patchhandler.BoundedContent(string(artifact.Content), input.Offset, input.MaxBytes)
	if err != nil {
		return server.ErrorResult(call.Name, missionID, "validation", fmt.Sprintf("bounded Section read failed: %v", err), false, nil)
	}
	return wire.ToolResult{ToolName: call.Name, MissionID: missionID, Content: map[string]any{
		"section_index":  input.SectionIndex,
		"content":        content,
		"offset":         offset,
		"next_offset":    nextOffset,
		"content_length": len(artifact.Content),
		"truncated":      truncated,
	}}
}
