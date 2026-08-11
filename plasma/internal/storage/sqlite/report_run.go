package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reportrun"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite/reportrunrepo"
)

// BackfillReportRuns applies an application-built report-run registration
// without touching ledger payloads or raw artifact bodies.
func (s *Store) BackfillReportRuns(ctx context.Context, registration reportrun.Registration) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := reportrunrepo.ApplyRegistrationTx(ctx, tx, registration); err != nil {
		return err
	}
	return tx.Commit()
}

// LoadReportRunDeleteFacts returns the current delete snapshot for a report artifact.
func (s *Store) LoadReportRunDeleteFacts(ctx context.Context, missionID string, artifactID string) (reportrun.DeleteFacts, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return reportrun.DeleteFacts{}, err
	}
	defer tx.Rollback()
	return reportrunrepo.DeleteFactsForArtifactTx(ctx, tx, missionID, artifactID)
}

// DeleteReportRun recomputes delete facts in a transaction, checks revision,
// and applies the caller-built purge decision.
func (s *Store) DeleteReportRun(
	ctx context.Context,
	missionID string,
	artifactID string,
	expectedRevision int64,
	expectedDeleteFactsHash string,
	decide func(reportrun.DeleteFacts) (reportrun.DeleteDecision, error),
) (reportrun.DeletePreview, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return reportrun.DeletePreview{}, err
	}
	defer tx.Rollback()
	facts, err := reportrunrepo.DeleteFactsForArtifactTx(ctx, tx, missionID, artifactID)
	if err != nil {
		return reportrun.DeletePreview{}, err
	}
	if facts.Run.Revision != expectedRevision {
		return reportrun.DeletePreview{}, fmt.Errorf("%w: report run revision changed", app.ErrConflict)
	}
	if strings.TrimSpace(expectedDeleteFactsHash) == "" {
		return reportrun.DeletePreview{}, fmt.Errorf("%w: delete_facts_hash is required", app.ErrInvalidInput)
	}
	if reportrun.DeleteFactsHash(facts) != strings.TrimSpace(expectedDeleteFactsHash) {
		return reportrun.DeletePreview{}, fmt.Errorf("%w: report delete facts changed", app.ErrConflict)
	}
	decision, err := decide(facts)
	if err != nil {
		return reportrun.DeletePreview{}, err
	}
	if len(decision.Preview.Blockers) > 0 {
		return decision.Preview, fmt.Errorf("%w: report is not eligible for delete", app.ErrConflict)
	}
	if err := reportrunrepo.DeleteRunTx(ctx, tx, decision); err != nil {
		return reportrun.DeletePreview{}, err
	}
	if err := tx.Commit(); err != nil {
		return reportrun.DeletePreview{}, err
	}
	return decision.Preview, nil
}
