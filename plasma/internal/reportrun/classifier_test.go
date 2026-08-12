package reportrun

import (
	"encoding/json"
	"fmt"
	"slices"
	"testing"
	"time"
)

func TestBuildRegistrationKeepsRetryLineageInOneRun(t *testing.T) {
	now := time.Date(2026, 8, 9, 1, 2, 3, 0, time.UTC)
	events := []Event{
		testEvent("evt_pending_1", "report.draft.pending", map[string]any{"title": "Root"}),
		testEvent("evt_plan_1", "report.plan.created", map[string]any{"pending_event_id": "evt_pending_1", "artifact_id": "art_plan_1"}),
		testEvent("evt_assembly_1", "report.final_assembly.created", map[string]any{"plan_event_id": "evt_plan_1", "artifact_id": "art_assembly_1"}),
		testEvent("evt_failed_1", "report.draft.failed", map[string]any{"pending_event_id": "evt_pending_1", "kind": "agent_error"}),
		testEvent("evt_pending_2", "report.draft.pending", map[string]any{
			"title":                     "Retry",
			"origin_pending_event_id":   "evt_pending_1",
			"retry_of_pending_event_id": "evt_pending_1",
			"retry_strategy":            "resume_failed",
		}),
		testEvent("evt_final_2", "report.artifact.created", map[string]any{
			"pending_event_id": "evt_pending_2",
			"artifact_id":      "art_final_2",
		}),
	}
	events = sequencedEvents(events)

	registration, err := BuildRegistration(events, RegistrationNative, now)
	if err != nil {
		t.Fatalf("BuildRegistration returned error: %v", err)
	}
	if len(registration.Runs) != 1 {
		t.Fatalf("expected one run, got %#v", registration.Runs)
	}
	run := registration.Runs[0]
	if run.RunID != "evt_pending_1" || run.RootPendingEventID != "evt_pending_1" {
		t.Fatalf("unexpected run identity: %#v", run)
	}
	if run.LifecycleState != LifecycleCompleted || run.FinalArtifactID != "art_final_2" {
		t.Fatalf("unexpected run state: %#v", run)
	}
	if !hasArtifactMembership(registration.Artifacts, "art_assembly_1", ArtifactRoleIntermediate, OwnershipCreated) {
		t.Fatalf("missing plan-linked assembly membership: %#v", registration.Artifacts)
	}
	if !hasArtifactMembership(registration.Artifacts, "art_final_2", ArtifactRoleFinal, OwnershipCreated) {
		t.Fatalf("missing retry final membership: %#v", registration.Artifacts)
	}
	if hasArtifactMembership(registration.Artifacts, "art_plan_1", ArtifactRoleIntermediate, OwnershipCreated) {
		t.Fatalf("plan event must not claim artifact ownership: %#v", registration.Artifacts)
	}
}

func TestBuildRegistrationTreatsSectionEvidenceGapAsNonArtifactStageData(t *testing.T) {
	now := time.Date(2026, 8, 9, 1, 2, 3, 0, time.UTC)
	registration, err := BuildRegistration(sequencedEvents([]Event{
		testEvent("evt_pending", "report.draft.pending", map[string]any{"title": "Root"}),
		testEvent("evt_plan", "report.plan.created", map[string]any{"pending_event_id": "evt_pending", "artifact_id": "art_plan"}),
		testEvent("evt_gap", "report.section.evidence_gap", map[string]any{
			"pending_event_id": "evt_pending", "plan_event_id": "evt_plan",
			"part_index": 1, "section_index": 1, "attempt_number": 1,
			"reason_code": "inadequate_section_evidence",
		}),
	}), RegistrationNative, now)
	if err != nil {
		t.Fatalf("BuildRegistration returned error: %v", err)
	}
	if len(registration.Runs) != 1 || registration.Runs[0].LifecycleState != LifecycleActive {
		t.Fatalf("unexpected run: %#v", registration.Runs)
	}
	if len(registration.Artifacts) != 0 {
		t.Fatalf("evidence gap must not claim artifacts: %#v", registration.Artifacts)
	}
	found := false
	for _, membership := range registration.Events {
		if membership.EventID == "evt_gap" && membership.EventRole == "stage" && membership.AttemptEventID == "evt_pending" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing evidence gap stage membership: %#v", registration.Events)
	}
}

