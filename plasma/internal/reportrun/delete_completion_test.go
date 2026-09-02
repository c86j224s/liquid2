package reportrun

import (
	"encoding/json"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/agentusage"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

const (
	deleteRunID      = "evt_root"
	deleteMissionID  = "mis_1"
	deleteRootID     = "evt_root"
	deleteFinalID    = "evt_final"
	deleteArtifactID = "art_final"
)

func producerSystem() ledger.Producer {
	return ledger.Producer{Type: "system", ID: "report-completion"}
}

func member(runID, missionID, eventID, role, attempt string, event ledger.Event) MemberEvent {
	return MemberEvent{Membership: EventMembership{RunID: runID, EventID: eventID, MissionID: missionID, EventRole: role, AttemptEventID: attempt}, Event: Event{EventID: eventID, MissionID: missionID, Sequence: event.Sequence, EventType: event.EventType, Producer: event.Producer, CausationEventID: event.CausationEventID, CorrelationID: event.CorrelationID, Payload: event.Payload, CreatedAt: event.CreatedAt}}
}

func validCompletionFacts() DeleteFacts {
	root := member(deleteRunID, deleteMissionID, deleteRootID, "draft_pending", deleteRootID, ledger.Event{EventID: deleteRootID, MissionID: deleteMissionID, EventType: "report.draft.pending", Payload: []byte(`{"retry_strategy":"initial"}`)})
	final := member(deleteRunID, deleteMissionID, deleteFinalID, "final", deleteRootID, ledger.Event{EventID: deleteFinalID, MissionID: deleteMissionID, EventType: "report.artifact.created", Payload: []byte(`{"pending_event_id":"evt_root","artifact_id":"art_final"}`)})
	completion := member(deleteRunID, deleteMissionID, completionDeleteEventID(deleteRunID), "completion", deleteRootID, ledger.Event{EventID: completionDeleteEventID(deleteRunID), MissionID: deleteMissionID, EventType: "report.run.completed", Producer: producerSystem(), CausationEventID: deleteFinalID, CorrelationID: deleteRunID, Payload: []byte(`{"kind":"report_run_completed","schema_version":"plasma.report_run_completion.v1","run_id":"evt_root","pending_event_id":"evt_root","canonical_event_id":"evt_final","artifact_id":"art_final","delayed_usage_target_count":0,"usage_recorded_count":0,"usage_unavailable_count":0}`)})
	return DeleteFacts{Run: Run{RunID: deleteRunID, RootPendingEventID: deleteRootID, MissionID: deleteMissionID, LifecycleState: LifecycleCompleted, FinalArtifactID: deleteArtifactID}, Events: []MemberEvent{root, final, completion}}
}

func TestPreviewDeleteAcceptsExactZeroTargetCompletionMember(t *testing.T) {
	preview := PreviewDelete(validCompletionFacts(), "")
	if !preview.Eligible {
		t.Fatalf("valid completion blocked: %#v", preview.Blockers)
	}
}

func completionFactsWithUsage() DeleteFacts {
	facts := validCompletionFacts()
	requirementsID := "evt_requirements"
	requirementsPayload := []byte(`{"pending_event_id":"evt_root","previous_provider_session_id":"ses-req","agent_executor":"codex","agent_model":"model","agent_reasoning_effort":"high"}`)
	requirements := member(deleteRunID, deleteMissionID, requirementsID, "stage", deleteRootID, ledger.Event{EventID: requirementsID, MissionID: deleteMissionID, EventType: "report.requirements.mapped", Payload: requirementsPayload})
	usageID := "evt_report_usage_requirements"
	usage := agentusage.New("", "codex", "model", "high", "").WithSurface("report_requirements").WithSession("ses-req", "ses-req", false, false).WithProviderUsage(agentusage.ProviderUsage{Scope: agentusage.UsageScopeCall, InputTokens: 3, OutputTokens: 2}, "provider")
	usagePayload := usageEventPayload(deleteRootID, requirementsID, "", usage)
	usageMember := member(deleteRunID, deleteMissionID, usageID, "stage", deleteRootID, ledger.Event{EventID: usageID, MissionID: deleteMissionID, EventType: "report.agent_usage.recorded", Producer: ledger.Producer{Type: "agent_session", ID: "ses-req"}, CausationEventID: requirementsID, CorrelationID: deleteRootID, Payload: usagePayload})
	partID := "evt_part"
	part := member(deleteRunID, deleteMissionID, partID, "stage", deleteRootID, ledger.Event{EventID: partID, MissionID: deleteMissionID, EventType: "report.part.edited", Payload: []byte(`{"pending_event_id":"evt_root","provider_session_id":"ses-part","previous_provider_session_id":"ses-prev","agent_executor":"codex","agent_model":"model","agent_reasoning_effort":"high"}`)})
	partUsageID := "evt_report_usage_part"
	partUsage := agentusage.New("", "codex", "model", "high", "").WithSurface("report_part_edit").WithSession("ses-prev", "ses-part", false, false).WithUnavailable("provider unavailable for test")
	partUsageMember := member(deleteRunID, deleteMissionID, partUsageID, "stage", deleteRootID, ledger.Event{EventID: partUsageID, MissionID: deleteMissionID, EventType: "report.agent_usage.recorded", Producer: ledger.Producer{Type: "agent_session", ID: "ses-part"}, CausationEventID: partID, CorrelationID: deleteRootID, Payload: usageEventPayload(deleteRootID, partID, "", partUsage)})
	facts.Events = append(facts.Events[:2], requirements, usageMember, part, partUsageMember, facts.Events[2])
	facts.Events[len(facts.Events)-1].Membership.AttemptEventID = deleteRootID
	facts.Events[len(facts.Events)-1].Event.Payload = []byte(`{"kind":"report_run_completed","schema_version":"plasma.report_run_completion.v1","run_id":"evt_root","pending_event_id":"evt_root","canonical_event_id":"evt_final","artifact_id":"art_final","delayed_usage_target_count":2,"usage_recorded_count":1,"usage_unavailable_count":1}`)
	facts.Events[len(facts.Events)-1].Membership.AttemptEventID = deleteRootID
	return facts
}

func usageEventPayload(pendingID, targetID, fork string, usage agentusage.AgentUsage) []byte {
	payload := map[string]any{"kind": "report_agent_usage", "pending_event_id": pendingID, "correlation_event_id": targetID, "agent_usage": usage}
	if fork != "" {
		payload["fork_source_agent_session_id"] = fork
	}
	encoded, _ := json.Marshal(payload)
	return encoded
}

func TestPreviewDeleteAcceptsRecordedAndUnavailableUsageOutcomes(t *testing.T) {
	preview := PreviewDelete(completionFactsWithUsage(), "")
	if !preview.Eligible {
		t.Fatalf("valid usage outcomes blocked: %#v", preview.Blockers)
	}
}

func TestPreviewDeleteRejectsCompletionMembershipAndEnvelopeMutations(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*DeleteFacts)
	}{
		{"completion event id", func(f *DeleteFacts) { f.Events[len(f.Events)-1].Event.EventID = "evt_wrong" }},
		{"completion membership event id", func(f *DeleteFacts) { f.Events[len(f.Events)-1].Membership.EventID = "evt_wrong" }},
		{"completion membership mission", func(f *DeleteFacts) { f.Events[len(f.Events)-1].Membership.MissionID = "mis_other" }},
		{"completion event mission", func(f *DeleteFacts) { f.Events[len(f.Events)-1].Event.MissionID = "mis_other" }},
		{"completion run", func(f *DeleteFacts) { f.Events[len(f.Events)-1].Membership.RunID = "evt_other" }},
		{"completion role", func(f *DeleteFacts) { f.Events[len(f.Events)-1].Membership.EventRole = "stage" }},
		{"completion attempt", func(f *DeleteFacts) { f.Events[len(f.Events)-1].Membership.AttemptEventID = "evt_child" }},
		{"completion producer", func(f *DeleteFacts) {
			f.Events[len(f.Events)-1].Event.Producer = ledger.Producer{Type: "agent", ID: "x"}
		}},
		{"completion causation", func(f *DeleteFacts) { f.Events[len(f.Events)-1].Event.CausationEventID = deleteRootID }},
		{"completion correlation", func(f *DeleteFacts) { f.Events[len(f.Events)-1].Event.CorrelationID = "evt_child" }},
		{"completion kind", func(f *DeleteFacts) { mutateCompletionPayload(f, "kind", "wrong") }},
		{"completion schema", func(f *DeleteFacts) { mutateCompletionPayload(f, "schema_version", "wrong") }},
		{"completion artifact", func(f *DeleteFacts) { mutateCompletionPayload(f, "artifact_id", "art_other") }},
		{"completion counts", func(f *DeleteFacts) { mutateCompletionPayload(f, "usage_recorded_count", float64(2)) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			facts := completionFactsWithUsage()
			tc.mutate(&facts)
			assertCompletionPending(t, facts)
		})
	}
}

