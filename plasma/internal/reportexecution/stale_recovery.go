package reportexecution

import (
	"context"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

// RecoveryHooks preserves product-specific legacy defaults and finalized H5
// recovery without introducing provider or HTTP dependencies into the runner.
type RecoveryHooks struct {
	Defaults                 RecoveryDefaults
	PendingExecutor          func(ledger.Event) string
	PendingMode              func(ledger.Event) string
	RecoverHumanizeFinalized func(context.Context, string, ledger.Event) (bool, error)
}

// ResumeLegacyDraft uses the historical Web decoder rather than the general
// pending decoder; nonrecoverable entries remain active and visible.
func (runner Runner) ResumeLegacyDraft(ctx context.Context, missionID string, pending ledger.Event, hooks RecoveryHooks) error {
	if !DraftPendingRecoverable(pending) {
		return nil
	}
	req, err := LegacyRecoveryDraftRequest(pending, hooks.Defaults)
	if err != nil {
		_, failErr := runner.AppendDraftFailed(ctx, missionID, pending.EventID, hooks.PendingExecutor(pending), hooks.PendingMode(pending), err)
		return failErr
	}
	return runner.RunDraft(context.Background(), missionID, req, pending.EventID)
}

// ResumeStaleReportOperations scans newest first and stops at the first unowned
// operation, preserving finalized-humanize recovery before provider replay.
func (runner Runner) ResumeStaleReportOperations(ctx context.Context, missionID string, hooks RecoveryHooks) error {
	events, err := runner.Service.ListEvents(ctx, missionID)
	if err != nil {
		return err
	}
	completed := CompletedPendingEventIDs(events)
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if _, ok := completed[event.EventID]; ok {
			continue
		}
		switch event.EventType {
		case "report.draft.pending":
			if !runner.InFlight.Owns(missionID, event.EventID) {
				return runner.ResumeLegacyDraft(ctx, missionID, event, hooks)
			}
		case "report.humanize.pending":
			if runner.InFlight.Owns(missionID, event.EventID) {
				continue
			}
			return runner.RetireHumanizePending(ctx, missionID, event)
		case "report.patch.pending":
			if !runner.InFlight.Owns(missionID, event.EventID) {
				return runner.ResumePatch(ctx, missionID, event)
			}
		}
	}
	return nil
}
