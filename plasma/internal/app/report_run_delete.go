package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/reportrun"
)

// ReportRunStore is the storage port for logical report-run membership and
// explicit report artifact deletion.
type ReportRunStore interface {
	BackfillReportRuns(context.Context, reportrun.Registration) error
	LoadReportRunDeleteFacts(context.Context, string, string) (reportrun.DeleteFacts, error)
	DeleteReportRun(context.Context, string, string, int64, string, func(reportrun.DeleteFacts) (reportrun.DeleteDecision, error)) (reportrun.DeletePreview, error)
}

// ReportDeletePreviewRequest describes the artifact-card preview target.
type ReportDeletePreviewRequest struct {
	MissionID            string
	ArtifactID           string
	ActivePendingEventID string
}

// ReportDeleteRequest is the explicit report deletion command.
type ReportDeleteRequest struct {
	MissionID            string
	ArtifactID           string
	ConfirmArtifactID    string
	ExpectedRevision     int64
	DeleteFactsHash      string
	ActivePendingEventID string
	Producer             Producer
}

// ReportDeleteResult is returned after a successful report-run purge.
type ReportDeleteResult struct {
	RunID   string                  `json:"run_id"`
	Deleted bool                    `json:"deleted"`
	Preview reportrun.DeletePreview `json:"preview"`
}

// PreviewReportDelete returns the current deletion impact for a completed
// Markdown report artifact. It backfills legacy report-run membership using
// explicit lineage before loading the preview facts.
func (s *Service) PreviewReportDelete(ctx context.Context, req ReportDeletePreviewRequest) (reportrun.DeletePreview, error) {
	missionID, artifactID, err := validateReportDeleteTarget(req.MissionID, req.ArtifactID)
	if err != nil {
		return reportrun.DeletePreview{}, err
	}
	store, err := s.reportRunStore()
	if err != nil {
		return reportrun.DeletePreview{}, err
	}
	if err := s.ensureReportRuns(ctx, missionID, store); err != nil {
		return reportrun.DeletePreview{}, err
	}
	facts, err := store.LoadReportRunDeleteFacts(ctx, missionID, artifactID)
	if err != nil {
		return reportrun.DeletePreview{}, err
	}
	return reportrun.PreviewDelete(facts, strings.TrimSpace(req.ActivePendingEventID)), nil
}

// DeleteReport purges a completed report run after artifact confirmation and
// revision matching. The delete transaction recomputes the preview facts.
func (s *Service) DeleteReport(ctx context.Context, req ReportDeleteRequest) (ReportDeleteResult, error) {
	missionID, artifactID, err := validateReportDeleteTarget(req.MissionID, req.ArtifactID)
	if err != nil {
		return ReportDeleteResult{}, err
	}
	if strings.TrimSpace(req.ConfirmArtifactID) != artifactID {
		return ReportDeleteResult{}, fmt.Errorf("%w: confirmation artifact_id does not match", ErrInvalidInput)
	}
	if req.ExpectedRevision <= 0 {
		return ReportDeleteResult{}, fmt.Errorf("%w: expected_revision is required", ErrInvalidInput)
	}
	deleteFactsHash := strings.TrimSpace(req.DeleteFactsHash)
	if deleteFactsHash == "" {
		return ReportDeleteResult{}, fmt.Errorf("%w: delete_facts_hash is required", ErrInvalidInput)
	}
	if req.Producer.Type != "user" {
		return ReportDeleteResult{}, fmt.Errorf("%w: report delete requires a user producer", ErrInvalidInput)
	}
	store, err := s.reportRunStore()
	if err != nil {
		return ReportDeleteResult{}, err
	}
	if err := s.ensureReportRuns(ctx, missionID, store); err != nil {
		return ReportDeleteResult{}, err
	}
	activePendingID := strings.TrimSpace(req.ActivePendingEventID)
	now := time.Now().UTC()
	preview, err := store.DeleteReportRun(ctx, missionID, artifactID, req.ExpectedRevision, deleteFactsHash, func(facts reportrun.DeleteFacts) (reportrun.DeleteDecision, error) {
		return reportrun.BuildDeleteDecision(facts, activePendingID, req.Producer.Type, req.Producer.ID, now), nil
	})
	if err != nil {
		return ReportDeleteResult{}, err
	}
	return ReportDeleteResult{RunID: preview.RunID, Deleted: true, Preview: preview}, nil
}

func (s *Service) reportRunStore() (ReportRunStore, error) {
	store, ok := s.store.(ReportRunStore)
	if !ok {
		return nil, fmt.Errorf("%w: report-run storage is not configured", ErrInvalidInput)
	}
	return store, nil
}

func (s *Service) ensureReportRuns(ctx context.Context, missionID string, store ReportRunStore) error {
	events, err := s.store.ListLedgerEvents(ctx, missionID)
	if err != nil {
		return err
	}
	registration, err := reportrun.BuildRegistration(appReportRunEvents(events), reportrun.RegistrationBackfilled, time.Now().UTC())
	if err != nil {
		return err
	}
	return store.BackfillReportRuns(ctx, registration)
}

func validateReportDeleteTarget(missionID string, artifactID string) (string, string, error) {
	missionID = strings.TrimSpace(missionID)
	artifactID = strings.TrimSpace(artifactID)
	if err := validateID("mis_", missionID); err != nil {
		return "", "", err
	}
	if err := validateID("art_", artifactID); err != nil {
		return "", "", err
	}
	return missionID, artifactID, nil
}

func appReportRunEvents(events []LedgerEvent) []reportrun.Event {
	out := make([]reportrun.Event, 0, len(events))
	for _, event := range events {
		out = append(out, reportrun.Event{
			EventID: event.EventID, MissionID: event.MissionID,
			Sequence: event.Sequence, EventType: event.EventType, Payload: event.Payload,
			CreatedAt: event.CreatedAt,
		})
	}
	return out
}
