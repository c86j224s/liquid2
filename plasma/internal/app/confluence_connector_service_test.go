package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestSearchConfluenceSourcesNormalizesAndDelegates(t *testing.T) {
	connector := &fakeConfluenceConnector{
		searchResult: ConfluenceSourceSearchResult{
			Candidates: []ConfluenceSourceCandidate{{
				Connector: ConnectorRef{ExternalSourceID: ConfluenceExternalSourceID("cloud_1", "123")},
				Title:     " Roadmap ",
				Summary:   " private excerpt ",
			}},
		},
	}
	svc := NewService(fakeStore{})
	result, err := svc.SearchConfluenceSources(context.Background(), connector, ConfluenceSourceSearchRequest{
		MissionID: " mis_1 ",
		CloudID:   " cloud_1 ",
		Query:     " roadmap ",
		Limit:     500,
		SpaceKey:  " ENG ",
	})
	if err != nil {
		t.Fatalf("SearchConfluenceSources returned error: %v", err)
	}
	if connector.searchRequest.MissionID != "mis_1" ||
		connector.searchRequest.CloudID != "cloud_1" ||
		connector.searchRequest.Query != "roadmap" ||
		connector.searchRequest.SpaceKey != "ENG" {
		t.Fatalf("request was not normalized: %#v", connector.searchRequest)
	}
	if connector.searchRequest.Limit != maxConfluenceSearchLimit {
		t.Fatalf("expected capped limit, got %d", connector.searchRequest.Limit)
	}
	if result.CloudID != "cloud_1" || len(result.Candidates) != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
	candidate := result.Candidates[0]
	if candidate.Connector.ConnectorID != ConfluenceConnectorID ||
		candidate.Connector.ExternalURI != "" ||
		candidate.Summary != "" ||
		!candidate.CanSnapshot {
		t.Fatalf("candidate was not normalized: %#v", candidate)
	}
}

func TestSnapshotConfluenceSourcePersistsArtifactAndSnapshot(t *testing.T) {
	updatedAt := time.Date(2026, 7, 2, 5, 10, 0, 0, time.UTC)
	store := &confluenceSnapshotFakeStore{}
	connector := &fakeConfluenceConnector{
		page: ConfluenceSourcePage{
			CloudID:     "cloud_1",
			SiteURL:     "https://example.atlassian.net/wiki",
			PageID:      "123",
			SpaceID:     "987",
			SpaceKey:    "ENG",
			Title:       "Confluence roadmap",
			WebURL:      "https://example.atlassian.net/wiki/spaces/ENG/pages/123",
			Version:     7,
			UpdatedAt:   updatedAt,
			BodyStorage: "<p>Hello <strong>research</strong></p>",
			PlainText:   "Hello research",
			Metadata:    json.RawMessage(`{"status":"current"}`),
		},
	}
	svc := NewService(store)
	result, err := svc.SnapshotConfluenceSource(context.Background(), connector, SnapshotConfluenceSourceRequest{
		MissionID:       "mis_1",
		ArtifactID:      "art_1",
		SnapshotID:      "src_1",
		CloudID:         "cloud_1",
		PageID:          "123",
		ExpectedVersion: 7,
		Reason:          "support claim",
	})
	if err != nil {
		t.Fatalf("SnapshotConfluenceSource returned error: %v", err)
	}
	if connector.readRequest.CloudID != "cloud_1" || connector.readRequest.PageID != "123" {
		t.Fatalf("unexpected read request: %#v", connector.readRequest)
	}
	if result.Artifact.MediaType != ConfluenceSnapshotMediaType ||
		result.Artifact.Producer.Type != "connector" ||
		result.Artifact.Producer.ID != ConfluenceConnectorID {
		t.Fatalf("unexpected artifact: %#v", result.Artifact)
	}
	content := string(result.Artifact.Content)
	for _, want := range []string{
		`"schema_version":"plasma.confluence.snapshot.v1"`,
		`"cloud_id":"cloud_1"`,
		`"format":"confluence_storage"`,
		`"content":"Hello research"`,
		`"storage_sha256":`,
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("artifact content missing %q: %s", want, content)
		}
	}
	if result.Snapshot.Connector.ExternalSourceID != ConfluenceExternalSourceID("cloud_1", "123") ||
		result.Snapshot.Connector.ConnectorVersion != ConfluenceHTTPConnectorV1 ||
		result.Snapshot.ExternalUpdatedAt != updatedAt {
		t.Fatalf("unexpected snapshot connector metadata: %#v", result.Snapshot)
	}
	if !strings.Contains(string(result.Snapshot.Locators), `"locator_type":"confluence_page_body"`) {
		t.Fatalf("snapshot locators were not recorded: %s", string(result.Snapshot.Locators))
	}
	if !strings.Contains(string(result.Snapshot.Locators), `"site_url":"https://example.atlassian.net/wiki"`) {
		t.Fatalf("snapshot locators missing site URL: %s", string(result.Snapshot.Locators))
	}
}

