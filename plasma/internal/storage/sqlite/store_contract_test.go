package sqlite

import (
	"github.com/c86j224s/liquid2/plasma/internal/confluenceaccess"
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/researchrecords"
	"github.com/c86j224s/liquid2/plasma/internal/source"
)

type conditionalRawArtifactStoreContract interface {
	CommitRawArtifactWithEventConditionally(
		context.Context,
		artifactcontract.Raw,
		func([]ledger.Event) (ledger.Event, bool, error),
	) (artifactcontract.Raw, ledger.Event, bool, error)
}

type conditionalDesignedReportHTMLExportStoreContract interface {
	CommitDesignedReportHTMLExportConditionally(
		context.Context,
		string,
		artifactcontract.Raw,
		artifactcontract.Raw,
		func([]ledger.Event) ([]ledger.Event, bool, error),
	) (artifactcontract.Raw, artifactcontract.Raw, ledger.Event, bool, error)
}

type reportRedpenRevisionStoreContract interface {
	CommitReportRedpenRevision(
		context.Context,
		artifactcontract.Raw,
		func([]ledger.Event, artifactcontract.Raw, string) (ledger.Event, bool, error),
	) (artifactcontract.Raw, ledger.Event, bool, error)
}

var _ app.Store = (*Store)(nil)
var _ app.AtomicWriteStore = (*Store)(nil)
var _ app.MissionListStore = (*Store)(nil)
var _ app.MissionActivityListStore = (*Store)(nil)
var _ app.ReportListStore = (*Store)(nil)
var _ app.ResearchRecordListStore = (*Store)(nil)
var _ researchrecords.QuestionStore = (*Store)(nil)
var _ researchrecords.QuestionListStore = (*Store)(nil)
var _ researchrecords.OptionStore = (*Store)(nil)
var _ researchrecords.OptionListStore = (*Store)(nil)
var _ app.RawArtifactListStore = (*Store)(nil)
var _ artifactcontract.Store = (*Store)(nil)
var _ source.Store = (*Store)(nil)
var _ source.ListStore = (*Store)(nil)
var _ app.ConditionalLedgerStore = (*Store)(nil)
var _ app.ModelDefaultsStore = (*Store)(nil)
var _ app.MissionHardDeleteStore = (*Store)(nil)
var _ confluenceaccess.ConnectionStore = (*Store)(nil)
var _ conditionalRawArtifactStoreContract = (*Store)(nil)
var _ conditionalDesignedReportHTMLExportStoreContract = (*Store)(nil)
var _ reportRedpenRevisionStoreContract = (*Store)(nil)
