package researchcatalog

import (
	"context"

	artifact "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mission"
	"github.com/c86j224s/liquid2/plasma/internal/researchrecords"
	"github.com/c86j224s/liquid2/plasma/internal/source"
)

// Dependencies supplies the app-owned reads and projections used by Service.
// Callbacks are intentionally invoked lazily in the historical operation order.
type Dependencies struct {
	EnsureReferenceTargetVisible func(context.Context, string, string, string, bool, map[string]struct{}) error
	ReadReferenceSummary         func(context.Context, string, string, string, bool) (ObjectSummary, error)

	GetMissionProjection                         func(context.Context, string) (mission.Projection, error)
	ListSourceSnapshots                          func(context.Context, string) ([]source.Snapshot, error)
	ListVisibleRawArtifacts                      func(context.Context, string, bool) ([]artifact.Raw, error)
	RawArtifactReadKind                          func(artifact.Raw) string
	ListEvents                                   func(context.Context, string) ([]ledger.Event, error)
	ListVisibleLedgerEvents                      func(context.Context, string, bool) ([]ledger.Event, error)
	ReportArtifactIDsHiddenFromResearchDiscovery func(context.Context, string, bool) (map[string]struct{}, error)
	FilterReportArtifactRefs                     func([]ObjectRef, map[string]struct{}) []ObjectRef
	FilterReportArtifactSummaryRefs              func([]ObjectSummary, map[string]struct{}) []ObjectSummary

	ListEvidenceRecords   func(context.Context, string) ([]researchrecords.EvidenceRecord, error)
	ListClaimRecords      func(context.Context, string) ([]researchrecords.ClaimRecord, error)
	ListQuestionRecords   func(context.Context, string) ([]researchrecords.QuestionRecord, error)
	ListOptionRecords     func(context.Context, string) ([]researchrecords.OptionRecord, error)
	ListProposalSummaries func(context.Context, string) ([]ObjectSummary, error)

	ListReportSummaries        func(context.Context, string) ([]ObjectSummary, error)
	ListReportVersionSummaries func(context.Context, string) ([]ObjectSummary, error)
	ListReportBlockSummaries   func(context.Context, string) ([]ObjectSummary, error)
}