func TestSnapshotConfluenceSourceWithEventBuildsSourceSnapshottedEvent(t *testing.T) {
	updatedAt := time.Date(2026, 7, 2, 5, 10, 0, 0, time.UTC)
	store := &confluenceSnapshotFakeStore{}
	connector := &fakeConfluenceConnector{
		page: ConfluenceSourcePage{
			CloudID:     "cloud_1",
			PageID:      "123",
			Title:       "Confluence roadmap",
			Version:     7,
			UpdatedAt:   updatedAt,
			BodyStorage: "<p>Hello <strong>research</strong></p>",
			PlainText:   "Hello research",
			Metadata:    json.RawMessage(`{"status":"current"}`),
		},
	}
	svc := NewService(store)
	result, err := svc.SnapshotConfluenceSourceWithEvent(context.Background(), connector, SnapshotConfluenceSourceWithEventRequest{
		Snapshot: SnapshotConfluenceSourceRequest{
			MissionID:       "mis_1",
			ArtifactID:      "art_1",
			SnapshotID:      "src_1",
			CloudID:         "cloud_1",
			PageID:          "123",
			ExpectedVersion: 7,
			Reason:          " support claim ",
		},
		EventID:  "evt_1",
		Producer: Producer{Type: "user", ID: "plasma-ui"},
	})
	if err != nil {
		t.Fatalf("SnapshotConfluenceSourceWithEvent returned error: %v", err)
	}
	if result.Event.EventID != "evt_1" || result.Event.EventType != "source.snapshotted" ||
		result.Event.Producer.Type != "user" || result.Event.Producer.ID != "plasma-ui" {
		t.Fatalf("unexpected event shell: %#v", result.Event)
	}
	assertJSONPayloadIncludes(t, result.Event.Payload, map[string]any{
		"snapshot_id":  "src_1",
		"artifact_ids": []any{"art_1"},
		"reason":       "support claim",
		"connector": map[string]any{
			"connector_id":       ConfluenceConnectorID,
			"connector_type":     ConfluenceConnectorType,
			"external_source_id": ConfluenceExternalSourceID("cloud_1", "123"),
			"external_uri":       ConfluenceExternalURI("cloud_1", "123"),
			"external_version":   "",
			"connector_version":  ConfluenceHTTPConnectorV1,
		},
	})
}

