package mcp

import (
	"context"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func (server *Server) callMissionGet(ctx context.Context, call ToolCall) ToolResult {
	var input missionGetInput
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
	projection, err := server.service.GetProjection(ctx, missionID)
	if err != nil {
		return errorFromErr(call.Name, missionID, err, nil)
	}
	var sources []sourceSnapshotOutput
	if includeRequested(input.Include, "sources") {
		snapshots, err := server.service.ListSourceSnapshotsWithState(ctx, app.ListSourceSnapshotsRequest{MissionID: missionID})
		if err != nil {
			return errorFromErr(call.Name, missionID, err, nil)
		}
		sources = sourceSnapshotsFromApp(snapshots)
	}
	var evidence []app.EvidenceRecord
	if includeRequested(input.Include, "evidence") || includeRequested(input.Include, "records") {
		evidence, err = server.service.ListEvidenceRecords(ctx, missionID)
		if err != nil {
			return errorFromErr(call.Name, missionID, err, nil)
		}
	}
	var claims []app.ClaimRecord
	if includeRequested(input.Include, "claims") || includeRequested(input.Include, "records") {
		claims, err = server.service.ListClaimRecords(ctx, missionID)
		if err != nil {
			return errorFromErr(call.Name, missionID, err, nil)
		}
	}
	var questions []app.QuestionRecord
	if includeRequested(input.Include, "questions") || includeRequested(input.Include, "records") {
		questions, err = server.service.ListQuestionRecords(ctx, missionID)
		if err != nil {
			return errorFromErr(call.Name, missionID, err, nil)
		}
	}
	return ToolResult{
		ToolName:  call.Name,
		MissionID: missionID,
		Content: missionGetOutput{
			MissionProjection:   projection,
			Sources:             sources,
			Evidence:            evidence,
			Claims:              claims,
			OpenQuestions:       questions,
			ActiveReportVersion: nil,
		},
	}
}