func TestBuildRegistrationTreatsSectionPlanRepairAsNonArtifactStageData(t *testing.T) {
	now := time.Date(2026, 8, 9, 1, 2, 3, 0, time.UTC)
	registration, err := BuildRegistration(sequencedEvents([]Event{
		testEvent("evt_pending", "report.draft.pending", map[string]any{"title": "Root"}),
		testEvent("evt_plan", "report.plan.created", map[string]any{"pending_event_id": "evt_pending", "artifact_id": "art_plan"}),
		testEvent("evt_repair", "report.plan.section_repair.completed", map[string]any{
			"pending_event_id": "evt_pending", "plan_event_id": "evt_plan", "repair_round": 1,
			"replacements": []any{map[string]any{
				"part_index": 1, "section_index": 1,
				"section": map[string]any{"title": "Replacement", "purpose": "Explain supported facts."},
			}},
		}),
	}), RegistrationNative, now)
	if err != nil {
		t.Fatalf("BuildRegistration returned error: %v", err)
	}
	if len(registration.Runs) != 1 || registration.Runs[0].LifecycleState != LifecycleActive {
		t.Fatalf("unexpected run: %#v", registration.Runs)
	}
	if len(registration.Artifacts) != 0 {
		t.Fatalf("Section plan repair must not claim artifacts: %#v", registration.Artifacts)
	}
	found := false
	for _, membership := range registration.Events {
		if membership.EventID == "evt_repair" && membership.EventRole == "stage" && membership.AttemptEventID == "evt_pending" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing Section plan repair stage membership: %#v", registration.Events)
	}
}

func TestBuildRegistrationMarksOrphanReportArtifactAmbiguous(t *testing.T) {
	now := time.Date(2026, 8, 9, 1, 2, 3, 0, time.UTC)
	registration, err := BuildRegistration([]Event{
		testEvent("evt_orphan_final", "report.artifact.created", map[string]any{"artifact_id": "art_orphan"}),
	}, RegistrationBackfilled, now)
	if err != nil {
		t.Fatalf("BuildRegistration returned error: %v", err)
	}
	if len(registration.Runs) != 1 {
		t.Fatalf("expected one ambiguous run, got %#v", registration.Runs)
	}
	if registration.Runs[0].LifecycleState != LifecycleAmbiguous {
		t.Fatalf("expected ambiguous run, got %#v", registration.Runs[0])
	}
	if !hasArtifactMembership(registration.Artifacts, "art_orphan", ArtifactRoleFinal, OwnershipCreated) {
		t.Fatalf("missing ambiguous artifact membership: %#v", registration.Artifacts)
	}
}

