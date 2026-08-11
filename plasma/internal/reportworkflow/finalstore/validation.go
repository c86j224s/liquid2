package finalstore

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func validateBase(input BaseInput) error {
	if strings.TrimSpace(input.MissionID) == "" || strings.TrimSpace(input.PendingEventID) == "" {
		return fmt.Errorf("%w: finalstore mission and pending event are required", producterror.ErrInvalidInput)
	}
	return nil
}

func validateOneTakeCandidate(candidate OneTakeCandidate) error {
	if strings.TrimSpace(candidate.ArtifactID) == "" || strings.TrimSpace(candidate.ToolSessionID) == "" ||
		strings.TrimSpace(candidate.ReportSessionID) == "" {
		return fmt.Errorf("%w: one_take finalstore identity is incomplete", producterror.ErrInvalidInput)
	}
	if strings.TrimSpace(candidate.Markdown) == "" {
		return fmt.Errorf("%w: one_take finalstore Markdown is required", producterror.ErrInvalidInput)
	}
	if candidate.StartedAt.IsZero() {
		return fmt.Errorf("%w: one_take finalstore start time is required", producterror.ErrInvalidInput)
	}
	return nil
}

func validatePlannedCandidate(candidate PlannedCandidate) error {
	if strings.TrimSpace(candidate.ArtifactID) == "" || strings.TrimSpace(candidate.ToolSessionID) == "" ||
		strings.TrimSpace(candidate.PlanEventID) == "" || strings.TrimSpace(candidate.PlanToolSessionID) == "" ||
		strings.TrimSpace(candidate.ReportPlanSessionID) == "" || strings.TrimSpace(candidate.ReportSessionID) == "" {
		return fmt.Errorf("%w: planned finalstore identity is incomplete", producterror.ErrInvalidInput)
	}
	if strings.TrimSpace(candidate.Markdown) == "" {
		return fmt.Errorf("%w: planned finalstore Markdown is required", producterror.ErrInvalidInput)
	}
	if candidate.WorkflowStartedAt.IsZero() {
		return fmt.Errorf("%w: planned finalstore start time is required", producterror.ErrInvalidInput)
	}
	return nil
}

func validateStoredArtifact(raw artifact.Raw, missionID string, artifactID string, markdown string) error {
	if strings.TrimSpace(raw.ArtifactID) != strings.TrimSpace(artifactID) ||
		strings.TrimSpace(raw.MissionID) != strings.TrimSpace(missionID) {
		return fmt.Errorf("%w: finalstore artifact envelope differs", producterror.ErrConflict)
	}
	if string(raw.Content) != markdown {
		return fmt.Errorf("%w: finalstore artifact content differs", producterror.ErrConflict)
	}
	if raw.SHA256 != "" && raw.SHA256 != sha256Hex(raw.Content) {
		return fmt.Errorf("%w: finalstore artifact sha differs", producterror.ErrConflict)
	}
	return nil
}

func validateStoredEvent(event ledger.Event, missionID string, pendingEventID string, artifactID string, planEventID string) error {
	if strings.TrimSpace(event.EventID) == "" || event.EventType != "report.artifact.created" ||
		strings.TrimSpace(event.MissionID) != strings.TrimSpace(missionID) {
		return fmt.Errorf("%w: finalstore terminal event envelope differs", producterror.ErrConflict)
	}
	payload, err := decodePayload(event.Payload)
	if err != nil {
		return err
	}
	if strings.TrimSpace(payload.ArtifactID) != strings.TrimSpace(artifactID) ||
		strings.TrimSpace(payload.PendingEventID) != strings.TrimSpace(pendingEventID) {
		return fmt.Errorf("%w: finalstore event payload differs from artifact envelope", producterror.ErrConflict)
	}
	if strings.TrimSpace(planEventID) != "" && strings.TrimSpace(payload.PlanEventID) != strings.TrimSpace(planEventID) {
		return fmt.Errorf("%w: finalstore event plan lineage differs", producterror.ErrConflict)
	}
	return nil
}

type artifactCreatedPayload struct {
	ArtifactID     string `json:"artifact_id"`
	PendingEventID string `json:"pending_event_id"`
	PlanEventID    string `json:"plan_event_id"`
}

func decodePayload(payload json.RawMessage) (artifactCreatedPayload, error) {
	if len(payload) == 0 {
		return artifactCreatedPayload{}, fmt.Errorf("%w: finalstore event payload is required", producterror.ErrInvalidInput)
	}
	var decoded artifactCreatedPayload
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return artifactCreatedPayload{}, fmt.Errorf("%w: finalstore event payload is invalid", producterror.ErrInvalidInput)
	}
	return decoded, nil
}