func TestPreviewDeleteRejectsRootCanonicalTargetUsageAndOutcomeMutations(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*DeleteFacts)
	}{
		{"root membership event id", func(f *DeleteFacts) { f.Events[0].Membership.EventID = "evt_other" }},
		{"root membership mission", func(f *DeleteFacts) { f.Events[0].Membership.MissionID = "mis_other" }},
		{"root event mission", func(f *DeleteFacts) { f.Events[0].Event.MissionID = "mis_other" }},
		{"root run", func(f *DeleteFacts) { f.Events[0].Membership.RunID = "evt_other" }},
		{"root role", func(f *DeleteFacts) { f.Events[0].Membership.EventRole = "final" }},
		{"root attempt", func(f *DeleteFacts) { f.Events[0].Membership.AttemptEventID = "evt_child" }},
		{"canonical membership event id", func(f *DeleteFacts) { f.Events[1].Membership.EventID = "evt_other" }},
		{"canonical membership mission", func(f *DeleteFacts) { f.Events[1].Membership.MissionID = "mis_other" }},
		{"canonical event mission", func(f *DeleteFacts) { f.Events[1].Event.MissionID = "mis_other" }},
		{"canonical attempt", func(f *DeleteFacts) { f.Events[1].Membership.AttemptEventID = "evt_child" }},
		{"canonical pending", func(f *DeleteFacts) { mutateEventPayload(f, 1, "pending_event_id", "evt_child") }},
		{"canonical artifact", func(f *DeleteFacts) { mutateEventPayload(f, 1, "artifact_id", "art_other") }},
		{"target attempt", func(f *DeleteFacts) { f.Events[2].Membership.AttemptEventID = "evt_child" }},
		{"target pending", func(f *DeleteFacts) { mutateEventPayload(f, 2, "pending_event_id", "evt_child") }},
		{"target malformed", func(f *DeleteFacts) { f.Events[2].Event.Payload = []byte("{") }},
		{"usage missing", func(f *DeleteFacts) { f.Events = append(f.Events[:3], f.Events[4:]...) }},
		{"usage duplicate", func(f *DeleteFacts) { f.Events = append(f.Events, f.Events[3]) }},
		{"usage wrong id", func(f *DeleteFacts) { f.Events[3].Event.EventID = "evt_extra" }},
		{"usage wrong producer", func(f *DeleteFacts) { f.Events[3].Event.Producer = ledger.Producer{Type: "agent_session", ID: "other"} }},
		{"usage wrong causation", func(f *DeleteFacts) { f.Events[3].Event.CausationEventID = deleteRootID }},
		{"usage wrong correlation", func(f *DeleteFacts) { f.Events[3].Event.CorrelationID = "evt_child" }},
		{"usage empty unavailable reason", func(f *DeleteFacts) {
			mutateUsage(f, 5, func(u *agentusage.AgentUsage) { u.UsageUnavailableReason = "" })
		}},
		{"usage wrong session", func(f *DeleteFacts) {
			mutateUsage(f, 3, func(u *agentusage.AgentUsage) { u.Session.AgentSessionID = "other" })
		}},
		{"usage wrong previous", func(f *DeleteFacts) {
			mutateUsage(f, 3, func(u *agentusage.AgentUsage) { u.Session.PreviousAgentSessionID = "other" })
		}},
		{"usage wrong surface", func(f *DeleteFacts) { mutateUsage(f, 3, func(u *agentusage.AgentUsage) { u.Surface = "other" }) }},
		{"usage wrong schema", func(f *DeleteFacts) { mutateUsage(f, 3, func(u *agentusage.AgentUsage) { u.SchemaVersion = 1 }) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			facts := completionFactsWithUsage()
			tc.mutate(&facts)
			assertCompletionPending(t, facts)
		})
	}
}

