package web

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

type sectionFanoutConcurrencyAgent struct {
	active  atomic.Int64
	max     atomic.Int64
	started atomic.Int64
	reached chan struct{}
	release chan struct{}
	once    sync.Once
}

func (agent *sectionFanoutConcurrencyAgent) Run(ctx context.Context, req AgentRequest) (AgentResult, error) {
	active := agent.active.Add(1)
	defer agent.active.Add(-1)
	for {
		current := agent.max.Load()
		if active <= current || agent.max.CompareAndSwap(current, active) {
			break
		}
	}
	if agent.started.Add(1) == int64(sectionFanoutWorkerLimit) {
		agent.once.Do(func() { close(agent.reached) })
	}
	select {
	case <-agent.release:
	case <-ctx.Done():
		return AgentResult{Log: "context canceled"}, ctx.Err()
	}
	return AgentResult{Text: fmt.Sprintf("section body for %s", req.PreviousSessionID), SessionID: req.PreviousSessionID, Resumed: req.PreviousSessionID != ""}, nil
}

func TestRunSectionFanoutTasksAllowsEightConcurrentSectionWorkers(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "plasma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := app.NewService(store)
	if _, err := svc.CreateMission(ctx, app.CreateMissionRequest{MissionID: "mis_fanout_concurrency", Title: "Fanout concurrency"}); err != nil {
		t.Fatal(err)
	}
	server := &Server{service: svc}
	agent := &sectionFanoutConcurrencyAgent{reached: make(chan struct{}), release: make(chan struct{})}
	req := sectionFanoutLongFormRequest{
		missionID:          "mis_fanout_concurrency",
		title:              "Fanout concurrency",
		executorName:       "codex",
		pendingEventID:     "evt_pending_fanout_concurrency",
		postReportHumanize: "disabled",
	}
	state := sectionFanoutPlanState{
		planEvent:           app.LedgerEvent{EventID: "evt_plan_fanout_concurrency"},
		reportSessionPolicy: reportSessionPolicyIsolatedFork,
		sessionChainKind:    "section_fanout_report",
		reportPlanSessionID: "report-plan-session",
		forkSourceSessionID: "report-plan-session",
	}
	tasks := make([]sectionFanoutTask, sectionFanoutWorkerLimit+2)
	for index := range tasks {
		tasks[index] = sectionFanoutTask{
			partIndex:       0,
			sectionIndex:    index,
			part:            agentReportPart{Title: "Part", Purpose: "Part purpose"},
			section:         agentReportSection{Title: fmt.Sprintf("Section %d", index+1), Purpose: "Section purpose"},
			previousSession: fmt.Sprintf("section-session-%d", index+1),
			toolSessionID:   fmt.Sprintf("tool-session-%d", index+1),
			sourceSessionID: "report-plan-session",
		}
	}

	results := make(chan []sectionFanoutResult, 1)
	errs := make(chan error, 1)
	go func() {
		got, err := server.runSectionFanoutTasks(ctx, req, state, tasks, agent)
		if err != nil {
			errs <- err
			return
		}
		results <- got
	}()

	select {
	case <-agent.reached:
	case err := <-errs:
		t.Fatalf("runSectionFanoutTasks returned before reaching worker limit: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatalf("section fanout workers did not reach concurrency limit %d; max=%d started=%d", sectionFanoutWorkerLimit, agent.max.Load(), agent.started.Load())
	}
	if got := agent.max.Load(); got != int64(sectionFanoutWorkerLimit) {
		t.Fatalf("worker concurrency = %d, want %d", got, sectionFanoutWorkerLimit)
	}
	close(agent.release)

	select {
	case err := <-errs:
		if cause := errors.Unwrap(err); cause != nil {
			t.Fatalf("%v: %v", err, cause)
		}
		t.Fatal(err)
	case got := <-results:
		if len(got) != len(tasks) {
			t.Fatalf("results = %d, want %d", len(got), len(tasks))
		}
	case <-time.After(3 * time.Second):
		t.Fatal("section fanout tasks did not finish after release")
	}
}
