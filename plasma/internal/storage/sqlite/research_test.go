package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestResearchRecordsRoundTrip(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	svc := newResearchTestService(t, store)
	createResearchSource(t, ctx, svc)

	evidence, err := svc.CreateEvidenceRecord(ctx, app.CreateEvidenceRecordRequest{
		EvidenceID:   "evd_1",
		MissionID:    "mis_1",
		Summary:      "Pinned source quote.",
		EvidenceType: "quote",
		SnapshotRefs: []app.SnapshotRef{{
			SnapshotID: "src_1",
			ArtifactID: "art_1",
			Locator:    []byte(`{"locator_type":"text_quote","exact":"snapshot"}`),
		}},
		Confidence:     app.Confidence{Level: "medium", Rationale: "Source snapshot is pinned."},
		Producer:       app.Producer{Type: "autopilot", ID: "ses_auto"},
		CreatedEventID: "evt_evidence",
	})
	if err != nil {
		t.Fatalf("CreateEvidenceRecord returned error: %v", err)
	}
	gotEvidence, err := svc.GetEvidenceRecord(ctx, evidence.EvidenceID)
	if err != nil {
		t.Fatalf("GetEvidenceRecord returned error: %v", err)
	}
	if gotEvidence.SchemaVersion != app.EvidenceRecordSchemaVersion || len(gotEvidence.SnapshotRefs) != 1 {
		t.Fatalf("unexpected evidence round trip: %#v", gotEvidence)
	}

	claim, err := svc.CreateClaimRecord(ctx, app.CreateClaimRecordRequest{
		ClaimID:               "clm_1",
		MissionID:             "mis_1",
		Text:                  "Research records must point at pinned evidence.",
		ClaimType:             "decision",
		SupportingEvidenceIDs: []string{evidence.EvidenceID},
		Confidence:            app.Confidence{Level: "high"},
		CreatedEventID:        "evt_claim",
	})
	if err != nil {
		t.Fatalf("CreateClaimRecord returned error: %v", err)
	}
	gotClaim, err := svc.GetClaimRecord(ctx, claim.ClaimID)
	if err != nil {
		t.Fatalf("GetClaimRecord returned error: %v", err)
	}
	if gotClaim.State != "proposed" || gotClaim.Approval.State != "pending" {
		t.Fatalf("unexpected claim round trip: %#v", gotClaim)
	}

	question, err := svc.CreateQuestionRecord(ctx, app.CreateQuestionRecordRequest{
		QuestionID:         "qst_1",
		MissionID:          "mis_1",
		Text:               "Should this claim enter the accepted projection?",
		Priority:           "high",
		Blocking:           true,
		RelatedEvidenceIDs: []string{evidence.EvidenceID},
		RelatedClaimIDs:    []string{claim.ClaimID},
		CreatedEventID:     "evt_question",
	})
	if err != nil {
		t.Fatalf("CreateQuestionRecord returned error: %v", err)
	}
	gotQuestion, err := svc.GetQuestionRecord(ctx, question.QuestionID)
	if err != nil {
		t.Fatalf("GetQuestionRecord returned error: %v", err)
	}
	if !gotQuestion.Blocking || gotQuestion.Priority != "high" {
		t.Fatalf("unexpected question round trip: %#v", gotQuestion)
	}

	option, err := svc.CreateOptionRecord(ctx, app.CreateOptionRecordRequest{
		OptionID:           "opt_1",
		MissionID:          "mis_1",
		Title:              "Approve as working conclusion",
		Description:        "Accept the claim after user review.",
		Pros:               []string{"Unblocks report generation."},
		Cons:               []string{"Requires approval bookkeeping."},
		SupportingClaimIDs: []string{claim.ClaimID},
		RiskLevel:          "medium",
		CreatedEventID:     "evt_option",
	})
	if err != nil {
		t.Fatalf("CreateOptionRecord returned error: %v", err)
	}
	gotOption, err := svc.GetOptionRecord(ctx, option.OptionID)
	if err != nil {
		t.Fatalf("GetOptionRecord returned error: %v", err)
	}
	if len(gotOption.Pros) != 1 || gotOption.SupportingClaimIDs[0] != claim.ClaimID {
		t.Fatalf("unexpected option round trip: %#v", gotOption)
	}

	proposal, err := svc.CreateProposalBundle(ctx, app.CreateProposalBundleRequest{
		ProposalID: "prp_1",
		MissionID:  "mis_1",
		Title:      "Review claim package",
		ObjectRefs: []app.ObjectRef{
			{ObjectKind: app.EvidenceRecordObjectKind, ObjectID: evidence.EvidenceID},
			{ObjectKind: app.ClaimRecordObjectKind, ObjectID: claim.ClaimID},
			{ObjectKind: app.QuestionRecordObjectKind, ObjectID: question.QuestionID},
			{ObjectKind: app.OptionRecordObjectKind, ObjectID: option.OptionID},
		},
		RequestedDecision: "approve",
		CreatedEventID:    "evt_proposal",
	})
	if err != nil {
		t.Fatalf("CreateProposalBundle returned error: %v", err)
	}
	approved, err := svc.UpdateProposalBundleState(ctx, app.UpdateProposalBundleStateRequest{
		ProposalID:      proposal.ProposalID,
		State:           "approved",
		DecisionEventID: "evt_approval",
	})
	if err != nil {
		t.Fatalf("UpdateProposalBundleState returned error: %v", err)
	}
	if approved.State != "approved" || approved.DecisionEventID != "evt_approval" {
		t.Fatalf("unexpected proposal state: %#v", approved)
	}
}

