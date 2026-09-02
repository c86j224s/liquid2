package app

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
	"github.com/c86j224s/liquid2/plasma/internal/source"
)

func TestAppendReportILLongFormProgressPersistsPlanAndConcurrentCoordinates(t *testing.T) {
	store := &reportILProgressStore{}
	service := NewService(store)
	ctx := context.Background()
	plan := reportilcontract.LongFormPlan{
		Parts: []reportilcontract.LongFormPart{
			{
				PartKey: "part_001", Title: "First Part", Purpose: "Internal purpose",
				Sections: []reportilcontract.LongFormSection{
					{SectionKey: "part_001.section_001", Title: "First Section", Purpose: "Internal section purpose", EvidenceSourceKeys: []string{"source_private"}},
					{SectionKey: "part_001.section_002", Title: "Second Section", EditorialAccountKeys: []string{"account_private"}},
				},
			},
		},
	}
	if err := service.AppendReportILLongFormProgress(
		ctx, "mis_long_form_progress", "evt_long_form_pending", "plan", "completed",
		&plan, 0, 0, "Report",
	); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for section := 1; section <= 2; section++ {
		section := section
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := service.AppendReportILLongFormProgress(
				ctx, "mis_long_form_progress", "evt_long_form_pending", "section", "started",
				nil, 1, section, "Section",
			); err != nil {
				t.Errorf("append Section %d progress: %v", section, err)
			}
		}()
	}
	wg.Wait()

	events, err := store.ListLedgerEvents(ctx, "mis_long_form_progress")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 {
		t.Fatalf("stored long-form progress event count = %d, want 3", len(events))
	}
	if events[0].EventType != "report.il_long_form_plan.created" || events[0].CausationEventID != "evt_long_form_pending" || events[0].CorrelationID != "evt_long_form_pending" {
		t.Fatalf("plan progress lineage = %#v", events[0])
	}
	var planPayload map[string]any
	if err := json.Unmarshal(events[0].Payload, &planPayload); err != nil {
		t.Fatal(err)
	}
	encoded := string(events[0].Payload)
	for _, forbidden := range []string{"source_private", "account_private", "Internal purpose", "Internal section purpose", "part_001"} {
		if strings.Contains(encoded, forbidden) {
			t.Fatalf("plan inventory leaked %q: %s", forbidden, encoded)
		}
	}
	projectedPlan, ok := planPayload["plan"].(map[string]any)
	if !ok {
		t.Fatalf("plan inventory payload = %#v", planPayload)
	}
	parts, ok := projectedPlan["parts"].([]any)
	if !ok || len(parts) != 1 {
		t.Fatalf("plan parts = %#v", projectedPlan["parts"])
	}
	sections := map[int]bool{}
	for _, event := range events[1:] {
		if event.EventType != "report.il_long_form_section.started" || event.CausationEventID != "evt_long_form_pending" || event.CorrelationID != "evt_long_form_pending" {
			t.Fatalf("Section progress lineage = %#v", event)
		}
		var payload struct {
			Part    int `json:"part_index"`
			Section int `json:"section_index"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.Part != 1 {
			t.Fatalf("Section part coordinate = %#v", payload)
		}
		sections[payload.Section] = true
	}
	if !sections[1] || !sections[2] {
		t.Fatalf("Section coordinates = %#v", sections)
	}
}

func TestAppendReportILLongFormProgressRejectsMalformedEvents(t *testing.T) {
	service := NewService(&reportILProgressStore{})
	for _, tc := range []struct {
		kind, status  string
		part, section int
	}{
		{kind: "plan", status: "started"},
		{kind: "section", status: "completed", part: 0, section: 1},
		{kind: "part_edit", status: "started", part: 1, section: 1},
		{kind: "unknown", status: "completed"},
	} {
		if err := service.AppendReportILLongFormProgress(
			context.Background(), "mis_long_form_progress", "evt_long_form_pending",
			tc.kind, tc.status, nil, tc.part, tc.section, "",
		); err == nil {
			t.Fatalf("malformed long-form progress was accepted: %#v", tc)
		}
	}
}

func TestReportILSourceAdapterPreservesHostOnlyCitationMetadata(t *testing.T) {
	store := &reportILAdapterFakeStore{sources: []SourceSnapshot{{
		SnapshotID: "src_public",
		MissionID:  "mis_public",
		Title:      "Public source title",
		Connector: ConnectorRef{
			ConnectorType: "url",
			ExternalURI:   "https://example.com/report",
		},
		ArtifactIDs: []string{"art_public"},
		ContentHash: ContentHash{Value: "snapshot-hash"},
		Access: SourceAccess{
			RetrievalPolicy: source.RetrievalPolicySnapshotOnly,
		},
	}}}
	reader := NewService(store).ReportILSourceReader()

	snapshots, err := reader.ListSourceSnapshots(context.Background(), "mis_public")
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshots) != 1 || snapshots[0].Title != "Public source title" || snapshots[0].ConnectorType != "url" || snapshots[0].ExternalURI != "https://example.com/report" || !snapshots[0].Active {
		t.Fatalf("Report IL snapshot projection = %#v", snapshots)
	}
}

type reportILProgressStore struct {
	fakeStore
	mu     sync.Mutex
	events []LedgerEvent
}

func (s *reportILProgressStore) AppendLedgerEvent(_ context.Context, event LedgerEvent) (LedgerEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	event.Sequence = int64(len(s.events) + 1)
	s.events = append(s.events, event)
	return event, nil
}

func (s *reportILProgressStore) ListLedgerEvents(_ context.Context, missionID string) ([]LedgerEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	events := make([]LedgerEvent, 0, len(s.events))
	for _, event := range s.events {
		if event.MissionID == missionID {
			events = append(events, event)
		}
	}
	return events, nil
}

type reportILAdapterFakeStore struct {
	fakeStore
	sources []SourceSnapshot
}

func (f *reportILAdapterFakeStore) ListSourceSnapshots(context.Context, string) ([]SourceSnapshot, error) {
	return append([]SourceSnapshot(nil), f.sources...), nil
}
