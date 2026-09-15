package reportfinaledit

import (
	"encoding/json"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"sync"
	"time"
)

type State struct {
	LegacyDrafts     map[string]*LongFormEditDraft
	StageDrafts      map[string]*LongFormStageEditDraft
	ValidationDrafts map[string]*ReadOnlyValidationDraft
}
type Handler struct {
	State                  *State
	Mu                     *sync.Mutex
	Service                reporting.FinalEditStageStore
	StageBinding           func() reporting.FinalEditStageBinding
	FinalizeBinding        func() reporting.LongFormFinalizeBinding
	RequireStageBinding    func(wire.CommonMutatingInput, string) (reporting.FinalEditStageBinding, error)
	RequireLongFormBinding func(wire.CommonMutatingInput) (reporting.LongFormFinalizeBinding, error)
	FinalizeAvailable      func(reporting.LongFormFinalizeBinding) bool
	MissionID              func() string
	Decode                 func(json.RawMessage, any) error
	NormalizeInput         func(wire.CommonMutatingInput) (wire.CommonMutatingInput, ledger.Producer, error)
	ErrorResult            func(string, string, string, string, bool, []string) wire.ToolResult
	ErrorFromErr           func(string, string, error, []string) wire.ToolResult
	ValidateID             func(string, string) error
	NewID                  func(string) string
}

func nowUTC() time.Time { return time.Now().UTC() }
