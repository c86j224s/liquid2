package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

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

func errorFromErr(toolName, missionID string, err error, related []string) ToolResult {
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

func errorResult(toolName, missionID, kind, message string, retryable bool, related []string) ToolResult {
	return ToolResult{
		ToolName:  toolName,
		MissionID: strings.TrimSpace(missionID),
		Error: &ToolError{
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

func mustJSON(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}
