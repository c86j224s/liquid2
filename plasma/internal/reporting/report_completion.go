package reporting

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mission"
	"github.com/c86j224s/liquid2/plasma/internal/reportrun"
	"github.com/c86j224s/liquid2/plasma/internal/reportusage"
)

const (
	ReportRunCompletedEventType = "report.run.completed"
	ReportRunCompletionSchema   = "plasma.report_run_completion.v1"
	ReportRunCompletionReason   = "provider usage was lost before durable report completion"
)

type ReportCompletionStore interface {
	AppendEventsConditionally(context.Context, string, func([]ledger.Event) ([]ledger.AppendRequest, error)) ([]ledger.Event, error)
	ListEvents(context.Context, string) ([]ledger.Event, error)
}

type ReportCompletionMissionStore interface {
	ReportCompletionStore
	ListMissionsWithState(context.Context, mission.ListRequest) ([]mission.Mission, error)
}

type ReportCompletionRequest struct {
	MissionID        string
	CanonicalEventID string
	ActualUsage      *ReportAgentUsageRequest
}

type completionTarget struct {
	Event   ledger.Event
	Pending ledger.Event
	Meta    reportusage.Target
}

type completionState struct {
	Root      string
	Canonical ledger.Event
	Pending   ledger.Event
	Targets   []completionTarget
}

// CompleteReportRun conditionally records all target outcomes and the completion
// boundary from one latest ledger snapshot.
func CompleteReportRun(ctx context.Context, store ReportCompletionStore, req ReportCompletionRequest) (ledger.Event, error) {
	event, _, err := completeReportRun(ctx, store, req)
	return event, err
}

func completeReportRun(ctx context.Context, store ReportCompletionStore, req ReportCompletionRequest) (ledger.Event, bool, error) {
	missionID := strings.TrimSpace(req.MissionID)
	canonicalID := strings.TrimSpace(req.CanonicalEventID)
	if missionID == "" || canonicalID == "" {
		return ledger.Event{}, false, fmt.Errorf("report completion requires mission and canonical event")
	}
	var replay ledger.Event
	var created bool
	appended, err := store.AppendEventsConditionally(ctx, missionID, func(events []ledger.Event) ([]ledger.AppendRequest, error) {
		state, err := findCompletionState(events, canonicalID)
		if err != nil {
			return nil, err
		}
		completionID := completionEventID(state.Root)
		if existing, ok := eventByID(events, completionID); ok {
			var payload struct {
				RunID            string `json:"run_id"`
				CanonicalEventID string `json:"canonical_event_id"`
			}
			if err := json.Unmarshal(existing.Payload, &payload); err != nil || strings.TrimSpace(payload.RunID) != state.Root || strings.TrimSpace(payload.CanonicalEventID) == "" {
				return nil, fmt.Errorf("invalid existing report completion")
			}
			existingState, err := findCompletionState(events, strings.TrimSpace(payload.CanonicalEventID))
			if err != nil || existingState.Root != state.Root {
				return nil, fmt.Errorf("existing report completion belongs to a different canonical run")
			}
			if err := validateCompletion(existing, existingState, events); err != nil {
				return nil, err
			}
			replay = existing
			return nil, nil
		}
		reqs, err := completionRequests(state, events, req.ActualUsage)
		if err != nil {
			return nil, err
		}
		created = true
		return reqs, nil
	})
	if err != nil {
		return ledger.Event{}, false, err
	}
	if replay.EventID != "" {
		return replay, false, nil
	}
	var completion ledger.Event
	for _, event := range appended {
		if event.EventType != ReportRunCompletedEventType {
			continue
		}
		if completion.EventID != "" {
			return ledger.Event{}, false, fmt.Errorf("report completion batch contains multiple completion events")
		}
		completion = event
	}
	if completion.EventID == "" {
		return ledger.Event{}, false, fmt.Errorf("report completion was not appended")
	}
	return completion, created, nil
}

