package reportparts

import (
	"context"
	"encoding/json"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"sync"
	"time"
)

type State struct {
	EditDrafts     map[string]*PartEditDraft
	AssemblyDrafts map[string]*PartAssemblyDraft
}
type Handler struct {
	State   *State
	Mu      *sync.Mutex
	Service interface {
		reporting.PartEditStore
		AppendEvent(context.Context, ledger.AppendRequest) (ledger.Event, error)
	}
	AssemblyBinding         func(wire.CommonMutatingInput) (reporting.PartAssemblyBinding, error)
	CurrentAssemblyBinding  func() reporting.PartAssemblyBinding
	ValidateAssemblyBinding func(reporting.PartAssemblyBinding) error
	RequireSession          func(wire.CommonMutatingInput) error
	EditBinding             func(wire.CommonMutatingInput) (reporting.PartEditBinding, error)
	Decode                  func(json.RawMessage, any) error
	NormalizeInput          func(wire.CommonMutatingInput) (wire.CommonMutatingInput, ledger.Producer, error)
	ErrorResult             func(string, string, string, string, bool, []string) wire.ToolResult
	ErrorFromErr            func(string, string, error, []string) wire.ToolResult
	ValidateID              func(string, string) error
	NewID                   func(string) string
}

func nowUTC() time.Time { return time.Now().UTC() }
