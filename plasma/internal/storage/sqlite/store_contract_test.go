package sqlite

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

type conditionalRawArtifactStoreContract interface {
	CommitRawArtifactWithEventConditionally(
		context.Context,
		app.RawArtifact,
		func([]app.LedgerEvent) (app.LedgerEvent, bool, error),
	) (app.RawArtifact, app.LedgerEvent, bool, error)
}

type conditionalDesignedReportHTMLExportStoreContract interface {
	CommitDesignedReportHTMLExportConditionally(
		context.Context,
		string,
		app.RawArtifact,
		app.RawArtifact,
		func([]app.LedgerEvent) ([]app.LedgerEvent, bool, error),
	) (app.RawArtifact, app.RawArtifact, app.LedgerEvent, bool, error)
}

type reportRedpenRevisionStoreContract interface {
	CommitReportRedpenRevision(
		context.Context,
		app.RawArtifact,
		func([]app.LedgerEvent, app.RawArtifact, string) (app.LedgerEvent, bool, error),
	) (app.RawArtifact, app.LedgerEvent, bool, error)
}

var _ app.Store = (*Store)(nil)
var _ app.AtomicWriteStore = (*Store)(nil)
var _ app.MissionListStore = (*Store)(nil)
var _ app.MissionActivityListStore = (*Store)(nil)
var _ app.ReportListStore = (*Store)(nil)
var _ app.ResearchRecordListStore = (*Store)(nil)
var _ app.RawArtifactListStore = (*Store)(nil)
var _ app.SourceSnapshotListStore = (*Store)(nil)
var _ app.ConditionalLedgerStore = (*Store)(nil)
var _ app.ModelDefaultsStore = (*Store)(nil)
var _ app.MissionHardDeleteStore = (*Store)(nil)
var _ app.ConfluenceConnectionStore = (*Store)(nil)
var _ conditionalRawArtifactStoreContract = (*Store)(nil)
var _ conditionalDesignedReportHTMLExportStoreContract = (*Store)(nil)
var _ reportRedpenRevisionStoreContract = (*Store)(nil)
