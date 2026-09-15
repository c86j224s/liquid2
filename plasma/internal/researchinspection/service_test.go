package researchinspection

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/researchcatalog"
)

func TestReadSequencesCallbacksAndForwardsReportPage(t *testing.T) {
	var calls []string
	var gotLegacy bool
	var gotLimit int
	var gotCursor string
	svc := NewService(Dependencies{
		ReadChunked: func(context.Context, string, string, string, int, int) (ObjectRead, bool, error) {
			calls = append(calls, "chunked")
			return ObjectRead{}, false, nil
		},
		ReadPayload: func(_ context.Context, missionID, kind, id string, legacy bool) (researchcatalog.ObjectSummary, []byte, error) {
			calls = append(calls, "payload:"+missionID+":"+kind+":"+id)
			gotLegacy = legacy
			return researchcatalog.ObjectSummary{Summary: "summary"}, []byte("hello"), nil
		},
		ReportVersionChildren: func(_ context.Context, missionID, id string, limit int, cursor string) (researchcatalog.Page, error) {
			calls = append(calls, "children:"+missionID+":"+id)
			gotLimit, gotCursor = limit, cursor
			return researchcatalog.Page{MissionID: missionID, ObjectKind: researchcatalog.ObjectReportBlock}, nil
		},
	})
	read, err := svc.Read(context.Background(), ReadRequest{MissionID: " mis_1 ", ObjectKind: researchcatalog.ObjectReportVersion, ObjectID: " ver_1 ", MaxBytes: 3, Limit: 7, Cursor: "next", Legacy: true})
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}
	if want := []string{"chunked", "payload:mis_1:report_version:ver_1", "children:mis_1:ver_1"}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %#v, want %#v", calls, want)
	}
	if !gotLegacy || gotLimit != 7 || gotCursor != "next" || read.Data != "hel" || read.Children == nil {
		t.Fatalf("read = %#v, legacy=%v limit=%d cursor=%q", read, gotLegacy, gotLimit, gotCursor)
	}
}

func TestReadInvalidInputDoesNotCallCallbacks(t *testing.T) {
	called := false
	svc := NewService(Dependencies{
		ReadChunked: func(context.Context, string, string, string, int, int) (ObjectRead, bool, error) {
			called = true
			return ObjectRead{}, false, nil
		},
		ReadPayload: func(context.Context, string, string, string, bool) (researchcatalog.ObjectSummary, []byte, error) {
			called = true
			return researchcatalog.ObjectSummary{}, nil, nil
		},
		ReportVersionChildren: func(context.Context, string, string, int, string) (researchcatalog.Page, error) {
			called = true
			return researchcatalog.Page{}, nil
		},
	})
	for _, req := range []ReadRequest{{MissionID: "bad", ObjectID: "obj"}, {MissionID: "mis_1", ObjectID: ""}, {MissionID: "mis_1", ObjectID: "obj", Offset: -1}} {
		if _, err := svc.Read(context.Background(), req); !errors.Is(err, producterror.ErrInvalidInput) {
			t.Errorf("Read(%#v) error = %v, want invalid input", req, err)
		}
	}
	if called {
		t.Fatal("callbacks were called for invalid input")
	}
}

func TestReadChunkedShortCircuitAndError(t *testing.T) {
	wantErr := errors.New("chunked failed")
	calls := 0
	svc := NewService(Dependencies{
		ReadChunked: func(context.Context, string, string, string, int, int) (ObjectRead, bool, error) {
			calls++
			return ObjectRead{Data: "ready"}, true, nil
		},
		ReadPayload: func(context.Context, string, string, string, bool) (researchcatalog.ObjectSummary, []byte, error) {
			t.Fatal("payload called after handled chunk")
			return researchcatalog.ObjectSummary{}, nil, nil
		},
	})
	read, err := svc.Read(context.Background(), ReadRequest{MissionID: "mis_1", ObjectID: "obj"})
	if err != nil || read.Data != "ready" || calls != 1 {
		t.Fatalf("short circuit = %#v, %v, calls=%d", read, err, calls)
	}
	svc = NewService(Dependencies{ReadChunked: func(context.Context, string, string, string, int, int) (ObjectRead, bool, error) {
		return ObjectRead{}, true, wantErr
	}})
	if _, err := svc.Read(context.Background(), ReadRequest{MissionID: "mis_1", ObjectID: "obj"}); !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}

func TestChunkBytesUTF8AndBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name        string
		data        []byte
		offset, max int
		want        string
		truncated   bool
		next        int
		wantErr     bool
	}{
		{name: "nil", data: nil, want: ""},
		{name: "empty", data: []byte{}, want: ""},
		{name: "multibyte", data: []byte("aéb"), max: 2, want: "a", truncated: true, next: 1},
		{name: "complete", data: []byte("aéb"), max: 3, want: "aé", truncated: true, next: 3},
		{name: "at end", data: []byte("é"), offset: 2, max: 1, want: ""},
		{name: "incomplete offset", data: []byte("é"), offset: 1, max: 1, wantErr: true},
		{name: "beyond", data: []byte("x"), offset: 2, max: 1, wantErr: true},
		{name: "invalid", data: []byte{0xff}, max: 1, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, truncated, next, err := ChunkBytes(tc.data, tc.offset, tc.max)
			if tc.wantErr {
				if !errors.Is(err, producterror.ErrInvalidInput) {
					t.Fatalf("error = %v", err)
				}
				return
			}
			if err != nil || string(got) != tc.want || truncated != tc.truncated || next != tc.next {
				t.Fatalf("got %q, truncated=%v next=%d err=%v", got, truncated, next, err)
			}
		})
	}
}

