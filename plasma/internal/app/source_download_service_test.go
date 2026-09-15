package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
	"github.com/c86j224s/liquid2/plasma/internal/source/confluencesource"
	"github.com/c86j224s/liquid2/plasma/internal/source/liquid2source"
	"testing"

	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

type sourceDownloadTestStore struct {
	fakeStore
	events    []ledger.Event
	artifacts map[string]artifactcontract.Raw
	snapshot  sourcecontract.Snapshot
	snapshots []sourcecontract.Snapshot
}

func (s *sourceDownloadTestStore) ListLedgerEvents(context.Context, string) ([]ledger.Event, error) {
	return s.events, nil
}
func (s *sourceDownloadTestStore) GetRawArtifact(_ context.Context, id string) (artifactcontract.Raw, error) {
	a, ok := s.artifacts[id]
	if !ok {
		return artifactcontract.Raw{}, sql.ErrNoRows
	}
	return a, nil
}
func (s *sourceDownloadTestStore) GetSourceSnapshot(_ context.Context, id string) (sourcecontract.Snapshot, error) {
	if s.snapshot.SnapshotID == "" || s.snapshot.SnapshotID != id {
		return sourcecontract.Snapshot{}, sql.ErrNoRows
	}
	return s.snapshot, nil
}
func (s *sourceDownloadTestStore) ListSourceSnapshots(context.Context, string) ([]sourcecontract.Snapshot, error) {
	return s.snapshots, nil
}

func TestResolveSourceCandidateDownloadLifecycleAndConsumption(t *testing.T) {
	ctx := context.Background()
	staged := candidateDownloadEvent("evt_staged", 3, "https://EXAMPLE.com/doc#fragment", "art_candidate", "evt_proposed")
	store := &sourceDownloadTestStore{events: []ledger.Event{staged}, artifacts: map[string]artifactcontract.Raw{
		"art_candidate": {ArtifactID: "art_candidate", MissionID: "mis_1", Content: []byte("exact bytes")},
	}}
	svc := NewService(store)
	got, err := svc.ResolveSourceCandidateDownload(ctx, SourceCandidateDownloadRequest{MissionID: "mis_1", StagedEventID: "evt_staged", ArtifactID: "art_candidate"})
	if err != nil || string(got.Content) != "exact bytes" {
		t.Fatalf("success: got=%q err=%v", got.Content, err)
	}

	cases := []struct {
		name   string
		mutate func(*sourceDownloadTestStore)
	}{
		{"old staged", func(s *sourceDownloadTestStore) {
			s.events = append(s.events, candidateDownloadEvent("evt_new", 4, "https://example.com/doc", "art_new", "evt_proposed"))
		}},
		{"newer fetching", func(s *sourceDownloadTestStore) {
			s.events = append(s.events, candidateTerminal("source.candidate.staging_started", "evt_fetching", 4, "https://example.com/doc"))
		}},
		{"newer failed", func(s *sourceDownloadTestStore) {
			s.events = append(s.events, candidateTerminal("source.candidate.staging_failed", "evt_failed", 4, "https://example.com/doc"))
		}},
		{"missing artifact", func(s *sourceDownloadTestStore) { delete(s.artifacts, "art_candidate") }},
		{"cross mission artifact", func(s *sourceDownloadTestStore) {
			s.artifacts["art_candidate"] = artifactcontract.Raw{ArtifactID: "art_candidate", MissionID: "mis_2", Content: []byte("x")}
		}},
		{"latest reject", func(s *sourceDownloadTestStore) {
			s.events = append(s.events, candidateDecision("source.candidate.rejected", "evt_reject", 4, "https://example.com/doc"))
		}},
		{"persisted same artifact consumed", func(s *sourceDownloadTestStore) {
			s.snapshots = []sourcecontract.Snapshot{{SnapshotID: "src_1", MissionID: "mis_1", ArtifactIDs: []string{"art_candidate"}}}
		}},
		{"persisted same URL different proposal consumed", func(s *sourceDownloadTestStore) {
			s.snapshots = []sourcecontract.Snapshot{{SnapshotID: "src_1", MissionID: "mis_1", Connector: sourcecontract.ConnectorRef{ExternalURI: "https://example.com/doc"}}}
		}},
		{"persisted Confluence locator consumed", func(s *sourceDownloadTestStore) {
			s.events[0] = candidateDownloadEvent("evt_staged", 3, "https://example.atlassian.net/wiki/spaces/ENG/pages/123/title", "art_candidate", "evt_proposed")
			s.snapshots = []sourcecontract.Snapshot{{SnapshotID: "src_1", MissionID: "mis_1", Locators: json.RawMessage(`[{"site_url":"https://example.atlassian.net/wiki","page_id":"123"}]`)}}
		}},
		{"Confluence candidate download excluded", func(s *sourceDownloadTestStore) {
			s.events[0] = candidateDownloadEvent("evt_staged", 3, "https://example.atlassian.net/wiki/spaces/ENG/pages/123/title", "art_candidate", "evt_proposed")
		}},
		{"encoded Confluence candidate download excluded", func(s *sourceDownloadTestStore) {
			s.events[0] = candidateDownloadEvent("evt_staged", 3, "https://example.atlassian.net/wiki/%70ages/123/title", "art_candidate", "evt_proposed")
		}},
		{"encoded Confluence edit candidate download excluded", func(s *sourceDownloadTestStore) {
			s.events[0] = candidateDownloadEvent("evt_staged", 3, "https://example.atlassian.net/wiki/%65dit-v2/123", "art_candidate", "evt_proposed")
		}},
		{"unrelated persisted snapshot remains downloadable", func(s *sourceDownloadTestStore) {
			s.snapshots = []sourcecontract.Snapshot{{SnapshotID: "src_other", MissionID: "mis_1", Connector: sourcecontract.ConnectorRef{ExternalURI: "https://example.com/other"}}}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fresh := cloneSourceDownloadStore(store)
			tc.mutate(fresh)
			got, err := NewService(fresh).ResolveSourceCandidateDownload(ctx, SourceCandidateDownloadRequest{MissionID: "mis_1", StagedEventID: "evt_staged", ArtifactID: "art_candidate"})
			if tc.name == "unrelated persisted snapshot remains downloadable" {
				if err != nil || string(got.Content) != "exact bytes" {
					t.Fatalf("unrelated snapshot blocked download: %q %v", got.Content, err)
				}
				return
			}
			if !errors.Is(err, ErrSourceDownloadNotFound) {
				t.Fatalf("expected uniform not found, got %v", err)
			}
		})
	}
}

