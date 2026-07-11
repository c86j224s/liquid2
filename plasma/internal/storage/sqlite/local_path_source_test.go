package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/sources/localpath"
)

func TestLocalPathSourceAttachObserveRemoveRestore(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "guide.txt"), []byte("alpha\nneedle beta\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "note.txt"), []byte("needle in note"), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := newLocalPathSQLiteService(t, root)

	attached, err := svc.AttachLocalPathSource(ctx, app.AttachLocalPathSourceRequest{
		MissionID:    "mis_1",
		RootID:       "docs",
		RelativePath: "guide.txt",
		Producer:     app.Producer{Type: "user", ID: "test"},
	})
	if err != nil {
		t.Fatalf("AttachLocalPathSource returned error: %v", err)
	}
	if attached.Snapshot.Access.RetrievalPolicy != app.SourceRetrievalPolicyLiveReference ||
		attached.Snapshot.Connector.ConnectorType != app.SourceConnectorTypeLocalPath ||
		len(attached.Snapshot.ArtifactIDs) != 0 {
		t.Fatalf("unexpected live source: %#v", attached.Snapshot)
	}
	duplicate, err := svc.AttachLocalPathSource(ctx, app.AttachLocalPathSourceRequest{
		MissionID:    "mis_1",
		RootID:       "docs",
		RelativePath: "guide.txt",
	})
	if err != nil {
		t.Fatalf("duplicate AttachLocalPathSource returned error: %v", err)
	}
	if !duplicate.Existing || duplicate.Snapshot.SnapshotID != attached.Snapshot.SnapshotID {
		t.Fatalf("expected active duplicate to return existing source, got %#v", duplicate)
	}
	read, err := svc.ReadLocalPathSource(ctx, app.ReadLocalPathSourceRequest{
		MissionID:  "mis_1",
		SnapshotID: attached.Snapshot.SnapshotID,
		MaxBytes:   6,
		Producer:   app.Producer{Type: "agent_session", ID: "ses_1"},
	})
	if err != nil {
		t.Fatalf("ReadLocalPathSource returned error: %v", err)
	}
	if read.Read.Content != "alpha\n" || read.ObservationEvent == nil {
		t.Fatalf("expected read content and observation event, got %#v", read)
	}
	assertNoRootLeakInJSON(t, root, read.Read, read.ObservationEvent)
	if _, err := svc.ReadLocalPathSource(ctx, app.ReadLocalPathSourceRequest{
		MissionID:  "mis_1",
		SnapshotID: attached.Snapshot.SnapshotID,
		Subpath:    "other.txt",
		Producer:   app.Producer{Type: "agent_session", ID: "ses_1"},
	}); !errors.Is(err, app.ErrInvalidInput) {
		t.Fatalf("expected file source subpath rejection, got %v", err)
	}
	if _, err := svc.ReadLocalPathSource(ctx, app.ReadLocalPathSourceRequest{
		MissionID:  "mis_1",
		SnapshotID: attached.Snapshot.SnapshotID,
		Subpath:    ".",
		Producer:   app.Producer{Type: "agent_session", ID: "ses_1"},
	}); !errors.Is(err, app.ErrInvalidInput) {
		t.Fatalf("expected file source explicit subpath rejection, got %v", err)
	}
	grep, err := svc.GrepLocalPathSource(ctx, app.GrepLocalPathSourceRequest{
		MissionID:   "mis_1",
		SnapshotID:  attached.Snapshot.SnapshotID,
		Query:       "needle",
		MaxSnippets: 5,
		Producer:    app.Producer{Type: "agent_session", ID: "ses_1"},
	})
	if err != nil {
		t.Fatalf("GrepLocalPathSource returned error: %v", err)
	}
	if len(grep.Grep.Matches) != 1 || grep.ObservationEvent == nil {
		t.Fatalf("expected grep match and observation event, got %#v", grep)
	}
	if _, err := svc.RemoveSource(ctx, app.RemoveSourceRequest{MissionID: "mis_1", SnapshotID: attached.Snapshot.SnapshotID, Reason: "wrong source"}); err != nil {
		t.Fatalf("RemoveSource returned error: %v", err)
	}
	if _, err := svc.ReadLocalPathSource(ctx, app.ReadLocalPathSourceRequest{MissionID: "mis_1", SnapshotID: attached.Snapshot.SnapshotID}); !errors.Is(err, app.ErrInvalidInput) {
		t.Fatalf("expected removed source read rejection, got %v", err)
	}
	outline, err := svc.OutlineMission(ctx, "mis_1")
	if err != nil {
		t.Fatalf("OutlineMission after remove returned error: %v", err)
	}
	if outline.Counts[app.ResearchIDEObjectSourceSnapshot] != 0 {
		t.Fatalf("research outline should hide removed sources, got %#v", outline)
	}
	page, err := svc.ListMissionObjects(ctx, "mis_1", app.ResearchIDEObjectSourceSnapshot, 10, "")
	if err != nil {
		t.Fatalf("ListMissionObjects after remove returned error: %v", err)
	}
	if len(page.Items) != 0 {
		t.Fatalf("research list should hide removed sources, got %#v", page)
	}
	grepAfterRemove, err := svc.GrepMissionObjects(ctx, "mis_1", "alpha", 10, "")
	if err != nil {
		t.Fatalf("GrepMissionObjects after remove should skip removed source, got %v", err)
	}
	if len(grepAfterRemove.Matches) != 0 {
		t.Fatalf("research grep should hide removed source matches, got %#v", grepAfterRemove)
	}
	_, err = svc.AttachLocalPathSource(ctx, app.AttachLocalPathSourceRequest{
		MissionID:    "mis_1",
		RootID:       "docs",
		RelativePath: "guide.txt",
	})
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("expected restore-required conflict, got %v", err)
	}
	restored, err := svc.AttachLocalPathSource(ctx, app.AttachLocalPathSourceRequest{
		MissionID:    "mis_1",
		RootID:       "docs",
		RelativePath: "guide.txt",
		Restore:      true,
	})
	if err != nil {
		t.Fatalf("restore AttachLocalPathSource returned error: %v", err)
	}
	if !restored.Restored || restored.Snapshot.SnapshotID != attached.Snapshot.SnapshotID {
		t.Fatalf("expected restored existing source, got %#v", restored)
	}
	events, err := svc.ListEvents(ctx, "mis_1")
	if err != nil {
		t.Fatal(err)
	}
	for _, eventType := range []string{app.SourceLocalPathAttachedEvent, app.SourceObservedEvent, app.SourceRemovedEvent, app.SourceRestoredEvent} {
		if countEventType(events, eventType) == 0 {
			t.Fatalf("expected event %s in %#v", eventType, events)
		}
	}
	for _, event := range events {
		assertNoRootLeakInJSON(t, root, event)
	}
}

