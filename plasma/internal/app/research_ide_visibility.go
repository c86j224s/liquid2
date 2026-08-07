package app

import (
	"encoding/json"
	"strings"
)

// researchIDEVisibleLedgerEvents keeps legacy discovery unchanged and removes
// report workflow events only from the non-legacy research discovery surface.
func researchIDEVisibleLedgerEvents(events []LedgerEvent, legacy bool) []LedgerEvent {
	if legacy {
		return events
	}
	visible := make([]LedgerEvent, 0, len(events))
	for _, event := range events {
		if researchIDEReportLedgerEvent(event) {
			continue
		}
		visible = append(visible, event)
	}
	return visible
}

// researchIDEReportArtifactIDs returns report output artifacts by ledger
// lineage. It deliberately ignores filenames, media types, producers, and
// payload artifact arrays because this boundary is defined by report.* events
// carrying a singular payload.artifact_id.
func researchIDEReportArtifactIDs(events []LedgerEvent) map[string]struct{} {
	artifactIDs := map[string]struct{}{}
	for _, event := range events {
		if !researchIDEReportLedgerEvent(event) {
			continue
		}
		var payload struct {
			ArtifactID string `json:"artifact_id"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		if artifactID := strings.TrimSpace(payload.ArtifactID); artifactID != "" {
			artifactIDs[artifactID] = struct{}{}
		}
	}
	return artifactIDs
}

func researchIDEReportLedgerEvent(event LedgerEvent) bool {
	return strings.HasPrefix(strings.TrimSpace(event.EventType), "report.")
}

func researchIDEFilterReportArtifactRefs(refs []ResearchIDEObjectRef, reportArtifactIDs map[string]struct{}) []ResearchIDEObjectRef {
	if len(reportArtifactIDs) == 0 {
		return refs
	}
	filtered := make([]ResearchIDEObjectRef, 0, len(refs))
	for _, ref := range refs {
		if ref.ObjectKind == ResearchIDEObjectRawArtifact {
			if _, ok := reportArtifactIDs[ref.ObjectID]; ok {
				continue
			}
		}
		filtered = append(filtered, ref)
	}
	return filtered
}

func researchIDEFilterReportArtifactSummaryRefs(summaries []ResearchIDEObjectSummary, reportArtifactIDs map[string]struct{}) []ResearchIDEObjectSummary {
	if len(reportArtifactIDs) == 0 {
		return summaries
	}
	filtered := make([]ResearchIDEObjectSummary, 0, len(summaries))
	for _, summary := range summaries {
		summary.Refs = researchIDEFilterReportArtifactRefs(summary.Refs, reportArtifactIDs)
		filtered = append(filtered, summary)
	}
	return filtered
}