func TestPreviewDeleteAllowsPatchLatestFinalArtifactToDiffer(t *testing.T) {
	facts := completionFactsWithUsage()
	facts.Run.FinalArtifactID = "art_later_patch"
	preview := PreviewDelete(facts, "")
	if !preview.Eligible {
		t.Fatalf("own canonical completion should remain valid after patch promotion: %#v", preview.Blockers)
	}
}

func assertCompletionPending(t *testing.T, facts DeleteFacts) {
	t.Helper()
	preview := PreviewDelete(facts, "")
	if preview.Eligible {
		t.Fatal("malformed completion was accepted")
	}
	found := false
	for _, blocker := range preview.Blockers {
		if blocker.ReasonCode == BlockerCompletionPending {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing completion_pending blocker: %#v", preview.Blockers)
	}
}

func mutateCompletionPayload(f *DeleteFacts, key string, value any) {
	mutateEventPayload(f, len(f.Events)-1, key, value)
}
func mutateEventPayload(f *DeleteFacts, index int, key string, value any) {
	var payload map[string]any
	_ = json.Unmarshal(f.Events[index].Event.Payload, &payload)
	payload[key] = value
	f.Events[index].Event.Payload, _ = json.Marshal(payload)
}
func mutateUsage(f *DeleteFacts, index int, mutate func(*agentusage.AgentUsage)) {
	var payload struct {
		AgentUsage agentusage.AgentUsage `json:"agent_usage"`
	}
	_ = json.Unmarshal(f.Events[index].Event.Payload, &payload)
	mutate(&payload.AgentUsage)
	var raw map[string]any
	_ = json.Unmarshal(f.Events[index].Event.Payload, &raw)
	raw["agent_usage"] = payload.AgentUsage
	f.Events[index].Event.Payload, _ = json.Marshal(raw)
}

func TestPreviewDeleteRequiresCompletionMemberOnCompletedRun(t *testing.T) {
	facts := completionFactsWithUsage()
	facts.Events = facts.Events[:len(facts.Events)-1]
	assertCompletionPending(t, facts)
}

func TestPreviewDeleteRejectsCompletionLineagePayloadMutations(t *testing.T) {
	for _, tc := range []struct {
		name  string
		key   string
		value any
	}{
		{"run id", "run_id", "evt_other"},
		{"pending id", "pending_event_id", "evt_child"},
		{"canonical id", "canonical_event_id", "evt_other_final"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			facts := completionFactsWithUsage()
			mutateCompletionPayload(&facts, tc.key, tc.value)
			assertCompletionPending(t, facts)
		})
	}
}