func TestResolveSourceCandidateDownloadLatestDecisionOrder(t *testing.T) {
	base := &sourceDownloadTestStore{events: []ledger.Event{candidateDecision("source.candidate.rejected", "evt_reject", 1, "https://example.com/doc"), candidateDownloadEvent("evt_staged", 2, "https://example.com/doc", "art_candidate", "evt_proposed")}, artifacts: map[string]artifactcontract.Raw{"art_candidate": {ArtifactID: "art_candidate", MissionID: "mis_1", Content: []byte("ok")}}}
	if _, err := NewService(base).ResolveSourceCandidateDownload(context.Background(), SourceCandidateDownloadRequest{MissionID: "mis_1", StagedEventID: "evt_staged", ArtifactID: "art_candidate"}); !errors.Is(err, ErrSourceDownloadNotFound) {
		t.Fatal("latest rejection before staging must reject")
	}
	base.events = append(base.events, candidateDecision("source.candidate.restored", "evt_restore", 3, "https://example.com/doc"))
	if _, err := NewService(base).ResolveSourceCandidateDownload(context.Background(), SourceCandidateDownloadRequest{MissionID: "mis_1", StagedEventID: "evt_staged", ArtifactID: "art_candidate"}); err != nil {
		t.Fatalf("latest restore must reopen: %v", err)
	}
}

