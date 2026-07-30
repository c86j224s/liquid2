package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type reportRedpenTestStore struct {
	fakeStore
	events    []LedgerEvent
	artifacts map[string]RawArtifact
}

func (s *reportRedpenTestStore) GetRawArtifact(_ context.Context, artifactID string) (RawArtifact, error) {
	artifact, ok := s.artifacts[artifactID]
	if !ok {
		return RawArtifact{}, errors.New("artifact not found")
	}
	return artifact, nil
}

func (s *reportRedpenTestStore) ListLedgerEvents(_ context.Context, missionID string) ([]LedgerEvent, error) {
	events := make([]LedgerEvent, 0, len(s.events))
	for _, event := range s.events {
		if event.MissionID == missionID {
			events = append(events, event)
		}
	}
	return events, nil
}

func (s *reportRedpenTestStore) CommitReportRedpenRevision(
	_ context.Context,
	candidate RawArtifact,
	build func([]LedgerEvent, RawArtifact) (LedgerEvent, bool, error),
) (RawArtifact, LedgerEvent, bool, error) {
	target := candidate
	exists := false
	for _, artifact := range s.artifacts {
		if artifact.MissionID == candidate.MissionID && artifact.SHA256 == candidate.SHA256 {
			target = artifact
			exists = true
			break
		}
	}
	event, appendEvent, err := build(append([]LedgerEvent(nil), s.events...), target)
	if err != nil {
		return RawArtifact{}, LedgerEvent{}, false, err
	}
	if !appendEvent {
		if !exists {
			return RawArtifact{}, LedgerEvent{}, false, errors.New("unstored no-op")
		}
		return target, event, false, nil
	}
	if !exists {
		s.artifacts[candidate.ArtifactID] = candidate
	}
	event.Sequence = int64(len(s.events) + 1)
	s.events = append(s.events, event)
	return target, event, true, nil
}