func completionFactsWithFinalEditTarget() DeleteFacts {
	facts := validCompletionFacts()
	targetID := "evt_final_edit_writer"
	target := member(deleteRunID, deleteMissionID, targetID, "stage", deleteRootID, ledger.Event{
		EventID: targetID, MissionID: deleteMissionID, EventType: "report.final_edit.writer.submitted",
		Payload: []byte(`{"pending_event_id":"evt_root","stage":"final_write","provider_session_id":"ses-writer","previous_provider_session_id":"ses-prev","agent_executor":"codex","agent_model":"model","agent_reasoning_effort":"high"}`),
	})
	usageID := "evt_report_usage_final_edit_writer"
	usage := agentusage.New("", "codex", "model", "high", "").WithSurface("report_final_write").WithSession("ses-writer", "ses-writer", false, false).WithProviderUsage(agentusage.ProviderUsage{Scope: agentusage.UsageScopeCall, InputTokens: 1, OutputTokens: 1}, "provider")
	usageMember := member(deleteRunID, deleteMissionID, usageID, "stage", deleteRootID, ledger.Event{
		EventID: usageID, MissionID: deleteMissionID, EventType: "report.agent_usage.recorded", Producer: ledger.Producer{Type: "agent_session", ID: "ses-writer"}, CausationEventID: targetID, CorrelationID: deleteRootID, Payload: usageEventPayload(deleteRootID, targetID, "", usage),
	})
	completion := facts.Events[len(facts.Events)-1]
	facts.Events = append(facts.Events[:len(facts.Events)-1], target, usageMember, completion)
	mutateCompletionPayload(&facts, "delayed_usage_target_count", float64(1))
	mutateCompletionPayload(&facts, "usage_recorded_count", float64(1))
	return facts
}

func TestPreviewDeleteRejectsFinalEditInvalidStage(t *testing.T) {
	facts := completionFactsWithFinalEditTarget()
	mutateEventPayload(&facts, 2, "stage", "reader_edit")
	assertCompletionPending(t, facts)
}