func TestResolveSourceSnapshotDownloadEligibility(t *testing.T) {
	ctx := context.Background()
	fresh := func() *sourceDownloadTestStore {
		return &sourceDownloadTestStore{
			snapshot: sourcecontract.Snapshot{SnapshotID: "src_1", MissionID: "mis_1", ArtifactIDs: []string{"art_pdf"}, Access: sourcecontract.Access{RetrievalPolicy: sourcecontract.RetrievalPolicySnapshotOnly}},
			artifacts: map[string]artifactcontract.Raw{
				"art_pdf":   {ArtifactID: "art_pdf", MissionID: "mis_1", Content: []byte("pdf")},
				"art_other": {ArtifactID: "art_other", MissionID: "mis_1", Content: []byte("other")},
			},
		}
	}
	cases := []struct {
		name    string
		request SourceSnapshotDownloadRequest
		mutate  func(*sourceDownloadTestStore)
		want    string
	}{
		{name: "implicit one artifact", request: SourceSnapshotDownloadRequest{MissionID: "mis_1", SnapshotID: "src_1"}, want: "pdf"},
		{name: "explicit attached artifact", request: SourceSnapshotDownloadRequest{MissionID: "mis_1", SnapshotID: "src_1", ArtifactID: "art_pdf"}, want: "pdf"},
		{name: "multi artifact explicit attached selector", request: SourceSnapshotDownloadRequest{MissionID: "mis_1", SnapshotID: "src_1", ArtifactID: "art_other"}, mutate: func(s *sourceDownloadTestStore) { s.snapshot.ArtifactIDs = []string{"art_pdf", "art_other"} }},
		{name: "multi artifact unattached selector", request: SourceSnapshotDownloadRequest{MissionID: "mis_1", SnapshotID: "src_1", ArtifactID: "art_missing"}, mutate: func(s *sourceDownloadTestStore) { s.snapshot.ArtifactIDs = []string{"art_pdf", "art_other"} }},
		{name: "missing snapshot", request: SourceSnapshotDownloadRequest{MissionID: "mis_1", SnapshotID: "src_missing"}},
		{name: "missing artifact", request: SourceSnapshotDownloadRequest{MissionID: "mis_1", SnapshotID: "src_1", ArtifactID: "art_missing"}},
		{name: "malformed snapshot", request: SourceSnapshotDownloadRequest{MissionID: "mis_1", SnapshotID: "src_1"}, mutate: func(s *sourceDownloadTestStore) {
			s.snapshot = sourcecontract.Snapshot{SnapshotID: "src_1", MissionID: "mis_1", ArtifactIDs: []string{"art_pdf"}, Access: sourcecontract.Access{RetrievalPolicy: sourcecontract.RetrievalPolicySnapshotOnly}}
			s.snapshot.SnapshotID = ""
		}},
		{name: "removed", request: SourceSnapshotDownloadRequest{MissionID: "mis_1", SnapshotID: "src_1"}, mutate: func(s *sourceDownloadTestStore) {
			s.events = []ledger.Event{{EventID: "evt_removed", Sequence: 10, EventType: SourceRemovedEvent, Payload: json.RawMessage(`{"snapshot_id":"src_1"}`)}}
			s.snapshot.State = sourcecontract.State{State: sourcecontract.StateRemoved, Removed: true}
		}},
		{name: "superseded", request: SourceSnapshotDownloadRequest{MissionID: "mis_1", SnapshotID: "src_1"}, mutate: func(s *sourceDownloadTestStore) {
			s.events = []ledger.Event{{EventID: "evt_updated", Sequence: 10, EventType: ConfluenceUpdatedEvent, Payload: json.RawMessage(`{"old_snapshot_id":"src_1","new_snapshot_id":"src_2"}`)}}
			s.snapshot.State = sourcecontract.State{State: sourcecontract.StateActive, Superseded: true}
		}},
		{name: "live reference", request: SourceSnapshotDownloadRequest{MissionID: "mis_1", SnapshotID: "src_1"}, mutate: func(s *sourceDownloadTestStore) {
			s.snapshot.Access.RetrievalPolicy = sourcecontract.RetrievalPolicyLiveReference
		}},
		{name: "Confluence connector snapshot", request: SourceSnapshotDownloadRequest{MissionID: "mis_1", SnapshotID: "src_1"}, mutate: func(s *sourceDownloadTestStore) {
			s.snapshot.Connector = sourcecontract.ConnectorRef{ConnectorID: confluencesource.ConfluenceConnectorID, ConnectorType: confluencesource.ConfluenceConnectorType}
		}},
		{name: "Liquid2 connector snapshot", request: SourceSnapshotDownloadRequest{MissionID: "mis_1", SnapshotID: "src_1"}, mutate: func(s *sourceDownloadTestStore) {
			s.snapshot.Connector = sourcecontract.ConnectorRef{ConnectorID: liquid2source.Liquid2ConnectorID, ConnectorType: liquid2source.Liquid2ConnectorType}
		}},
		{name: "browser rendered HTML remains downloadable", request: SourceSnapshotDownloadRequest{MissionID: "mis_1", SnapshotID: "src_1"}, mutate: func(s *sourceDownloadTestStore) {
			s.snapshot.Connector = sourcecontract.ConnectorRef{ConnectorID: "url", ConnectorType: "url", ExternalURI: "https://example.com/app"}
			s.snapshot.Locators = json.RawMessage(`[{"locator_type":"full_document","retrieval_method":"browser_render"}]`)
			s.artifacts["art_pdf"] = artifactcontract.Raw{ArtifactID: "art_pdf", MissionID: "mis_1", MediaType: "text/html", Content: []byte("<html>rendered</html>")}
		}, want: "<html>rendered</html>"},
		{name: "cross mission snapshot", request: SourceSnapshotDownloadRequest{MissionID: "mis_1", SnapshotID: "src_1"}, mutate: func(s *sourceDownloadTestStore) { s.snapshot.MissionID = "mis_2" }},
		{name: "cross mission artifact", request: SourceSnapshotDownloadRequest{MissionID: "mis_1", SnapshotID: "src_1"}, mutate: func(s *sourceDownloadTestStore) {
			s.artifacts["art_pdf"] = artifactcontract.Raw{ArtifactID: "art_pdf", MissionID: "mis_2"}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := fresh()
			if tc.mutate != nil {
				tc.mutate(store)
			}
			got, err := NewService(store).ResolveSourceSnapshotDownload(ctx, tc.request)
			if tc.want == "" {
				if !errors.Is(err, ErrSourceDownloadNotFound) {
					t.Fatalf("expected source rejection, got %v", err)
				}
				return
			}
			if err != nil || string(got.Content) != tc.want {
				t.Fatalf("source success: %q %v", got.Content, err)
			}
		})
	}
}

