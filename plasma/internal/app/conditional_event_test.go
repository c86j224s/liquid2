package app

import (
	"context"
	"errors"
	"testing"
)

type conditionalAdapterStore struct {
	fakeStore
	events []LedgerEvent
	calls  int
	fail   bool
}

func (s *conditionalAdapterStore) AppendLedgerEventsConditionally(_ context.Context, mission string, build func([]LedgerEvent) ([]LedgerEvent, error)) ([]LedgerEvent, error) {
	s.calls++
	built, err := build(append([]LedgerEvent(nil), s.events...))
	if err != nil || s.fail {
		if err == nil {
			err = errors.New("rollback")
		}
		return nil, err
	}
	for _, event := range built {
		if event.MissionID != mission {
			return nil, errors.New("mission")
		}
		s.events = append(s.events, event)
	}
	return built, nil
}
func (s *conditionalAdapterStore) ListLedgerEvents(context.Context, string) ([]LedgerEvent, error) {
	return append([]LedgerEvent(nil), s.events...), nil
}
func validConditionalRequest(mission, id string) AppendEventRequest {
	return AppendEventRequest{EventID: id, MissionID: mission, EventType: "report.draft.pending", Producer: Producer{Type: "user", ID: "u"}, Payload: []byte(`{"title":"test"}`)}
}
func TestAppendEventsConditionallyBatchAndNoop(t *testing.T) {
	store := &conditionalAdapterStore{}
	svc := NewService(store)
	got, err := svc.AppendEventsConditionally(context.Background(), "mis_1", func([]LedgerEvent) ([]AppendEventRequest, error) {
		return []AppendEventRequest{validConditionalRequest("mis_1", "evt_a"), validConditionalRequest("mis_1", "evt_b")}, nil
	})
	if err != nil || len(got) != 2 || len(store.events) != 2 {
		t.Fatalf("got=%#v err=%v", got, err)
	}
	got, err = svc.AppendEventsConditionally(context.Background(), "mis_1", func([]LedgerEvent) ([]AppendEventRequest, error) { return nil, nil })
	if err != nil || len(got) != 0 || len(store.events) != 2 {
		t.Fatalf("noop got=%#v err=%v", got, err)
	}
}
func TestAppendEventsConditionallyRejectsUnsupportedStoreWithBuilder(t *testing.T) {
	svc := NewService(&fakeStore{})
	_, err := svc.AppendEventsConditionally(context.Background(), "mis_1", func([]LedgerEvent) ([]AppendEventRequest, error) { return nil, nil })
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("unsupported store err=%v", err)
	}
}

func TestAppendEventsConditionallyPropagatesBuilderAndBuiltRequestErrors(t *testing.T) {
	store := &conditionalAdapterStore{}
	svc := NewService(store)
	builderErr := errors.New("builder")
	_, err := svc.AppendEventsConditionally(context.Background(), "mis_1", func([]LedgerEvent) ([]AppendEventRequest, error) { return nil, builderErr })
	if !errors.Is(err, builderErr) {
		t.Fatalf("builder err=%v", err)
	}
	_, err = svc.AppendEventsConditionally(context.Background(), "mis_1", func([]LedgerEvent) ([]AppendEventRequest, error) {
		return []AppendEventRequest{{MissionID: "mis_1", EventID: "bad"}}, nil
	})
	if err == nil {
		t.Fatal("invalid built request accepted")
	}
}

func TestAppendEventsConditionallyRejectsAgentExecutorConflict(t *testing.T) {
	store := &conditionalAdapterStore{events: []LedgerEvent{{EventID: "evt_existing", MissionID: "mis_1", EventType: "report.draft.pending", Producer: Producer{Type: "user", ID: "u"}, Payload: []byte(`{"agent_executor":"codex"}`)}}}
	svc := NewService(store)
	_, err := svc.AppendEventsConditionally(context.Background(), "mis_1", func([]LedgerEvent) ([]AppendEventRequest, error) {
		return []AppendEventRequest{{EventID: "evt_new", MissionID: "mis_1", EventType: "report.artifact.created", Producer: Producer{Type: "agent", ID: "a"}, Payload: []byte(`{"agent_executor":"claude"}`)}}, nil
	})
	if err == nil {
		t.Fatal("executor conflict accepted")
	}
}

func TestAppendEventsConditionallyRejectsNilUnsupportedCrossMissionAndRollsBack(t *testing.T) {
	svc := NewService(&fakeStore{})
	if _, err := svc.AppendEventsConditionally(context.Background(), "mis_1", nil); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("nil err=%v", err)
	}
	store := &conditionalAdapterStore{fail: true}
	svc = NewService(store)
	_, err := svc.AppendEventsConditionally(context.Background(), "mis_1", func([]LedgerEvent) ([]AppendEventRequest, error) {
		return []AppendEventRequest{validConditionalRequest("mis_2", "evt_x")}, nil
	})
	if err == nil || len(store.events) != 0 {
		t.Fatalf("cross mission err=%v events=%d", err, len(store.events))
	}
}