func TestPreviewDeleteRejectsUsageMetadataAndMembershipMutations(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*DeleteFacts)
	}{
		{"fork source", func(f *DeleteFacts) {
			mutateUsage(f, 3, func(u *agentusage.AgentUsage) { u.UsageUnavailableReason = "" })
			var raw map[string]any
			_ = json.Unmarshal(f.Events[3].Event.Payload, &raw)
			raw["fork_source_agent_session_id"] = "fork"
			f.Events[3].Event.Payload, _ = json.Marshal(raw)
		}},
		{"executor", func(f *DeleteFacts) { mutateUsage(f, 3, func(u *agentusage.AgentUsage) { u.Executor = "other" }) }},
		{"model", func(f *DeleteFacts) { mutateUsage(f, 3, func(u *agentusage.AgentUsage) { u.Model = "other" }) }},
		{"reasoning", func(f *DeleteFacts) {
			mutateUsage(f, 3, func(u *agentusage.AgentUsage) { u.ReasoningEffort = "other" })
		}},
		{"membership event id", func(f *DeleteFacts) { f.Events[3].Membership.EventID = "evt_other" }},
		{"membership mission", func(f *DeleteFacts) { f.Events[3].Membership.MissionID = "mis_other" }},
		{"membership run", func(f *DeleteFacts) { f.Events[3].Membership.RunID = "evt_other" }},
		{"membership role", func(f *DeleteFacts) { f.Events[3].Membership.EventRole = "completion" }},
		{"membership attempt", func(f *DeleteFacts) { f.Events[3].Membership.AttemptEventID = "evt_child" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			facts := completionFactsWithUsage()
			tc.mutate(&facts)
			assertCompletionPending(t, facts)
		})
	}
}

func retryCompletionFacts() DeleteFacts {
	root := member(deleteRunID, deleteMissionID, deleteRootID, "draft_pending", deleteRootID, ledger.Event{EventID: deleteRootID, MissionID: deleteMissionID, EventType: "report.draft.pending", Payload: []byte(`{"retry_strategy":"initial"}`)})
	failed := member(deleteRunID, deleteMissionID, "evt_root_failed", "terminal", deleteRootID, ledger.Event{EventID: "evt_root_failed", MissionID: deleteMissionID, EventType: "report.draft.failed", Payload: []byte(`{"pending_event_id":"evt_root"}`)})
	childID := "evt_child"
	child := member(deleteRunID, deleteMissionID, childID, "draft_pending", childID, ledger.Event{EventID: childID, MissionID: deleteMissionID, EventType: "report.draft.pending", Payload: []byte(`{"origin_pending_event_id":"evt_root","retry_of_pending_event_id":"evt_root","retry_strategy":"resume_failed"}`)})
	rootTargetID := "evt_root_target"
	rootTarget := member(deleteRunID, deleteMissionID, rootTargetID, "stage", deleteRootID, ledger.Event{EventID: rootTargetID, MissionID: deleteMissionID, EventType: "report.requirements.mapped", Payload: []byte(`{"pending_event_id":"evt_root","previous_provider_session_id":"ses-root","agent_executor":"codex","agent_model":"model","agent_reasoning_effort":"high"}`)})
	rootUsage := agentusage.New("", "codex", "model", "high", "").WithSurface("report_requirements").WithSession("ses-root", "ses-root", false, false).WithProviderUsage(agentusage.ProviderUsage{Scope: agentusage.UsageScopeCall, InputTokens: 1, OutputTokens: 1}, "provider")
	rootUsageID := "evt_report_usage_root_target"
	rootUsageMember := member(deleteRunID, deleteMissionID, rootUsageID, "stage", deleteRootID, ledger.Event{EventID: rootUsageID, MissionID: deleteMissionID, EventType: "report.agent_usage.recorded", Producer: ledger.Producer{Type: "agent_session", ID: "ses-root"}, CausationEventID: rootTargetID, CorrelationID: deleteRootID, Payload: usageEventPayload(deleteRootID, rootTargetID, "", rootUsage)})
	childTargetID := "evt_child_target"
	childTarget := member(deleteRunID, deleteMissionID, childTargetID, "stage", childID, ledger.Event{EventID: childTargetID, MissionID: deleteMissionID, EventType: "report.part.edited", Payload: []byte(`{"pending_event_id":"evt_child","provider_session_id":"ses-child","previous_provider_session_id":"ses-prev","agent_executor":"codex","agent_model":"model","agent_reasoning_effort":"high"}`)})
	childUsage := agentusage.New("", "codex", "model", "high", "").WithSurface("report_part_edit").WithSession("ses-prev", "ses-child", false, false).WithUnavailable("provider unavailable for retry child")
	childUsageID := "evt_report_usage_child_target"
	childUsageMember := member(deleteRunID, deleteMissionID, childUsageID, "stage", childID, ledger.Event{EventID: childUsageID, MissionID: deleteMissionID, EventType: "report.agent_usage.recorded", Producer: ledger.Producer{Type: "agent_session", ID: "ses-child"}, CausationEventID: childTargetID, CorrelationID: childID, Payload: usageEventPayload(childID, childTargetID, "", childUsage)})
	finalID := "evt_child_final"
	final := member(deleteRunID, deleteMissionID, finalID, "final", childID, ledger.Event{EventID: finalID, MissionID: deleteMissionID, EventType: "report.artifact.created", Payload: []byte(`{"pending_event_id":"evt_child","artifact_id":"art_child"}`)})
	completion := member(deleteRunID, deleteMissionID, completionDeleteEventID(deleteRunID), "completion", childID, ledger.Event{EventID: completionDeleteEventID(deleteRunID), MissionID: deleteMissionID, EventType: "report.run.completed", Producer: producerSystem(), CausationEventID: finalID, CorrelationID: deleteRunID, Payload: []byte(`{"kind":"report_run_completed","schema_version":"plasma.report_run_completion.v1","run_id":"evt_root","pending_event_id":"evt_child","canonical_event_id":"evt_child_final","artifact_id":"art_child","delayed_usage_target_count":2,"usage_recorded_count":1,"usage_unavailable_count":1}`)})
	return DeleteFacts{Run: Run{RunID: deleteRunID, RootPendingEventID: deleteRootID, MissionID: deleteMissionID, LifecycleState: LifecycleCompleted, FinalArtifactID: "art_child"}, Events: []MemberEvent{root, failed, child, rootTarget, rootUsageMember, childTarget, childUsageMember, final, completion}}
}

