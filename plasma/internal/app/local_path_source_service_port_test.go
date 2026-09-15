package app

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
	"github.com/c86j224s/liquid2/plasma/internal/sources/localpath"
)

type localPathPortFake struct {
	roots      []sourcecontract.LocalPathRoot
	tree       sourcecontract.LocalPathTreeResult
	read       sourcecontract.LocalPathReadResult
	pdf        sourcecontract.LocalPathReadResult
	grep       sourcecontract.LocalPathGrepResult
	isPDF      bool
	readReq    sourcecontract.LocalPathReadRequest
	pdfReq     sourcecontract.LocalPathReadRequest
	treeReq    sourcecontract.LocalPathTreeRequest
	grepReq    sourcecontract.LocalPathGrepRequest
	readCalls  int
	pdfCalls   int
	isPDFCalls int
	treeCalls  int
	grepCalls  int
}

func (f *localPathPortFake) Roots() []sourcecontract.LocalPathRoot {
	return f.roots
}
func (f *localPathPortFake) Inspect(context.Context, string, string) (sourcecontract.LocalPathMetadata, error) {
	return sourcecontract.LocalPathMetadata{}, nil
}
func (f *localPathPortFake) IsPDF(_ context.Context, rootID, relativePath string) (bool, error) {
	f.isPDFCalls++
	return f.isPDF, nil
}
func (f *localPathPortFake) ReadFile(_ context.Context, req sourcecontract.LocalPathReadRequest) (sourcecontract.LocalPathReadResult, error) {
	f.readCalls++
	f.readReq = req
	return f.read, nil
}
func (f *localPathPortFake) ReadPDFText(_ context.Context, req sourcecontract.LocalPathReadRequest) (sourcecontract.LocalPathReadResult, error) {
	f.pdfCalls++
	f.pdfReq = req
	return f.pdf, nil
}
func (f *localPathPortFake) Tree(_ context.Context, req sourcecontract.LocalPathTreeRequest) (sourcecontract.LocalPathTreeResult, error) {
	f.treeCalls++
	f.treeReq = req
	return f.tree, nil
}
func (f *localPathPortFake) Grep(_ context.Context, req sourcecontract.LocalPathGrepRequest) (sourcecontract.LocalPathGrepResult, error) {
	f.grepCalls++
	f.grepReq = req
	return f.grep, nil
}

func TestServiceLocalPathPortIsReplaceable(t *testing.T) {
	fake := &localPathPortFake{
		roots: []sourcecontract.LocalPathRoot{{RootID: "fake", Alias: "Fake"}},
		tree:  sourcecontract.LocalPathTreeResult{RootID: "fake", RelativePath: "docs", Entries: []sourcecontract.LocalPathTreeEntry{{Name: "note.txt"}}},
	}
	svc := NewServiceWithLocalPathEngine(fakeStore{}, fake)
	roots, err := svc.ListLocalPathRoots(context.Background())
	if err != nil || !reflect.DeepEqual(roots, fake.roots) {
		t.Fatalf("ListLocalPathRoots = %#v, %v", roots, err)
	}
	tree, err := svc.BrowseLocalPathRoot(context.Background(), BrowseLocalPathRootRequest{RootID: "fake", RelativePath: "docs", Depth: 2, Limit: 5})
	if err != nil || !reflect.DeepEqual(tree, fake.tree) {
		t.Fatalf("BrowseLocalPathRoot = %#v, %v", tree, err)
	}
	if fake.treeCalls != 1 || fake.treeReq.RootID != "fake" || fake.treeReq.RelativePath != "docs" || fake.treeReq.Depth != 2 || fake.treeReq.Limit != 5 {
		t.Fatalf("unexpected tree delegation: calls=%d req=%#v", fake.treeCalls, fake.treeReq)
	}
}

func TestServiceLocalPathPortTypedNilIsUnconfigured(t *testing.T) {
	var typedNilFake *localPathPortFake
	var typedNilEngine *localpath.Engine
	for _, makeService := range []func() *Service{
		func() *Service { return NewServiceWithLocalPathEngine(fakeStore{}, typedNilFake) },
		func() *Service {
			svc := NewService(fakeStore{})
			svc.SetLocalPathEngine(typedNilFake)
			return svc
		},
		func() *Service { return NewServiceWithLocalPathEngine(fakeStore{}, typedNilEngine) },
		func() *Service {
			svc := NewService(fakeStore{})
			svc.SetLocalPathEngine(typedNilEngine)
			return svc
		},
		func() *Service { return NewServiceWithLocalPathEngine(fakeStore{}, nil) },
		func() *Service {
			svc := NewServiceWithLocalPathEngine(fakeStore{}, &localPathPortFake{})
			svc.SetLocalPathEngine(nil)
			return svc
		},
	} {
		_, err := makeService().ListLocalPathRoots(context.Background())
		if !errors.Is(err, ErrInvalidInput) || err.Error() != "invalid input: local path roots are not configured" {
			t.Fatalf("typed nil error = %v", err)
		}
	}
}

func localPathTestSnapshot(pathKind string) sourcecontract.Snapshot {
	locators, _ := json.Marshal([]sourcecontract.LocalPathLocator{{LocatorType: sourcecontract.LocatorTypeLocalPath, RootID: "fake", RelativePath: "docs", PathKind: pathKind}})
	return sourcecontract.Snapshot{
		SnapshotID: "src_1", MissionID: "mis_1",
		Connector: sourcecontract.ConnectorRef{ConnectorType: sourcecontract.ConnectorTypeLocalPath},
		Locators:  locators,
		Access:    sourcecontract.Access{RetrievalPolicy: sourcecontract.RetrievalPolicyLiveReference},
		State:     sourcecontract.State{State: sourcecontract.StateActive},
	}
}

