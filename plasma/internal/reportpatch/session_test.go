package reportpatch

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
)

type fakeExecutor struct{}

func (fakeExecutor) Run(context.Context, agentexec.AgentRequest) (agentexec.AgentResult, error) {
	return agentexec.AgentResult{}, nil
}

type fakeForkExecutor struct {
	fakeExecutor
	readyErr      error
	forkErr       error
	forkSessionID string
	forkSourceID  string
	forkCalls     []string
	readyCalls    []string
}

func (executor *fakeForkExecutor) CheckForkSession(_ context.Context, sourceSessionID string) error {
	executor.readyCalls = append(executor.readyCalls, sourceSessionID)
	return executor.readyErr
}

func (executor *fakeForkExecutor) ForkSession(_ context.Context, sourceSessionID string) (agentexec.AgentSessionForkResult, error) {
	executor.forkCalls = append(executor.forkCalls, sourceSessionID)
	if executor.forkErr != nil {
		return agentexec.AgentSessionForkResult{}, executor.forkErr
	}
	sessionID := strings.TrimSpace(executor.forkSessionID)
	if sessionID == "" {
		sessionID = "forked-session"
	}
	return agentexec.AgentSessionForkResult{
		SessionID:       sessionID,
		SourceSessionID: executor.forkSourceID,
	}, nil
}

func TestSelectSessionExplicitSameSession(t *testing.T) {
	selection, err := SelectSession(context.Background(), fakeExecutor{}, " source-session ", " same_session ")
	if err != nil {
		t.Fatalf("SelectSession returned error: %v", err)
	}
	expected := PatchSessionSelection{
		SessionID:                    "source-session",
		PreviousAgentSessionID:       "source-session",
		ReportSessionPolicy:          reportexecution.SessionPolicySameSession,
		ReportSessionPolicySelection: reportexecution.SessionPolicySelectionExplicitSameSession,
		SessionChainKind:             "same_report_session_patch",
	}
	if !reflect.DeepEqual(selection, expected) {
		t.Fatalf("selection mismatch\n got: %#v\nwant: %#v", selection, expected)
	}
}

func TestSelectSessionExplicitIsolatedForkSuccess(t *testing.T) {
	executor := &fakeForkExecutor{forkSessionID: "forked-session", forkSourceID: "source-session"}
	selection, err := SelectSession(context.Background(), executor, "source-session", "isolated_fork")
	if err != nil {
		t.Fatalf("SelectSession returned error: %v", err)
	}
	expected := PatchSessionSelection{
		SessionID:                    "forked-session",
		PreviousAgentSessionID:       "forked-session",
		ForkSourceAgentSessionID:     "source-session",
		ReportSessionPolicy:          reportexecution.SessionPolicyIsolatedFork,
		ReportSessionPolicySelection: reportexecution.SessionPolicySelectionExplicitIsolatedFork,
		SessionChainKind:             "isolated_fork_report_patch",
	}
	if !reflect.DeepEqual(selection, expected) {
		t.Fatalf("selection mismatch\n got: %#v\nwant: %#v", selection, expected)
	}
	if !reflect.DeepEqual(executor.readyCalls, []string{"source-session"}) || !reflect.DeepEqual(executor.forkCalls, []string{"source-session"}) {
		t.Fatalf("unexpected readiness/fork calls: ready=%v fork=%v", executor.readyCalls, executor.forkCalls)
	}
}

func TestSelectSessionExplicitFreshSessionRejectedWithoutForkSideEffects(t *testing.T) {
	executor := &fakeForkExecutor{forkSessionID: "forked-session", forkSourceID: "source-session"}
	_, err := SelectSession(context.Background(), executor, "source-session", "fresh_session")
	if err == nil {
		t.Fatalf("SelectSession returned nil error")
	}
	if !errors.Is(err, producterror.ErrInvalidInput) || !strings.Contains(err.Error(), "automatic-only") {
		t.Fatalf("expected automatic-only invalid input, got %v", err)
	}
	if len(executor.readyCalls) != 0 || len(executor.forkCalls) != 0 {
		t.Fatalf("fresh_session must not check readiness or fork: ready=%v fork=%v", executor.readyCalls, executor.forkCalls)
	}
}

