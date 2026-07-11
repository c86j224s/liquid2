package sqlite

import (
	"context"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func TestCreateEvidenceProposalCommitsAtomically(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	svc := newResearchTestService(t, store)
	createResearchSource(t, ctx, svc)

	result, err := svc.CreateEvidenceProposal(ctx, app.CreateEvidenceProposalRequest{
		EvidenceEvent: app.AppendEventRequest{
			EventID:   "evt_atomic_evidence",
			MissionID: "mis_1",
			EventType: "evidence.proposed",
			Producer:  app.Producer{Type: "agent_session", ID: "ses_1"},
			Payload:   []byte(`{"evidence_id":"evd_atomic","proposal_id":"prp_atomic"}`),
		},
		Evidence: app.CreateEvidenceRecordRequest{
			EvidenceID:   "evd_atomic",
			MissionID:    "mis_1",
			State:        "proposed",
			Summary:      "Atomic evidence.",
			EvidenceType: "quote",
			SnapshotRefs: []app.SnapshotRef{{
				SnapshotID: "src_1",
				ArtifactID: "art_1",
			}},
			Producer:       app.Producer{Type: "agent_session", ID: "ses_1"},
			CreatedEventID: "evt_atomic_evidence",
		},
		ProposalEvent: app.AppendEventRequest{
			EventID:   "evt_atomic_proposal",
			MissionID: "mis_1",
			EventType: "proposal.submitted",
			Producer:  app.Producer{Type: "agent_session", ID: "ses_1"},
			Payload:   []byte(`{"proposal_id":"prp_atomic"}`),
		},
		Proposal: app.CreateProposalBundleRequest{
			ProposalID:        "prp_atomic",
			MissionID:         "mis_1",
			State:             "pending_review",
			Title:             "Atomic proposal",
			ObjectRefs:        []app.ObjectRef{{ObjectKind: app.EvidenceRecordObjectKind, ObjectID: "evd_atomic"}},
			RequestedDecision: "approve",
			CreatedEventID:    "evt_atomic_proposal",
		},
	})
	if err != nil {
		t.Fatalf("CreateEvidenceProposal returned error: %v", err)
	}
	if result.EvidenceEvent.Sequence == 0 || result.ProposalEvent.Sequence != result.EvidenceEvent.Sequence+1 {
		t.Fatalf("unexpected event sequences: %#v %#v", result.EvidenceEvent, result.ProposalEvent)
	}
	if _, err := svc.GetEvidenceRecord(ctx, "evd_atomic"); err != nil {
		t.Fatalf("GetEvidenceRecord returned error: %v", err)
	}
	if proposal, err := svc.GetProposalBundle(ctx, "prp_atomic"); err != nil || proposal.State != "pending_review" {
		t.Fatalf("unexpected proposal bundle: %#v err=%v", proposal, err)
	}
}

func TestCreateEvidenceProposalRollsBackWhenBundleInsertFails(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	svc := newResearchTestService(t, store)
	createResearchSource(t, ctx, svc)
	if _, err := svc.CreateEvidenceRecord(ctx, app.CreateEvidenceRecordRequest{
		EvidenceID:   "evd_1",
		MissionID:    "mis_1",
		Summary:      "Existing evidence.",
		EvidenceType: "quote",
		SnapshotRefs: []app.SnapshotRef{{
			SnapshotID: "src_1",
			ArtifactID: "art_1",
			Locator:    []byte(`{"locator_type":"text_quote"}`),
		}},
		Producer:       app.Producer{Type: "autopilot", ID: "ses_auto"},
		CreatedEventID: "evt_evidence",
	}); err != nil {
		t.Fatalf("CreateEvidenceRecord returned error: %v", err)
	}
	if _, err := svc.CreateProposalBundle(ctx, app.CreateProposalBundleRequest{
		ProposalID:        "prp_1",
		MissionID:         "mis_1",
		Title:             "Existing proposal",
		ObjectRefs:        []app.ObjectRef{{ObjectKind: app.EvidenceRecordObjectKind, ObjectID: "evd_1"}},
		RequestedDecision: "approve",
		CreatedEventID:    "evt_proposal",
	}); err != nil {
		t.Fatalf("CreateProposalBundle returned error: %v", err)
	}

	_, err := svc.CreateEvidenceProposal(ctx, app.CreateEvidenceProposalRequest{
		EvidenceEvent: app.AppendEventRequest{
			EventID:   "evt_atomic_evidence",
			MissionID: "mis_1",
			EventType: "evidence.proposed",
			Producer:  app.Producer{Type: "agent_session", ID: "ses_1"},
			Payload:   []byte(`{"evidence_id":"evd_atomic","proposal_id":"prp_1"}`),
		},
		Evidence: app.CreateEvidenceRecordRequest{
			EvidenceID:   "evd_atomic",
			MissionID:    "mis_1",
			State:        "proposed",
			Summary:      "This write should roll back.",
			EvidenceType: "quote",
			SnapshotRefs: []app.SnapshotRef{{
				SnapshotID: "src_1",
				ArtifactID: "art_1",
			}},
			Producer:       app.Producer{Type: "agent_session", ID: "ses_1"},
			CreatedEventID: "evt_atomic_evidence",
		},
		ProposalEvent: app.AppendEventRequest{
			EventID:   "evt_atomic_proposal",
			MissionID: "mis_1",
			EventType: "proposal.submitted",
			Producer:  app.Producer{Type: "agent_session", ID: "ses_1"},
			Payload:   []byte(`{"proposal_id":"prp_1"}`),
		},
		Proposal: app.CreateProposalBundleRequest{
			ProposalID:        "prp_1",
			MissionID:         "mis_1",
			State:             "pending_review",
			Title:             "Duplicate proposal",
			ObjectRefs:        []app.ObjectRef{{ObjectKind: app.EvidenceRecordObjectKind, ObjectID: "evd_atomic"}},
			RequestedDecision: "approve",
			CreatedEventID:    "evt_atomic_proposal",
		},
	})
	if err == nil {
		t.Fatalf("expected duplicate proposal failure")
	}
	assertLedgerEventMissing(t, svc, "evt_atomic_evidence")
	assertLedgerEventMissing(t, svc, "evt_atomic_proposal")
	if _, err := svc.GetEvidenceRecord(ctx, "evd_atomic"); err == nil {
		t.Fatalf("atomic evidence record was not rolled back")
	}
}

func TestCreateQuestionProposalRollsBackWhenBundleInsertFails(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	svc := newResearchTestService(t, store)
	createResearchSource(t, ctx, svc)
	if _, err := svc.CreateEvidenceRecord(ctx, app.CreateEvidenceRecordRequest{
		EvidenceID:   "evd_1",
		MissionID:    "mis_1",
		Summary:      "Existing evidence.",
		EvidenceType: "quote",
		SnapshotRefs: []app.SnapshotRef{{
			SnapshotID: "src_1",
			ArtifactID: "art_1",
			Locator:    []byte(`{"locator_type":"text_quote"}`),
		}},
		Producer:       app.Producer{Type: "autopilot", ID: "ses_auto"},
		CreatedEventID: "evt_evidence",
	}); err != nil {
		t.Fatalf("CreateEvidenceRecord returned error: %v", err)
	}
	if _, err := svc.CreateProposalBundle(ctx, app.CreateProposalBundleRequest{
		ProposalID:        "prp_1",
		MissionID:         "mis_1",
		Title:             "Existing proposal",
		ObjectRefs:        []app.ObjectRef{{ObjectKind: app.EvidenceRecordObjectKind, ObjectID: "evd_1"}},
		RequestedDecision: "approve",
		CreatedEventID:    "evt_proposal",
	}); err != nil {
		t.Fatalf("CreateProposalBundle returned error: %v", err)
	}

	_, err := svc.CreateQuestionProposal(ctx, app.CreateQuestionProposalRequest{
		QuestionEvent: app.AppendEventRequest{
			EventID:   "evt_atomic_question",
			MissionID: "mis_1",
			EventType: "question.proposed",
			Producer:  app.Producer{Type: "agent_session", ID: "ses_1"},
			Payload:   []byte(`{"question_id":"qst_atomic","proposal_id":"prp_1"}`),
		},
		Question: app.CreateQuestionRecordRequest{
			QuestionID:     "qst_atomic",
			MissionID:      "mis_1",
			State:          "open",
			Text:           "This question should roll back.",
			Priority:       "medium",
			CreatedEventID: "evt_atomic_question",
		},
		ProposalEvent: app.AppendEventRequest{
			EventID:   "evt_atomic_question_proposal",
			MissionID: "mis_1",
			EventType: "proposal.submitted",
			Producer:  app.Producer{Type: "agent_session", ID: "ses_1"},
			Payload:   []byte(`{"proposal_id":"prp_1"}`),
		},
		Proposal: app.CreateProposalBundleRequest{
			ProposalID:        "prp_1",
			MissionID:         "mis_1",
			State:             "pending_review",
			Title:             "Duplicate proposal",
			ObjectRefs:        []app.ObjectRef{{ObjectKind: app.QuestionRecordObjectKind, ObjectID: "qst_atomic"}},
			RequestedDecision: "approve",
			CreatedEventID:    "evt_atomic_question_proposal",
		},
	})
	if err == nil {
		t.Fatalf("expected duplicate proposal failure")
	}
	assertLedgerEventMissing(t, svc, "evt_atomic_question")
	assertLedgerEventMissing(t, svc, "evt_atomic_question_proposal")
	if _, err := svc.GetQuestionRecord(ctx, "qst_atomic"); err == nil {
		t.Fatalf("atomic question record was not rolled back")
	}
}

func TestSnapshotLiquid2SourceWithEventRollsBackWhenSnapshotInsertFails(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	svc := newResearchTestService(t, store)
	createResearchSource(t, ctx, svc)

	_, err := svc.SnapshotLiquid2SourceWithEvent(ctx, atomicFakeLiquid2Connector{}, app.SnapshotLiquid2SourceWithEventRequest{
		Snapshot: app.SnapshotLiquid2SourceRequest{
			MissionID:        "mis_1",
			ArtifactID:       "art_atomic",
			SnapshotID:       "src_1",
			ExternalSourceID: "doc_atomic",
			Reason:           "duplicate snapshot should fail",
		},
		EventID:  "evt_atomic_snapshot",
		Producer: app.Producer{Type: "agent_session", ID: "ses_1"},
	})
	if err == nil {
		t.Fatalf("expected duplicate snapshot failure")
	}
	assertLedgerEventMissing(t, svc, "evt_atomic_snapshot")
	if _, err := svc.GetRawArtifact(ctx, "art_atomic"); err == nil {
		t.Fatalf("atomic raw artifact was not rolled back")
	}
}

func TestSnapshotLiquid2SourceWithEventCommitsAtomically(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	svc := newResearchTestService(t, store)

	result, err := svc.SnapshotLiquid2SourceWithEvent(ctx, atomicFakeLiquid2Connector{}, app.SnapshotLiquid2SourceWithEventRequest{
		Snapshot: app.SnapshotLiquid2SourceRequest{
			MissionID:        "mis_1",
			ArtifactID:       "art_atomic",
			SnapshotID:       "src_atomic",
			ExternalSourceID: "doc_atomic",
			Reason:           "atomic snapshot",
		},
		EventID:  "evt_atomic_snapshot",
		Producer: app.Producer{Type: "agent_session", ID: "ses_1"},
	})
	if err != nil {
		t.Fatalf("SnapshotLiquid2SourceWithEvent returned error: %v", err)
	}
	if result.Event.Sequence == 0 || result.Snapshot.SnapshotID != "src_atomic" || result.Artifact.ArtifactID != "art_atomic" {
		t.Fatalf("unexpected atomic snapshot result: %#v", result)
	}
	if _, err := svc.GetRawArtifact(ctx, "art_atomic"); err != nil {
		t.Fatalf("GetRawArtifact returned error: %v", err)
	}
	if _, err := svc.GetSourceSnapshot(ctx, "src_atomic"); err != nil {
		t.Fatalf("GetSourceSnapshot returned error: %v", err)
	}
}

func assertLedgerEventMissing(t *testing.T, svc *app.Service, eventID string) {
	t.Helper()
	events, err := svc.ListEvents(context.Background(), "mis_1")
	if err != nil {
		t.Fatalf("ListEvents returned error: %v", err)
	}
	for _, event := range events {
		if event.EventID == eventID {
			t.Fatalf("event %s was not rolled back", eventID)
		}
	}
}

type atomicFakeLiquid2Connector struct{}

func (atomicFakeLiquid2Connector) SearchLiquid2Sources(
	context.Context,
	app.Liquid2SourceSearchRequest,
) (app.Liquid2SourceSearchResult, error) {
	return app.Liquid2SourceSearchResult{}, nil
}

func (atomicFakeLiquid2Connector) ReadLiquid2Source(
	context.Context,
	app.Liquid2SourceReadRequest,
) (app.Liquid2SourceDocument, error) {
	return app.Liquid2SourceDocument{
		Connector: app.ConnectorRef{ExternalSourceID: "doc_atomic"},
		Title:     "Atomic source",
		Contents: []app.Liquid2SourceContent{{
			ContentID: "content_1",
			Role:      "extracted",
			Format:    "text",
			Content:   "atomic source body",
		}},
	}, nil
}
