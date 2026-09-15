package reportrun

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportusage"
)

func findCompletionState(events []ledger.Event, canonicalID string) (completionState, error) {
	registration, err := BuildRegistration(reportRunEvents(events), RegistrationBackfilled, time.Now().UTC())
	if err != nil {
		return completionState{}, err
	}
	canonical, ok := eventByID(events, canonicalID)
	if !ok || canonical.EventType != "report.artifact.created" {
		return completionState{}, fmt.Errorf("canonical completion artifact is missing")
	}
	var canonicalPayload struct {
		PendingEventID string `json:"pending_event_id"`
		ArtifactID     string `json:"artifact_id"`
	}
	if err := json.Unmarshal(canonical.Payload, &canonicalPayload); err != nil {
		return completionState{}, err
	}
	if strings.TrimSpace(canonicalPayload.PendingEventID) == "" || strings.TrimSpace(canonicalPayload.ArtifactID) == "" {
		return completionState{}, fmt.Errorf("canonical completion artifact payload is incomplete")
	}
	for _, run := range registration.Runs {
		if run.LifecycleState == LifecycleAmbiguous {
			continue
		}
		var canonicalMember, pendingMember *EventMembership
		for index := range registration.Events {
			member := &registration.Events[index]
			if member.RunID != run.RunID {
				continue
			}
			if member.EventID == canonical.EventID && member.EventRole == "final" {
				canonicalMember = member
			}
			if member.EventID == strings.TrimSpace(canonicalPayload.PendingEventID) && member.EventRole == "draft_pending" {
				pendingMember = member
			}
		}
		if canonicalMember == nil || pendingMember == nil || canonicalMember.AttemptEventID != pendingMember.EventID || pendingMember.AttemptEventID != pendingMember.EventID {
			continue
		}
		pending, ok := eventByID(events, pendingMember.EventID)
		if !ok || pending.EventType != "report.draft.pending" {
			continue
		}
		state := completionState{Root: run.RunID, Canonical: canonical, Pending: pending}
		for index := range registration.Events {
			member := registration.Events[index]
			if member.RunID != run.RunID {
				continue
			}
			targetEvent, targetOK := eventByID(events, member.EventID)
			if !targetOK {
				continue
			}
			meta, isTarget, targetErr := reportusage.TargetForEvent(targetEvent)
			if targetErr != nil {
				return completionState{}, targetErr
			}
			if !isTarget {
				continue
			}
			var targetPayload struct {
				PendingEventID string `json:"pending_event_id"`
			}
			if err := json.Unmarshal(targetEvent.Payload, &targetPayload); err != nil {
				return completionState{}, err
			}
			targetPending, found := eventByID(events, strings.TrimSpace(targetPayload.PendingEventID))
			if !found || targetPending.EventType != "report.draft.pending" || member.AttemptEventID != targetPending.EventID || !memberInRun(registration.Events, run.RunID, targetPending.EventID, "draft_pending") {
				return completionState{}, fmt.Errorf("target %s has invalid report pending lineage", targetEvent.EventID)
			}
			state.Targets = append(state.Targets, completionTarget{Event: targetEvent, Pending: targetPending, Meta: meta})
		}
		sort.SliceStable(state.Targets, func(i, j int) bool {
			left, right := state.Targets[i].Event, state.Targets[j].Event
			if left.Sequence != right.Sequence {
				return left.Sequence < right.Sequence
			}
			if !left.CreatedAt.Equal(right.CreatedAt) {
				return left.CreatedAt.Before(right.CreatedAt)
			}
			return left.EventID < right.EventID
		})
		return state, nil
	}
	return completionState{}, fmt.Errorf("canonical artifact is not an authoritative report run final")
}

func memberInRun(members []EventMembership, runID, eventID, role string) bool {
	for _, member := range members {
		if member.RunID == runID && member.EventID == eventID && member.EventRole == role {
			return true
		}
	}
	return false
}

func reportRunEvents(events []ledger.Event) []Event {
	out := make([]Event, 0, len(events))
	for _, event := range events {
		out = append(out, Event{EventID: event.EventID, MissionID: event.MissionID, Sequence: event.Sequence, EventType: event.EventType, Producer: event.Producer, CausationEventID: event.CausationEventID, CorrelationID: event.CorrelationID, Payload: event.Payload, CreatedAt: event.CreatedAt})
	}
	return out
}

