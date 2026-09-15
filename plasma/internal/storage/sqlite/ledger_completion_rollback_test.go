package sqlite

import (
	"context"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mission"
	"testing"
)

func TestLedgerConditionalAppendRollsBackWholeCompletionBatchOnDuplicate(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	if err := store.CreateMission(ctx, mission.Mission{MissionID: "mis_completion_rollback", Title: "Completion rollback"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AppendLedgerEvent(ctx, ledger.Event{EventID: "evt_existing", MissionID: "mis_completion_rollback", EventType: "report.draft.pending", Producer: ledger.Producer{Type: "user", ID: "u"}, Payload: []byte(`{}`)}); err != nil {
		t.Fatal(err)
	}
	beforeProjection := countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_report_runs WHERE mission_id = ?`, "mis_completion_rollback")
	_, err := store.AppendLedgerEventsConditionally(ctx, "mis_completion_rollback", func([]ledger.Event) ([]ledger.Event, error) {
		return []ledger.Event{{EventID: "evt_new_first", MissionID: "mis_completion_rollback", EventType: "report.agent_usage.recorded", Producer: ledger.Producer{Type: "agent_session", ID: "s"}, Payload: []byte(`{}`)}, {EventID: "evt_existing", MissionID: "mis_completion_rollback", EventType: "report.run.completed", Producer: ledger.Producer{Type: "system", ID: "report-completion"}, Payload: []byte(`{}`)}}, nil
	})
	if err == nil {
		t.Fatal("duplicate batch unexpectedly committed")
	}
	events, err := store.ListLedgerEvents(ctx, "mis_completion_rollback")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].EventID != "evt_existing" {
		t.Fatalf("partial batch persisted: %#v", events)
	}
	if beforeProjection != 1 {
		t.Fatalf("unexpected baseline report-run projection count: %d", beforeProjection)
	}
	if countRows(t, ctx, store, `SELECT COUNT(*) FROM plasma_report_runs WHERE mission_id = ?`, "mis_completion_rollback") != beforeProjection {
		t.Fatal("report-run projection persisted after rollback")
	}
}
