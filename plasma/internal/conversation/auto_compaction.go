package conversation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

const (
	autoCompactionPhaseCompactCall    = "compact_call"
	autoCompactionPhaseCompactSession = "compact_session"
	autoCompactionPhaseCompactAppend  = "compact_append"
	autoCompactionPhaseRetryCall      = "retry_call"
	autoCompactionPhaseRetrySession   = "retry_session"
	autoCompactionPhaseCompleted      = "completed"
)

// AutoCompactionRequest는 기존 세션의 자동 압축과 원래 턴 재시도를 묶는 실행 요청이다.
// 이 계약은 Web의 자동 턴 복구만 소유하며 workflow의 별도 fallback 규칙은 변경하지 않는다.
type AutoCompactionRequest struct {
	CompactRequest    agentexec.AgentRequest
	RetryRequest      agentexec.AgentRequest
	PreviousSessionID string
}

// AutoCompactionCallbacks는 각 단계의 provider 실행과 압축 이벤트 append를 호출자에게 위임한다.
// 콜백은 실행되는 단계에서 반드시 제공되어야 하며, 이 함수는 nil fallback을 만들지 않는다.
type AutoCompactionCallbacks struct {
	RunCompact      func(context.Context, agentexec.AgentRequest) (agentexec.AgentResult, error)
	RunRetry        func(context.Context, agentexec.AgentRequest) (agentexec.AgentResult, error)
	AppendCompacted func(context.Context, agentexec.AgentResult, int64) (ledger.Event, error)
}

// AutoCompactionResult는 자동 압축 시도의 결과와 단계별 시간·세션 정보를 담는다.
type AutoCompactionResult struct {
	CompactResult     agentexec.AgentResult
	RetryResult       agentexec.AgentResult
	CompactEvent      ledger.Event
	CompactDurationMS int64
	RetryDurationMS   int64
	ReturnedSessionID string
	Phase             string
}

// ShouldAutoCompactAfterError는 기존 세션에서 컨텍스트 창 고갈 오류가 발생했는지 판정한다.
func ShouldAutoCompactAfterError(previousSessionID string, err error, result agentexec.AgentResult) bool {
	if strings.TrimSpace(previousSessionID) == "" || err == nil {
		return false
	}
	text := strings.ToLower(err.Error() + "\n" + result.Log)
	return strings.Contains(text, "ran out of room in the model's context window")
}

// ValidateSameSessionResult는 세션 재개 결과를 엄격하게 검증한다.
// 새 세션을 허용하지 않으며, 빈 세션과 다른 세션에는 동일한 product error 계약을 반환한다.
func ValidateSameSessionResult(result agentexec.AgentResult, previousSessionID string) (agentexec.AgentResult, error) {
	previousSessionID = strings.TrimSpace(previousSessionID)
	result.SessionID = strings.TrimSpace(result.SessionID)
	if previousSessionID == "" {
		if result.SessionID == "" {
			return result, invalidSessionResultError("agent did not return a session id")
		}
		return result, nil
	}
	if result.SessionID == "" {
		return result, invalidSessionResultError("agent did not return a session id for resumed session")
	}
	if result.SessionID != previousSessionID {
		result.SessionID = ""
		return result, invalidSessionResultError("agent returned a different session id")
	}
	return result, nil
}

func invalidSessionResultError(message string) error {
	return fmt.Errorf("%w: %s", producterror.ErrInvalidInput, message)
}

// RunAutoCompaction runs exactly one compact call, append, and retry call in order.
// It returns the original callback error and records the phase at which that error occurred.
func RunAutoCompaction(ctx context.Context, req AutoCompactionRequest, callbacks AutoCompactionCallbacks) (AutoCompactionResult, error) {
	result := AutoCompactionResult{Phase: autoCompactionPhaseCompactCall}
	compactStarted := time.Now()
	compactResult, err := callbacks.RunCompact(ctx, req.CompactRequest)
	result.CompactDurationMS = time.Since(compactStarted).Milliseconds()
	result.CompactResult = compactResult
	result.ReturnedSessionID = strings.TrimSpace(compactResult.SessionID)
	if err != nil {
		return result, err
	}

	result.Phase = autoCompactionPhaseCompactSession
	compactResult, err = ValidateSameSessionResult(compactResult, req.PreviousSessionID)
	result.CompactResult = compactResult
	if err != nil {
		return result, err
	}

	result.Phase = autoCompactionPhaseCompactAppend
	compactEvent, err := callbacks.AppendCompacted(ctx, compactResult, result.CompactDurationMS)
	result.CompactEvent = compactEvent
	if err != nil {
		return result, err
	}

	result.Phase = autoCompactionPhaseRetryCall
	retryStarted := time.Now()
	retryResult, err := callbacks.RunRetry(ctx, req.RetryRequest)
	result.RetryDurationMS = time.Since(retryStarted).Milliseconds()
	result.RetryResult = retryResult
	result.ReturnedSessionID = strings.TrimSpace(retryResult.SessionID)
	if err != nil {
		return result, err
	}

	result.Phase = autoCompactionPhaseRetrySession
	retryResult, err = ValidateSameSessionResult(retryResult, req.PreviousSessionID)
	result.RetryResult = retryResult
	if err != nil {
		return result, err
	}

	result.Phase = autoCompactionPhaseCompleted
	return result, nil
}