func canonicalArtifactID(event ledger.Event) string {
	var payload struct {
		ArtifactID string `json:"artifact_id"`
	}
	_ = json.Unmarshal(event.Payload, &payload)
	return strings.TrimSpace(payload.ArtifactID)
}

func completionPayload(state completionState, usageRecorded, usageUnavailable int) []byte {
	var payload struct {
		ArtifactID string `json:"artifact_id"`
	}
	_ = json.Unmarshal(state.Canonical.Payload, &payload)
	encoded, _ := json.Marshal(map[string]any{"kind": "report_run_completed", "schema_version": ReportRunCompletionSchema, "run_id": state.Root, "pending_event_id": state.Pending.EventID, "canonical_event_id": state.Canonical.EventID, "artifact_id": strings.TrimSpace(payload.ArtifactID), "delayed_usage_target_count": len(state.Targets), "usage_recorded_count": usageRecorded, "usage_unavailable_count": usageUnavailable})
	return encoded
}

func validateExistingCompletion(event ledger.Event, root string, events []ledger.Event) error {
	var payload struct {
		RunID            string `json:"run_id"`
		CanonicalEventID string `json:"canonical_event_id"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil || strings.TrimSpace(payload.RunID) != root || strings.TrimSpace(payload.CanonicalEventID) == "" {
		return fmt.Errorf("invalid existing report completion")
	}
	existingState, err := findCompletionState(events, strings.TrimSpace(payload.CanonicalEventID))
	if err != nil || existingState.Root != root {
		return fmt.Errorf("existing report completion belongs to a different canonical run")
	}
	return validateCompletion(event, existingState, events)
}

func validateCompletion(event ledger.Event, state completionState, events []ledger.Event) error {
	if event.EventID != completionEventID(state.Root) || event.EventType != ReportRunCompletedEventType || event.Producer != (ledger.Producer{Type: "system", ID: "report-completion"}) || event.CausationEventID != state.Canonical.EventID || event.CorrelationID != state.Root {
		return fmt.Errorf("invalid report completion envelope")
	}
	var payload struct {
		Kind             string `json:"kind"`
		SchemaVersion    string `json:"schema_version"`
		RunID            string `json:"run_id"`
		PendingEventID   string `json:"pending_event_id"`
		CanonicalEventID string `json:"canonical_event_id"`
		ArtifactID       string `json:"artifact_id"`
		TargetCount      int    `json:"delayed_usage_target_count"`
		Recorded         int    `json:"usage_recorded_count"`
		Unavailable      int    `json:"usage_unavailable_count"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}
	if payload.Kind != "report_run_completed" || payload.SchemaVersion != ReportRunCompletionSchema || payload.RunID != state.Root || payload.PendingEventID != state.Pending.EventID || payload.CanonicalEventID != state.Canonical.EventID || payload.ArtifactID != canonicalArtifactID(state.Canonical) || payload.TargetCount < 0 || payload.Recorded < 0 || payload.Unavailable < 0 || payload.TargetCount != len(state.Targets) || payload.Recorded+payload.Unavailable != payload.TargetCount {
		return fmt.Errorf("invalid report completion payload")
	}
	usage, err := validateUsageEvents(state, events)
	if err != nil {
		return err
	}
	recorded := 0
	unavailable := 0
	for _, target := range state.Targets {
		value, ok := usage[target.Event.EventID]
		if !ok {
			return fmt.Errorf("report completion is missing usage outcome")
		}
		if value.ProviderUsage != nil && !value.UsageUnavailable {
			recorded++
		} else {
			unavailable++
		}
	}
	if payload.Recorded != recorded || payload.Unavailable != unavailable {
		return fmt.Errorf("report completion counts differ")
	}
	return nil
}

func completionEventID(root string) string {
	return "evt_report_run_completed_" + strings.TrimPrefix(strings.TrimSpace(root), "evt_")
}
func eventByID(events []ledger.Event, id string) (ledger.Event, bool) {
	for _, event := range events {
		if event.EventID == id {
			return event, true
		}
	}
	return ledger.Event{}, false
}
