package app

import (
	"github.com/c86j224s/liquid2/plasma/internal/mission"
	"github.com/c86j224s/liquid2/plasma/internal/researchcatalog"
)

// ResearchIDEOutline은 agent가 미션 전체 구조를 가볍게 파악하기 위한 summary view다.
type ResearchIDEOutline struct {
	MissionID               string                          `json:"mission_id"`
	LastSequence            int64                           `json:"last_sequence"`
	Title                   string                          `json:"title"`
	Objective               string                          `json:"objective,omitempty"`
	Scope                   mission.Scope                   `json:"scope"`
	Counts                  map[string]int                  `json:"counts"`
	ActiveReportVersionID   string                          `json:"active_report_version_id,omitempty"`
	RecentLedgerEvents      []researchcatalog.ObjectSummary `json:"recent_ledger_events,omitempty"`
	NextSuggestedObjectRefs []researchcatalog.ObjectRef     `json:"next_suggested_object_refs,omitempty"`
}

// ResearchIDEChangesRequest identifies the durable mission ledger high-water
// mark an agent has already observed. Only later meaningful changes are
// returned.
type ResearchIDEChangesRequest struct {
	MissionID     string
	AfterSequence int64
	Limit         int
}

// ResearchIDEChanges is a bounded change feed over mission content and source
// state. Internal execution telemetry is omitted, while CurrentSequence still
// advances across it so callers do not inspect the same ledger range twice.
type ResearchIDEChanges struct {
	MissionID         string                          `json:"mission_id"`
	AfterSequence     int64                           `json:"after_sequence"`
	CurrentSequence   int64                           `json:"current_sequence"`
	Items             []researchcatalog.ObjectSummary `json:"items"`
	NextAfterSequence int64                           `json:"next_after_sequence"`
	Limit             int                             `json:"limit"`
	Truncated         bool                            `json:"truncated"`
	ResyncRequired    bool                            `json:"resync_required"`
}