func TestPreviewDeleteAcceptsRetryRootAndChildTargets(t *testing.T) {
	preview := PreviewDelete(retryCompletionFacts(), "")
	if !preview.Eligible {
		t.Fatalf("valid retry completion blocked: %#v", preview.Blockers)
	}
}

func TestPreviewDeleteRejectsRetryCompletionRootAttemptMutation(t *testing.T) {
	facts := retryCompletionFacts()
	facts.Events[len(facts.Events)-1].Membership.AttemptEventID = deleteRootID
	assertCompletionPending(t, facts)
}

func TestPreviewDeleteAllowsRealisticPatchPromotionWithLaterFinalMember(t *testing.T) {
	facts := completionFactsWithUsage()
	patchPending := member(deleteRunID, deleteMissionID, "evt_patch_pending", "operation_pending", "", ledger.Event{EventID: "evt_patch_pending", MissionID: deleteMissionID, EventType: "report.patch.pending", Payload: []byte(`{"base_artifact_id":"art_final"}`)})
	patchFinal := member(deleteRunID, deleteMissionID, "evt_patch_final", "patch_finalized", "evt_patch_pending", ledger.Event{EventID: "evt_patch_final", MissionID: deleteMissionID, EventType: "report.patch.finalized", Payload: []byte(`{"pending_event_id":"evt_patch_pending","artifact_id":"art_patch"}`)})
	patchArtifact := member(deleteRunID, deleteMissionID, "evt_patch_artifact", "final", "evt_patch_pending", ledger.Event{EventID: "evt_patch_artifact", MissionID: deleteMissionID, EventType: "report.artifact.created", Payload: []byte(`{"pending_event_id":"evt_patch_pending","artifact_id":"art_patch"}`)})
	completion := facts.Events[len(facts.Events)-1]
	facts.Events = append(facts.Events[:len(facts.Events)-1], patchPending, patchFinal, patchArtifact, completion)
	facts.Run.FinalArtifactID = "art_patch"
	preview := PreviewDelete(facts, "")
	if !preview.Eligible {
		t.Fatalf("realistic patch promotion blocked: %#v", preview.Blockers)
	}
}
