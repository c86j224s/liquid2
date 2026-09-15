package researchrecords

import (
	"context"
	"encoding/json"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/source"
)

const (
	EvidenceRecordSchemaVersion = "plasma.evidence_record.v1"
	EvidenceRecordObjectKind    = "evidence_record"
)

// Confidence is the confidence metadata attached to an evidence record.
type Confidence struct {
	Level             string   `json:"level"`
	Rationale         string   `json:"rationale"`
	OpenRisks         []string `json:"open_risks,omitempty"`
	NeedsVerification bool     `json:"needs_verification"`
}

// SnapshotRef identifies an artifact and locator within a source snapshot.
type SnapshotRef struct {
	SnapshotID string          `json:"snapshot_id"`
	ArtifactID string          `json:"artifact_id"`
	Locator    json.RawMessage `json:"locator"`
}

// EvidenceRecord is evidence extracted from a source snapshot.
type EvidenceRecord struct {
	SchemaVersion  string          `json:"schema_version"`
	ObjectKind     string          `json:"object_kind"`
	EvidenceID     string          `json:"evidence_id"`
	MissionID      string          `json:"mission_id"`
	State          string          `json:"state"`
	Summary        string          `json:"summary"`
	EvidenceType   string          `json:"evidence_type"`
	SnapshotRefs   []SnapshotRef   `json:"snapshot_refs"`
	Confidence     Confidence      `json:"confidence"`
	Producer       ledger.Producer `json:"producer"`
	CreatedEventID string          `json:"created_event_id"`
	CreatedAt      time.Time       `json:"created_at"`
}

// CreateEvidenceRecordRequest contains the caller-owned evidence values.
type CreateEvidenceRecordRequest struct {
	EvidenceID     string
	MissionID      string
	State          string
	Summary        string
	EvidenceType   string
	SnapshotRefs   []SnapshotRef
	Confidence     Confidence
	Producer       ledger.Producer
	CreatedEventID string
}

// SnapshotReader is the only source dependency needed to validate evidence refs.
type SnapshotReader interface {
	GetSourceSnapshot(context.Context, string) (source.Snapshot, error)
}

// EvidenceStore persists and reads evidence records.
type EvidenceStore interface {
	CreateEvidenceRecord(context.Context, EvidenceRecord) error
	GetEvidenceRecord(context.Context, string) (EvidenceRecord, error)
}

// EvidenceListStore lists evidence records for one mission.
type EvidenceListStore interface {
	ListEvidenceRecords(context.Context, string) ([]EvidenceRecord, error)
}
