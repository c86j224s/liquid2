package reportexperiment

import (
	"context"
	"fmt"
	"sync"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportworkflow"
)

// NodeObservation은 제품 workflow observer에서 받은 content-free node 실행 기록이다.
type NodeObservation struct {
	Index      int    `json:"index"`
	NodeID     string `json:"node_id"`
	StartedAt  string `json:"started_at"`
	DurationMS int64  `json:"duration_ms,omitempty"`
	Attempt    int    `json:"attempt"`
	Replay     bool   `json:"replay"`
	Failed     bool   `json:"failed"`
	Terminal   bool   `json:"terminal"`
}

type nodeRecorder struct {
	mu     sync.Mutex
	values []NodeObservation
}

func (recorder *nodeRecorder) Observe(observation reportworkflow.NodeObservation) {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	startedAt := observation.StartedAt.UTC().Format(timeFormat)
	terminal := observation.DurationMS > 0 || observation.Failed || observation.Replay
	if !terminal {
		for _, existing := range recorder.values {
			if existing.NodeID == observation.NodeID && existing.StartedAt == startedAt {
				terminal = true
				break
			}
		}
	}
	recorder.values = append(recorder.values, NodeObservation{
		Index: len(recorder.values) + 1, NodeID: observation.NodeID, StartedAt: startedAt,
		DurationMS: observation.DurationMS, Attempt: observation.Attempt, Replay: observation.Replay, Failed: observation.Failed,
		Terminal: terminal,
	})
}

func (recorder *nodeRecorder) Snapshot() []NodeObservation {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	return append([]NodeObservation(nil), recorder.values...)
}

// AgentRequestObservation은 provider 요청의 prompt-free 실행 receipt다.
type AgentRequestObservation struct {
	Index                             int      `json:"index"`
	Stage                             string   `json:"stage,omitempty"`
	PromptSHA256                      string   `json:"prompt_sha256"`
	Tools                             []string `json:"tools,omitempty"`
	ReplaceMCPTools                   bool     `json:"replace_mcp_tools"`
	Model                             string   `json:"model,omitempty"`
	ReasoningEffort                   string   `json:"reasoning_effort,omitempty"`
	MissionID                         string   `json:"mission_id,omitempty"`
	ToolSessionID                     string   `json:"tool_session_id,omitempty"`
	PreviousSessionID                 string   `json:"previous_session_id,omitempty"`
	DisableTools                      bool     `json:"disable_tools,omitempty"`
	IgnoreUserConfig                  bool     `json:"ignore_user_config,omitempty"`
	ProviderSessionID                 string   `json:"provider_session_id,omitempty"`
	PreviousProviderSessionID         string   `json:"previous_provider_session_id,omitempty"`
	ForkSourceAgentSessionID          string   `json:"fork_source_agent_session_id,omitempty"`
	LongFormFinalize                  bool     `json:"long_form_finalize"`
	LongFormFinalizeToolSessionID     string   `json:"long_form_finalize_tool_session_id,omitempty"`
	LongFormFinalizeProviderSessionID string   `json:"long_form_finalize_provider_session_id,omitempty"`
}

type requestRecorder struct {
	delegate agentexec.AgentExecutor
	mu       sync.Mutex
	values   []AgentRequestObservation
}

func newRequestRecorder(delegate agentexec.AgentExecutor) *requestRecorder {
	return &requestRecorder{delegate: delegate}
}

func (recorder *requestRecorder) Run(ctx context.Context, req agentexec.AgentRequest) (agentexec.AgentResult, error) {
	recorder.record(req)
	return recorder.delegate.Run(ctx, req)
}

func (recorder *requestRecorder) ForkSession(ctx context.Context, sourceSessionID string) (agentexec.AgentSessionForkResult, error) {
	forker, ok := recorder.delegate.(agentexec.AgentSessionForker)
	if !ok {
		return agentexec.AgentSessionForkResult{}, fmt.Errorf("%w: final edit requires a fork-capable executor", producterror.ErrInvalidInput)
	}
	return forker.ForkSession(ctx, sourceSessionID)
}

func (recorder *requestRecorder) Snapshot() []AgentRequestObservation {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	return append([]AgentRequestObservation(nil), recorder.values...)
}

func (recorder *requestRecorder) record(req agentexec.AgentRequest) {
	observation := AgentRequestObservation{
		PromptSHA256:      bytesSHA256([]byte(req.Prompt)),
		Tools:             append([]string(nil), req.ExtraMCPTools...),
		ReplaceMCPTools:   req.ReplaceMCPTools,
		Model:             req.Model,
		ReasoningEffort:   req.ReasoningEffort,
		MissionID:         req.MissionID,
		ToolSessionID:     req.ToolSessionID,
		PreviousSessionID: req.PreviousSessionID,
		DisableTools:      req.DisableTools,
		IgnoreUserConfig:  req.IgnoreUserConfig,
	}
	if req.FinalEditStage != nil {
		observation.Stage = req.FinalEditStage.Stage
		observation.ProviderSessionID = req.FinalEditStage.ProviderSessionID
		observation.PreviousProviderSessionID = req.FinalEditStage.PreviousProviderSessionID
		observation.ForkSourceAgentSessionID = req.FinalEditStage.ForkSourceAgentSessionID
	}
	if req.LongFormFinalize != nil {
		observation.LongFormFinalize = true
		observation.LongFormFinalizeToolSessionID = req.LongFormFinalize.ToolSessionID
		observation.LongFormFinalizeProviderSessionID = req.LongFormFinalize.ProviderSessionID
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	observation.Index = len(recorder.values) + 1
	recorder.values = append(recorder.values, observation)
}
