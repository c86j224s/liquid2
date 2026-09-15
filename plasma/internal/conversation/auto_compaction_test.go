package conversation

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func TestRunAutoCompactionForwardsRequestsInOrderAndRetainsMetadata(t *testing.T) {
	compactReq := agentexec.AgentRequest{UserText: "compact", MissionID: "mis", Compaction: true}
	retryReq := agentexec.AgentRequest{UserText: "ask", MissionID: "mis"}
	compactResult := agentexec.AgentResult{Text: "summary", SessionID: "ses", Resumed: true, Log: "compact log"}
	retryResult := agentexec.AgentResult{Text: "answer", SessionID: "ses", Resumed: true, Log: "retry log"}
	var calls []string
	var gotCompact, gotRetry agentexec.AgentRequest
	var gotAppend agentexec.AgentResult
	var gotDuration int64
	wantEvent := ledger.Event{EventID: "evt_compact", MissionID: "mis", EventType: "turn.agent.compacted"}
	got, err := RunAutoCompaction(context.Background(), AutoCompactionRequest{CompactRequest: compactReq, RetryRequest: retryReq, PreviousSessionID: "ses"}, AutoCompactionCallbacks{
		RunCompact: func(_ context.Context, req agentexec.AgentRequest) (agentexec.AgentResult, error) {
			calls = append(calls, "compact")
			gotCompact = req
			return compactResult, nil
		},
		AppendCompacted: func(_ context.Context, result agentexec.AgentResult, durationMS int64) (ledger.Event, error) {
			calls = append(calls, "append")
			gotAppend = result
			gotDuration = durationMS
			return wantEvent, nil
		},
		RunRetry: func(_ context.Context, req agentexec.AgentRequest) (agentexec.AgentResult, error) {
			calls = append(calls, "retry")
			gotRetry = req
			return retryResult, nil
		},
	})
	if err != nil || got.Phase != autoCompactionPhaseCompleted {
		t.Fatalf("RunAutoCompaction() = %#v, %v", got, err)
	}
	if !reflect.DeepEqual(calls, []string{"compact", "append", "retry"}) || !reflect.DeepEqual(gotCompact, compactReq) || !reflect.DeepEqual(gotRetry, retryReq) {
		t.Fatalf("unexpected call forwarding: calls=%v compact=%#v retry=%#v", calls, gotCompact, gotRetry)
	}
	if !reflect.DeepEqual(got.CompactEvent, wantEvent) || !reflect.DeepEqual(gotAppend, compactResult) || !reflect.DeepEqual(got.RetryResult, retryResult) || gotDuration < 0 || got.CompactDurationMS < 0 || got.RetryDurationMS < 0 {
		t.Fatalf("metadata/duration not retained: %#v duration=%d append=%#v", got, gotDuration, gotAppend)
	}
}

func TestRunAutoCompactionReturnsOriginalErrorAndPhase(t *testing.T) {
	tests := []struct {
		name      string
		phase     string
		configure func(*AutoCompactionCallbacks)
	}{
		{name: "compact call", phase: autoCompactionPhaseCompactCall, configure: func(callbacks *AutoCompactionCallbacks) {
			callbacks.RunCompact = func(context.Context, agentexec.AgentRequest) (agentexec.AgentResult, error) {
				return agentexec.AgentResult{}, errors.New("compact failed")
			}
		}},
		{name: "compact session", phase: autoCompactionPhaseCompactSession, configure: func(callbacks *AutoCompactionCallbacks) {
			callbacks.RunCompact = func(context.Context, agentexec.AgentRequest) (agentexec.AgentResult, error) {
				return agentexec.AgentResult{SessionID: "other"}, nil
			}
		}},
		{name: "append", phase: autoCompactionPhaseCompactAppend, configure: func(callbacks *AutoCompactionCallbacks) {
			callbacks.RunCompact = func(context.Context, agentexec.AgentRequest) (agentexec.AgentResult, error) {
				return agentexec.AgentResult{SessionID: "ses"}, nil
			}
			callbacks.AppendCompacted = func(context.Context, agentexec.AgentResult, int64) (ledger.Event, error) {
				return ledger.Event{}, errors.New("append failed")
			}
		}},
		{name: "retry call", phase: autoCompactionPhaseRetryCall, configure: func(callbacks *AutoCompactionCallbacks) {
			callbacks.RunCompact = func(context.Context, agentexec.AgentRequest) (agentexec.AgentResult, error) {
				return agentexec.AgentResult{SessionID: "ses"}, nil
			}
			callbacks.AppendCompacted = func(context.Context, agentexec.AgentResult, int64) (ledger.Event, error) {
				return ledger.Event{EventID: "evt"}, nil
			}
			callbacks.RunRetry = func(context.Context, agentexec.AgentRequest) (agentexec.AgentResult, error) {
				return agentexec.AgentResult{}, errors.New("retry failed")
			}
		}},
		{name: "retry session", phase: autoCompactionPhaseRetrySession, configure: func(callbacks *AutoCompactionCallbacks) {
			callbacks.RunCompact = func(context.Context, agentexec.AgentRequest) (agentexec.AgentResult, error) {
				return agentexec.AgentResult{SessionID: "ses"}, nil
			}
			callbacks.AppendCompacted = func(context.Context, agentexec.AgentResult, int64) (ledger.Event, error) {
				return ledger.Event{EventID: "evt"}, nil
			}
			callbacks.RunRetry = func(context.Context, agentexec.AgentRequest) (agentexec.AgentResult, error) {
				return agentexec.AgentResult{SessionID: "other"}, nil
			}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			callbacks := AutoCompactionCallbacks{
				RunCompact: func(context.Context, agentexec.AgentRequest) (agentexec.AgentResult, error) {
					return agentexec.AgentResult{SessionID: "ses"}, nil
				},
				RunRetry: func(context.Context, agentexec.AgentRequest) (agentexec.AgentResult, error) {
					return agentexec.AgentResult{SessionID: "ses"}, nil
				},
				AppendCompacted: func(context.Context, agentexec.AgentResult, int64) (ledger.Event, error) {
					return ledger.Event{EventID: "evt"}, nil
				},
			}
			test.configure(&callbacks)
			got, err := RunAutoCompaction(context.Background(), AutoCompactionRequest{PreviousSessionID: "ses"}, callbacks)
			if err == nil || got.Phase != test.phase || !strings.Contains(err.Error(), "failed") && !strings.Contains(err.Error(), "different session") {
				t.Fatalf("RunAutoCompaction() = %#v, %v; want phase %s", got, err, test.phase)
			}
		})
	}
}