func completionRequests(state completionState, events []ledger.Event, actual *ReportAgentUsageRequest) ([]ledger.AppendRequest, error) {
	usageByTarget, err := validateUsageEvents(state, events)
	if err != nil {
		return nil, err
	}
	requests := make([]ledger.AppendRequest, 0, len(state.Targets)+1)
	actualTarget := ""
	if actual != nil {
		actualTarget = strings.TrimSpace(actual.CanonicalEventID)
		known := false
		for _, target := range state.Targets {
			if target.Event.EventID == actualTarget {
				known = true
				break
			}
		}
		if !known {
			return nil, fmt.Errorf("actual report usage does not match a completion target")
		}
	}
	actualRecorded := false
	actualUnavailable := false
	for _, target := range state.Targets {
		if _, ok := usageByTarget[target.Event.EventID]; ok {
			if actualTarget == target.Event.EventID && actual != nil {
				request, err := actualUsageAppendRequest(state, target, *actual)
				if err != nil {
					return nil, err
				}
				if existing, found := eventByID(events, request.EventID); found && (existing.EventType != request.EventType || existing.Producer != request.Producer || existing.CausationEventID != request.CausationEventID || existing.CorrelationID != request.CorrelationID || !bytes.Equal(existing.Payload, request.Payload)) {
					return nil, fmt.Errorf("actual report usage conflicts with existing record")
				}
			}
			continue
		}
		if actualTarget == target.Event.EventID {
			request, err := actualUsageAppendRequest(state, target, *actual)
			if err != nil {
				return nil, err
			}
			requests = append(requests, request)
			var usagePayload struct {
				AgentUsage agentusage.AgentUsage `json:"agent_usage"`
			}
			if err := json.Unmarshal(request.Payload, &usagePayload); err != nil {
				return nil, err
			}
			actualRecorded = usagePayload.AgentUsage.ProviderUsage != nil && !usagePayload.AgentUsage.UsageUnavailable
			actualUnavailable = !actualRecorded
			continue
		}
		request, err := unavailableUsageRequest(state, target)
		if err != nil {
			return nil, err
		}
		requests = append(requests, request)
	}
	usageRecorded := 0
	usageUnavailable := 0
	for _, target := range state.Targets {
		if usage, ok := usageByTarget[target.Event.EventID]; ok {
			if usage.ProviderUsage != nil && !usage.UsageUnavailable {
				usageRecorded++
			} else {
				usageUnavailable++
			}
			continue
		}
		if actualRecorded && actualTarget == target.Event.EventID {
			usageRecorded++
		} else if actualUnavailable && actualTarget == target.Event.EventID {
			usageUnavailable++
		} else {
			usageUnavailable++
		}
	}
	if usageRecorded+usageUnavailable != len(state.Targets) {
		return nil, fmt.Errorf("report completion has incomplete usage outcomes")
	}
	requests = append(requests, ledger.AppendRequest{
		EventID:          completionEventID(state.Root),
		MissionID:        state.Canonical.MissionID,
		EventType:        ReportRunCompletedEventType,
		Producer:         ledger.Producer{Type: "system", ID: "report-completion"},
		CausationEventID: state.Canonical.EventID,
		CorrelationID:    state.Root,
		Payload:          completionPayload(state, usageRecorded, usageUnavailable),
	})
	return requests, nil
}

