package app

import (
	"testing"
)

func TestBuildProjectionAppliesUserSteering(t *testing.T) {
	projection, err := BuildProjection("mis_1", []LedgerEvent{
		{
			EventID:   "evt_1",
			MissionID: "mis_1",
			Sequence:  1,
			EventType: "mission.created",
			Producer:  Producer{Type: "user", ID: "ses_1"},
			Payload:   []byte(`{"title":"Initial","objective":"Draft","scope":{"included":["A"]}}`),
		},
		{
			EventID:   "evt_2",
			MissionID: "mis_1",
			Sequence:  2,
			EventType: "mission.steered",
			Producer:  Producer{Type: "user", ID: "ses_1"},
			Payload:   []byte(`{"objective":"Updated","scope":{"excluded":["B"]}}`),
		},
	})
	if err != nil {
		t.Fatalf("BuildProjection returned error: %v", err)
	}
	if projection.Objective != "Updated" {
		t.Fatalf("unexpected objective: %q", projection.Objective)
	}
	if len(projection.Scope.Excluded) != 1 || projection.Scope.Excluded[0] != "B" {
		t.Fatalf("unexpected scope: %#v", projection.Scope)
	}
	if projection.NeedsReview {
		t.Fatalf("did not expect review flag: %#v", projection.NeedsReviewReasons)
	}
}

func TestBuildProjectionRejectsAutopilotSteeringWithoutApproval(t *testing.T) {
	projection, err := BuildProjection("mis_1", []LedgerEvent{
		{
			EventID:   "evt_1",
			MissionID: "mis_1",
			Sequence:  1,
			EventType: "mission.created",
			Producer:  Producer{Type: "user", ID: "ses_1"},
			Payload:   []byte(`{"objective":"Original"}`),
		},
		{
			EventID:   "evt_2",
			MissionID: "mis_1",
			Sequence:  2,
			EventType: "mission.steered",
			Producer:  Producer{Type: "autopilot", ID: "ses_2"},
			Payload:   []byte(`{"objective":"Hidden mutation"}`),
		},
	})
	if err != nil {
		t.Fatalf("BuildProjection returned error: %v", err)
	}
	if projection.Objective != "Original" {
		t.Fatalf("autopilot steering mutated objective: %q", projection.Objective)
	}
	if !projection.NeedsReview {
		t.Fatal("expected needs_review for autopilot steering")
	}
}

func TestBuildProjectionAppliesApprovedObjects(t *testing.T) {
	projection, err := BuildProjection("mis_1", []LedgerEvent{
		{
			EventID:   "evt_1",
			MissionID: "mis_1",
			Sequence:  1,
			EventType: "session.attached",
			Producer:  Producer{Type: "user", ID: "ses_1"},
			Payload:   []byte(`{"session_id":"ses_1"}`),
		},
		{
			EventID:   "evt_2",
			MissionID: "mis_1",
			Sequence:  2,
			EventType: "claim.approved",
			Producer:  Producer{Type: "user", ID: "ses_1"},
			Payload:   []byte(`{"claim_id":"clm_1"}`),
		},
		{
			EventID:   "evt_3",
			MissionID: "mis_1",
			Sequence:  3,
			EventType: "question.proposed",
			Producer:  Producer{Type: "autopilot", ID: "ses_2"},
			Payload:   []byte(`{"question_id":"qst_1"}`),
		},
		{
			EventID:   "evt_4",
			MissionID: "mis_1",
			Sequence:  4,
			EventType: "report.promoted",
			Producer:  Producer{Type: "user", ID: "ses_1"},
			Payload:   []byte(`{"report_version_id":"rvn_1"}`),
		},
	})
	if err != nil {
		t.Fatalf("BuildProjection returned error: %v", err)
	}
	if projection.ActiveSessionIDs[0] != "ses_1" {
		t.Fatalf("unexpected sessions: %#v", projection.ActiveSessionIDs)
	}
	if projection.AcceptedClaimIDs[0] != "clm_1" {
		t.Fatalf("unexpected claims: %#v", projection.AcceptedClaimIDs)
	}
	if projection.OpenQuestionIDs[0] != "qst_1" {
		t.Fatalf("unexpected questions: %#v", projection.OpenQuestionIDs)
	}
	if projection.ActiveReportVersionID != "rvn_1" {
		t.Fatalf("unexpected report version: %q", projection.ActiveReportVersionID)
	}
}