func TestResolveSourceCandidateDownloadHistoricalSnapshotConsumption(t *testing.T) {
	ctx := context.Background()
	candidate := candidateDownloadEvent("evt_staged", 3, "https://example.com/doc", "art_candidate", "evt_proposed")
	newStore := func(events []ledger.Event) *sourceDownloadTestStore {
		return &sourceDownloadTestStore{events: append([]ledger.Event{candidate}, events...), artifacts: map[string]artifactcontract.Raw{"art_candidate": {ArtifactID: "art_candidate", MissionID: "mis_1", Content: []byte("exact")}}}
	}
	for _, tc := range []struct {
		name   string
		events []ledger.Event
		wantOK bool
	}{
		{name: "removed historical approval", events: []ledger.Event{
			{EventID: "evt_source", Sequence: 4, EventType: "source.snapshotted", Payload: json.RawMessage(`{"snapshot_id":"src_1","source_candidate_proposal_event_id":"evt_proposed","url":"https://example.com/doc"}`)},
			{EventID: "evt_removed", Sequence: 5, EventType: SourceRemovedEvent, Payload: json.RawMessage(`{"snapshot_id":"src_1"}`)},
		}},
		{name: "superseded historical approval", events: []ledger.Event{
			{EventID: "evt_source", Sequence: 4, EventType: "source.snapshotted", Payload: json.RawMessage(`{"snapshot_id":"src_1","source_candidate_proposal_event_id":"evt_proposed","url":"https://example.com/doc"}`)},
			{EventID: "evt_updated", Sequence: 5, EventType: ConfluenceUpdatedEvent, Payload: json.RawMessage(`{"old_snapshot_id":"src_1","new_snapshot_id":"src_2"}`)},
		}},
		{name: "browser-render historical approval", events: []ledger.Event{
			{EventID: "evt_source", Sequence: 4, EventType: "source.snapshotted", Payload: json.RawMessage(`{"snapshot_id":"src_1","source_candidate_proposal_event_id":"evt_proposed","url":"https://example.com/doc","retrieval_method":"browser_render","raw_fetch_artifact_id":"art_raw"}`)},
		}},
		{name: "malformed historical event", events: []ledger.Event{{EventID: "evt_bad", Sequence: 4, EventType: "source.snapshotted", Payload: json.RawMessage(`{"source_candidate_proposal_event_id":123}`)}}, wantOK: true},
		{name: "unrelated approved source", events: []ledger.Event{{EventID: "evt_other", Sequence: 4, EventType: "source.snapshotted", Payload: json.RawMessage(`{"source_candidate_proposal_event_id":"evt_other_proposed","url":"https://example.com/other"}`)}}, wantOK: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := newStore(tc.events)
			got, err := NewService(store).ResolveSourceCandidateDownload(ctx, SourceCandidateDownloadRequest{MissionID: "mis_1", StagedEventID: "evt_staged", ArtifactID: "art_candidate"})
			if tc.wantOK {
				if err != nil || string(got.Content) != "exact" {
					t.Fatalf("unrelated snapshot consumed candidate: %q %v", got.Content, err)
				}
				return
			}
			if !errors.Is(err, ErrSourceDownloadNotFound) {
				t.Fatalf("historical approval must consume candidate, got %v", err)
			}
		})
	}
}

