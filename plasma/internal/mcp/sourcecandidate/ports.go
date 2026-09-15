package sourcecandidate

import (
	"context"
	"encoding/json"
	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"github.com/c86j224s/liquid2/plasma/internal/sourcecandidates"
	"github.com/c86j224s/liquid2/plasma/internal/sourceretrieval"
)

type Service interface {
	sourcecandidates.Store
	GetRawArtifact(context.Context, string) (artifact.Raw, error)
}
type Handler struct {
	Service                           Service
	CurrentUserEventID, AgentExecutor func() string
	RequireSession                    func(wire.CommonMutatingInput) error
	EnforceMission                    func(string) error
	Fetcher                           func() func(context.Context, string) (sourceretrieval.Fetched, error)
	Decode                            func(json.RawMessage, any) error
	NormalizeInput                    func(wire.CommonMutatingInput) (wire.CommonMutatingInput, ledger.Producer, error)
	ErrorResult                       func(string, string, string, string, bool, []string) wire.ToolResult
	ErrorFromErr                      func(string, string, error, []string) wire.ToolResult
	NewID                             func(string) string
	ValidateID                        func(string, string) error
	ArtifactOutput                    func(artifact.Raw) wire.RawArtifactOutput
	BoundedRead                       func([]byte, int, int) (string, int, int, bool, error)
}