func validateUsageEvents(state completionState, events []ledger.Event) (map[string]agentusage.AgentUsage, error) {
	targets := make(map[string]completionTarget, len(state.Targets))
	for _, target := range state.Targets {
		targets[target.Event.EventID] = target
	}
	usageByTarget := make(map[string]agentusage.AgentUsage, len(targets))
	usageEventByTarget := make(map[string]string, len(targets))
	for _, event := range events {
		if event.EventType != ReportAgentUsageRecordedEventType {
			continue
		}
		var payload struct {
			Kind                     string                `json:"kind"`
			PendingEventID           string                `json:"pending_event_id"`
			CorrelationEventID       string                `json:"correlation_event_id"`
			ForkSourceAgentSessionID string                `json:"fork_source_agent_session_id"`
			AgentUsage               agentusage.AgentUsage `json:"agent_usage"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return nil, fmt.Errorf("decode report usage %s: %w", event.EventID, err)
		}
		target, isTarget := targets[strings.TrimSpace(payload.CorrelationEventID)]
		if !isTarget {
			continue
		}
		if err := validateUsageEvent(event, payload.Kind, payload.PendingEventID, payload.CorrelationEventID, target); err != nil {
			return nil, err
		}
		if priorID, exists := usageEventByTarget[target.Event.EventID]; exists {
			if priorID != event.EventID {
				return nil, fmt.Errorf("conflicting report usage events for %s", target.Event.EventID)
			}
			continue
		}
		usageEventByTarget[target.Event.EventID] = event.EventID
		usageByTarget[target.Event.EventID] = payload.AgentUsage
	}
	return usageByTarget, nil
}

func validateUsageEvent(event ledger.Event, kind, pendingID, correlationID string, target completionTarget) error {
	if event.EventID != reportAgentUsageEventID(correlationID) || event.MissionID != target.Event.MissionID || event.EventType != ReportAgentUsageRecordedEventType || event.CausationEventID != target.Event.EventID || event.CorrelationID != pendingID || strings.TrimSpace(kind) != "report_agent_usage" || strings.TrimSpace(pendingID) != target.Pending.EventID || strings.TrimSpace(correlationID) != target.Event.EventID {
		return fmt.Errorf("invalid report usage envelope for %s", target.Event.EventID)
	}
	if target.Meta.AgentSessionID == "" || event.Producer != (ledger.Producer{Type: "agent_session", ID: target.Meta.AgentSessionID}) {
		return fmt.Errorf("invalid report usage producer for %s", target.Event.EventID)
	}
	var payload struct {
		ForkSourceAgentSessionID string                `json:"fork_source_agent_session_id"`
		AgentUsage               agentusage.AgentUsage `json:"agent_usage"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil || payload.AgentUsage.SchemaVersion != agentusage.SchemaVersion || payload.AgentUsage.Empty() || (payload.AgentUsage.ProviderUsage == nil && !payload.AgentUsage.UsageUnavailable) || (payload.AgentUsage.ProviderUsage != nil && payload.AgentUsage.UsageUnavailable) || (payload.AgentUsage.UsageUnavailable && strings.TrimSpace(payload.AgentUsage.UsageUnavailableReason) == "") {
		return fmt.Errorf("invalid report usage schema for %s", target.Event.EventID)
	}
	if payload.AgentUsage.Session.AgentSessionID != target.Meta.AgentSessionID || payload.AgentUsage.Session.PreviousAgentSessionID != target.Meta.PreviousAgentSessionID || payload.AgentUsage.Surface != target.Meta.Surface || payload.AgentUsage.Executor != target.Meta.AgentExecutor || payload.AgentUsage.Model != target.Meta.AgentModel || payload.AgentUsage.ReasoningEffort != target.Meta.AgentReasoningEffort || strings.TrimSpace(payload.ForkSourceAgentSessionID) != target.Meta.ForkSourceAgentSessionID {
		return fmt.Errorf("report usage metadata differs for %s", target.Event.EventID)
	}
	return nil
}

func actualUsageAppendRequest(state completionState, target completionTarget, actual ReportAgentUsageRequest) (ledger.AppendRequest, error) {
	if strings.TrimSpace(actual.MissionID) != target.Event.MissionID || strings.TrimSpace(actual.CanonicalEventID) != target.Event.EventID {
		return ledger.AppendRequest{}, fmt.Errorf("actual report usage does not match target")
	}
	if strings.TrimSpace(actual.AgentSessionID) != target.Meta.AgentSessionID ||
		strings.TrimSpace(actual.PreviousAgentSessionID) != target.Meta.PreviousAgentSessionID ||
		strings.TrimSpace(actual.Surface) != target.Meta.Surface ||
		strings.TrimSpace(actual.ForkSourceAgentSessionID) != target.Meta.ForkSourceAgentSessionID ||
		strings.TrimSpace(actual.Usage.Executor) != target.Meta.AgentExecutor ||
		strings.TrimSpace(actual.Usage.Model) != target.Meta.AgentModel ||
		strings.TrimSpace(actual.Usage.ReasoningEffort) != target.Meta.AgentReasoningEffort {
		return ledger.AppendRequest{}, fmt.Errorf("actual report usage metadata differs from target")
	}
	if actual.Usage.SchemaVersion != agentusage.SchemaVersion || actual.Usage.Empty() ||
		(actual.Usage.ProviderUsage == nil && !actual.Usage.UsageUnavailable) ||
		(actual.Usage.ProviderUsage != nil && actual.Usage.UsageUnavailable) ||
		(actual.Usage.UsageUnavailable && strings.TrimSpace(actual.Usage.UsageUnavailableReason) == "") {
		return ledger.AppendRequest{}, fmt.Errorf("actual report usage has invalid outcome")
	}
	actual.PendingEventID = target.Pending.EventID
	actual.Surface = target.Meta.Surface
	actual.AgentSessionID = target.Meta.AgentSessionID
	actual.PreviousAgentSessionID = target.Meta.PreviousAgentSessionID
	actual.ForkSourceAgentSessionID = target.Meta.ForkSourceAgentSessionID
	actual.Usage.Executor = target.Meta.AgentExecutor
	actual.Usage.Model = target.Meta.AgentModel
	actual.Usage.ReasoningEffort = target.Meta.AgentReasoningEffort
	if actual.AgentSessionID == "" {
		return ledger.AppendRequest{}, fmt.Errorf("report usage target has no agent session")
	}
	request, ok, err := buildReportAgentUsageAppendRequest(actual)
	if err != nil {
		return ledger.AppendRequest{}, err
	}
	if !ok {
		return ledger.AppendRequest{}, fmt.Errorf("actual report usage is empty")
	}
	return request, nil
}

func unavailableUsageRequest(state completionState, target completionTarget) (ledger.AppendRequest, error) {
	if target.Meta.AgentSessionID == "" || target.Meta.Surface == "report_" {
		return ledger.AppendRequest{}, fmt.Errorf("report usage target metadata is incomplete")
	}
	usage := agentusage.New("", target.Meta.AgentExecutor, target.Meta.AgentModel, target.Meta.AgentReasoningEffort, "")
	usage = usage.WithSurface(target.Meta.Surface).WithSession(target.Meta.PreviousAgentSessionID, target.Meta.AgentSessionID, false, false).WithUnavailable(ReportRunCompletionReason)
	encoded, _ := json.Marshal(map[string]any{
		"kind": "report_agent_usage", "pending_event_id": target.Pending.EventID,
		"correlation_event_id": target.Event.EventID, "agent_usage": usage,
		"fork_source_agent_session_id": target.Meta.ForkSourceAgentSessionID,
	})
	return ledger.AppendRequest{
		EventID: reportAgentUsageEventID(target.Event.EventID), MissionID: target.Event.MissionID,
		EventType: ReportAgentUsageRecordedEventType, Producer: ledger.Producer{Type: "agent_session", ID: target.Meta.AgentSessionID},
		CausationEventID: target.Event.EventID, CorrelationID: target.Pending.EventID, Payload: encoded,
	}, nil
}

func findCompletionState(events []ledger.Event, canonicalID string) (completionState, error) {
	registration, err := reportrun.BuildRegistration(reportRunEvents(events), reportrun.RegistrationBackfilled, time.Now().UTC())
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
		if run.LifecycleState == reportrun.LifecycleAmbiguous {
			continue
		}
		var canonicalMember, pendingMember *reportrun.EventMembership
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

func memberInRun(members []reportrun.EventMembership, runID, eventID, role string) bool {
	for _, member := range members {
		if member.RunID == runID && member.EventID == eventID && member.EventRole == role {
			return true
		}
	}
	return false
}

func reportRunEvents(events []ledger.Event) []reportrun.Event {
	out := make([]reportrun.Event, 0, len(events))
	for _, event := range events {
		out = append(out, reportrun.Event{EventID: event.EventID, MissionID: event.MissionID, Sequence: event.Sequence, EventType: event.EventType, Producer: event.Producer, CausationEventID: event.CausationEventID, CorrelationID: event.CorrelationID, Payload: event.Payload, CreatedAt: event.CreatedAt})
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

func RecoverMission(ctx context.Context, store ReportCompletionStore, missionID string) (int, error) {
	events, err := store.ListEvents(ctx, missionID)
	if err != nil {
		return 0, err
	}
	registration, err := reportrun.BuildRegistration(reportRunEvents(events), reportrun.RegistrationBackfilled, time.Now().UTC())
	if err != nil {
		return 0, err
	}
	changed := 0
	var errs []error
	seen := map[string]bool{}
	for _, run := range registration.Runs {
		if seen[run.RunID] || run.LifecycleState == reportrun.LifecycleAmbiguous || run.FinalArtifactID == "" {
			continue
		}
		seen[run.RunID] = true
		completionID := completionEventID(run.RunID)
		existing, exists := eventByID(events, completionID)
		canonicalID := ""
		if exists && existing.EventType == ReportRunCompletedEventType {
			var payload struct {
				CanonicalEventID string `json:"canonical_event_id"`
			}
			if json.Unmarshal(existing.Payload, &payload) == nil {
				canonicalID = strings.TrimSpace(payload.CanonicalEventID)
			}
			if canonicalID == "" {
				errs = append(errs, fmt.Errorf("recover report completion for %s: invalid existing report completion", run.RunID))
				continue
			}
		} else {
			var finals []ledger.Event
			for _, member := range registration.Events {
				if member.RunID != run.RunID || member.EventRole != "final" {
					continue
				}
				if event, found := eventByID(events, member.EventID); found && event.EventType == "report.artifact.created" {
					finals = append(finals, event)
				}
			}
			sort.Slice(finals, func(i, j int) bool {
				if finals[i].Sequence != finals[j].Sequence {
					return finals[i].Sequence < finals[j].Sequence
				}
				if !finals[i].CreatedAt.Equal(finals[j].CreatedAt) {
					return finals[i].CreatedAt.Before(finals[j].CreatedAt)
				}
				return finals[i].EventID < finals[j].EventID
			})
			if len(finals) > 0 {
				canonicalID = finals[len(finals)-1].EventID
			}
		}
		if canonicalID == "" {
			continue
		}
		_, created, runErr := completeReportRun(ctx, store, ReportCompletionRequest{MissionID: missionID, CanonicalEventID: canonicalID})
		if runErr != nil {
			errs = append(errs, fmt.Errorf("recover report completion for %s: %w", run.RunID, runErr))
			continue
		}
		if created {
			changed++
		}
	}
	return changed, errors.Join(errs...)
}

func RecoverAll(ctx context.Context, store ReportCompletionMissionStore) (int, error) {
	missions, err := store.ListMissionsWithState(ctx, mission.ListRequest{IncludeArchived: true})
	if err != nil {
		return 0, err
	}
	changed := 0
	var errs []error
	for _, current := range missions {
		count, runErr := RecoverMission(ctx, store, current.MissionID)
		changed += count
		if runErr != nil {
			errs = append(errs, fmt.Errorf("recover report completion for %s: %w", current.MissionID, runErr))
		}
	}
	return changed, errors.Join(errs...)
}