func TestSaveReportRedpenWorkcopyCreatesUpdatesAndReusesRevisions(t *testing.T) {
	ctx := context.Background()
	source := mustReportRedpenArtifact(t, "art_source", "mis_1", "report.md", "# 제목\n\n원문입니다.\n")
	store := &reportRedpenTestStore{
		artifacts: map[string]RawArtifact{source.ArtifactID: source},
		events:    []LedgerEvent{reportRedpenSourceEvent(source)},
	}
	svc := NewService(store)

	first := saveReportRedpenForTest(t, svc, SaveReportRedpenRequest{
		EventID: "evt_redpen_1", ArtifactID: "art_redpen_1", NewWorkcopyID: "rwc_1",
		MissionID: "mis_1", SourceArtifactID: source.ArtifactID,
		Producer: Producer{Type: "user", ID: "plasma-ui"}, Content: []byte("# 제목\n\n첫 교정입니다.\n"),
	})
	if !first.Exists || !first.Changed || first.Revision != 1 || first.WorkcopyID != "rwc_1" || first.PreviousArtifactID != source.ArtifactID {
		t.Fatalf("unexpected first redpen revision: %#v", first)
	}
	if got := string(store.artifacts[source.ArtifactID].Content); got != "# 제목\n\n원문입니다.\n" {
		t.Fatalf("source artifact was mutated: %q", got)
	}
	if strings.Contains(string(first.Event.Payload), "첫 교정입니다") || strings.Contains(string(first.Event.Payload), "content") {
		t.Fatalf("redpen event leaked report content: %s", first.Event.Payload)
	}

	noOp := saveReportRedpenForTest(t, svc, SaveReportRedpenRequest{
		EventID: "evt_redpen_noop", ArtifactID: "art_redpen_noop", NewWorkcopyID: "rwc_unused",
		MissionID: "mis_1", SourceArtifactID: source.ArtifactID, ExpectedCurrentArtifactID: first.Artifact.ArtifactID,
		Producer: Producer{Type: "user", ID: "plasma-ui"}, Content: append([]byte(nil), first.Artifact.Content...),
	})
	if noOp.Changed || noOp.Revision != 1 || reportRedpenEventCount(store.events) != 1 || len(store.artifacts) != 2 {
		t.Fatalf("same-content save should be a no-op: result=%#v events=%d artifacts=%d", noOp, reportRedpenEventCount(store.events), len(store.artifacts))
	}

	second := saveReportRedpenForTest(t, svc, SaveReportRedpenRequest{
		EventID: "evt_redpen_2", ArtifactID: "art_redpen_2", NewWorkcopyID: "rwc_unused_2",
		MissionID: "mis_1", SourceArtifactID: source.ArtifactID, ExpectedCurrentArtifactID: first.Artifact.ArtifactID,
		Producer: Producer{Type: "user", ID: "plasma-ui"}, Content: []byte("# 제목\n\n두 번째 교정입니다.\n"),
	})
	if !second.Changed || second.Revision != 2 || second.WorkcopyID != first.WorkcopyID || second.PreviousArtifactID != first.Artifact.ArtifactID {
		t.Fatalf("unexpected second redpen revision: %#v", second)
	}

	_, err := svc.SaveReportRedpenWorkcopy(ctx, SaveReportRedpenRequest{
		EventID: "evt_redpen_stale", ArtifactID: "art_redpen_stale", NewWorkcopyID: "rwc_unused_3",
		MissionID: "mis_1", SourceArtifactID: source.ArtifactID, ExpectedCurrentArtifactID: first.Artifact.ArtifactID,
		Producer: Producer{Type: "user", ID: "plasma-ui"}, Content: []byte("stale edit"),
	})
	if !errors.Is(err, ErrConflict) || reportRedpenEventCount(store.events) != 2 {
		t.Fatalf("expected stale save conflict without a new event, got %v events=%d", err, reportRedpenEventCount(store.events))
	}

	reverted := saveReportRedpenForTest(t, svc, SaveReportRedpenRequest{
		EventID: "evt_redpen_3", ArtifactID: "art_redpen_3", NewWorkcopyID: "rwc_unused_4",
		MissionID: "mis_1", SourceArtifactID: source.ArtifactID, ExpectedCurrentArtifactID: second.Artifact.ArtifactID,
		Producer: Producer{Type: "user", ID: "plasma-ui"}, Content: append([]byte(nil), source.Content...),
	})
	if reverted.Artifact.ArtifactID != source.ArtifactID || reverted.Revision != 3 || reverted.Filename != "report-redpen.md" || len(store.artifacts) != 3 {
		t.Fatalf("revert should reuse the source artifact: result=%#v artifacts=%d", reverted, len(store.artifacts))
	}

	loaded, err := svc.GetReportRedpenWorkcopy(ctx, "mis_1", source.ArtifactID)
	if err != nil {
		t.Fatalf("GetReportRedpenWorkcopy returned error: %v", err)
	}
	if !loaded.Exists || loaded.Revision != 3 || loaded.Artifact.ArtifactID != source.ArtifactID || string(loaded.Artifact.Content) != string(source.Content) {
		t.Fatalf("unexpected loaded redpen workcopy: %#v", loaded)
	}
}

