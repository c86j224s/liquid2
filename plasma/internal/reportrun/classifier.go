package reportrun

import (
	"strings"
	"time"
)

type eventPayload map[string]any

type runBuild struct {
	run       Run
	events    map[string]EventMembership
	artifacts map[string]ArtifactMembership
	ambiguous bool
}

// BuildRegistration deterministically projects report events into run
// membership. It uses only explicit lineage fields and marks unresolved report
// lineage ambiguous instead of guessing from title, time, producer, or name.
func BuildRegistration(events []Event, status string, now time.Time) (Registration, error) {
	if status == "" {
		status = RegistrationBackfilled
	}
	payloads := map[string]eventPayload{}
	runs := map[string]*runBuild{}
	eventRun := map[string]string{}
	artifactRuns := map[string]map[string]bool{}
	ambiguousArtifacts := map[string]bool{}
	var reportEvents []Event

	for _, event := range events {
		if !isReportEvent(event.EventType) {
			continue
		}
		payload, err := parsePayload(event)
		if err != nil {
			runID := event.EventID
			runs[runID] = newRunBuild(runID, event.MissionID, "", status, now)
			runs[runID].ambiguous = true
			continue
		}
		payloads[event.EventID] = payload
		reportEvents = append(reportEvents, event)
	}
	lineage := buildPendingLineage(reportEvents, payloads)

	for _, event := range reportEvents {
		if event.EventType != "report.draft.pending" {
			continue
		}
		payload := payloads[event.EventID]
		if lineage.invalid[event.EventID] {
			run := ensureRun(runs, event.EventID, event.MissionID, payloadString(payload, "title"), status, now)
			run.ambiguous = true
			addEvent(run, event, "ambiguous", event.EventID, now)
			eventRun[event.EventID] = event.EventID
			continue
		}
		rootID := firstNonEmpty(lineage.roots[event.EventID], event.EventID)
		run := ensureRun(runs, rootID, event.MissionID, payloadString(payload, "title"), status, now)
		addEvent(run, event, "draft_pending", event.EventID, now)
		eventRun[event.EventID] = rootID
	}
	for runID := range lineage.ambiguousRuns {
		if run := runs[runID]; run != nil {
			run.ambiguous = true
		}
	}

	changed := true
	for changed {
		changed = false
		for _, event := range reportEvents {
			if eventRun[event.EventID] != "" {
				continue
			}
			payload, ok := payloads[event.EventID]
			if !ok {
				continue
			}
			runID, attemptID, ok := eventLineageRun(payload, eventRun, artifactRuns)
			if !ok {
				continue
			}
			run := ensureRun(runs, runID, event.MissionID, "", status, now)
			for _, artifactID := range addClassifiedEvent(run, event, payload, attemptID, now) {
				indexArtifact(artifactRuns, artifactID, runID)
			}
			eventRun[event.EventID] = runID
			changed = true
		}
	}

	for _, event := range reportEvents {
		if eventRun[event.EventID] != "" {
			continue
		}
		payload, ok := payloads[event.EventID]
		if !ok {
			continue
		}
		runID := firstNonEmpty(payloadString(payload, "pending_event_id"), payloadString(payload, "source_artifact_id"), payloadString(payload, "base_artifact_id"), event.EventID)
		run := ensureRun(runs, runID, event.MissionID, payloadString(payload, "title"), status, now)
		run.ambiguous = true
		addEvent(run, event, "ambiguous", payloadString(payload, "pending_event_id"), now)
		addAmbiguousArtifacts(run, event, payload, now)
		eventRun[event.EventID] = runID
		for _, artifactID := range artifactIDs(payload) {
			ambiguousArtifacts[artifactID] = true
		}
	}

	for artifactID := range ambiguousArtifacts {
		if owners := artifactRuns[artifactID]; len(owners) > 0 {
			for runID := range owners {
				if run := runs[runID]; run != nil {
					run.ambiguous = true
				}
			}
		}
	}

	registration := Registration{}
	for _, run := range runs {
		finalizeRun(run, eventRun)
		registration.Runs = append(registration.Runs, run.run)
		for _, membership := range run.events {
			registration.Events = append(registration.Events, membership)
		}
		for _, membership := range run.artifacts {
			registration.Artifacts = append(registration.Artifacts, membership)
		}
	}
	return registration, nil
}