func TestSelectSessionExplicitIsolatedForkErrors(t *testing.T) {
	tests := []struct {
		name     string
		executor agentexec.AgentExecutor
		contains string
	}{
		{
			name:     "no forker",
			executor: fakeExecutor{},
			contains: "requires a forkable executor",
		},
		{
			name:     "readiness fails",
			executor: &fakeForkExecutor{readyErr: errors.New("not ready")},
			contains: "is not ready for fork",
		},
		{
			name:     "fork fails",
			executor: &fakeForkExecutor{forkErr: errors.New("fork unavailable")},
			contains: "report patch session fork failed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := SelectSession(context.Background(), tt.executor, "source-session", "isolated_fork")
			if err == nil {
				t.Fatalf("SelectSession returned nil error")
			}
			if !strings.Contains(err.Error(), tt.contains) {
				t.Fatalf("error %q does not contain %q", err.Error(), tt.contains)
			}
			if tt.name != "fork fails" && !errors.Is(err, producterror.ErrInvalidInput) {
				t.Fatalf("expected invalid input classification, got %v", err)
			}
		})
	}
}

func TestSelectSessionAutomaticFallbacks(t *testing.T) {
	tests := []struct {
		name      string
		executor  agentexec.AgentExecutor
		selection string
	}{
		{
			name:      "no forker",
			executor:  fakeExecutor{},
			selection: reportexecution.SessionPolicySelectionAutoSameSessionNoForker,
		},
		{
			name:      "readiness fails",
			executor:  &fakeForkExecutor{readyErr: errors.New("not ready")},
			selection: reportexecution.SessionPolicySelectionAutoSameSessionForkFailed,
		},
		{
			name:      "fork fails",
			executor:  &fakeForkExecutor{forkErr: errors.New("fork unavailable")},
			selection: reportexecution.SessionPolicySelectionAutoSameSessionForkFailed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selection, err := SelectSession(context.Background(), tt.executor, "source-session", "")
			if err != nil {
				t.Fatalf("SelectSession returned error: %v", err)
			}
			expected := PatchSessionSelection{
				SessionID:                    "source-session",
				PreviousAgentSessionID:       "source-session",
				ReportSessionPolicy:          reportexecution.SessionPolicySameSession,
				ReportSessionPolicySelection: tt.selection,
				SessionChainKind:             "same_report_session_patch",
			}
			if !reflect.DeepEqual(selection, expected) {
				t.Fatalf("selection mismatch\n got: %#v\nwant: %#v", selection, expected)
			}
		})
	}
}

func TestSelectSessionAutomaticForkSuccess(t *testing.T) {
	executor := &fakeForkExecutor{forkSessionID: "forked-session"}
	selection, err := SelectSession(context.Background(), executor, "source-session", "")
	if err != nil {
		t.Fatalf("SelectSession returned error: %v", err)
	}
	expected := PatchSessionSelection{
		SessionID:                    "forked-session",
		PreviousAgentSessionID:       "forked-session",
		ForkSourceAgentSessionID:     "source-session",
		ReportSessionPolicy:          reportexecution.SessionPolicyIsolatedFork,
		ReportSessionPolicySelection: reportexecution.SessionPolicySelectionAutoIsolatedFork,
		SessionChainKind:             "isolated_fork_report_patch",
	}
	if !reflect.DeepEqual(selection, expected) {
		t.Fatalf("selection mismatch\n got: %#v\nwant: %#v", selection, expected)
	}
}

func TestSelectSessionBlankSourceSessionError(t *testing.T) {
	_, err := SelectSession(context.Background(), fakeExecutor{}, "   ", "")
	if err == nil {
		t.Fatalf("SelectSession returned nil error")
	}
	if !errors.Is(err, producterror.ErrInvalidInput) || !strings.Contains(err.Error(), "requires a previous report session") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSelectSessionLiteralAutoUnsupported(t *testing.T) {
	_, err := SelectSession(context.Background(), fakeExecutor{}, "source-session", "auto")
	if err == nil {
		t.Fatalf("SelectSession returned nil error")
	}
	if !errors.Is(err, producterror.ErrInvalidInput) || !strings.Contains(err.Error(), "unsupported report session policy") {
		t.Fatalf("unexpected error: %v", err)
	}
}
