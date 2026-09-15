package reportilphase0

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

type retryCheckpointStoreFake struct {
	loads  []string
	result map[string]*reportilcontract.ResumeCheckpoint
	errs   map[string]error
	append int
}

func (store *retryCheckpointStoreFake) LoadReportILResumeCheckpoint(_ context.Context, _, pendingID string) (*reportilcontract.ResumeCheckpoint, error) {
	store.loads = append(store.loads, pendingID)
	if resume := store.result[pendingID]; resume != nil {
		return resume, nil
	}
	if err := store.errs[pendingID]; err != nil {
		return nil, err
	}
	return nil, errors.New("report IL retry has no durable checkpoint")
}

func (store *retryCheckpointStoreFake) AppendReportILCheckpoint(_ context.Context, _ string, _ reportilcontract.ProductCheckpoint) error {
	store.append++
	return nil
}

func TestResolveRetryCheckpointReusesFirstStoredAncestorWithoutLaterLoads(t *testing.T) {
	want := &reportilcontract.ResumeCheckpoint{}
	store := &retryCheckpointStoreFake{result: map[string]*reportilcontract.ResumeCheckpoint{"evt_parent": want}}
	events := []ledger.Event{
		retryCheckpointPending("evt_root", ""),
		retryCheckpointPending("evt_parent", "evt_root"),
		retryCheckpointPending("evt_child", "evt_parent"),
	}

	got, err := ResolveRetryCheckpoint(context.Background(), "mis_1", "evt_child", events, store, nil, nil)
	if err != nil || got != want {
		t.Fatalf("resume=%p want=%p err=%v", got, want, err)
	}
	if !reflect.DeepEqual(store.loads, []string{"evt_child", "evt_parent"}) {
		t.Fatalf("loads=%v", store.loads)
	}
	if store.append != 0 {
		t.Fatalf("stored checkpoint was appended %d times", store.append)
	}
}

func TestResolveRetryCheckpointStopsOnStorageError(t *testing.T) {
	store := &retryCheckpointStoreFake{errs: map[string]error{"evt_child": errors.New("storage unavailable")}}
	events := []ledger.Event{retryCheckpointPending("evt_child", "")}

	_, err := ResolveRetryCheckpoint(context.Background(), "mis_1", "evt_child", events, store, nil, nil)
	if err == nil || err.Error() != "storage unavailable" {
		t.Fatalf("err=%v", err)
	}
	if !reflect.DeepEqual(store.loads, []string{"evt_child"}) {
		t.Fatalf("loads=%v", store.loads)
	}
}

func TestResolveRetryCheckpointCycleReturnsNilWithoutStoreRead(t *testing.T) {
	store := &retryCheckpointStoreFake{}
	events := []ledger.Event{
		retryCheckpointPending("evt_one", "evt_two"),
		retryCheckpointPending("evt_two", "evt_one"),
	}

	got, err := ResolveRetryCheckpoint(context.Background(), "mis_1", "evt_one", events, store, nil, nil)
	if err != nil || got != nil {
		t.Fatalf("resume=%#v err=%v", got, err)
	}
	if len(store.loads) != 0 {
		t.Fatalf("cycle unexpectedly loaded checkpoints: %v", store.loads)
	}
}

func TestReportILRetryPendingLineageCapsAt64Entries(t *testing.T) {
	events := make([]ledger.Event, 0, 65)
	for index := 0; index < 65; index++ {
		parent := ""
		if index > 0 {
			parent = fmt.Sprintf("evt_%02d", index-1)
		}
		events = append(events, retryCheckpointPending(fmt.Sprintf("evt_%02d", index), parent))
	}

	lineage := reportILRetryPendingLineage(events, "evt_64")
	if len(lineage) != 64 || lineage[0] != "evt_64" || lineage[len(lineage)-1] != "evt_01" {
		t.Fatalf("lineage length/order=%v", lineage)
	}
}

func TestReportILRetryPendingLineageMatchesOriginalPartialDecodeOnDuplicateTypeError(t *testing.T) {
	payload := []byte(`{"retry_of_pending_event_id":7,"retry_of_pending_event_id":" evt_parent "}`)
	var decoded struct {
		RetryOf string `json:"retry_of_pending_event_id"`
	}
	if err := json.Unmarshal(payload, &decoded); err == nil {
		t.Fatal("duplicate type mismatch unexpectedly decoded without error")
	}
	if decoded.RetryOf != " evt_parent " {
		t.Fatalf("stdlib partial decode=%q want %q", decoded.RetryOf, " evt_parent ")
	}
	lineage := reportILRetryPendingLineage([]ledger.Event{
		{EventID: "evt_child", EventType: "report.draft.pending", Payload: payload},
	}, "evt_child")
	if !reflect.DeepEqual(lineage, []string{"evt_child", "evt_parent"}) {
		t.Fatalf("lineage=%v want child then partially decoded parent", lineage)
	}
}

func TestReportILRetryPendingLineageDuplicateEventIDMalformedLaterPayloadClearsParent(t *testing.T) {
	lineage := reportILRetryPendingLineage([]ledger.Event{
		retryCheckpointPending("evt_child", "evt_parent"),
		{EventID: "evt_child", EventType: "report.draft.pending", Payload: []byte(`{"retry_of_pending_event_id":`)},
	}, "evt_child")
	if !reflect.DeepEqual(lineage, []string{"evt_child"}) {
		t.Fatalf("lineage=%v want duplicate later malformed payload to overwrite parent with empty", lineage)
	}
}

func TestReportILRetryPendingLineageSyntaxTruncatedUsesStdlibDecodeState(t *testing.T) {
	payload := []byte(`{"retry_of_pending_event_id":" evt_parent `)
	var decoded struct {
		RetryOf string `json:"retry_of_pending_event_id"`
	}
	_ = json.Unmarshal(payload, &decoded)
	lineage := reportILRetryPendingLineage([]ledger.Event{
		{EventID: "evt_child", EventType: "report.draft.pending", Payload: payload},
	}, "evt_child")
	want := []string{"evt_child"}
	if strings.TrimSpace(decoded.RetryOf) != "" {
		want = []string{"evt_child", strings.TrimSpace(decoded.RetryOf)}
	}
	if !reflect.DeepEqual(lineage, want) {
		t.Fatalf("lineage=%v want stdlib-derived lineage=%v decoded=%q", lineage, want, decoded.RetryOf)
	}
}

func retryCheckpointPending(id, parent string) ledger.Event {
	payload := fmt.Sprintf(`{"retry_of_pending_event_id":%q}`, parent)
	return ledger.Event{EventID: id, EventType: "report.draft.pending", Payload: []byte(payload)}
}

var _ RetryCheckpointStore = (*retryCheckpointStoreFake)(nil)