func TestGrepForwardsInputsAndPreservesMatchSemantics(t *testing.T) {
	var gotMission, gotQuery string
	var gotLegacy bool
	calls := 0
	svc := NewService(Dependencies{GrepCandidates: func(_ context.Context, missionID, query string, legacy bool) ([]GrepCandidate, error) {
		calls++
		gotMission, gotQuery, gotLegacy = missionID, query, legacy
		return []GrepCandidate{{Summary: researchcatalog.ObjectSummary{ObjectKind: researchcatalog.ObjectRawArtifact, ObjectID: "art_1"}, Text: "AaAa"}}, nil
	}})
	result, err := svc.Grep(context.Background(), " mis_1 ", " aa ", 2, "", true)
	if err != nil {
		t.Fatalf("Grep returned error: %v", err)
	}
	if calls != 1 || gotMission != "mis_1" || gotQuery != "aa" || !gotLegacy || len(result.Matches) != 2 || result.Matches[0].Position != 0 || result.Matches[1].Position != 2 {
		t.Fatalf("result=%#v calls=%d mission=%q query=%q legacy=%v", result, calls, gotMission, gotQuery, gotLegacy)
	}
}

func TestGrepInvalidInputDoesNotCallCandidates(t *testing.T) {
	calls := 0
	svc := NewService(Dependencies{GrepCandidates: func(context.Context, string, string, bool) ([]GrepCandidate, error) {
		calls++
		return nil, nil
	}})
	for _, tc := range []struct{ mission, query, cursor string }{{"bad", "x", ""}, {"mis_1", "", ""}, {"mis_1", "x", "bad"}} {
		if _, err := svc.Grep(context.Background(), tc.mission, tc.query, 1, tc.cursor, false); !errors.Is(err, producterror.ErrInvalidInput) {
			t.Errorf("Grep(%#v) error=%v, want invalid input", tc, err)
		}
	}
	if calls != 0 {
		t.Fatalf("candidate callback calls=%d, want 0", calls)
	}
}

func TestGrepEmptyCandidatesReturnNilMatches(t *testing.T) {
	svc := NewService(Dependencies{GrepCandidates: func(context.Context, string, string, bool) ([]GrepCandidate, error) {
		return nil, nil
	}})
	result, err := svc.Grep(context.Background(), "mis_1", "x", 1, "", false)
	if err != nil || result.Matches != nil || result.NextCursor != "" || result.Truncated {
		t.Fatalf("empty result=%#v err=%v", result, err)
	}
}

func TestGrepPropagatesCandidateErrorAndExhaustedCursor(t *testing.T) {
	wantErr := errors.New("candidate failed")
	svc := NewService(Dependencies{GrepCandidates: func(context.Context, string, string, bool) ([]GrepCandidate, error) { return nil, wantErr }})
	if _, err := svc.Grep(context.Background(), "mis_1", "x", 1, "", false); !errors.Is(err, wantErr) {
		t.Fatalf("error=%v, want %v", err, wantErr)
	}
	svc = NewService(Dependencies{GrepCandidates: func(context.Context, string, string, bool) ([]GrepCandidate, error) {
		return []GrepCandidate{{Summary: researchcatalog.ObjectSummary{ObjectID: "art_1"}, Text: "x"}}, nil
	}})
	result, err := svc.Grep(context.Background(), "mis_1", "x", 1, "1", false)
	if err != nil || result.Matches != nil || result.NextCursor != "" || result.Truncated {
		t.Fatalf("exhausted result=%#v err=%v", result, err)
	}
}

func TestGrepPositionsAndSnippetUseExistingByteBehavior(t *testing.T) {
	svc := NewService(Dependencies{GrepCandidates: func(context.Context, string, string, bool) ([]GrepCandidate, error) {
		return []GrepCandidate{{Summary: researchcatalog.ObjectSummary{ObjectID: "art_1"}, Text: "éAAé"}}, nil
	}})
	result, err := svc.Grep(context.Background(), "mis_1", "aa", 0, "", false)
	if err != nil || len(result.Matches) != 1 || result.Matches[0].Position != 2 || result.Matches[0].Snippet != "éAAé" || result.Limit != 20 {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestClampBytesLimits(t *testing.T) {
	for input, want := range map[int]int{0: DefaultBytes, -1: DefaultBytes, 1: 1, MaxBytes: MaxBytes, MaxBytes + 1: MaxBytes} {
		if got := ClampBytes(input); got != want {
			t.Errorf("ClampBytes(%d) = %d, want %d", input, got, want)
		}
	}
}