func TestBuildRegistrationInvalidRetryLineageFailsClosed(t *testing.T) {
	tests := []struct {
		name         string
		events       []Event
		invalidRunID string
	}{
		{
			name: "missing parent",
			events: []Event{
				testEvent("evt_root", "report.draft.pending", map[string]any{"title": "Root"}),
				testEvent("evt_retry", "report.draft.pending", map[string]any{
					"origin_pending_event_id":   "evt_root",
					"retry_of_pending_event_id": "evt_missing",
					"retry_strategy":            "resume_failed",
				}),
			},
			invalidRunID: "evt_retry",
		},
		{
			name: "cross-root parent",
			events: []Event{
				testEvent("evt_root_a", "report.draft.pending", map[string]any{"title": "A"}),
				testEvent("evt_root_b", "report.draft.pending", map[string]any{"title": "B"}),
				testEvent("evt_root_b_failed", "report.draft.failed", map[string]any{"pending_event_id": "evt_root_b", "kind": "agent_error"}),
				testEvent("evt_retry", "report.draft.pending", map[string]any{
					"origin_pending_event_id":   "evt_root_a",
					"retry_of_pending_event_id": "evt_root_b",
					"retry_strategy":            "restart",
				}),
			},
			invalidRunID: "evt_retry",
		},
		{
			name: "cycle",
			events: []Event{
				testEvent("evt_root", "report.draft.pending", map[string]any{"title": "Root"}),
				testEvent("evt_retry_a", "report.draft.pending", map[string]any{
					"origin_pending_event_id":   "evt_root",
					"retry_of_pending_event_id": "evt_retry_b",
					"retry_strategy":            "resume_failed",
				}),
				testEvent("evt_retry_b", "report.draft.pending", map[string]any{
					"origin_pending_event_id":   "evt_root",
					"retry_of_pending_event_id": "evt_retry_a",
					"retry_strategy":            "resume_failed",
				}),
			},
			invalidRunID: "evt_retry_a",
		},
		{
			name:         "depth",
			events:       retryDepthEvents(65),
			invalidRunID: "evt_retry_64",
		},
		{
			name: "nonfailed parent",
			events: []Event{
				testEvent("evt_root", "report.draft.pending", map[string]any{"title": "Root"}),
				testEvent("evt_root_final", "report.artifact.created", map[string]any{"pending_event_id": "evt_root", "artifact_id": "art_root"}),
				testEvent("evt_retry", "report.draft.pending", map[string]any{
					"origin_pending_event_id":   "evt_root",
					"retry_of_pending_event_id": "evt_root",
					"retry_strategy":            "resume_failed",
				}),
			},
			invalidRunID: "evt_retry",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			registration, err := BuildRegistration(sequencedEvents(tc.events), RegistrationBackfilled, time.Date(2026, 8, 9, 1, 2, 3, 0, time.UTC))
			if err != nil {
				t.Fatalf("BuildRegistration returned error: %v", err)
			}
			if run := runByID(registration.Runs, tc.invalidRunID); run == nil || run.LifecycleState != LifecycleAmbiguous {
				t.Fatalf("invalid retry should be represented separately as ambiguous: %#v", registration.Runs)
			}
			if run := runByID(registration.Runs, "evt_root"); run != nil && run.LifecycleState != LifecycleAmbiguous {
				t.Fatalf("claimed root should fail closed when present: %#v", run)
			}
		})
	}
}

func TestBuildRegistrationRetryLineageUsesLedgerSequence(t *testing.T) {
	now := time.Date(2026, 8, 9, 1, 2, 3, 0, time.UTC)
	t.Run("arbitrary slice order still follows sequence", func(t *testing.T) {
		events := []Event{
			testEventSeq(3, "evt_retry", "report.draft.pending", map[string]any{
				"origin_pending_event_id":   "evt_root",
				"retry_of_pending_event_id": "evt_root",
				"retry_strategy":            "resume_failed",
			}),
			testEventSeq(4, "evt_retry_final", "report.artifact.created", map[string]any{"pending_event_id": "evt_retry", "artifact_id": "art_retry"}),
			testEventSeq(2, "evt_root_failed", "report.draft.failed", map[string]any{"pending_event_id": "evt_root", "kind": "agent_error"}),
			testEventSeq(1, "evt_root", "report.draft.pending", map[string]any{"title": "Root"}),
		}
		registration, err := BuildRegistration(events, RegistrationBackfilled, now)
		if err != nil {
			t.Fatalf("BuildRegistration returned error: %v", err)
		}
		if len(registration.Runs) != 1 || registration.Runs[0].RunID != "evt_root" || registration.Runs[0].LifecycleState != LifecycleCompleted {
			t.Fatalf("sequence-ordered retry should merge into root: %#v", registration.Runs)
		}
	})
	t.Run("retry before failed terminal is ambiguous", func(t *testing.T) {
		events := []Event{
			testEventSeq(1, "evt_root", "report.draft.pending", map[string]any{"title": "Root"}),
			testEventSeq(3, "evt_root_failed", "report.draft.failed", map[string]any{"pending_event_id": "evt_root", "kind": "agent_error"}),
			testEventSeq(2, "evt_retry", "report.draft.pending", map[string]any{
				"origin_pending_event_id":   "evt_root",
				"retry_of_pending_event_id": "evt_root",
				"retry_strategy":            "resume_failed",
			}),
		}
		registration, err := BuildRegistration(events, RegistrationBackfilled, now)
		if err != nil {
			t.Fatalf("BuildRegistration returned error: %v", err)
		}
		if run := runByID(registration.Runs, "evt_retry"); run == nil || run.LifecycleState != LifecycleAmbiguous {
			t.Fatalf("out-of-order retry should be separate ambiguous run: %#v", registration.Runs)
		}
		if run := runByID(registration.Runs, "evt_root"); run == nil || run.LifecycleState != LifecycleAmbiguous {
			t.Fatalf("claimed root should be ambiguous after out-of-order retry: %#v", registration.Runs)
		}
	})
}

