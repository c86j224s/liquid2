package reportrun

import "time"

const (
	LifecycleActive    = "active"
	LifecycleCompleted = "completed"
	LifecycleFailed    = "failed"
	LifecycleCanceled  = "canceled"
	LifecycleAmbiguous = "ambiguous"
	LifecyclePurged    = "purged"

	RegistrationNative     = "native"
	RegistrationBackfilled = "backfilled"

	ArtifactRoleInput        = "input"
	ArtifactRoleIntermediate = "intermediate"
	ArtifactRoleFinal        = "final"
	ArtifactRoleDerivative   = "derivative"

	OwnershipCreated    = "created"
	OwnershipReferenced = "referenced"

	UsageAggregationVersion = "report_usage.v2"
)

// UsageAggregate is the compact usage tombstone retained after report purge.
//
// The aggregate counts each member event at most once and converts cumulative
// provider snapshots into call increments. Missing or discontinuous usage marks
// the aggregate as partial without preserving prompts, provider responses, or
// report content.
type UsageAggregate struct {
	UsageRecordCount      int64  `json:"usage_record_count"`
	UsageAvailableCount   int64  `json:"usage_available_count"`
	UsageUnavailableCount int64  `json:"usage_unavailable_count"`
	InputTokens           int64  `json:"input_tokens"`
	CachedInputTokens     int64  `json:"cached_input_tokens"`
	UncachedInputTokens   int64  `json:"uncached_input_tokens"`
	OutputTokens          int64  `json:"output_tokens"`
	ReasoningOutputTokens int64  `json:"reasoning_output_tokens"`
	TotalTokens           int64  `json:"total_tokens"`
	UsagePartial          bool   `json:"usage_partial"`
	AggregationVersion    string `json:"aggregation_version"`
}

// Run is the report-run aggregate root persisted in plasma_report_runs.
//
// RunID is always the root/original report.draft.pending event ID. Revision is
// monotonic and must advance on membership or lifecycle changes.
type Run struct {
	RunID              string
	MissionID          string
	RootPendingEventID string
	LifecycleState     string
	Revision           int64
	Title              string
	FinalArtifactID    string
	RegistrationStatus string
	PurgedAt           time.Time
	PurgedByType       string
	PurgedByID         string
	Usage              UsageAggregate
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// EventMembership links an existing ledger event to one report run.
type EventMembership struct {
	RunID          string
	EventID        string
	MissionID      string
	EventRole      string
	AttemptEventID string
	CreatedAt      time.Time
}

// ArtifactMembership links an existing raw artifact to one report run.
type ArtifactMembership struct {
	RunID          string
	ArtifactID     string
	MissionID      string
	ArtifactRole   string
	Ownership      string
	AttemptEventID string
	SourceEventID  string
	CreatedAt      time.Time
}

// Registration is the complete report-run membership projection for a mission.
type Registration struct {
	Runs      []Run
	Events    []EventMembership
	Artifacts []ArtifactMembership
}

// Event is the ledger subset needed by report-run classification.
type Event struct {
	EventID   string
	MissionID string
	Sequence  int64
	EventType string
	Payload   []byte
	CreatedAt time.Time
}

// Artifact is the raw artifact metadata needed by report-run deletion.
type Artifact struct {
	ArtifactID string
	MissionID  string
	MediaType  string
	ByteSize   int64
}

// MemberEvent is a run event membership with its ledger event.
type MemberEvent struct {
	Membership EventMembership
	Event      Event
}

// MemberArtifact is a run artifact membership with its raw artifact metadata.
type MemberArtifact struct {
	Membership ArtifactMembership
	Artifact   Artifact
}

// SharedArtifact records why a run-created artifact must be preserved.
type SharedArtifact struct {
	ArtifactID string   `json:"artifact_id"`
	ByteSize   int64    `json:"byte_size"`
	Reasons    []string `json:"reasons"`
}

// Blocker records one stable reason that report-run deletion cannot proceed.
type Blocker struct {
	ReasonCode string `json:"reason_code"`
	Message    string `json:"message"`
}

// DeleteFacts is the storage snapshot used by preview and delete.
type DeleteFacts struct {
	Run                Run
	Events             []MemberEvent
	Artifacts          []MemberArtifact
	SharedArtifacts    []SharedArtifact
	MalformedReference bool
}

// DeletePreview is the user-facing deletion impact view.
type DeletePreview struct {
	RunID                  string           `json:"run_id"`
	State                  string           `json:"state"`
	Revision               int64            `json:"revision"`
	Eligible               bool             `json:"eligible"`
	DeletableEventCount    int64            `json:"deletable_event_count"`
	DeletableArtifactCount int64            `json:"deletable_artifact_count"`
	DeletableArtifactBytes int64            `json:"deletable_artifact_bytes"`
	SharedArtifactCount    int64            `json:"shared_artifact_count"`
	SharedArtifacts        []SharedArtifact `json:"shared_artifacts"`
	Blockers               []Blocker        `json:"blockers"`
	Usage                  UsageAggregate   `json:"usage"`
	DeleteFactsHash        string           `json:"delete_facts_hash"`
}

// DeleteDecision is the result of applying product deletion rules to facts.
type DeleteDecision struct {
	Preview           DeletePreview
	DeleteEventIDs    []string
	DeleteArtifactIDs []string
	RetainedUsage     UsageAggregate
	PurgedAt          time.Time
	PurgedByType      string
	PurgedByID        string
}