func TestResolveSourceCandidateDownloadHistoricalSnapshotURLFallback(t *testing.T) {
	store := &sourceDownloadTestStore{
		events: []ledger.Event{
			candidateDownloadEvent("evt_staged", 1, "https://EXAMPLE.com/doc#fragment", "art_candidate", "evt_proposed"),
			{EventID: "evt_source", Sequence: 2, EventType: "source.snapshotted", Payload: json.RawMessage(`{"url":"https://example.com/doc"}`)},
		},
		artifacts: map[string]artifactcontract.Raw{"art_candidate": {ArtifactID: "art_candidate", MissionID: "mis_1", Content: []byte("x")}},
	}
	_, err := NewService(store).ResolveSourceCandidateDownload(context.Background(), SourceCandidateDownloadRequest{MissionID: "mis_1", StagedEventID: "evt_staged", ArtifactID: "art_candidate"})
	if !errors.Is(err, ErrSourceDownloadNotFound) {
		t.Fatalf("URL fallback must consume historical snapshot, got %v", err)
	}
}

func TestResolveSourceCandidateDownloadHistoricalConfluenceIdentityFallback(t *testing.T) {
	store := &sourceDownloadTestStore{
		events: []ledger.Event{
			candidateDownloadEvent("evt_staged", 1, "https://docs.atlassian.net/wiki/spaces/ENG/pages/123/Roadmap", "art_candidate", "evt_proposed"),
			{EventID: "evt_source", Sequence: 2, EventType: "source.snapshotted", Payload: json.RawMessage(`{"connector":{"connector_id":"confluence","connector_type":"confluence_cloud","external_source_id":"site_docs.atlassian.net:123"}}`)},
		},
		artifacts: map[string]artifactcontract.Raw{"art_candidate": {ArtifactID: "art_candidate", MissionID: "mis_1", Content: []byte("x")}},
	}
	_, err := NewService(store).ResolveSourceCandidateDownload(context.Background(), SourceCandidateDownloadRequest{MissionID: "mis_1", StagedEventID: "evt_staged", ArtifactID: "art_candidate"})
	if !errors.Is(err, ErrSourceDownloadNotFound) {
		t.Fatalf("legacy Confluence identity must consume historical snapshot, got %v", err)
	}
}

func candidateDownloadEvent(id string, sequence int64, rawURL, artifactID, proposalID string) ledger.Event {
	payload, _ := json.Marshal(map[string]any{"url": rawURL, "artifact_id": artifactID, "proposal_event_id": proposalID})
	return ledger.Event{EventID: id, Sequence: sequence, EventType: "source.candidate.staged", Payload: payload}
}
func candidateTerminal(kind, id string, sequence int64, rawURL string) ledger.Event {
	payload, _ := json.Marshal(map[string]string{"url": rawURL})
	return ledger.Event{EventID: id, Sequence: sequence, EventType: kind, Payload: payload}
}
func candidateDecision(kind, id string, sequence int64, rawURL string) ledger.Event {
	return candidateTerminal(kind, id, sequence, rawURL)
}
func cloneSourceDownloadStore(s *sourceDownloadTestStore) *sourceDownloadTestStore {
	c := *s
	c.events = append([]ledger.Event(nil), s.events...)
	c.artifacts = map[string]artifactcontract.Raw{}
	for k, v := range s.artifacts {
		c.artifacts[k] = v
	}
	c.snapshots = append([]sourcecontract.Snapshot(nil), s.snapshots...)
	return &c
}
