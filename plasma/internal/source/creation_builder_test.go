package source

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

type recordingArtifactReader struct {
	artifacts map[string]artifact.Raw
	err       error
	calls     []string
}

func (r *recordingArtifactReader) GetRawArtifact(_ context.Context, id string) (artifact.Raw, error) {
	r.calls = append(r.calls, id)
	if r.err != nil {
		return artifact.Raw{}, r.err
	}
	value, ok := r.artifacts[id]
	if !ok {
		return artifact.Raw{}, errors.New("missing artifact")
	}
	return value, nil
}

func rawArtifact(id, mission, sha string) artifact.Raw {
	return artifact.Raw{ArtifactID: id, MissionID: mission, SHA256: sha}
}

func snapshotRequest(ids []string) CreateRequest {
	return CreateRequest{SnapshotID: " src_1 ", MissionID: " mis_1 ", Connector: ConnectorRef{ConnectorID: " c1 ", ConnectorType: " type ", ExternalSourceID: " doc "}, ArtifactIDs: ids}
}

func TestBuildSnapshotValidationOrderAndReaderRouting(t *testing.T) {
	readerErr := errors.New("reader failed")
	reader := &recordingArtifactReader{err: readerErr}
	cases := []struct {
		name string
		req  CreateRequest
		want string
	}{
		{"snapshot before mission", CreateRequest{SnapshotID: "bad", MissionID: "bad"}, "id must start with src_"},
		{"connector before external", CreateRequest{SnapshotID: "src_1", MissionID: "mis_1"}, "connector id and type are required"},
		{"locator before policy", func() CreateRequest {
			r := snapshotRequest([]string{"art_1"})
			r.Locators = json.RawMessage("{")
			r.Access.RetrievalPolicy = "bad"
			return r
		}(), "locators must be valid JSON"},
		{"policy before lookup", func() CreateRequest {
			r := snapshotRequest([]string{"art_1"})
			r.Access.RetrievalPolicy = "bad"
			return r
		}(), "unsupported source retrieval policy"},
		{"artifact id before duplicate", func() CreateRequest { r := snapshotRequest([]string{"bad", "bad"}); return r }(), "id must start with art_"},
		{"duplicate before lookup", snapshotRequest([]string{"art_1", " art_1 "}), "duplicate snapshot artifact id"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			caseReader := &recordingArtifactReader{artifacts: map[string]artifact.Raw{"art_1": rawArtifact("art_1", "mis_1", strings.Repeat("a", 64))}}
			_, err := BuildSnapshot(context.Background(), caseReader, tc.req, nil)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %v, want %q", err, tc.want)
			}
		})
	}
	reader = &recordingArtifactReader{err: readerErr}
	_, err := BuildSnapshot(context.Background(), reader, snapshotRequest([]string{"art_1"}), nil)
	if !errors.Is(err, readerErr) || !reflect.DeepEqual(reader.calls, []string{"art_1"}) {
		t.Fatalf("reader calls=%v err=%v", reader.calls, err)
	}
}

func TestBuildSnapshotPendingPrecedesReaderAndValidatesPending(t *testing.T) {
	sha := strings.Repeat("a", 64)
	reader := &recordingArtifactReader{artifacts: map[string]artifact.Raw{"art_1": rawArtifact("art_1", "mis_1", sha)}}
	pending := []artifact.Raw{rawArtifact("art_1", "mis_1", sha)}
	got, err := BuildSnapshot(context.Background(), reader, snapshotRequest([]string{" art_1 "}), pending)
	if err != nil || len(reader.calls) != 0 || !reflect.DeepEqual(got.ArtifactIDs, []string{"art_1"}) {
		t.Fatalf("got=%#v calls=%v err=%v", got, reader.calls, err)
	}
	pending[0].MissionID = "mis_other"
	if _, err := BuildSnapshot(context.Background(), reader, snapshotRequest([]string{"art_1"}), pending); !errors.Is(err, producterror.ErrInvalidInput) || !strings.Contains(err.Error(), "another mission") {
		t.Fatalf("pending validation error=%v", err)
	}
}

func TestBuildSnapshotHashesDefaultsAndCopiesLocators(t *testing.T) {
	sha1, sha2 := strings.Repeat("a", 64), strings.Repeat("b", 64)
	reader := &recordingArtifactReader{artifacts: map[string]artifact.Raw{"art_1": rawArtifact("art_1", "mis_1", sha1), "art_2": rawArtifact("art_2", "mis_1", sha2)}}
	locators := json.RawMessage(`[{"kind":"x"}]`)
	req := snapshotRequest([]string{"art_2", "art_1"})
	req.Locators = locators
	got, err := BuildSnapshot(context.Background(), reader, req, nil)
	if err != nil || got.ContentHash.Algorithm != "sha256" || got.Locators[0] != '[' || got.Access.Visibility != "private" || got.Access.License != "unknown" {
		t.Fatalf("snapshot=%#v err=%v", got, err)
	}
	locators[0] = '{'
	if got.Locators[0] != '[' {
		t.Fatal("locators alias caller storage")
	}
	want := snapshotHashValue([]artifact.Raw{reader.artifacts["art_2"], reader.artifacts["art_1"]})
	if got.ContentHash.Value != want {
		t.Fatalf("hash=%q want %q", got.ContentHash.Value, want)
	}
}

