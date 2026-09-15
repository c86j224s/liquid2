package mission

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	canonicalmission "github.com/c86j224s/liquid2/plasma/internal/mission"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/researchrecords"
	"github.com/c86j224s/liquid2/plasma/internal/source"
)

// ErrorResult adapts a validation or policy error into the root MCP envelope.
type ErrorResult func(string, string, string, string, bool, []string) wire.ToolResult

// ErrorFromErr delegates application error classification to the root policy.
type ErrorFromErr func(string, string, error, []string) wire.ToolResult

// Handler owns mission tool decoding, validation, and canonical port calls.
type Handler struct {
	reader       Reader
	updater      MetadataUpdater
	boundMission func(string) error
	newID        func(string) string
	mapSource    func(source.Snapshot) any
	errorResult  ErrorResult
	errorFrom    ErrorFromErr
}

// NewHandler binds only the narrow mission ports and explicit root policy callbacks.
func NewHandler(reader Reader, updater MetadataUpdater, boundMission func(string) error, newID func(string) string, mapSource func(source.Snapshot) any, errorResult ErrorResult, errorFrom ErrorFromErr) *Handler {
	return &Handler{reader: reader, updater: updater, boundMission: boundMission, newID: newID, mapSource: mapSource, errorResult: errorResult, errorFrom: errorFrom}
}

// CallGet handles plasma.mission.get while preserving the established call order.
func (handler *Handler) CallGet(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input getInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return handler.validation(call.Name, input.MissionID, err)
	}
	missionID := strings.TrimSpace(input.MissionID)
	if err := validateID(missionID); err != nil {
		return handler.validation(call.Name, missionID, err)
	}
	if err := handler.boundMission(missionID); err != nil {
		return handler.validation(call.Name, missionID, err)
	}
	projection, err := handler.reader.GetProjection(ctx, missionID)
	if err != nil {
		return handler.errorFrom(call.Name, missionID, err, nil)
	}
	var sources []any
	if includeRequested(input.Include, "sources") {
		snapshots, err := handler.reader.ListSourceSnapshotsWithState(ctx, source.ListRequest{MissionID: missionID})
		if err != nil {
			return handler.errorFrom(call.Name, missionID, err, nil)
		}
		sources = make([]any, 0, len(snapshots))
		for _, snapshot := range snapshots {
			sources = append(sources, handler.mapSource(snapshot))
		}
	}
	var evidence []researchrecords.EvidenceRecord
	if includeRequested(input.Include, "evidence") || includeRequested(input.Include, "records") {
		evidence, err = handler.reader.ListEvidenceRecords(ctx, missionID)
		if err != nil {
			return handler.errorFrom(call.Name, missionID, err, nil)
		}
	}
	var claims []researchrecords.ClaimRecord
	if includeRequested(input.Include, "claims") || includeRequested(input.Include, "records") {
		claims, err = handler.reader.ListClaimRecords(ctx, missionID)
		if err != nil {
			return handler.errorFrom(call.Name, missionID, err, nil)
		}
	}
	var questions []researchrecords.QuestionRecord
	if includeRequested(input.Include, "questions") || includeRequested(input.Include, "records") {
		questions, err = handler.reader.ListQuestionRecords(ctx, missionID)
		if err != nil {
			return handler.errorFrom(call.Name, missionID, err, nil)
		}
	}
	return wire.ToolResult{ToolName: call.Name, MissionID: missionID, Content: GetOutput{
		MissionProjection: projection, Sources: sources, Evidence: evidence, Claims: claims, OpenQuestions: questions, ActiveReportVersion: nil,
	}}
}

// CallUpdate handles plasma.mission.update after root idempotency and session guards.
func (handler *Handler) CallUpdate(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	var input UpdateInput
	if err := decodeArgs(call.Arguments, &input); err != nil {
		return handler.validation(call.Name, input.MissionID, err)
	}
	missionID := strings.TrimSpace(input.MissionID)
	if input.Title == nil && input.Objective == nil && input.Scope == nil {
		return handler.validation(call.Name, missionID, errors.New("at least one metadata field is required"))
	}
	if handler.updater == nil {
		return handler.errorResult(call.Name, missionID, "internal", "mission metadata service is unavailable", false, nil)
	}
	result, err := handler.updater.UpdateMissionMetadata(ctx, canonicalmission.UpdateMissionMetadataRequest{
		EventID: handler.newID("evt"), MissionID: missionID,
		Producer: ledger.Producer{Type: strings.TrimSpace(input.Producer.Type), ID: strings.TrimSpace(input.Producer.ID)},
		Title:    input.Title, Objective: input.Objective, Scope: input.Scope,
	})
	if err != nil {
		return handler.errorFrom(call.Name, missionID, err, nil)
	}
	return wire.ToolResult{ToolName: call.Name, MissionID: missionID, CreatedEventIDs: []string{result.Event.EventID}, Content: result}
}

func (handler *Handler) validation(toolName, missionID string, err error) wire.ToolResult {
	return handler.errorResult(toolName, missionID, "validation", err.Error(), false, nil)
}

func validateID(id string) error {
	if !strings.HasPrefix(id, "mis_") || len(id) <= len("mis_") {
		return fmt.Errorf("%w: id must start with mis_", producterror.ErrInvalidInput)
	}
	return nil
}

func includeRequested(includes []string, target string) bool {
	for _, include := range includes {
		normalized := strings.TrimSpace(include)
		if normalized == target || normalized == "*" || normalized == "all" {
			return true
		}
	}
	return false
}
