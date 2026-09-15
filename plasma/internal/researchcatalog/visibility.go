package researchcatalog

import (
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"strings"
)

func ReportLedgerEvent(event ledger.Event) bool {
	return strings.HasPrefix(strings.TrimSpace(event.EventType), "report.")
}

func FilterReportArtifactRefs(refs []ObjectRef, reportArtifactIDs map[string]struct{}) []ObjectRef {
	if len(reportArtifactIDs) == 0 {
		return refs
	}
	filtered := make([]ObjectRef, 0, len(refs))
	for _, ref := range refs {
		if ref.ObjectKind == ObjectRawArtifact {
			if _, ok := reportArtifactIDs[ref.ObjectID]; ok {
				continue
			}
		}
		filtered = append(filtered, ref)
	}
	return filtered
}

func FilterReportArtifactSummaryRefs(summaries []ObjectSummary, reportArtifactIDs map[string]struct{}) []ObjectSummary {
	if len(reportArtifactIDs) == 0 {
		return summaries
	}
	filtered := make([]ObjectSummary, 0, len(summaries))
	for _, summary := range summaries {
		summary.Refs = FilterReportArtifactRefs(summary.Refs, reportArtifactIDs)
		filtered = append(filtered, summary)
	}
	return filtered
}