func addClassifiedEvent(run *runBuild, event Event, payload eventPayload, attemptID string, now time.Time) []string {
	role := eventRole(event.EventType, payload)
	addEvent(run, event, role, attemptID, now)
	for _, inputID := range referencedArtifactIDs(payload) {
		addArtifact(run, inputID, event.MissionID, ArtifactRoleInput, OwnershipReferenced, attemptID, event.EventID, now)
	}
	switch event.EventType {
	case "report.artifact.created":
		artifactID := payloadString(payload, "artifact_id")
		addArtifact(run, artifactID, event.MissionID, ArtifactRoleFinal, OwnershipCreated, attemptID, event.EventID, now)
		run.run.FinalArtifactID = artifactID
		return []string{artifactID}
	case "report.artifact.exported":
		artifactID := payloadString(payload, "artifact_id")
		contentModelArtifactID := payloadString(payload, "content_model_artifact_id")
		addArtifact(run, artifactID, event.MissionID, ArtifactRoleDerivative, OwnershipCreated, attemptID, event.EventID, now)
		addArtifact(run, contentModelArtifactID, event.MissionID, ArtifactRoleIntermediate, OwnershipCreated, attemptID, event.EventID, now)
		return []string{artifactID, contentModelArtifactID}
	case "report.redpen.saved":
		artifactID := payloadString(payload, "artifact_id")
		ownership, ok := redpenArtifactOwnership(payload)
		if !ok {
			run.ambiguous = true
			ownership = OwnershipReferenced
		}
		addArtifact(run, artifactID, event.MissionID, ArtifactRoleIntermediate, ownership, attemptID, event.EventID, now)
		if ownership == OwnershipCreated {
			return []string{artifactID}
		}
		return nil
	default:
		artifactID := payloadString(payload, "artifact_id")
		if isCreatedIntermediateEvent(event.EventType) {
			addArtifact(run, artifactID, event.MissionID, ArtifactRoleIntermediate, OwnershipCreated, attemptID, event.EventID, now)
			return []string{artifactID}
		}
		if artifactID != "" && !isKnownNonCreatorReportEvent(event.EventType) {
			run.ambiguous = true
			addArtifact(run, artifactID, event.MissionID, ArtifactRoleIntermediate, OwnershipReferenced, attemptID, event.EventID, now)
		}
	}
	return nil
}

func addAmbiguousArtifacts(run *runBuild, event Event, payload eventPayload, now time.Time) {
	attemptID := payloadString(payload, "pending_event_id")
	for _, inputID := range referencedArtifactIDs(payload) {
		addArtifact(run, inputID, event.MissionID, ArtifactRoleInput, OwnershipReferenced, attemptID, event.EventID, now)
	}
	switch event.EventType {
	case "report.artifact.created":
		addArtifact(run, payloadString(payload, "artifact_id"), event.MissionID, ArtifactRoleFinal, OwnershipCreated, attemptID, event.EventID, now)
	case "report.artifact.exported":
		addArtifact(run, payloadString(payload, "artifact_id"), event.MissionID, ArtifactRoleDerivative, OwnershipCreated, attemptID, event.EventID, now)
		addArtifact(run, payloadString(payload, "content_model_artifact_id"), event.MissionID, ArtifactRoleIntermediate, OwnershipCreated, attemptID, event.EventID, now)
	case "report.redpen.saved":
		ownership, ok := redpenArtifactOwnership(payload)
		if !ok {
			ownership = OwnershipReferenced
		}
		addArtifact(run, payloadString(payload, "artifact_id"), event.MissionID, ArtifactRoleIntermediate, ownership, attemptID, event.EventID, now)
	default:
		if artifactID := payloadString(payload, "artifact_id"); artifactID != "" {
			addArtifact(run, artifactID, event.MissionID, ArtifactRoleIntermediate, OwnershipReferenced, attemptID, event.EventID, now)
		}
	}
}

func redpenArtifactOwnership(payload eventPayload) (string, bool) {
	switch payloadString(payload, "artifact_ownership") {
	case OwnershipCreated:
		return OwnershipCreated, true
	case OwnershipReferenced:
		return OwnershipReferenced, true
	default:
		return "", false
	}
}