func TestBuildRegistrationUnknownArtifactEventDoesNotClaimOwnership(t *testing.T) {
	for _, eventType := range []string{
		"report.future.created",
		"report.future.started",
		"report.future.pending",
		"report.future.failed",
		"report.future.skipped",
		"report.future.rejected",
	} {
		t.Run(eventType, func(t *testing.T) {
			registration, err := BuildRegistration([]Event{
				testEvent("evt_pending", "report.draft.pending", map[string]any{"title": "Root"}),
				testEvent("evt_unknown", eventType, map[string]any{
					"pending_event_id": "evt_pending",
					"artifact_id":      "art_existing",
				}),
			}, RegistrationBackfilled, time.Date(2026, 8, 9, 1, 2, 3, 0, time.UTC))
			if err != nil {
				t.Fatalf("BuildRegistration returned error: %v", err)
			}
			if run := runByID(registration.Runs, "evt_pending"); run == nil || run.LifecycleState != LifecycleAmbiguous {
				t.Fatalf("unknown artifact event should mark the run ambiguous: %#v", registration.Runs)
			}
			if !hasArtifactMembership(registration.Artifacts, "art_existing", ArtifactRoleIntermediate, OwnershipReferenced) {
				t.Fatalf("unknown artifact event should only reference the artifact: %#v", registration.Artifacts)
			}
			if hasArtifactMembership(registration.Artifacts, "art_existing", ArtifactRoleIntermediate, OwnershipCreated) {
				t.Fatalf("unknown artifact event must not claim ownership: %#v", registration.Artifacts)
			}
		})
	}
}

func TestBuildRegistrationRedpenUsesExplicitArtifactOwnership(t *testing.T) {
	tests := []struct {
		name          string
		ownership     any
		wantOwnership string
		wantAmbiguous bool
	}{
		{name: "created", ownership: OwnershipCreated, wantOwnership: OwnershipCreated},
		{name: "referenced", ownership: OwnershipReferenced, wantOwnership: OwnershipReferenced},
		{name: "missing", wantOwnership: OwnershipReferenced, wantAmbiguous: true},
		{name: "invalid", ownership: "owned", wantOwnership: OwnershipReferenced, wantAmbiguous: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			payload := map[string]any{
				"pending_event_id":   "evt_pending",
				"source_artifact_id": "art_final",
				"artifact_id":        "art_redpen",
			}
			if tc.ownership != nil {
				payload["artifact_ownership"] = tc.ownership
			}
			registration, err := BuildRegistration(sequencedEvents([]Event{
				testEvent("evt_pending", "report.draft.pending", map[string]any{"title": "Root"}),
				testEvent("evt_final", "report.artifact.created", map[string]any{
					"pending_event_id": "evt_pending",
					"artifact_id":      "art_final",
				}),
				testEvent("evt_redpen", "report.redpen.saved", payload),
			}), RegistrationBackfilled, time.Date(2026, 8, 9, 1, 2, 3, 0, time.UTC))
			if err != nil {
				t.Fatalf("BuildRegistration returned error: %v", err)
			}
			run := runByID(registration.Runs, "evt_pending")
			if run == nil {
				t.Fatalf("missing run: %#v", registration.Runs)
			}
			if tc.wantAmbiguous && run.LifecycleState != LifecycleAmbiguous {
				t.Fatalf("redpen invalid ownership should mark ambiguous: %#v", run)
			}
			if !tc.wantAmbiguous && run.LifecycleState != LifecycleCompleted {
				t.Fatalf("redpen valid ownership should keep completed run: %#v", run)
			}
			if !hasArtifactMembership(registration.Artifacts, "art_redpen", ArtifactRoleIntermediate, tc.wantOwnership) {
				t.Fatalf("redpen membership ownership mismatch: %#v", registration.Artifacts)
			}
		})
	}
}

