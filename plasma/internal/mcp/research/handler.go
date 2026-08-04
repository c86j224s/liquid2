package research

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
)

// Handler owns research-specific MCP request decoding, validation, and app port
// calls. It does not decide whether tools are exposed or idempotently cached.
type Handler struct {
	reader             Reader
	legacyReader       LegacyReader
	proposalWriter     ProposalWriter
	boundMissionID     string
	legacyResearchLoop bool
}

// NewHandler binds narrow research ports from the root service without changing
// the root NewServer signature.
func NewHandler(service any, boundMissionID string, legacyResearchLoop bool) *Handler {
	handler := &Handler{boundMissionID: strings.TrimSpace(boundMissionID), legacyResearchLoop: legacyResearchLoop}
	if reader, ok := service.(Reader); ok {
		handler.reader = reader
	}
	if legacyReader, ok := service.(LegacyReader); ok {
		handler.legacyReader = legacyReader
	}
	if writer, ok := service.(ProposalWriter); ok {
		handler.proposalWriter = writer
	}
	return handler
}

func decodeArgs(args json.RawMessage, target any) error {
	if len(args) == 0 {
		args = json.RawMessage(`{}`)
	}
	if !json.Valid(args) {
		return fmt.Errorf("%w: tool arguments must be valid JSON", app.ErrInvalidInput)
	}
	if err := json.Unmarshal(args, target); err != nil {
		return fmt.Errorf("%w: decode tool arguments: %v", app.ErrInvalidInput, err)
	}
	return nil
}

func validateID(prefix, id string) error {
	trimmed := strings.TrimSpace(id)
	if !strings.HasPrefix(trimmed, prefix) || len(trimmed) <= len(prefix) {
		return fmt.Errorf("%w: id must start with %s", app.ErrInvalidInput, prefix)
	}
	return nil
}

func (handler *Handler) enforceBoundMission(missionID string) error {
	if handler.boundMissionID == "" {
		return nil
	}
	if missionID != handler.boundMissionID {
		return fmt.Errorf("%w: tool call mission_id is outside this MCP session", app.ErrInvalidInput)
	}
	return nil
}

func (handler *Handler) enforceLegacyResearchRead(legacy bool) error {
	if legacy && !handler.legacyResearchLoop {
		return fmt.Errorf("%w: legacy research reads require legacy research loop mode", app.ErrInvalidInput)
	}
	return nil
}

func errorFromErr(toolName, missionID string, err error, related []string) wire.ToolResult {
	if confluenceErr, ok := app.ConfluenceErrorDetails(err); ok {
		return errorResult(toolName, missionID, confluenceErr.Category, confluenceErr.Error(), confluenceErr.HTTPStatus == 429 || confluenceErr.HTTPStatus >= 500, related)
	}
	kind := "internal"
	retryable := false
	if errors.Is(err, app.ErrInvalidInput) {
		kind = "validation"
	} else if errors.Is(err, app.ErrConflict) {
		kind = "conflict"
	}
	return errorResult(toolName, missionID, kind, err.Error(), retryable, related)
}

func errorResult(toolName, missionID, kind, message string, retryable bool, related []string) wire.ToolResult {
	return wire.ToolResult{
		ToolName:  toolName,
		MissionID: strings.TrimSpace(missionID),
		Error: &wire.ToolError{
			ErrorKind:        kind,
			Message:          message,
			Retryable:        retryable,
			RelatedObjectIDs: normalizeRelatedIDs(related),
		},
	}
}

func normalizeRelatedIDs(ids []string) []string {
	normalized := []string{}
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	return normalized
}
