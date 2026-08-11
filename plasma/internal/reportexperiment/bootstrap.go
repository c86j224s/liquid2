package reportexperiment

import (
	"context"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

const (
	bootstrapResponseToken = "PLASMA_REPORT_EXPERIMENT_BOOTSTRAP_OK"
	bootstrapPrompt        = "Reply with exactly this fixed token and no punctuation or extra words:\n" + bootstrapResponseToken
)

// BootstrapReceipt는 finalization-only harness가 실제 Codex plan session을 확보한 증거다.
//
// Prompt 본문과 provider raw response는 archive에 남기지 않는다. 이후 seed와
// PrefixOutput은 SessionID를 제품 plan session lineage처럼 사용하지만, 이 bootstrap은
// 제품 planning stage를 대체하지 않는다.
type BootstrapReceipt struct {
	PromptSHA256        string `json:"prompt_sha256"`
	SessionID           string `json:"session_id"`
	ResponseTokenSHA256 string `json:"response_token_sha256"`
}

// bootstrapPlanSession은 실제 Codex 실행으로 fork 가능한 fresh session ID만 확보한다.
//
// 이 요청은 tools-disabled여야 하며, 반환 텍스트는 고정 토큰으로만 검증한다.
// 호출자는 이 세션 ID를 finalization-only seed lineage에 사용하되 제품 planning
// stage가 실행된 것으로 해석하면 안 된다.
func bootstrapPlanSession(ctx context.Context, executor agentexec.AgentExecutor, model, reasoningEffort string) (BootstrapReceipt, error) {
	receipt := BootstrapReceipt{
		PromptSHA256:        bytesSHA256([]byte(bootstrapPrompt)),
		ResponseTokenSHA256: bytesSHA256([]byte(bootstrapResponseToken)),
	}
	result, err := executor.Run(ctx, agentexec.AgentRequest{
		UserText:         "bootstrap fixed-Part finalization plan session",
		Prompt:           bootstrapPrompt,
		Model:            model,
		ReasoningEffort:  reasoningEffort,
		AgentExecutor:    executorCodex,
		DisableTools:     true,
		IgnoreUserConfig: true,
		ReplaceMCPTools:  true,
	})
	if err != nil {
		return BootstrapReceipt{}, err
	}
	if strings.TrimSpace(result.Text) != bootstrapResponseToken {
		return BootstrapReceipt{}, fmt.Errorf("%w: Codex bootstrap response did not match the fixed token", producterror.ErrConflict)
	}
	receipt.SessionID = strings.TrimSpace(result.SessionID)
	if receipt.SessionID == "" {
		return BootstrapReceipt{}, fmt.Errorf("%w: Codex bootstrap did not return a provider session ID", producterror.ErrConflict)
	}
	return receipt, nil
}