func TestRunAutoCompactionReportsCancellationPhase(t *testing.T) {
	for _, test := range []struct {
		name  string
		phase string
		setup func(*AutoCompactionCallbacks)
	}{
		{name: "compact", phase: autoCompactionPhaseCompactCall, setup: func(callbacks *AutoCompactionCallbacks) {
			callbacks.RunCompact = func(context.Context, agentexec.AgentRequest) (agentexec.AgentResult, error) {
				return agentexec.AgentResult{}, context.Canceled
			}
		}},
		{name: "retry", phase: autoCompactionPhaseRetryCall, setup: func(callbacks *AutoCompactionCallbacks) {
			callbacks.RunRetry = func(context.Context, agentexec.AgentRequest) (agentexec.AgentResult, error) {
				return agentexec.AgentResult{}, context.Canceled
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			callbacks := AutoCompactionCallbacks{
				RunCompact: func(context.Context, agentexec.AgentRequest) (agentexec.AgentResult, error) {
					return agentexec.AgentResult{SessionID: "ses"}, nil
				},
				AppendCompacted: func(context.Context, agentexec.AgentResult, int64) (ledger.Event, error) {
					return ledger.Event{EventID: "evt"}, nil
				},
				RunRetry: func(context.Context, agentexec.AgentRequest) (agentexec.AgentResult, error) {
					return agentexec.AgentResult{SessionID: "ses"}, nil
				},
			}
			test.setup(&callbacks)
			got, err := RunAutoCompaction(context.Background(), AutoCompactionRequest{PreviousSessionID: "ses"}, callbacks)
			if !errors.Is(err, context.Canceled) || got.Phase != test.phase {
				t.Fatalf("RunAutoCompaction() = %#v, %v; want phase %s and cancellation", got, err, test.phase)
			}
		})
	}
}

func TestRunAutoCompactionDoesNotRetryAfterAppendFailure(t *testing.T) {
	retries := 0
	got, err := RunAutoCompaction(context.Background(), AutoCompactionRequest{PreviousSessionID: "ses"}, AutoCompactionCallbacks{
		RunCompact: func(context.Context, agentexec.AgentRequest) (agentexec.AgentResult, error) {
			return agentexec.AgentResult{SessionID: "ses"}, nil
		},
		AppendCompacted: func(context.Context, agentexec.AgentResult, int64) (ledger.Event, error) {
			return ledger.Event{}, errors.New("append failed")
		},
		RunRetry: func(context.Context, agentexec.AgentRequest) (agentexec.AgentResult, error) {
			retries++
			return agentexec.AgentResult{SessionID: "ses"}, nil
		},
	})
	if err == nil || got.Phase != autoCompactionPhaseCompactAppend || retries != 0 {
		t.Fatalf("append failure = %#v, %v; retries=%d", got, err, retries)
	}
}

func TestValidateSameSessionResultUsesProductErrorContract(t *testing.T) {
	for _, result := range []agentexec.AgentResult{{}, {SessionID: "other"}} {
		_, err := ValidateSameSessionResult(result, "ses")
		if !errors.Is(err, producterror.ErrInvalidInput) {
			t.Fatalf("ValidateSameSessionResult(%#v) error=%v", result, err)
		}
	}
	if _, err := ValidateSameSessionResult(agentexec.AgentResult{}, ""); !errors.Is(err, producterror.ErrInvalidInput) {
		t.Fatalf("empty new session error=%v", err)
	}
}

func TestShouldAutoCompactAfterError(t *testing.T) {
	if !ShouldAutoCompactAfterError("ses", errors.New("provider failed"), agentexec.AgentResult{Log: "ran out of room in the model's context window"}) {
		t.Fatal("expected context exhaustion to trigger compaction")
	}
	if ShouldAutoCompactAfterError("", errors.New("ran out of room in the model's context window"), agentexec.AgentResult{}) {
		t.Fatal("empty previous session must not trigger compaction")
	}
	started := time.Now()
	if time.Since(started) < 0 {
		t.Fatal("clock moved backwards")
	}
}