func TestIsReportEventTypeExcludesOnlyCanvasEvents(t *testing.T) {
	for _, eventType := range []string{"report.promoted", "report.exported"} {
		if IsReportEventType(eventType) {
			t.Fatalf("%s should not be report-run lineage", eventType)
		}
	}
	for _, eventType := range []string{
		"report.artifact.created",
		"report.artifact.exported",
		"report.redpen.saved",
		"report.patch.pending",
		"report.patch.finalized",
		"report.design.pending",
		"report.humanize.pending",
		"report.humanize.skipped",
	} {
		if !IsReportEventType(eventType) {
			t.Fatalf("%s should remain report-run lineage", eventType)
		}
	}
}

func TestPreviewDeleteBlocksActiveRunWithOpenPending(t *testing.T) {
	preview := PreviewDelete(DeleteFacts{
		Run: Run{RunID: "evt_pending_1", LifecycleState: LifecycleActive, Revision: 2},
		Events: []MemberEvent{{
			Event: Event{EventID: "evt_pending_1", EventType: "report.draft.pending", Payload: jsonPayload(t, map[string]any{})},
		}},
	}, "evt_pending_1")
	if preview.Eligible {
		t.Fatalf("active run preview should not be eligible: %#v", preview)
	}
	got := blockerCodes(preview.Blockers)
	for _, want := range []string{BlockerActiveRun, BlockerInFlight, BlockerOpenPending, BlockerNotCompleted} {
		if !slices.Contains(got, want) {
			t.Fatalf("missing blocker %s in %#v", want, got)
		}
	}
}

func retryDepthEvents(count int) []Event {
	events := []Event{testEvent("evt_root", "report.draft.pending", map[string]any{"title": "Root"})}
	previous := "evt_root"
	for i := 1; i < count; i++ {
		failedID := previous + "_failed"
		events = append(events, testEvent(failedID, "report.draft.failed", map[string]any{
			"pending_event_id": previous,
			"kind":             "agent_error",
		}))
		id := fmt.Sprintf("evt_retry_%02d", i)
		events = append(events, testEvent(id, "report.draft.pending", map[string]any{
			"origin_pending_event_id":   "evt_root",
			"retry_of_pending_event_id": previous,
			"retry_strategy":            "resume_failed",
		}))
		previous = id
	}
	return events
}

func sequencedEvents(events []Event) []Event {
	out := append([]Event(nil), events...)
	for i := range out {
		out[i].Sequence = int64(i + 1)
	}
	return out
}

func testEvent(id string, eventType string, payload map[string]any) Event {
	return testEventSeq(0, id, eventType, payload)
}

func testEventSeq(sequence int64, id string, eventType string, payload map[string]any) Event {
	return Event{
		EventID:   id,
		MissionID: "mis_report",
		Sequence:  sequence,
		EventType: eventType,
		Payload:   jsonPayload(nil, payload),
		CreatedAt: time.Date(2026, 8, 9, 1, 2, 3, 0, time.UTC),
	}
}

func jsonPayload(t *testing.T, payload map[string]any) []byte {
	out, err := json.Marshal(payload)
	if err != nil {
		if t != nil {
			t.Fatalf("marshal payload: %v", err)
		}
		panic(err)
	}
	return out
}

func runByID(runs []Run, runID string) *Run {
	for i := range runs {
		if runs[i].RunID == runID {
			return &runs[i]
		}
	}
	return nil
}

func hasArtifactMembership(items []ArtifactMembership, artifactID string, role string, ownership string) bool {
	for _, item := range items {
		if item.ArtifactID == artifactID && item.ArtifactRole == role && item.Ownership == ownership {
			return true
		}
	}
	return false
}

func blockerCodes(items []Blocker) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.ReasonCode)
	}
	return out
}