func TestLocalPathDirectoryAttachTreeAndResearchIDE(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "a.txt"), []byte("needle directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "docs", "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "nested", "b.txt"), []byte("needle nested"), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := newLocalPathSQLiteService(t, root)
	attached, err := svc.AttachLocalPathSource(ctx, app.AttachLocalPathSourceRequest{
		MissionID:    "mis_1",
		RootID:       "docs",
		RelativePath: "docs",
	})
	if err != nil {
		t.Fatalf("AttachLocalPathSource returned error: %v", err)
	}
	tree, err := svc.TreeLocalPathSource(ctx, app.TreeLocalPathSourceRequest{
		MissionID:  "mis_1",
		SnapshotID: attached.Snapshot.SnapshotID,
		Depth:      2,
		Limit:      10,
		Producer:   app.Producer{Type: "agent_session", ID: "ses_1"},
	})
	if err != nil {
		t.Fatalf("TreeLocalPathSource returned error: %v", err)
	}
	if len(tree.Tree.Entries) != 3 || tree.ObservationEvent == nil {
		t.Fatalf("expected tree entries and observation event, got %#v", tree)
	}
	read, err := svc.ReadLocalPathSource(ctx, app.ReadLocalPathSourceRequest{
		MissionID:  "mis_1",
		SnapshotID: attached.Snapshot.SnapshotID,
		Subpath:    "a.txt",
		Producer:   app.Producer{Type: "agent_session", ID: "ses_1"},
	})
	if err != nil {
		t.Fatalf("ReadLocalPathSource subpath returned error: %v", err)
	}
	if read.Read.Content != "needle directory" || read.Read.Metadata.Subpath != "a.txt" || read.Read.Metadata.RelativePath != "docs/a.txt" {
		t.Fatalf("unexpected subpath read result: %#v", read)
	}
	var readPayload map[string]any
	if err := json.Unmarshal(read.ObservationEvent.Payload, &readPayload); err != nil {
		t.Fatal(err)
	}
	if readPayload["subpath"] != "a.txt" || readPayload["relative_path"] != "docs/a.txt" {
		t.Fatalf("expected subpath observation payload, got %#v", readPayload)
	}
	subtree, err := svc.TreeLocalPathSource(ctx, app.TreeLocalPathSourceRequest{
		MissionID:  "mis_1",
		SnapshotID: attached.Snapshot.SnapshotID,
		Subpath:    "nested",
		Depth:      1,
		Limit:      10,
		Producer:   app.Producer{Type: "agent_session", ID: "ses_1"},
	})
	if err != nil {
		t.Fatalf("TreeLocalPathSource subpath returned error: %v", err)
	}
	if subtree.Tree.Metadata.Subpath != "nested" || len(subtree.Tree.Entries) != 1 || subtree.Tree.Entries[0].RelativePath != "docs/nested/b.txt" {
		t.Fatalf("unexpected subpath tree result: %#v", subtree)
	}
	sourceGrep, err := svc.GrepLocalPathSource(ctx, app.GrepLocalPathSourceRequest{
		MissionID:   "mis_1",
		SnapshotID:  attached.Snapshot.SnapshotID,
		Subpath:     "nested",
		Query:       "needle",
		MaxSnippets: 5,
		Producer:    app.Producer{Type: "agent_session", ID: "ses_1"},
	})
	if err != nil {
		t.Fatalf("GrepLocalPathSource subpath returned error: %v", err)
	}
	if sourceGrep.Grep.Metadata.Subpath != "nested" || len(sourceGrep.Grep.Matches) != 1 || sourceGrep.Grep.Matches[0].RelativePath != "docs/nested/b.txt" {
		t.Fatalf("unexpected subpath grep result: %#v", sourceGrep)
	}
	if _, err := svc.ReadLocalPathSource(ctx, app.ReadLocalPathSourceRequest{
		MissionID:  "mis_1",
		SnapshotID: attached.Snapshot.SnapshotID,
		Subpath:    "../guide.txt",
		Producer:   app.Producer{Type: "agent_session", ID: "ses_1"},
	}); !errors.Is(err, app.ErrInvalidInput) {
		t.Fatalf("expected traversal subpath rejection, got %v", err)
	}
	if _, err := svc.ReadLocalPathSource(ctx, app.ReadLocalPathSourceRequest{
		MissionID:  "mis_1",
		SnapshotID: attached.Snapshot.SnapshotID,
		Subpath:    "nested/../a.txt",
		Producer:   app.Producer{Type: "agent_session", ID: "ses_1"},
	}); !errors.Is(err, app.ErrInvalidInput) {
		t.Fatalf("expected normalized traversal subpath rejection, got %v", err)
	}
	page, err := svc.ListMissionObjects(ctx, "mis_1", app.ResearchIDEObjectSourceSnapshot, 10, "")
	if err != nil {
		t.Fatalf("ListMissionObjects returned error: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].Metadata["retrieval_policy"] != app.SourceRetrievalPolicyLiveReference {
		t.Fatalf("expected live source metadata in research list, got %#v", page)
	}
	grep, err := svc.GrepMissionObjects(ctx, "mis_1", "needle directory", 10, "")
	if err != nil {
		t.Fatalf("GrepMissionObjects returned error: %v", err)
	}
	if len(grep.Matches) != 1 {
		t.Fatalf("expected research grep match through local path engine, got %#v", grep)
	}
	refs, err := svc.ListObjectReferences(ctx, "mis_1", app.ResearchIDEObjectSourceSnapshot, attached.Snapshot.SnapshotID, 20, "")
	if err != nil {
		t.Fatalf("ListObjectReferences returned error: %v", err)
	}
	if !hasRef(refs.Backward, app.ResearchIDEObjectLedgerEvent) {
		t.Fatalf("expected source observation ledger event reference, got %#v", refs)
	}
}

