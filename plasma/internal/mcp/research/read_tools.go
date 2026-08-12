package research

import (
	"context"
	"errors"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
)

func (handler *Handler) CallOutline(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input researchOutlineInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	if err := handler.validateRead(missionID, input.Legacy, input.Legacy); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	var outline app.ResearchIDEOutline
	var err error
	if input.Legacy {
		outline, err = handler.legacyReader.OutlineMissionLegacy(ctx, missionID)
	} else {
		outline, err = handler.reader.OutlineMission(ctx, missionID)
	}
	if err != nil {
		return errorFromErr(call.Name, missionID, err, nil)
	}
	return wire.ToolResult{ToolName: call.Name, MissionID: missionID, Content: outline}
}

func (handler *Handler) CallChanges(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input researchChangesInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	if err := handler.validateRead(missionID, false, false); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	changes, err := handler.reader.ListMissionChanges(ctx, app.ResearchIDEChangesRequest{
		MissionID: missionID, AfterSequence: input.AfterSequence, Limit: input.Limit,
	})
	if err != nil {
		return errorFromErr(call.Name, missionID, err, nil)
	}
	return wire.ToolResult{ToolName: call.Name, MissionID: missionID, Content: changes}
}

func (handler *Handler) CallList(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input researchListInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	if err := handler.validateRead(missionID, input.Legacy, input.Legacy); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	var page app.ResearchIDEPage
	var err error
	if input.Legacy {
		page, err = handler.legacyReader.ListMissionObjectsLegacy(ctx, missionID, input.ObjectKind, input.Limit, input.Cursor)
	} else {
		page, err = handler.reader.ListMissionObjects(ctx, missionID, input.ObjectKind, input.Limit, input.Cursor)
	}
	if err != nil {
		return errorFromErr(call.Name, missionID, err, nil)
	}
	return wire.ToolResult{ToolName: call.Name, MissionID: missionID, Content: page}
}

func (handler *Handler) CallRead(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input researchReadInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	if err := handler.validateRead(missionID, input.Legacy, false); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	read, err := handler.reader.ReadMissionObject(ctx, app.ResearchIDEReadRequest{
		MissionID:  missionID,
		ObjectKind: input.ObjectKind,
		ObjectID:   input.ObjectID,
		Offset:     input.Offset,
		MaxBytes:   input.MaxBytes,
		Limit:      input.Limit,
		Cursor:     input.Cursor,
		Legacy:     input.Legacy,
	})
	if err != nil {
		return errorFromErr(call.Name, missionID, err, []string{input.ObjectID})
	}
	return wire.ToolResult{ToolName: call.Name, MissionID: missionID, Content: read}
}

func (handler *Handler) CallGrep(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input researchGrepInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	if err := handler.validateRead(missionID, input.Legacy, input.Legacy); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	var result app.ResearchIDEGrepResult
	var err error
	if input.Legacy {
		result, err = handler.legacyReader.GrepMissionObjectsLegacy(ctx, missionID, input.Query, input.Limit, input.Cursor)
	} else {
		result, err = handler.reader.GrepMissionObjects(ctx, missionID, input.Query, input.Limit, input.Cursor)
	}
	if err != nil {
		return errorFromErr(call.Name, missionID, err, nil)
	}
	return wire.ToolResult{ToolName: call.Name, MissionID: missionID, Content: result}
}

func (handler *Handler) CallReferences(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input researchReferencesInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return errorResult(call.Name, input.MissionID, "validation", err.Error(), false, nil)
	}
	missionID := strings.TrimSpace(input.MissionID)
	if err := handler.validateRead(missionID, input.Legacy, input.Legacy); err != nil {
		return errorResult(call.Name, missionID, "validation", err.Error(), false, nil)
	}
	var refs app.ResearchIDEReferences
	var err error
	if input.Legacy {
		refs, err = handler.legacyReader.ListObjectReferencesLegacy(ctx, missionID, input.ObjectKind, input.ObjectID, input.Limit, input.Cursor)
	} else {
		refs, err = handler.reader.ListObjectReferences(ctx, missionID, input.ObjectKind, input.ObjectID, input.Limit, input.Cursor)
	}
	if err != nil {
		return errorFromErr(call.Name, missionID, err, []string{input.ObjectID})
	}
	return wire.ToolResult{ToolName: call.Name, MissionID: missionID, Content: refs}
}

func (handler *Handler) validateRead(missionID string, legacy bool, needsLegacyReader bool) error {
	if err := validateID("mis_", missionID); err != nil {
		return err
	}
	if err := handler.enforceBoundMission(missionID); err != nil {
		return err
	}
	if err := handler.enforceLegacyResearchRead(legacy); err != nil {
		return err
	}
	if needsLegacyReader {
		if handler.legacyReader == nil {
			return errors.New("legacy research reader is not available")
		}
		return nil
	}
	if handler.reader == nil {
		return errors.New("research reader is not available")
	}
	return nil
}