func TestSnapshotConfluenceSourceRejectsVersionDrift(t *testing.T) {
	connector := &fakeConfluenceConnector{
		page: ConfluenceSourcePage{
			CloudID:     "cloud_1",
			PageID:      "123",
			Version:     8,
			BodyStorage: "<p>Hello</p>",
			PlainText:   "Hello",
		},
	}
	svc := NewService(&confluenceSnapshotFakeStore{})
	_, err := svc.SnapshotConfluenceSource(context.Background(), connector, SnapshotConfluenceSourceRequest{
		MissionID:       "mis_1",
		ArtifactID:      "art_1",
		SnapshotID:      "src_1",
		CloudID:         "cloud_1",
		PageID:          "123",
		ExpectedVersion: 7,
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestSnapshotConfluenceSourceRejectsMissingExpectedVersion(t *testing.T) {
	connector := &fakeConfluenceConnector{
		page: ConfluenceSourcePage{
			CloudID:     "cloud_1",
			PageID:      "123",
			Title:       "Roadmap",
			Version:     7,
			BodyStorage: "<p>Hello</p>",
			PlainText:   "Hello",
		},
	}
	svc := NewService(&confluenceSnapshotFakeStore{})
	_, err := svc.SnapshotConfluenceSource(context.Background(), connector, SnapshotConfluenceSourceRequest{
		MissionID:  "mis_1",
		ArtifactID: "art_1",
		SnapshotID: "src_1",
		CloudID:    "cloud_1",
		PageID:     "123",
	})
	if err == nil || !strings.Contains(err.Error(), "page version") {
		t.Fatalf("expected missing expected version error, got %v", err)
	}
	if connector.readRequest.PageID != "" {
		t.Fatalf("connector should not be read before version validation: %#v", connector.readRequest)
	}
}

func TestSnapshotConfluenceSourceRejectsMissingPlainText(t *testing.T) {
	connector := &fakeConfluenceConnector{
		page: ConfluenceSourcePage{
			CloudID:     "cloud_1",
			PageID:      "123",
			Version:     7,
			BodyStorage: "<p>Hello</p>",
		},
	}
	svc := NewService(&confluenceSnapshotFakeStore{})
	_, err := svc.SnapshotConfluenceSource(context.Background(), connector, SnapshotConfluenceSourceRequest{
		MissionID:  "mis_1",
		ArtifactID: "art_1",
		SnapshotID: "src_1",
		CloudID:    "cloud_1",
		PageID:     "123",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestSnapshotConfluenceSourceRejectsOversizedBody(t *testing.T) {
	connector := &fakeConfluenceConnector{
		page: ConfluenceSourcePage{
			CloudID:     "cloud_1",
			PageID:      "123",
			Version:     7,
			BodyStorage: "<p>too large</p>",
			PlainText:   "too large",
		},
	}
	svc := NewService(&confluenceSnapshotFakeStore{})
	_, err := svc.SnapshotConfluenceSource(context.Background(), connector, SnapshotConfluenceSourceRequest{
		MissionID:    "mis_1",
		ArtifactID:   "art_1",
		SnapshotID:   "src_1",
		CloudID:      "cloud_1",
		PageID:       "123",
		MaxBodyBytes: 4,
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestPreviewConfluenceSourceDoesNotCreateSourceAndOffersRanges(t *testing.T) {
	store := &confluenceSnapshotFakeStore{}
	connector := &fakeConfluenceConnector{
		page: ConfluenceSourcePage{
			CloudID:     "cloud_1",
			PageID:      "123",
			Title:       "Large page",
			Version:     7,
			BodyStorage: "<p>" + strings.Repeat("large ", 20) + "</p>",
			PlainText:   strings.Repeat("large ", 20),
			Metadata:    json.RawMessage(`{"status":"current"}`),
		},
	}
	svc := NewService(store)
	result, err := svc.PreviewConfluenceSource(context.Background(), connector, ConfluenceSourcePreviewRequest{
		MissionID:    "mis_1",
		CloudID:      "cloud_1",
		PageID:       "123",
		MaxBodyBytes: 8,
		PreviewRunes: 12,
	})
	if err != nil {
		t.Fatalf("PreviewConfluenceSource returned error: %v", err)
	}
	if !result.FullBodyTooLarge || len(result.RangeOptions) == 0 || result.CandidateKind != "confluence_page_preview_result" {
		t.Fatalf("unexpected preview result: %#v", result)
	}
	if len(store.artifacts) != 0 || len(store.snapshots) != 0 || len(store.events) != 0 {
		t.Fatalf("preview should not create durable source data: %#v %#v %#v", store.artifacts, store.snapshots, store.events)
	}
	for _, option := range result.RangeOptions {
		body, err := confluenceRangeBody(connector.page.PlainText, ConfluenceRangeSelection{
			ContentID: option.ContentID,
			Start:     option.Start,
			End:       option.End,
		})
		if err != nil {
			t.Fatalf("range option was not readable: %#v: %v", option, err)
		}
		if got := len([]byte(body.Content)); got > 8 {
			t.Fatalf("range option exceeded max bytes: got %d option=%#v body=%q", got, option, body.Content)
		}
	}
}

func TestSnapshotConfluenceSourceRangeStoresPreciseLocator(t *testing.T) {
	connector := &fakeConfluenceConnector{
		page: ConfluenceSourcePage{
			CloudID:     "cloud_1",
			PageID:      "123",
			Title:       "Large page",
			Version:     7,
			BodyStorage: "<p>" + strings.Repeat("alpha ", 40) + "</p>",
			PlainText:   "alpha beta gamma delta",
			Metadata:    json.RawMessage(`{"status":"current"}`),
		},
	}
	svc := NewService(&confluenceSnapshotFakeStore{})
	result, err := svc.SnapshotConfluenceSource(context.Background(), connector, SnapshotConfluenceSourceRequest{
		MissionID:       "mis_1",
		ArtifactID:      "art_1",
		SnapshotID:      "src_1",
		CloudID:         "cloud_1",
		PageID:          "123",
		ExpectedVersion: 7,
		MaxBodyBytes:    12,
		Range:           ConfluenceRangeSelection{ContentID: "plain_text", Start: 6, End: 10},
	})
	if err != nil {
		t.Fatalf("SnapshotConfluenceSource range returned error: %v", err)
	}
	content := string(result.Artifact.Content)
	if strings.Contains(content, strings.Repeat("alpha ", 20)) {
		t.Fatalf("range artifact stored full body: %s", content)
	}
	for _, want := range []string{
		`"partial":true`,
		`"content":"beta"`,
		`"start":6`,
		`"end":10`,
		`"locator_type":"confluence_page_range"`,
	} {
		if !strings.Contains(content+"\n"+string(result.Snapshot.Locators), want) {
			t.Fatalf("range snapshot missing %q content=%s locators=%s", want, content, string(result.Snapshot.Locators))
		}
	}
}

func TestCheckConfluenceSourceUpdateRecordsSafeEventPayload(t *testing.T) {
	updatedAt := time.Date(2026, 7, 3, 2, 0, 0, 0, time.UTC)
	store := &confluenceSnapshotFakeStore{
		snapshots: map[string]SourceSnapshot{
			"src_1": {
				SnapshotID: "src_1",
				MissionID:  "mis_1",
				Connector: ConnectorRef{
					ConnectorID:      ConfluenceConnectorID,
					ConnectorType:    ConfluenceConnectorType,
					ExternalSourceID: ConfluenceExternalSourceID("cloud_1", "123"),
					ExternalVersion:  "7",
				},
				Locators: json.RawMessage(`[{"cloud_id":"cloud_1","page_id":"123"}]`),
			},
		},
	}
	connector := &fakeConfluenceConnector{
		version: ConfluenceSourceVersion{
			CloudID:   "cloud_1",
			PageID:    "123",
			Title:     "Roadmap",
			Version:   8,
			UpdatedAt: updatedAt,
		},
	}
	svc := NewService(store)
	result, err := svc.CheckConfluenceSourceUpdateWithEvent(context.Background(), connector, CheckConfluenceSourceUpdateRequest{
		MissionID:  "mis_1",
		SnapshotID: "src_1",
		EventID:    "evt_update_check",
		Producer:   Producer{Type: "user", ID: "plasma-cli"},
	})
	if err != nil {
		t.Fatalf("CheckConfluenceSourceUpdateWithEvent returned error: %v", err)
	}
	if !result.UpdateAvailable || result.LatestVersion != 8 {
		t.Fatalf("unexpected update result: %#v", result)
	}
	if len(store.events) != 1 || store.events[0].EventType != ConfluenceUpdateAvailableEvent {
		t.Fatalf("unexpected events: %#v", store.events)
	}
	payload := string(store.events[0].Payload)
	for _, leaked := range []string{"storage", "plain_text", "secret-token"} {
		if strings.Contains(payload, leaked) {
			t.Fatalf("event payload leaked %q: %s", leaked, payload)
		}
	}
	sources, err := svc.ListSourceSnapshotsWithState(context.Background(), ListSourceSnapshotsRequest{MissionID: "mis_1"})
	if err != nil {
		t.Fatalf("ListSourceSnapshotsWithState returned error: %v", err)
	}
	if len(sources) != 1 || sources[0].State.ConfluenceUpdate == nil {
		t.Fatalf("expected projected Confluence update state, got %#v", sources)
	}
	state := sources[0].State.ConfluenceUpdate
	if state.Status != ConfluenceUpdateStatusAvailable || state.CurrentVersion != 7 || state.LatestVersion != 8 {
		t.Fatalf("unexpected projected Confluence update state: %#v", state)
	}
}

func TestCheckConfluenceSourceUpdateRecordsNotFoundWithoutMarkingSnapshotDeleted(t *testing.T) {
	store := &confluenceSnapshotFakeStore{
		snapshots: map[string]SourceSnapshot{
			"src_1": confluenceTestSnapshot("src_1", "mis_1", "cloud_1", "123", "7"),
		},
	}
	connector := &fakeConfluenceConnector{
		versionErr: NewConfluenceHTTPError(404, "", "get_page_version"),
	}
	svc := NewService(store)
	_, err := svc.CheckConfluenceSourceUpdateWithEvent(context.Background(), connector, CheckConfluenceSourceUpdateRequest{
		MissionID:  "mis_1",
		SnapshotID: "src_1",
		EventID:    "evt_update_check_failed",
		Producer:   Producer{Type: "user", ID: "plasma-cli"},
	})
	if err == nil {
		t.Fatal("expected Confluence not-found error")
	}
	if len(store.events) != 1 || store.events[0].EventType != ConfluenceUpdateFailedEvent {
		t.Fatalf("unexpected failure events: %#v", store.events)
	}
	payload := string(store.events[0].Payload)
	for _, forbidden := range []string{"page body", "provider response", "source.deleted"} {
		if strings.Contains(payload, forbidden) {
			t.Fatalf("failure event leaked or invented %q: %s", forbidden, payload)
		}
	}
	sources, listErr := svc.ListSourceSnapshotsWithState(context.Background(), ListSourceSnapshotsRequest{MissionID: "mis_1"})
	if listErr != nil {
		t.Fatalf("ListSourceSnapshotsWithState returned error: %v", listErr)
	}
	if len(sources) != 1 || sources[0].State.ConfluenceUpdate == nil {
		t.Fatalf("expected failed Confluence update state, got %#v", sources)
	}
	state := sources[0].State
	if state.Removed || state.State != SourceStateActive {
		t.Fatalf("not-found observation must not remove the snapshot: %#v", state)
	}
	if state.ConfluenceUpdate.Status != ConfluenceUpdateStatusFailed ||
		state.ConfluenceUpdate.ErrorCategory != ConfluenceErrorCategoryNotFound ||
		state.ConfluenceUpdate.ErrorCode != ConfluenceErrorCodeNotFound {
		t.Fatalf("unexpected failed Confluence update state: %#v", state.ConfluenceUpdate)
	}
	store.events = append(store.events, LedgerEvent{
		EventID:   "evt_removed_after_check",
		MissionID: "mis_1",
		EventType: SourceRemovedEvent,
		Payload:   json.RawMessage(`{"snapshot_id":"src_1","reason":"user cleanup"}`),
		CreatedAt: time.Now().UTC(),
	})
	removedSources, listErr := svc.ListSourceSnapshotsWithState(context.Background(), ListSourceSnapshotsRequest{
		MissionID:      "mis_1",
		IncludeRemoved: true,
	})
	if listErr != nil {
		t.Fatalf("ListSourceSnapshotsWithState including removed returned error: %v", listErr)
	}
	if len(removedSources) != 1 || !removedSources[0].State.Removed || removedSources[0].State.ConfluenceUpdate == nil {
		t.Fatalf("removing a source must preserve its last Confluence check state: %#v", removedSources)
	}
}

func TestCheckConfluenceSourceUpdateDoesNotRecordLocalValidationFailure(t *testing.T) {
	store := &confluenceSnapshotFakeStore{}
	svc := NewService(store)
	_, err := svc.CheckConfluenceSourceUpdateWithEvent(context.Background(), &fakeConfluenceConnector{}, CheckConfluenceSourceUpdateRequest{
		MissionID:  "mis_1",
		SnapshotID: "src_missing",
		EventID:    "evt_update_check_failed",
		Producer:   Producer{Type: "user", ID: "plasma-cli"},
	})
	if err == nil {
		t.Fatal("expected missing snapshot error")
	}
	if len(store.events) != 0 {
		t.Fatalf("local validation failure must not create an observation event: %#v", store.events)
	}
}

func TestConfluenceUpdateFailureRejectsUnknownPublicErrorCode(t *testing.T) {
	err := validateConfluenceUpdateStateEventPayload(json.RawMessage(`{
		"old_snapshot_id":"src_1",
		"checked_at":"2026-07-14T01:02:03Z",
		"error_category":"confluence_auth",
		"error_code":"raw_provider_detail"
	}`))
	if err == nil {
		t.Fatal("expected unknown Confluence error code to be rejected")
	}
}

func TestConfluenceUpdatePathsRejectRemovedSnapshot(t *testing.T) {
	store := &confluenceSnapshotFakeStore{
		snapshots: map[string]SourceSnapshot{
			"src_1": confluenceTestSnapshot("src_1", "mis_1", "cloud_1", "123", "7"),
		},
		events: []LedgerEvent{{
			EventID:   "evt_removed",
			MissionID: "mis_1",
			EventType: SourceRemovedEvent,
			Payload:   json.RawMessage(`{"snapshot_id":"src_1","reason":"mistake"}`),
		}},
	}
	connector := &fakeConfluenceConnector{}
	svc := NewService(store)
	assertConfluenceUpdatePathRejected(t, svc, connector, "src_1", "source is removed")
	if connector.readRequest.PageID != "" {
		t.Fatalf("connector should not be read for removed snapshot: %#v", connector.readRequest)
	}
}

func TestConfluenceUpdatePathsRejectSupersededSnapshot(t *testing.T) {
	store := &confluenceSnapshotFakeStore{
		snapshots: map[string]SourceSnapshot{
			"src_old": confluenceTestSnapshot("src_old", "mis_1", "cloud_1", "123", "7"),
			"src_new": confluenceTestSnapshot("src_new", "mis_1", "cloud_1", "123", "8"),
		},
		events: []LedgerEvent{{
			EventID:   "evt_updated",
			MissionID: "mis_1",
			EventType: ConfluenceUpdatedEvent,
			Payload:   json.RawMessage(`{"old_snapshot_id":"src_old","new_snapshot_id":"src_new"}`),
		}},
	}
	connector := &fakeConfluenceConnector{}
	svc := NewService(store)
	assertConfluenceUpdatePathRejected(t, svc, connector, "src_old", "source has been superseded")
	if connector.readRequest.PageID != "" {
		t.Fatalf("connector should not be read for superseded snapshot: %#v", connector.readRequest)
	}
}

func TestConfluenceUpdatePathsRejectRestoredSupersededSnapshot(t *testing.T) {
	store := &confluenceSnapshotFakeStore{
		snapshots: map[string]SourceSnapshot{
			"src_old": confluenceTestSnapshot("src_old", "mis_1", "cloud_1", "123", "7"),
			"src_new": confluenceTestSnapshot("src_new", "mis_1", "cloud_1", "123", "8"),
		},
		events: []LedgerEvent{
			{
				EventID:   "evt_updated",
				MissionID: "mis_1",
				EventType: ConfluenceUpdatedEvent,
				Payload:   json.RawMessage(`{"old_snapshot_id":"src_old","new_snapshot_id":"src_new"}`),
			},
			{
				EventID:   "evt_removed",
				MissionID: "mis_1",
				EventType: SourceRemovedEvent,
				Payload:   json.RawMessage(`{"snapshot_id":"src_old","reason":"cleanup"}`),
			},
			{
				EventID:   "evt_restored",
				MissionID: "mis_1",
				EventType: SourceRestoredEvent,
				Payload:   json.RawMessage(`{"snapshot_id":"src_old"}`),
			},
		},
	}
	connector := &fakeConfluenceConnector{}
	svc := NewService(store)
	assertConfluenceUpdatePathRejected(t, svc, connector, "src_old", "source has been superseded")
	if connector.readRequest.PageID != "" {
		t.Fatalf("connector should not be read for restored superseded snapshot: %#v", connector.readRequest)
	}
}

func TestConfluenceUpdatePathsKeepOldSupersededWhenNewerSnapshotRemoved(t *testing.T) {
	store := &confluenceSnapshotFakeStore{
		snapshots: map[string]SourceSnapshot{
			"src_old": confluenceTestSnapshot("src_old", "mis_1", "cloud_1", "123", "7"),
			"src_new": confluenceTestSnapshot("src_new", "mis_1", "cloud_1", "123", "8"),
		},
		events: []LedgerEvent{
			{
				EventID:   "evt_updated",
				MissionID: "mis_1",
				EventType: ConfluenceUpdatedEvent,
				Payload:   json.RawMessage(`{"old_snapshot_id":"src_old","new_snapshot_id":"src_new"}`),
			},
			{
				EventID:   "evt_removed_new",
				MissionID: "mis_1",
				EventType: SourceRemovedEvent,
				Payload:   json.RawMessage(`{"snapshot_id":"src_new","reason":"bad update"}`),
			},
		},
	}
	svc := NewService(store)
	oldConnector := &fakeConfluenceConnector{}
	assertConfluenceUpdatePathRejected(t, svc, oldConnector, "src_old", "source has been superseded")
	if oldConnector.readRequest.PageID != "" {
		t.Fatalf("connector should not be read for old superseded snapshot: %#v", oldConnector.readRequest)
	}
	newConnector := &fakeConfluenceConnector{}
	assertConfluenceUpdatePathRejected(t, svc, newConnector, "src_new", "source is removed")
	if newConnector.readRequest.PageID != "" {
		t.Fatalf("connector should not be read for removed newer snapshot: %#v", newConnector.readRequest)
	}
}

func TestListSourceSnapshotsHidesSupersededByDefault(t *testing.T) {
	store := &confluenceSnapshotFakeStore{
		snapshots: map[string]SourceSnapshot{
			"src_old": confluenceTestSnapshot("src_old", "mis_1", "cloud_1", "123", "7"),
			"src_new": confluenceTestSnapshot("src_new", "mis_1", "cloud_1", "123", "8"),
		},
		events: []LedgerEvent{{
			EventID:   "evt_updated",
			MissionID: "mis_1",
			EventType: ConfluenceUpdatedEvent,
			Payload:   json.RawMessage(`{"old_snapshot_id":"src_old","new_snapshot_id":"src_new"}`),
		}},
	}
	svc := NewService(store)
	defaultSources, err := svc.ListSourceSnapshotsWithState(context.Background(), ListSourceSnapshotsRequest{MissionID: "mis_1"})
	if err != nil {
		t.Fatalf("ListSourceSnapshotsWithState returned error: %v", err)
	}
	defaultIDs := sourceSnapshotIDs(defaultSources)
	if defaultIDs["src_old"] {
		t.Fatalf("default source list should hide superseded snapshot: %#v", defaultSources)
	}
	if !defaultIDs["src_new"] {
		t.Fatalf("default source list should include current snapshot: %#v", defaultSources)
	}
	auditSources, err := svc.ListSourceSnapshotsWithState(context.Background(), ListSourceSnapshotsRequest{
		MissionID:         "mis_1",
		IncludeSuperseded: true,
	})
	if err != nil {
		t.Fatalf("ListSourceSnapshotsWithState audit returned error: %v", err)
	}
	auditIDs := sourceSnapshotIDs(auditSources)
	if !auditIDs["src_old"] || !auditIDs["src_new"] {
		t.Fatalf("include superseded should show both snapshots: %#v", auditSources)
	}
}

func TestConfluenceUpdatePathsRejectOlderActiveSnapshotForSameIdentity(t *testing.T) {
	store := &confluenceSnapshotFakeStore{
		snapshots: map[string]SourceSnapshot{
			"src_old": confluenceTestSnapshot("src_old", "mis_1", "cloud_1", "123", "7"),
			"src_new": confluenceTestSnapshot("src_new", "mis_1", "cloud_1", "123", "8"),
		},
	}
	connector := &fakeConfluenceConnector{}
	svc := NewService(store)
	assertConfluenceUpdatePathRejected(t, svc, connector, "src_old", "current active snapshot")
	if connector.readRequest.PageID != "" {
		t.Fatalf("connector should not be read for older active snapshot: %#v", connector.readRequest)
	}
}

func TestUpdateConfluenceSourceCreatesNewSnapshotAndSupersedesOld(t *testing.T) {
	store := &confluenceSnapshotFakeStore{
		snapshots: map[string]SourceSnapshot{
			"src_old": {
				SnapshotID: "src_old",
				MissionID:  "mis_1",
				Connector: ConnectorRef{
					ConnectorID:      ConfluenceConnectorID,
					ConnectorType:    ConfluenceConnectorType,
					ExternalSourceID: ConfluenceExternalSourceID("cloud_1", "123"),
					ExternalVersion:  "7",
				},
				Locators: json.RawMessage(`[{"cloud_id":"cloud_1","page_id":"123"}]`),
			},
		},
	}
	connector := &fakeConfluenceConnector{
		page: ConfluenceSourcePage{
			CloudID:     "cloud_1",
			PageID:      "123",
			Title:       "Roadmap",
			Version:     8,
			BodyStorage: "<p>Hello</p>",
			PlainText:   "Hello",
		},
	}
	svc := NewService(store)
	result, err := svc.UpdateConfluenceSourceWithEvent(context.Background(), connector, UpdateConfluenceSourceRequest{
		MissionID:          "mis_1",
		PreviousSnapshotID: "src_old",
		ArtifactID:         "art_new",
		SnapshotID:         "src_new",
		ExpectedVersion:    8,
		SnapshotEventID:    "evt_snapshot",
		UpdateEventID:      "evt_updated",
		Producer:           Producer{Type: "user", ID: "plasma-cli"},
	})
	if err != nil {
		t.Fatalf("UpdateConfluenceSourceWithEvent returned error: %v", err)
	}
	if result.PreviousSnapshot.SnapshotID != "src_old" || result.Snapshot.SnapshotID != "src_new" {
		t.Fatalf("unexpected update result: %#v", result)
	}
	if len(store.events) != 2 || store.events[1].EventType != ConfluenceUpdatedEvent {
		t.Fatalf("unexpected committed events: %#v", store.events)
	}
	state, err := svc.sourceState(context.Background(), "mis_1", "src_old")
	if err != nil {
		t.Fatalf("sourceState returned error: %v", err)
	}
	if !state.Superseded || state.SupersededBy != "src_new" {
		t.Fatalf("old snapshot was not superseded: %#v", state)
	}
}

func TestUpdateConfluenceSourceRejectsMissingExpectedVersion(t *testing.T) {
	store := &confluenceSnapshotFakeStore{
		snapshots: map[string]SourceSnapshot{
			"src_old": {
				SnapshotID: "src_old",
				MissionID:  "mis_1",
				Connector: ConnectorRef{
					ConnectorID:      ConfluenceConnectorID,
					ConnectorType:    ConfluenceConnectorType,
					ExternalSourceID: ConfluenceExternalSourceID("cloud_1", "123"),
					ExternalVersion:  "7",
				},
				Locators: json.RawMessage(`[{"cloud_id":"cloud_1","page_id":"123"}]`),
			},
		},
	}
	connector := &fakeConfluenceConnector{}
	svc := NewService(store)
	_, err := svc.UpdateConfluenceSourceWithEvent(context.Background(), connector, UpdateConfluenceSourceRequest{
		MissionID:          "mis_1",
		PreviousSnapshotID: "src_old",
		ArtifactID:         "art_new",
		SnapshotID:         "src_new",
		SnapshotEventID:    "evt_snapshot",
		UpdateEventID:      "evt_updated",
	})
	if err == nil || !strings.Contains(err.Error(), "page version") {
		t.Fatalf("expected missing expected version error, got %v", err)
	}
	if connector.readRequest.PageID != "" {
		t.Fatalf("connector should not be read before version validation: %#v", connector.readRequest)
	}
}

func assertConfluenceUpdatePathRejected(t *testing.T, svc *Service, connector *fakeConfluenceConnector, snapshotID string, message string) {
	t.Helper()
	_, err := svc.CheckConfluenceSourceUpdate(context.Background(), connector, CheckConfluenceSourceUpdateRequest{
		MissionID:  "mis_1",
		SnapshotID: snapshotID,
	})
	if err == nil || !strings.Contains(err.Error(), message) {
		t.Fatalf("expected check-update rejection containing %q, got %v", message, err)
	}
	_, err = svc.PreviewConfluenceSourceUpdate(context.Background(), connector, ConfluenceUpdatePreviewRequest{
		MissionID:  "mis_1",
		SnapshotID: snapshotID,
	})
	if err == nil || !strings.Contains(err.Error(), message) {
		t.Fatalf("expected update-preview rejection containing %q, got %v", message, err)
	}
	_, err = svc.UpdateConfluenceSourceWithEvent(context.Background(), connector, UpdateConfluenceSourceRequest{
		MissionID:          "mis_1",
		PreviousSnapshotID: snapshotID,
		ArtifactID:         "art_new",
		SnapshotID:         "src_newer",
		ExpectedVersion:    9,
		SnapshotEventID:    "evt_snapshot",
		UpdateEventID:      "evt_updated_newer",
	})
	if err == nil || !strings.Contains(err.Error(), message) {
		t.Fatalf("expected update rejection containing %q, got %v", message, err)
	}
}

func confluenceTestSnapshot(snapshotID string, missionID string, cloudID string, pageID string, version string) SourceSnapshot {
	return SourceSnapshot{
		SnapshotID: snapshotID,
		MissionID:  missionID,
		Connector: ConnectorRef{
			ConnectorID:      ConfluenceConnectorID,
			ConnectorType:    ConfluenceConnectorType,
			ExternalSourceID: ConfluenceExternalSourceID(cloudID, pageID),
			ExternalVersion:  version,
		},
		Locators: json.RawMessage(fmt.Sprintf(`[{"cloud_id":%q,"page_id":%q}]`, cloudID, pageID)),
	}
}

func sourceSnapshotIDs(snapshots []SourceSnapshot) map[string]bool {
	ids := map[string]bool{}
	for _, snapshot := range snapshots {
		ids[snapshot.SnapshotID] = true
	}
	return ids
}

type fakeConfluenceConnector struct {
	searchRequest ConfluenceSourceSearchRequest
	searchResult  ConfluenceSourceSearchResult
	searchErr     error
	readRequest   ConfluenceSourceReadRequest
	page          ConfluenceSourcePage
	readErr       error
	version       ConfluenceSourceVersion
	versionErr    error
}

func (f *fakeConfluenceConnector) SearchConfluenceSources(
	_ context.Context,
	req ConfluenceSourceSearchRequest,
) (ConfluenceSourceSearchResult, error) {
	f.searchRequest = req
	if f.searchErr != nil {
		return ConfluenceSourceSearchResult{}, f.searchErr
	}
	return f.searchResult, nil
}

func (f *fakeConfluenceConnector) ReadConfluenceSource(
	_ context.Context,
	req ConfluenceSourceReadRequest,
) (ConfluenceSourcePage, error) {
	f.readRequest = req
	if f.readErr != nil {
		return ConfluenceSourcePage{}, f.readErr
	}
	return f.page, nil
}

func (f *fakeConfluenceConnector) GetConfluenceSourceVersion(
	_ context.Context,
	req ConfluenceSourceReadRequest,
) (ConfluenceSourceVersion, error) {
	f.readRequest = req
	if f.versionErr != nil {
		return ConfluenceSourceVersion{}, f.versionErr
	}
	return f.version, nil
}

type confluenceSnapshotFakeStore struct {
	fakeStore
	artifacts map[string]RawArtifact
	snapshots map[string]SourceSnapshot
	events    []LedgerEvent
}

func (f *confluenceSnapshotFakeStore) CreateRawArtifact(_ context.Context, artifact RawArtifact) error {
	if f.artifacts == nil {
		f.artifacts = map[string]RawArtifact{}
	}
	f.artifacts[artifact.ArtifactID] = artifact
	return nil
}

func (f *confluenceSnapshotFakeStore) GetRawArtifact(_ context.Context, artifactID string) (RawArtifact, error) {
	if artifact, ok := f.artifacts[artifactID]; ok {
		return artifact, nil
	}
	return RawArtifact{}, errors.New("missing artifact")
}

func (f *confluenceSnapshotFakeStore) CreateSourceSnapshot(_ context.Context, snapshot SourceSnapshot) error {
	if f.snapshots == nil {
		f.snapshots = map[string]SourceSnapshot{}
	}
	f.snapshots[snapshot.SnapshotID] = snapshot
	return nil
}

func (f *confluenceSnapshotFakeStore) GetSourceSnapshot(_ context.Context, snapshotID string) (SourceSnapshot, error) {
	if snapshot, ok := f.snapshots[snapshotID]; ok {
		return snapshot, nil
	}
	return SourceSnapshot{}, errors.New("missing snapshot")
}

func (f *confluenceSnapshotFakeStore) ListSourceSnapshots(_ context.Context, missionID string) ([]SourceSnapshot, error) {
	var snapshots []SourceSnapshot
	for _, snapshot := range f.snapshots {
		if snapshot.MissionID == missionID {
			snapshots = append(snapshots, snapshot)
		}
	}
	return snapshots, nil
}

func (f *confluenceSnapshotFakeStore) CommitAtomicWrite(_ context.Context, write AtomicWrite) (AtomicWriteResult, error) {
	if f.artifacts == nil {
		f.artifacts = map[string]RawArtifact{}
	}
	if f.snapshots == nil {
		f.snapshots = map[string]SourceSnapshot{}
	}
	for i, event := range write.Events {
		event.Sequence = int64(len(f.events) + 1)
		f.events = append(f.events, event)
		write.Events[i] = event
	}
	for _, artifact := range write.RawArtifacts {
		f.artifacts[artifact.ArtifactID] = artifact
	}
	for _, snapshot := range write.SourceSnapshots {
		f.snapshots[snapshot.SnapshotID] = snapshot
	}
	return AtomicWriteResult{Events: write.Events}, nil
}

func (f *confluenceSnapshotFakeStore) ListLedgerEvents(context.Context, string) ([]LedgerEvent, error) {
	return append([]LedgerEvent(nil), f.events...), nil
}
