package reportil

import (
	"context"
	"encoding/json"
	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
	"github.com/c86j224s/liquid2/plasma/internal/reportilsource"
	"sync"
)

// SourceState is shared with editorial/document operations under the same mutex.
type SourceState struct {
	ReadBytes                 int
	ReadBySource, NextOffsets map[string]int
	Complete                  map[string]bool
	Quotes                    map[string]reportilcontract.SourceQuoteReceipt
	QuoteCount                int
}
type EditorialState struct {
	UsedAnchors      map[string]bool
	ReviewNextOffset int
	ReviewComplete   bool
	Workspaces       map[string]*EditorialMemoryWorkspace
}
type ArtifactService interface {
	ListEvents(context.Context, string) ([]ledger.Event, error)
	GetRawArtifact(context.Context, string) (artifact.Raw, error)
	CreateRawArtifact(context.Context, artifact.CreateRequest) (artifact.Raw, error)
}
type DocumentState struct {
	Workspaces           map[string]*DocumentWorkspace
	PlanReviewComplete   bool
	PlanReviewNextOffset int
}
type SourceHandler struct {
	Documents *DocumentState

	Editorial    *EditorialState
	Artifacts    ArtifactService
	NewID        func(string) string
	ErrorFromErr func(string, string, error, []string) wire.ToolResult

	Mu                   *sync.Mutex
	State                *SourceState
	MissionID, SessionID func() string
	AccessBinding        func() (reportilcontract.SourceAccessBinding, error)
	ReadableSource       func(context.Context, reportilcontract.SourceAccessBinding, reportilcontract.SourceCatalogEntry) (reportilsource.Readable, error)
	Decode               func(json.RawMessage, any) error
	ErrorResult          func(string, string, string, string, bool, []string) wire.ToolResult
	SHA256               func([]byte) string
	BoundedRead          func([]byte, int, int, int) (string, int, int, bool, error)
}

// State groups the mutually dependent IL workspaces; all access uses the supplied shared mutex.
type State struct {
	Source    SourceState
	Editorial EditorialState
	Documents DocumentState
}

func NewState() *State {
	return &State{Source: SourceState{ReadBySource: map[string]int{}, NextOffsets: map[string]int{}, Complete: map[string]bool{}, Quotes: map[string]reportilcontract.SourceQuoteReceipt{}}, Editorial: EditorialState{UsedAnchors: map[string]bool{}, Workspaces: map[string]*EditorialMemoryWorkspace{}}, Documents: DocumentState{Workspaces: map[string]*DocumentWorkspace{}}}
}