func TestParseLocalPathLocatorShapesAndNormalization(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    LocalPathLocator
		wantErr string
	}{
		{
			name: "object uses locator type",
			raw:  `{"locator_type":"local_path","root_id":" root_1 ","relative_path":" dir\\file ","path_kind":" file "}`,
			want: LocalPathLocator{LocatorType: LocatorTypeLocalPath, RootID: "root_1", RelativePath: "dir/file", PathKind: "file"},
		},
		{
			name: "one item array uses legacy kind",
			raw:  `[{"kind":"local_path","root_id":"root_1","relative_path":"a/./b","path_kind":"directory"}]`,
			want: LocalPathLocator{LocatorType: LocatorTypeLocalPath, RootID: "root_1", RelativePath: "a/b", PathKind: "directory"},
		},
		{name: "empty input", raw: "", wantErr: "locator is required"},
		{name: "invalid json", raw: "{", wantErr: "object or one-item array"},
		{name: "empty array", raw: "[]", wantErr: "exactly one locator"},
		{name: "multiple locators", raw: "[{},{}]", wantErr: "exactly one locator"},
		{name: "missing discriminator", raw: `{ "root_id":"root_1", "relative_path":"file", "path_kind":"file" }`, wantErr: "object or one-item array"},
		{name: "wrong discriminator", raw: `{ "locator_type":"media", "root_id":"root_1", "relative_path":"file", "path_kind":"file" }`, wantErr: "kind is required"},
		{name: "empty relative path normalizes to root", raw: `{ "locator_type":"local_path", "root_id":"root_1", "relative_path":"", "path_kind":"file" }`, want: LocalPathLocator{LocatorType: LocatorTypeLocalPath, RootID: "root_1", RelativePath: ".", PathKind: "file"}},
		{name: "traversal", raw: `{ "locator_type":"local_path", "root_id":"root_1", "relative_path":"../file", "path_kind":"file" }`, wantErr: "must not contain traversal"},
		{name: "absolute path", raw: `{ "locator_type":"local_path", "root_id":"root_1", "relative_path":"/file", "path_kind":"file" }`, wantErr: "must not be absolute"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseLocalPathLocator(json.RawMessage(tc.raw))
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error=%v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("locator=%#v err=%v, want %#v", got, err, tc.want)
			}
		})
	}
}

func TestParseMediaLocatorShapesAndMediaKinds(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    MediaLocator
		wantErr string
	}{
		{
			name: "object uses locator type",
			raw:  `{"locator_type":"media","media_kind":" video ","canonical_url":" https://example.com/watch ","title":" Title "}`,
			want: MediaLocator{LocatorType: LocatorTypeMedia, MediaKind: MediaKindVideo, CanonicalURL: "https://example.com/watch", Title: "Title"},
		},
		{
			name: "one item array uses legacy kind",
			raw:  `[{"kind":"media","media_kind":"audio","direct_media_url":"https://example.com/audio"}]`,
			want: MediaLocator{LocatorType: LocatorTypeMedia, MediaKind: MediaKindAudio, DirectMediaURL: "https://example.com/audio"},
		},
		{name: "image kind is accepted", raw: `{ "locator_type":"media", "media_kind":"image" }`, want: MediaLocator{LocatorType: LocatorTypeMedia, MediaKind: MediaKindImage}},
		{name: "empty input", raw: "", wantErr: "locator is required"},
		{name: "invalid json", raw: "{", wantErr: "object or one-item array"},
		{name: "empty array", raw: "[]", wantErr: "exactly one locator"},
		{name: "multiple locators", raw: "[{},{}]", wantErr: "exactly one locator"},
		{name: "missing discriminator", raw: `{ "media_kind":"video" }`, wantErr: "object or one-item array"},
		{name: "wrong discriminator", raw: `{ "locator_type":"local_path", "media_kind":"video" }`, wantErr: "kind is required"},
		{name: "unsupported media kind", raw: `{ "locator_type":"media", "media_kind":"document" }`, wantErr: "unsupported media kind"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseMediaLocator(json.RawMessage(tc.raw))
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error=%v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("locator=%#v err=%v, want %#v", got, err, tc.want)
			}
		})
	}
}

func TestBuildSnapshotLiveReferenceLocators(t *testing.T) {
	reader := &recordingArtifactReader{}
	local := snapshotRequest(nil)
	local.Access.RetrievalPolicy = RetrievalPolicyLiveReference
	local.Connector = ConnectorRef{ConnectorID: "local", ConnectorType: ConnectorTypeLocalPath, ExternalSourceID: "root:file.txt"}
	local.Locators = json.RawMessage(`{"locator_type":"local_path","root_id":"root","relative_path":"file.txt","path_kind":"file"}`)
	got, err := BuildSnapshot(context.Background(), reader, local, nil)
	if err != nil || got.ContentHash != (ContentHash{Algorithm: "none", Value: ""}) {
		t.Fatalf("local=%#v err=%v", got, err)
	}
	media := snapshotRequest(nil)
	media.Access.RetrievalPolicy = RetrievalPolicyLiveReference
	media.Connector = ConnectorRef{ConnectorID: "media", ConnectorType: ConnectorTypeMediaURL, ExternalURI: "https://example.com/watch"}
	media.Locators = json.RawMessage(`{"locator_type":"media","media_kind":"video","canonical_url":"https://example.com/watch"}`)
	if _, err := BuildSnapshot(context.Background(), reader, media, nil); err != nil {
		t.Fatalf("media live reference: %v", err)
	}
	media.Locators = json.RawMessage(`{"locator_type":"media","media_kind":"image","canonical_url":"https://example.com/image"}`)
	if _, err := BuildSnapshot(context.Background(), reader, media, nil); !errors.Is(err, producterror.ErrInvalidInput) {
		t.Fatalf("image live reference error=%v", err)
	}
}