func newLocalPathSQLiteService(t *testing.T, root string) *app.Service {
	t.Helper()
	store := newTestStore(t)
	engine, err := localpath.New(localpath.Config{Roots: []localpath.RootConfig{{RootID: "docs", Path: root, Alias: "docs"}}})
	if err != nil {
		t.Fatalf("localpath.New returned error: %v", err)
	}
	svc := app.NewServiceWithLocalPathEngine(store, engine)
	if _, err := svc.CreateMission(context.Background(), app.CreateMissionRequest{MissionID: "mis_1", Title: "Mission"}); err != nil {
		t.Fatalf("CreateMission returned error: %v", err)
	}
	return svc
}

func countEventType(events []app.LedgerEvent, eventType string) int {
	count := 0
	for _, event := range events {
		if event.EventType == eventType {
			count++
		}
	}
	return count
}

func assertNoRootLeakInJSON(t *testing.T, root string, values ...any) {
	t.Helper()
	encoded, err := json.Marshal(values)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	if strings.Contains(text, root) || strings.Contains(text, filepath.ToSlash(root)) {
		t.Fatalf("absolute root leaked in JSON: %s", text)
	}
}

func hasRef(refs []app.ResearchIDEObjectRef, kind string) bool {
	for _, ref := range refs {
		if ref.ObjectKind == kind {
			return true
		}
	}
	return false
}