type localPathSourceStore struct {
	fakeStore
	snapshot sourcecontract.Snapshot
	events   []ledger.Event
}

func (s *localPathSourceStore) GetSourceSnapshot(context.Context, string) (sourcecontract.Snapshot, error) {
	return s.snapshot, nil
}
func (s *localPathSourceStore) ListSourceSnapshots(context.Context, string) ([]sourcecontract.Snapshot, error) {
	return []sourcecontract.Snapshot{s.snapshot}, nil
}
func (s *localPathSourceStore) ListLedgerEvents(context.Context, string) ([]ledger.Event, error) {
	return s.events, nil
}
func (s *localPathSourceStore) AppendLedgerEvent(_ context.Context, event ledger.Event) (ledger.Event, error) {
	event.Sequence = int64(len(s.events) + 1)
	s.events = append(s.events, event)
	return event, nil
}

func TestServiceReadLocalPathPortPreservesPDFObservationSemantics(t *testing.T) {
	fake := &localPathPortFake{
		isPDF: true,
		read:  sourcecontract.LocalPathReadResult{Metadata: sourcecontract.LocalPathMetadata{RootID: "fake", RelativePath: "docs", PathKind: "file", Offset: 0, MaxBytes: 8, NextOffset: 8, Truncated: true}},
		pdf:   sourcecontract.LocalPathReadResult{Content: "extracted", Metadata: sourcecontract.LocalPathMetadata{RootID: "fake", RelativePath: "docs", PathKind: "file", Extraction: "pdf_text", PageCount: 1, NextOffset: 8, Truncated: true}},
	}
	store := &localPathSourceStore{snapshot: localPathTestSnapshot("file")}
	svc := NewServiceWithLocalPathEngine(store, fake)
	result, err := svc.ReadLocalPathSource(context.Background(), ReadLocalPathSourceRequest{MissionID: "mis_1", SnapshotID: "src_1", Offset: 0, MaxBytes: 8, ToolSessionID: "tool_1"})
	if err != nil || result.Read.Content != "extracted" || result.Read.Metadata.Extraction != "pdf_text" {
		t.Fatalf("ReadLocalPathSource = %#v, %v", result, err)
	}
	if fake.readCalls != 1 || fake.isPDFCalls != 1 || fake.pdfCalls != 1 || fake.pdfReq != fake.readReq {
		t.Fatalf("unexpected PDF delegation: read=%d detect=%d pdf=%d req=%#v pdfReq=%#v", fake.readCalls, fake.isPDFCalls, fake.pdfCalls, fake.readReq, fake.pdfReq)
	}
	if len(store.events) != 1 || store.events[0].EventType != SourceObservedEvent || !reflect.DeepEqual(result.ObservationEvent, &store.events[0]) {
		t.Fatalf("unexpected observation events: %#v", store.events)
	}
	if !stringsContains(string(store.events[0].Payload), "tool_1") || !stringsContains(string(store.events[0].Payload), "pdf_text") {
		t.Fatalf("observation payload = %s", store.events[0].Payload)
	}
}

func TestServiceTreeAndGrepLocalPathPortPreserveObservationSemantics(t *testing.T) {
	fake := &localPathPortFake{
		tree: sourcecontract.LocalPathTreeResult{RootID: "fake", RelativePath: "docs/nested", Entries: []sourcecontract.LocalPathTreeEntry{{Name: "note.txt"}}, Truncated: true, Metadata: sourcecontract.LocalPathMetadata{RootID: "fake", RelativePath: "docs/nested", Subpath: "nested", PathKind: "directory"}},
		grep: sourcecontract.LocalPathGrepResult{RootID: "fake", RelativePath: "docs/nested", Query: "needle", Matches: []sourcecontract.LocalPathGrepMatch{{RelativePath: "docs/nested/note.txt", Line: 1, Column: 1, Snippet: "needle"}}, Metadata: sourcecontract.LocalPathMetadata{RootID: "fake", RelativePath: "docs/nested", Subpath: "nested", PathKind: "directory"}},
	}
	store := &localPathSourceStore{snapshot: localPathTestSnapshot("directory")}
	svc := NewServiceWithLocalPathEngine(store, fake)
	tree, err := svc.TreeLocalPathSource(context.Background(), TreeLocalPathSourceRequest{MissionID: "mis_1", SnapshotID: "src_1", Subpath: "nested", Depth: 2, Limit: 4, ToolSessionID: "tool_tree"})
	if err != nil || !reflect.DeepEqual(tree.Tree, fake.tree) || fake.treeReq.Subpath != "nested" || fake.treeReq.Depth != 2 || fake.treeReq.Limit != 4 {
		t.Fatalf("tree = %#v, %v req=%#v", tree, err, fake.treeReq)
	}
	grep, err := svc.GrepLocalPathSource(context.Background(), GrepLocalPathSourceRequest{MissionID: "mis_1", SnapshotID: "src_1", Subpath: "nested", Query: "needle", MaxSnippets: 3, ToolSessionID: "tool_grep"})
	if err != nil || !reflect.DeepEqual(grep.Grep, fake.grep) || fake.grepReq.Subpath != "nested" || fake.grepReq.Query != "needle" || fake.grepReq.MaxSnippets != 3 {
		t.Fatalf("grep = %#v, %v req=%#v", grep, err, fake.grepReq)
	}
	if len(store.events) != 2 || store.events[0].EventType != SourceObservedEvent || store.events[1].EventType != SourceObservedEvent {
		t.Fatalf("observation events = %#v", store.events)
	}
}

func stringsContains(value, needle string) bool {
	return strings.Contains(value, needle)
}
