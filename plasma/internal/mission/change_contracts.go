package mission

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

// UpdateMissionMetadataRequest contains the optional mission metadata fields to change.
type UpdateMissionMetadataRequest struct {
	EventID   string
	MissionID string
	Producer  ledger.Producer
	Title     *string
	Objective *string
	Scope     *Scope
}

// UpdateMissionMetadataResult contains the metadata event and rebuilt projection.
type UpdateMissionMetadataResult struct {
	Event      ledger.Event `json:"event"`
	Projection Projection   `json:"projection"`
}

// MissionLifecycleChangeRequest contains the identity and reason for an archive or restore.
type MissionLifecycleChangeRequest struct {
	EventID   string
	MissionID string
	Producer  ledger.Producer
	Reason    string
}

// MissionLifecycleChangeResult contains the lifecycle event and rebuilt projection.
type MissionLifecycleChangeResult struct {
	Event      *ledger.Event `json:"event,omitempty"`
	Projection Projection    `json:"projection"`
	Idempotent bool          `json:"idempotent,omitempty"`
}

// BuildMetadataUpdate validates metadata policy and builds its ledger append request.
func BuildMetadataUpdate(req UpdateMissionMetadataRequest) (ledger.AppendRequest, error) {
	if req.Title == nil && req.Objective == nil && req.Scope == nil {
		return ledger.AppendRequest{}, fmt.Errorf("%w: at least one metadata field is required", producterror.ErrInvalidInput)
	}
	if req.Producer.Type != "user" {
		return ledger.AppendRequest{}, fmt.Errorf("%w: metadata updates require a user producer", producterror.ErrInvalidInput)
	}
	payload := make(map[string]any, 3)
	if req.Title != nil {
		value := strings.TrimSpace(*req.Title)
		if value == "" {
			return ledger.AppendRequest{}, fmt.Errorf("%w: title must not be blank", producterror.ErrInvalidInput)
		}
		payload["title"] = value
	}
	if req.Objective != nil {
		payload["objective"] = strings.TrimSpace(*req.Objective)
	}
	if req.Scope != nil {
		payload["scope"] = normalizeScope(*req.Scope)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return ledger.AppendRequest{}, err
	}
	return ledger.AppendRequest{
		EventID:   req.EventID,
		MissionID: req.MissionID,
		EventType: "mission.metadata.updated",
		Producer:  req.Producer,
		Payload:   body,
	}, nil
}

// BuildLifecycleChange validates lifecycle policy and builds its ledger append request.
func BuildLifecycleChange(req MissionLifecycleChangeRequest, targetState, eventType string) (ledger.AppendRequest, error) {
	if err := validateLifecycleID("evt_", req.EventID); err != nil {
		return ledger.AppendRequest{}, err
	}
	if err := validateLifecycleID("mis_", req.MissionID); err != nil {
		return ledger.AppendRequest{}, err
	}
	if req.Producer.Type != "user" {
		return ledger.AppendRequest{}, fmt.Errorf("%w: mission lifecycle updates require a user producer", producterror.ErrInvalidInput)
	}
	payload, err := json.Marshal(map[string]any{
		"lifecycle_state": targetState,
		"reason":          strings.TrimSpace(req.Reason),
	})
	if err != nil {
		return ledger.AppendRequest{}, err
	}
	return ledger.AppendRequest{
		EventID:   req.EventID,
		MissionID: req.MissionID,
		EventType: eventType,
		Producer:  req.Producer,
		Payload:   payload,
	}, nil
}

// LifecycleChangeNeeded reports whether the target lifecycle state differs from the projection.
func LifecycleChangeNeeded(missionID string, events []ledger.Event, targetState string) (bool, error) {
	if len(events) == 0 {
		return false, fmt.Errorf("%w: mission does not exist", producterror.ErrInvalidInput)
	}
	projection, err := BuildProjection(missionID, events)
	if err != nil {
		return false, err
	}
	return NormalizeLifecycleState(projection.LifecycleState) != targetState, nil
}

// NormalizeLifecycleState preserves the mission lifecycle rule: only archived is archived;
// every other state is treated as active.
func NormalizeLifecycleState(value string) string {
	if strings.TrimSpace(value) == LifecycleArchived {
		return LifecycleArchived
	}
	return LifecycleActive
}

func validateLifecycleID(prefix, id string) error {
	trimmed := strings.TrimSpace(id)
	if !strings.HasPrefix(trimmed, prefix) || len(trimmed) <= len(prefix) {
		return fmt.Errorf("%w: id must start with %s", producterror.ErrInvalidInput, prefix)
	}
	return nil
}