func TestReportRedpenWorkcopyRejectsInvalidSourceAndFirstSaveExpectation(t *testing.T) {
	ctx := context.Background()
	nonMarkdown := mustReportRedpenArtifact(t, "art_text", "mis_1", "report.txt", "plain")
	nonMarkdown.MediaType = "text/plain"
	store := &reportRedpenTestStore{artifacts: map[string]RawArtifact{nonMarkdown.ArtifactID: nonMarkdown}}
	svc := NewService(store)

	_, err := svc.SaveReportRedpenWorkcopy(ctx, SaveReportRedpenRequest{
		EventID: "evt_redpen", ArtifactID: "art_redpen", NewWorkcopyID: "rwc_1",
		MissionID: "mis_1", SourceArtifactID: nonMarkdown.ArtifactID,
		Producer: Producer{Type: "user", ID: "plasma-ui"}, Content: []byte("edit"),
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected non-Markdown source rejection, got %v", err)
	}

	source := mustReportRedpenArtifact(t, "art_source", "mis_1", "report.md", "original")
	store.artifacts[source.ArtifactID] = source
	_, err = svc.SaveReportRedpenWorkcopy(ctx, SaveReportRedpenRequest{
		EventID: "evt_redpen_unowned", ArtifactID: "art_redpen_unowned", NewWorkcopyID: "rwc_unowned",
		MissionID: "mis_1", SourceArtifactID: source.ArtifactID,
		Producer: Producer{Type: "user", ID: "plasma-ui"}, Content: []byte("edit"),
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected non-report Markdown source rejection, got %v", err)
	}
	store.events = append(store.events, reportRedpenSourceEvent(source))
	_, err = svc.SaveReportRedpenWorkcopy(ctx, SaveReportRedpenRequest{
		EventID: "evt_redpen_first", ArtifactID: "art_redpen_first", NewWorkcopyID: "rwc_2",
		MissionID: "mis_1", SourceArtifactID: source.ArtifactID, ExpectedCurrentArtifactID: source.ArtifactID,
		Producer: Producer{Type: "user", ID: "plasma-ui"}, Content: []byte("edit"),
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected first-save expectation conflict, got %v", err)
	}

	collision := mustReportRedpenArtifact(t, "art_plain_collision", "mis_1", "plain.txt", "collision")
	collision.MediaType = "text/plain"
	store.artifacts[collision.ArtifactID] = collision
	_, err = svc.SaveReportRedpenWorkcopy(ctx, SaveReportRedpenRequest{
		EventID: "evt_redpen_collision", ArtifactID: "art_redpen_collision", NewWorkcopyID: "rwc_collision",
		MissionID: "mis_1", SourceArtifactID: source.ArtifactID,
		Producer: Producer{Type: "user", ID: "plasma-ui"}, Content: []byte("collision"),
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected incompatible content-hash collision rejection, got %v", err)
	}
}

func TestGetReportRedpenWorkcopyRejectsCorruptedEvent(t *testing.T) {
	source := mustReportRedpenArtifact(t, "art_source", "mis_1", "report.md", "original")
	store := &reportRedpenTestStore{
		artifacts: map[string]RawArtifact{source.ArtifactID: source},
		events: []LedgerEvent{
			reportRedpenSourceEvent(source),
			{EventID: "evt_redpen_broken", MissionID: source.MissionID, EventType: ReportRedpenSavedEvent, Payload: json.RawMessage(`{"broken":true}`)},
		},
	}

	_, err := NewService(store).GetReportRedpenWorkcopy(context.Background(), source.MissionID, source.ArtifactID)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected corrupted redpen event rejection, got %v", err)
	}
}

func reportRedpenSourceEvent(artifact RawArtifact) LedgerEvent {
	payload, _ := json.Marshal(map[string]any{"kind": "markdown_report_artifact", "artifact_id": artifact.ArtifactID})
	return LedgerEvent{
		EventID: "evt_source_" + artifact.ArtifactID, MissionID: artifact.MissionID,
		EventType: "report.artifact.created", Producer: Producer{Type: "agent", ID: "reporter"}, Payload: payload,
	}
}

func reportRedpenEventCount(events []LedgerEvent) int {
	count := 0
	for _, event := range events {
		if event.EventType == ReportRedpenSavedEvent {
			count++
		}
	}
	return count
}

func saveReportRedpenForTest(t *testing.T, svc *Service, req SaveReportRedpenRequest) ReportRedpenWorkcopy {
	t.Helper()
	result, err := svc.SaveReportRedpenWorkcopy(context.Background(), req)
	if err != nil {
		t.Fatalf("SaveReportRedpenWorkcopy returned error: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(result.Event.Payload, &payload); err != nil {
		t.Fatalf("unmarshal redpen event: %v", err)
	}
	if payload["source_artifact_id"] != req.SourceArtifactID {
		t.Fatalf("unexpected redpen source payload: %#v", payload)
	}
	return result
}

func mustReportRedpenArtifact(t *testing.T, artifactID, missionID, filename, content string) RawArtifact {
	t.Helper()
	artifact, err := buildRawArtifact(CreateRawArtifactRequest{
		ArtifactID: artifactID, MissionID: missionID, MediaType: "text/markdown; charset=utf-8",
		Filename: filename, Producer: Producer{Type: "agent", ID: "reporter"}, Content: []byte(content),
	})
	if err != nil {
		t.Fatalf("buildRawArtifact returned error: %v", err)
	}
	return artifact
}