func TestRejectedResearchRecordsRemainQueryable(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	svc := newResearchTestService(t, store)

	rejected := app.EvidenceRecord{
		SchemaVersion:  app.EvidenceRecordSchemaVersion,
		ObjectKind:     app.EvidenceRecordObjectKind,
		EvidenceID:     "evd_rejected",
		MissionID:      "mis_1",
		State:          "rejected",
		Summary:        "User rejected this assertion but it stays auditable.",
		EvidenceType:   "user_assertion",
		SnapshotRefs:   []app.SnapshotRef{},
		Confidence:     app.Confidence{Level: "unknown"},
		Producer:       app.Producer{Type: "user", ID: "ses_user"},
		CreatedEventID: "evt_user",
		CreatedAt:      time.Now().UTC(),
	}
	if err := store.CreateEvidenceRecord(ctx, rejected); err != nil {
		t.Fatalf("CreateEvidenceRecord store returned error: %v", err)
	}
	got, err := svc.GetEvidenceRecord(ctx, rejected.EvidenceID)
	if err != nil {
		t.Fatalf("GetEvidenceRecord returned error: %v", err)
	}
	if got.State != "rejected" || got.Summary == "" {
		t.Fatalf("rejected evidence was not preserved: %#v", got)
	}
}

func newResearchTestService(t *testing.T, store *Store) *app.Service {
	t.Helper()
	svc := app.NewService(store)
	if _, err := svc.CreateMission(context.Background(), app.CreateMissionRequest{MissionID: "mis_1", Title: "Research Mission"}); err != nil {
		t.Fatalf("CreateMission returned error: %v", err)
	}
	for _, event := range []struct {
		id        string
		eventType string
		producer  app.Producer
		payload   []byte
	}{
		{id: "evt_user", eventType: "mission.steered", producer: app.Producer{Type: "user", ID: "ses_user"}},
		{id: "evt_evidence", eventType: "evidence.proposed", producer: app.Producer{Type: "autopilot", ID: "ses_auto"}, payload: []byte(`{"evidence_id":"evd_1","proposal_id":"prp_1"}`)},
		{id: "evt_claim", eventType: "claim.proposed", producer: app.Producer{Type: "autopilot", ID: "ses_auto"}, payload: []byte(`{"claim_id":"clm_1","proposal_id":"prp_1"}`)},
		{id: "evt_question", eventType: "question.proposed", producer: app.Producer{Type: "autopilot", ID: "ses_auto"}},
		{id: "evt_option", eventType: "option.proposed", producer: app.Producer{Type: "autopilot", ID: "ses_auto"}},
		{id: "evt_proposal", eventType: "proposal.submitted", producer: app.Producer{Type: "autopilot", ID: "ses_auto"}, payload: []byte(`{"proposal_id":"prp_1"}`)},
		{id: "evt_approval", eventType: "proposal.approved", producer: app.Producer{Type: "user", ID: "ses_user"}, payload: []byte(`{"proposal_id":"prp_1","approved_object_ids":["evd_1","clm_1","qst_1","opt_1"],"rejected_object_ids":[]}`)},
	} {
		if _, err := svc.AppendEvent(context.Background(), app.AppendEventRequest{
			EventID:   event.id,
			MissionID: "mis_1",
			EventType: event.eventType,
			Producer:  event.producer,
			Payload:   event.payload,
		}); err != nil {
			t.Fatalf("AppendEvent %s returned error: %v", event.id, err)
		}
	}
	return svc
}

func createResearchSource(t *testing.T, ctx context.Context, svc *app.Service) {
	t.Helper()
	artifact, err := svc.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_1",
		MissionID:  "mis_1",
		MediaType:  "text/plain",
		Producer:   app.Producer{Type: "connector", ID: "liquid2"},
		Content:    []byte("snapshot body"),
	})
	if err != nil {
		t.Fatalf("CreateRawArtifact returned error: %v", err)
	}
	if _, err := svc.CreateSourceSnapshot(ctx, app.CreateSourceSnapshotRequest{
		SnapshotID:  "src_1",
		MissionID:   "mis_1",
		Connector:   app.ConnectorRef{ConnectorID: "liquid2", ConnectorType: "liquid2", ExternalSourceID: "doc_1"},
		ArtifactIDs: []string{artifact.ArtifactID},
		ContentHash: app.ContentHash{Algorithm: "sha256", Value: artifact.SHA256},
	}); err != nil {
		t.Fatalf("CreateSourceSnapshot returned error: %v", err)
	}
}
