package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	ConfluenceUpdateStatusCurrent   = "current"
	ConfluenceUpdateStatusAvailable = "update_available"
	ConfluenceUpdateStatusFailed    = "check_failed"
)

type ConfluenceUpdateState struct {
	Status          string     `json:"status"`
	CheckedAt       time.Time  `json:"checked_at"`
	CurrentVersion  int        `json:"current_version,omitempty"`
	LatestVersion   int        `json:"latest_version,omitempty"`
	LatestUpdatedAt *time.Time `json:"latest_updated_at,omitempty"`
	ErrorCategory   string     `json:"error_category,omitempty"`
	ErrorCode       string     `json:"error_code,omitempty"`
	EventID         string     `json:"event_id,omitempty"`
}

type confluenceUpdateStateEventPayload struct {
	OldSnapshotID string `json:"old_snapshot_id"`
	OldVersion    int    `json:"old_version,omitempty"`
	NewVersion    int    `json:"new_version,omitempty"`
	NewUpdatedAt  string `json:"new_updated_at,omitempty"`
	CheckedAt     string `json:"checked_at,omitempty"`
	ErrorCategory string `json:"error_category,omitempty"`
	ErrorCode     string `json:"error_code,omitempty"`
}

func (s *Service) recordConfluenceUpdateCheckFailure(ctx context.Context, req CheckConfluenceSourceUpdateRequest, checkErr error) error {
	details, ok := ConfluenceErrorDetails(checkErr)
	if !ok || !durableConfluenceUpdateError(details.Category, details.Code) {
		return nil
	}
	checkedAt := time.Now().UTC()
	event, err := buildLedgerEvent(AppendEventRequest{
		EventID:   strings.TrimSpace(req.EventID),
		MissionID: strings.TrimSpace(req.MissionID),
		EventType: ConfluenceUpdateFailedEvent,
		Producer:  normalizeConfluenceUpdateProducer(req.Producer),
		Payload: mustMarshalJSON(confluenceUpdateStateEventPayload{
			OldSnapshotID: strings.TrimSpace(req.SnapshotID),
			CheckedAt:     checkedAt.Format(time.RFC3339Nano),
			ErrorCategory: strings.TrimSpace(details.Category),
			ErrorCode:     strings.TrimSpace(details.Code),
		}),
	})
	if err != nil {
		return err
	}
	_, err = s.commitAtomicWrite(ctx, AtomicWrite{Events: []LedgerEvent{event}})
	return err
}

func durableConfluenceUpdateError(category string, code string) bool {
	code = strings.TrimSpace(code)
	switch strings.TrimSpace(category) {
	case ConfluenceErrorCategoryAuth:
		return code == ConfluenceErrorCodeUnauthorized ||
			code == ConfluenceErrorCodeTokenExpired ||
			code == ConfluenceErrorCodeRevoked
	case ConfluenceErrorCategoryPermission:
		return code == ConfluenceErrorCodeForbidden
	case ConfluenceErrorCategoryNotFound:
		return code == ConfluenceErrorCodeNotFound
	case ConfluenceErrorCategoryRateLimited:
		return code == ConfluenceErrorCodeRateLimited
	case ConfluenceErrorCategoryUpstream:
		return code == ConfluenceErrorCodeUpstream
	default:
		return false
	}
}

func applyConfluenceUpdateState(states map[string]SourceState, event LedgerEvent) {
	payload, ok := decodeConfluenceUpdateStatePayload(event)
	if !ok {
		return
	}
	status := ConfluenceUpdateStatusCurrent
	if event.EventType == ConfluenceUpdateAvailableEvent {
		status = ConfluenceUpdateStatusAvailable
	} else if event.EventType == ConfluenceUpdateFailedEvent {
		status = ConfluenceUpdateStatusFailed
	}
	state := states[payload.OldSnapshotID]
	if state.State == "" {
		state.State = SourceStateActive
	}
	state.ConfluenceUpdate = &ConfluenceUpdateState{
		Status:          status,
		CheckedAt:       parseConfluenceUpdateTime(payload.CheckedAt, eventTime(event)),
		CurrentVersion:  payload.OldVersion,
		LatestVersion:   payload.NewVersion,
		LatestUpdatedAt: parseOptionalConfluenceUpdateTime(payload.NewUpdatedAt),
		ErrorCategory:   strings.TrimSpace(payload.ErrorCategory),
		ErrorCode:       strings.TrimSpace(payload.ErrorCode),
		EventID:         event.EventID,
	}
	states[payload.OldSnapshotID] = state
}

func decodeConfluenceUpdateStatePayload(event LedgerEvent) (confluenceUpdateStateEventPayload, bool) {
	var payload confluenceUpdateStateEventPayload
	if json.Unmarshal(event.Payload, &payload) != nil {
		return confluenceUpdateStateEventPayload{}, false
	}
	payload.OldSnapshotID = strings.TrimSpace(payload.OldSnapshotID)
	return payload, payload.OldSnapshotID != ""
}

func validateConfluenceUpdateStateEventPayload(payload json.RawMessage) error {
	var typed confluenceUpdateStateEventPayload
	if json.Unmarshal(payload, &typed) != nil {
		return fmt.Errorf("%w: invalid Confluence update check failure payload", ErrInvalidInput)
	}
	if err := validateID("src_", strings.TrimSpace(typed.OldSnapshotID)); err != nil {
		return err
	}
	if !durableConfluenceUpdateError(typed.ErrorCategory, typed.ErrorCode) {
		return fmt.Errorf("%w: invalid Confluence update check failure classification", ErrInvalidInput)
	}
	return nil
}

func parseConfluenceUpdateTime(value string, fallback time.Time) time.Time {
	parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}

func parseOptionalConfluenceUpdateTime(value string) *time.Time {
	parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
	if err != nil {
		return nil
	}
	return &parsed
}
