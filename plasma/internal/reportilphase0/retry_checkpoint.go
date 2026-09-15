package reportilphase0

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

// RetryCheckpointStore owns durable checkpoint lookup and append operations for
// the Report IL retry boundary. The phase-0 resolver does not own persistence.
type RetryCheckpointStore interface {
	LoadReportILResumeCheckpoint(context.Context, string, string) (*reportilcontract.ResumeCheckpoint, error)
	AppendReportILCheckpoint(context.Context, string, reportilcontract.ProductCheckpoint) error
}

type RetryCheckpointDocuments interface {
	ReadReportILEditorialMemory(context.Context, string, string, reportilcontract.SourceCatalog) (reportilcontract.EditorialMemory, reportilcontract.EditorialMemoryReceipt, error)
	ReadReportILLongFormPlan(context.Context, string, string, reportilcontract.SourceCatalog) (reportilcontract.LongFormPlan, reportilcontract.LongFormPlanReceipt, error)
	ReadReportILLongFormStageDocument(context.Context, string, string, string, reportilcontract.SourceCatalog) (reportilcontract.AuthorDocument, reportilcontract.AuthorWorkspaceReceipt, error)
}

// ResolveRetryCheckpoint searches the retry's private pending lineage for a
// stored checkpoint, then applies the existing phase-0 recovery fallbacks.
// Lineage intentionally preserves only parent order, cycle nil, and the depth
// cap; it does not perform normal report-execution lineage validation.
func ResolveRetryCheckpoint(
	ctx context.Context,
	missionID,
	retryPendingID string,
	events []ledger.Event,
	store RetryCheckpointStore,
	sources SourceReader,
	documents RetryCheckpointDocuments,
) (*reportilcontract.ResumeCheckpoint, error) {
	lineage := reportILRetryPendingLineage(events, retryPendingID)
	if lineage == nil {
		return nil, nil
	}
	var err error
	for _, pendingID := range lineage {
		var resume *reportilcontract.ResumeCheckpoint
		resume, err = store.LoadReportILResumeCheckpoint(ctx, missionID, pendingID)
		if err == nil {
			return resume, nil
		}
		if !strings.Contains(err.Error(), "no durable checkpoint") {
			return nil, err
		}
	}
	if err == nil || !strings.Contains(err.Error(), "no durable checkpoint") {
		return nil, err
	}
	resume, err := RecoverSourceSelectionCheckpoint(ctx, missionID, retryPendingID, events, sources)
	if err == nil {
		if appendErr := store.AppendReportILCheckpoint(ctx, missionID, resume.ProductCheckpoint); appendErr != nil {
			return nil, fmt.Errorf("append recovered checkpoint: %w", appendErr)
		}
		return resume, nil
	}
	if !strings.Contains(err.Error(), "not recoverable") {
		return nil, err
	}
	resume, err = RecoverPartsCheckpoint(ctx, missionID, retryPendingID, events, sources, documents)
	if err == nil {
		if appendErr := store.AppendReportILCheckpoint(ctx, missionID, resume.ProductCheckpoint); appendErr != nil {
			return nil, fmt.Errorf("append recovered checkpoint: %w", appendErr)
		}
		return resume, nil
	}
	if !strings.Contains(err.Error(), "not recoverable") {
		return nil, err
	}
	return RecoverLegacyCheckpoint(ctx, missionID, retryPendingID, events, sources, documents)
}

func reportILRetryPendingLineage(events []ledger.Event, pendingID string) []string {
	parents := map[string]string{}
	for _, event := range events {
		if event.EventType != "report.draft.pending" {
			continue
		}
		var payload struct {
			RetryOf string `json:"retry_of_pending_event_id"`
		}
		_ = json.Unmarshal(event.Payload, &payload)
		parents[event.EventID] = strings.TrimSpace(payload.RetryOf)
	}
	lineage := []string{}
	seen := map[string]bool{}
	for current, depth := pendingID, 0; current != "" && depth < 64; depth++ {
		if seen[current] {
			return nil
		}
		seen[current] = true
		lineage = append(lineage, current)
		current = parents[current]
	}
	return lineage
}