func eventLineageRun(payload eventPayload, eventRun map[string]string, artifactRuns map[string]map[string]bool) (string, string, bool) {
	pendingID := payloadString(payload, "pending_event_id")
	if pendingID != "" {
		if runID := eventRun[pendingID]; runID != "" {
			return runID, pendingID, true
		}
	}
	planEventID := payloadString(payload, "plan_event_id")
	if planEventID != "" {
		if runID := eventRun[planEventID]; runID != "" {
			return runID, pendingID, true
		}
	}
	for _, artifactID := range []string{payloadString(payload, "source_artifact_id"), payloadString(payload, "base_artifact_id")} {
		if runID, ok := singleArtifactRun(artifactRuns, artifactID); ok {
			return runID, pendingID, true
		}
	}
	return "", "", false
}

func finalizeRun(run *runBuild, eventRun map[string]string) {
	if run.ambiguous {
		run.run.LifecycleState = LifecycleAmbiguous
		return
	}
	open := map[string]bool{}
	terminal := map[string]string{}
	for _, membership := range run.events {
		if membership.EventRole == "draft_pending" || membership.EventRole == "operation_pending" {
			open[membership.EventID] = true
		}
	}
	for _, membership := range run.events {
		if membership.EventRole != "terminal" && membership.EventRole != "canceled" && membership.EventRole != "final" && membership.EventRole != "derivative" {
			continue
		}
		pendingID := membership.AttemptEventID
		if pendingID != "" {
			delete(open, pendingID)
			terminal[pendingID] = membership.EventRole
		}
	}
	switch {
	case len(open) > 0:
		run.run.LifecycleState = LifecycleActive
	case run.run.FinalArtifactID != "":
		run.run.LifecycleState = LifecycleCompleted
	case hasCanceledTerminal(run.events):
		run.run.LifecycleState = LifecycleCanceled
	case len(terminal) > 0:
		run.run.LifecycleState = LifecycleFailed
	default:
		run.run.LifecycleState = LifecycleActive
	}
}

func hasCanceledTerminal(events map[string]EventMembership) bool {
	for _, event := range events {
		if event.EventRole == "canceled" {
			return true
		}
	}
	return false
}

func newRunBuild(runID, missionID, title, status string, now time.Time) *runBuild {
	return &runBuild{
		run: Run{
			RunID: runID, MissionID: missionID, RootPendingEventID: runID,
			LifecycleState: LifecycleActive, Revision: 1, Title: strings.TrimSpace(title),
			RegistrationStatus: status, Usage: UsageAggregate{AggregationVersion: UsageAggregationVersion},
			CreatedAt: now, UpdatedAt: now,
		},
		events:    map[string]EventMembership{},
		artifacts: map[string]ArtifactMembership{},
	}
}

func ensureRun(runs map[string]*runBuild, runID, missionID, title, status string, now time.Time) *runBuild {
	run := runs[runID]
	if run == nil {
		run = newRunBuild(runID, missionID, title, status, now)
		runs[runID] = run
	}
	if run.run.Title == "" {
		run.run.Title = strings.TrimSpace(title)
	}
	return run
}

func addEvent(run *runBuild, event Event, role, attemptID string, now time.Time) {
	if event.EventID == "" {
		return
	}
	run.events[event.EventID] = EventMembership{RunID: run.run.RunID, EventID: event.EventID, MissionID: event.MissionID, EventRole: role, AttemptEventID: attemptID, CreatedAt: now}
}

func addArtifact(run *runBuild, artifactID, missionID, role, ownership, attemptID, sourceEventID string, now time.Time) {
	artifactID = strings.TrimSpace(artifactID)
	if artifactID == "" {
		return
	}
	existing := run.artifacts[artifactID]
	if existing.ArtifactID != "" && artifactRoleRank(existing.ArtifactRole) >= artifactRoleRank(role) && existing.Ownership == OwnershipCreated {
		return
	}
	run.artifacts[artifactID] = ArtifactMembership{RunID: run.run.RunID, ArtifactID: artifactID, MissionID: missionID, ArtifactRole: role, Ownership: ownership, AttemptEventID: attemptID, SourceEventID: sourceEventID, CreatedAt: now}
}

func indexArtifact(index map[string]map[string]bool, artifactID, runID string) {
	if strings.TrimSpace(artifactID) == "" || strings.TrimSpace(runID) == "" {
		return
	}
	if index[artifactID] == nil {
		index[artifactID] = map[string]bool{}
	}
	index[artifactID][runID] = true
}

func singleArtifactRun(index map[string]map[string]bool, artifactID string) (string, bool) {
	owners := index[strings.TrimSpace(artifactID)]
	if len(owners) != 1 {
		return "", false
	}
	for runID := range owners {
		return runID, true
	}
	return "", false
}
