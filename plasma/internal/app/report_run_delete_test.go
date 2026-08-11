package app

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/reportrun"
)

func TestPreviewReportDeleteBackfillsAndReturnsFacts(t *testing.T) {
	store := &reportRunDeleteStore{
		events: []LedgerEvent{
			{EventID: "evt_report_pending", MissionID: "mis_1", EventType: "report.draft.pending", Payload: []byte(`{"title":"Report"}`)},
			{EventID: "evt_report_final", MissionID: "mis_1", EventType: "report.artifact.created", Payload: []byte(`{"pending_event_id":"evt_report_pending","artifact_id":"art_1"}`)},
		},
		facts: completedReportDeleteFacts(),
	}
	svc := NewService(store)

	preview, err := svc.PreviewReportDelete(context.Background(), ReportDeletePreviewRequest{
		MissionID: "mis_1", ArtifactID: "art_1",
	})
	if err != nil {
		t.Fatalf("PreviewReportDelete returned error: %v", err)
	}
	if !preview.Eligible || preview.RunID != "evt_report_pending" {
		t.Fatalf("unexpected preview: %#v", preview)
	}
	if !store.backfilled || len(store.registration.Runs) != 1 ||
		store.registration.Runs[0].RegistrationStatus != reportrun.RegistrationBackfilled {
		t.Fatalf("backfill was not applied: %#v", store.registration)
	}
}

func TestDeleteReportRequiresLiteralConfirmationAndUserProducer(t *testing.T) {
	svc := NewService(&reportRunDeleteStore{facts: completedReportDeleteFacts()})
	_, err := svc.DeleteReport(context.Background(), ReportDeleteRequest{
		MissionID: "mis_1", ArtifactID: "art_1", ConfirmArtifactID: "art_other",
		ExpectedRevision: 3, DeleteFactsHash: "hash", Producer: Producer{Type: "user", ID: "test"},
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid confirmation error, got %v", err)
	}
	_, err = svc.DeleteReport(context.Background(), ReportDeleteRequest{
		MissionID: "mis_1", ArtifactID: "art_1", ConfirmArtifactID: "art_1",
		ExpectedRevision: 3, DeleteFactsHash: "hash", Producer: Producer{Type: "agent", ID: "test"},
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected non-user producer error, got %v", err)
	}
	_, err = svc.DeleteReport(context.Background(), ReportDeleteRequest{
		MissionID: "mis_1", ArtifactID: "art_1", ConfirmArtifactID: "art_1",
		ExpectedRevision: 3, Producer: Producer{Type: "user", ID: "test"},
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected missing delete facts hash error, got %v", err)
	}
}

func TestDeleteReportDelegatesExpectedRevisionAndReturnsDeletedResult(t *testing.T) {
	store := &reportRunDeleteStore{facts: completedReportDeleteFacts()}
	svc := NewService(store)

	result, err := svc.DeleteReport(context.Background(), ReportDeleteRequest{
		MissionID: "mis_1", ArtifactID: "art_1", ConfirmArtifactID: "art_1",
		ExpectedRevision: 3, DeleteFactsHash: "expected-hash", Producer: Producer{Type: "user", ID: "test"},
	})
	if err != nil {
		t.Fatalf("DeleteReport returned error: %v", err)
	}
	if !result.Deleted || result.RunID != "evt_report_pending" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if !store.deleted || store.expectedRevision != 3 || store.expectedDeleteFactsHash != "expected-hash" {
		t.Fatalf("delete store was not called correctly: deleted=%v revision=%d hash=%q", store.deleted, store.expectedRevision, store.expectedDeleteFactsHash)
	}
}

func TestPreviewReportDeletePropagatesMissingDeleteTarget(t *testing.T) {
	store := &reportRunDeleteStore{loadErr: sql.ErrNoRows}
	svc := NewService(store)

	_, err := svc.PreviewReportDelete(context.Background(), ReportDeletePreviewRequest{
		MissionID: "mis_1", ArtifactID: "art_1",
	})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected missing delete target to propagate, got %v", err)
	}
}

type reportRunDeleteStore struct {
	fakeStore
	events                  []LedgerEvent
	facts                   reportrun.DeleteFacts
	loadErr                 error
	backfilled              bool
	registration            reportrun.Registration
	deleted                 bool
	expectedRevision        int64
	expectedDeleteFactsHash string
}

func (s *reportRunDeleteStore) ListLedgerEvents(context.Context, string) ([]LedgerEvent, error) {
	return append([]LedgerEvent(nil), s.events...), nil
}

func (s *reportRunDeleteStore) BackfillReportRuns(_ context.Context, registration reportrun.Registration) error {
	s.backfilled = true
	s.registration = registration
	return nil
}

func (s *reportRunDeleteStore) LoadReportRunDeleteFacts(context.Context, string, string) (reportrun.DeleteFacts, error) {
	if s.loadErr != nil {
		return reportrun.DeleteFacts{}, s.loadErr
	}
	return s.facts, nil
}

func (s *reportRunDeleteStore) DeleteReportRun(
	_ context.Context,
	_ string,
	_ string,
	expectedRevision int64,
	expectedDeleteFactsHash string,
	decide func(reportrun.DeleteFacts) (reportrun.DeleteDecision, error),
) (reportrun.DeletePreview, error) {
	s.deleted = true
	s.expectedRevision = expectedRevision
	s.expectedDeleteFactsHash = expectedDeleteFactsHash
	decision, err := decide(s.facts)
	if err != nil {
		return reportrun.DeletePreview{}, err
	}
	return decision.Preview, nil
}

func completedReportDeleteFacts() reportrun.DeleteFacts {
	now := time.Date(2026, 8, 9, 1, 2, 3, 0, time.UTC)
	return reportrun.DeleteFacts{
		Run: reportrun.Run{
			RunID: "evt_report_pending", MissionID: "mis_1", RootPendingEventID: "evt_report_pending",
			LifecycleState: reportrun.LifecycleCompleted, Revision: 3, FinalArtifactID: "art_1",
		},
		Events: []reportrun.MemberEvent{{
			Event: reportrun.Event{EventID: "evt_report_pending", MissionID: "mis_1", EventType: "report.draft.pending", Payload: []byte(`{"title":"Report"}`), CreatedAt: now},
		}, {
			Event: reportrun.Event{EventID: "evt_report_final", MissionID: "mis_1", EventType: "report.artifact.created", Payload: []byte(`{"pending_event_id":"evt_report_pending","artifact_id":"art_1"}`), CreatedAt: now},
		}},
		Artifacts: []reportrun.MemberArtifact{{
			Membership: reportrun.ArtifactMembership{ArtifactID: "art_1", ArtifactRole: reportrun.ArtifactRoleFinal, Ownership: reportrun.OwnershipCreated},
			Artifact:   reportrun.Artifact{ArtifactID: "art_1", MissionID: "mis_1", ByteSize: 7},
		}},
	}
}