func TestBuildProjectionRejectsUnapprovedAcceptedTransitions(t *testing.T) {
	projection, err := BuildProjection("mis_1", []LedgerEvent{
		{
			EventID:   "evt_1",
			MissionID: "mis_1",
			Sequence:  1,
			EventType: "question.proposed",
			Producer:  Producer{Type: "autopilot", ID: "ses_2"},
			Payload:   []byte(`{"question_id":"qst_1"}`),
		},
		{
			EventID:   "evt_2",
			MissionID: "mis_1",
			Sequence:  2,
			EventType: "claim.approved",
			Producer:  Producer{Type: "autopilot", ID: "ses_2"},
			Payload:   []byte(`{"claim_id":"clm_1"}`),
		},
		{
			EventID:   "evt_3",
			MissionID: "mis_1",
			Sequence:  3,
			EventType: "question.answered",
			Producer:  Producer{Type: "autopilot", ID: "ses_2"},
			Payload:   []byte(`{"question_id":"qst_1"}`),
		},
		{
			EventID:   "evt_4",
			MissionID: "mis_1",
			Sequence:  4,
			EventType: "report.promoted",
			Producer:  Producer{Type: "system", ID: "worker_1"},
			Payload:   []byte(`{"report_version_id":"rvn_1"}`),
		},
	})
	if err != nil {
		t.Fatalf("BuildProjection returned error: %v", err)
	}
	if len(projection.AcceptedClaimIDs) != 0 {
		t.Fatalf("unapproved claim mutated projection: %#v", projection.AcceptedClaimIDs)
	}
	if len(projection.OpenQuestionIDs) != 1 || projection.OpenQuestionIDs[0] != "qst_1" {
		t.Fatalf("unapproved question answer mutated projection: %#v", projection.OpenQuestionIDs)
	}
	if projection.ActiveReportVersionID != "" {
		t.Fatalf("unapproved report promotion mutated projection: %q", projection.ActiveReportVersionID)
	}
	if !projection.NeedsReview {
		t.Fatal("expected needs_review for unapproved accepted-state transitions")
	}
}

func TestBuildProjectionMarksMalformedProjectionPayloads(t *testing.T) {
	projection, err := BuildProjection("mis_1", []LedgerEvent{
		{
			EventID:   "evt_1",
			MissionID: "mis_1",
			Sequence:  1,
			EventType: "claim.approved",
			Producer:  Producer{Type: "user", ID: "ses_1"},
			Payload:   []byte(`{"claim_id":`),
		},
		{
			EventID:   "evt_2",
			MissionID: "mis_1",
			Sequence:  2,
			EventType: "report.promoted",
			Producer:  Producer{Type: "user", ID: "ses_1"},
			Payload:   []byte(`{}`),
		},
	})
	if err != nil {
		t.Fatalf("BuildProjection returned error: %v", err)
	}
	if len(projection.AcceptedClaimIDs) != 0 || projection.ActiveReportVersionID != "" {
		t.Fatalf("malformed payload mutated projection: %#v", projection)
	}
	if !projection.NeedsReview {
		t.Fatal("expected needs_review for malformed payloads")
	}
}

func TestBuildProjectionMarksConflictingSteering(t *testing.T) {
	projection, err := BuildProjection("mis_1", []LedgerEvent{
		{
			EventID:   "evt_1",
			MissionID: "mis_1",
			Sequence:  1,
			EventType: "mission.created",
			Producer:  Producer{Type: "user", ID: "ses_1"},
			Payload:   []byte(`{"objective":"Original","scope":{"included":["A"]}}`),
		},
		{
			EventID:   "evt_2",
			MissionID: "mis_1",
			Sequence:  2,
			EventType: "mission.steered",
			Producer:  Producer{Type: "user", ID: "ses_2"},
			Payload:   []byte(`{"objective":"Conflicting","scope":{"included":["B"]}}`),
		},
	})
	if err != nil {
		t.Fatalf("BuildProjection returned error: %v", err)
	}
	if projection.Objective != "Original" {
		t.Fatalf("conflicting steering mutated objective: %q", projection.Objective)
	}
	if len(projection.Scope.Included) != 1 || projection.Scope.Included[0] != "A" {
		t.Fatalf("conflicting steering mutated scope: %#v", projection.Scope)
	}
	if !projection.NeedsReview {
		t.Fatal("expected needs_review for conflicting steering")
	}
}
