package source

import (
	"context"
	"encoding/json"
	"github.com/c86j224s/liquid2/plasma/internal/app"
	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
	"github.com/c86j224s/liquid2/plasma/internal/source/confluencesource"
	"github.com/c86j224s/liquid2/plasma/internal/source/liquid2source"
)

type Service interface {
	ListSourceSnapshotsWithState(context.Context, sourcecontract.ListRequest) ([]sourcecontract.Snapshot, error)
	GetSourceSnapshot(context.Context, string) (sourcecontract.Snapshot, error)
	GetRawArtifact(context.Context, string) (artifactcontract.Raw, error)
	ListLocalPathRoots(context.Context) ([]sourcecontract.LocalPathRoot, error)
	BrowseLocalPathRoot(context.Context, app.BrowseLocalPathRootRequest) (sourcecontract.LocalPathTreeResult, error)
	AttachLocalPathSource(context.Context, app.AttachLocalPathSourceRequest) (app.LocalPathSourceResult, error)
	ReadLocalPathSource(context.Context, app.ReadLocalPathSourceRequest) (app.ReadLocalPathSourceResult, error)
	TreeLocalPathSource(context.Context, app.TreeLocalPathSourceRequest) (app.TreeLocalPathSourceResult, error)
	GrepLocalPathSource(context.Context, app.GrepLocalPathSourceRequest) (app.GrepLocalPathSourceResult, error)
	RemoveSource(context.Context, app.RemoveSourceRequest) (app.SourceStateChangeResult, error)
	RestoreSource(context.Context, app.RestoreSourceRequest) (app.SourceStateChangeResult, error)
	SearchLiquid2Sources(context.Context, liquid2source.Liquid2SourceConnector, liquid2source.Liquid2SourceSearchRequest) (liquid2source.Liquid2SourceSearchResult, error)
	SearchConfluenceSources(context.Context, confluencesource.ConfluenceSourceConnector, confluencesource.ConfluenceSourceSearchRequest) (confluencesource.ConfluenceSourceSearchResult, error)
	GetMissionConnectorAccess(context.Context, string, string) (app.ConnectorAccessProjection, error)
}
type ConfluenceConnectorRequest struct{ ConnectionID, CloudID, SpaceKey string }
type Factory func(context.Context, ConfluenceConnectorRequest) (confluencesource.ConfluenceSourceConnector, error)
type Handler struct {
	ApprovalRequired    func(string, string, string, []string) wire.ToolResult
	Service             Service
	MissionID           func() string
	EnforceMission      func(string) error
	RequireSession      func(wire.CommonMutatingInput) error
	ObservationProducer func() (ledger.Producer, string, error)
	Liquid2Connector    func() (liquid2source.Liquid2SourceConnector, bool)
	ConfluenceFactory   func() Factory
	Decode              func(json.RawMessage, any) error
	NormalizeInput      func(wire.CommonMutatingInput) (wire.CommonMutatingInput, ledger.Producer, error)
	ErrorResult         func(string, string, string, string, bool, []string) wire.ToolResult
	ErrorFromErr        func(string, string, error, []string) wire.ToolResult
	NewID               func(string) string
	ValidateID          func(string, string) error
	ArtifactOutput      func(artifactcontract.Raw) wire.RawArtifactOutput
}
