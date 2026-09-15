package artifact

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func TestBuildRejectsInvalidInputInOrder(t *testing.T) {
	tests := []struct {
		name string
		req  CreateRequest
		want string
	}{
		{
			name: "artifact id",
			req:  validCreateRequest("bad", "mis_1"),
			want: "invalid input: id must start with art_",
		},
		{
			name: "mission id after artifact id",
			req:  validCreateRequest("art_1", "bad"),
			want: "invalid input: id must start with mis_",
		},
		{
			name: "media type after ids",
			req: func() CreateRequest {
				req := validCreateRequest("art_1", "mis_1")
				req.MediaType = "   "
				return req
			}(),
			want: "invalid input: media type is required",
		},
		{
			name: "producer after media type",
			req: func() CreateRequest {
				req := validCreateRequest("art_1", "mis_1")
				req.Producer = ledger.Producer{Type: "", ID: ""}
				return req
			}(),
			want: "invalid input: producer type and id are required",
		},
		{
			name: "content after producer",
			req: func() CreateRequest {
				req := validCreateRequest("art_1", "mis_1")
				req.Content = nil
				return req
			}(),
			want: "invalid input: artifact content is required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Build(test.req)
			if err == nil || err.Error() != test.want {
				t.Fatalf("Build() error = %v, want %q", err, test.want)
			}
			if !errors.Is(err, producterror.ErrInvalidInput) {
				t.Fatalf("Build() error does not preserve producterror.ErrInvalidInput: %v", err)
			}
		})
	}
}

func TestBuildAcceptsHashCaseAndWhitespace(t *testing.T) {
	req := validCreateRequest(" art_1 ", " mis_1 ")
	req.MediaType = " text/plain "
	req.Filename = " source.txt "
	req.Producer = ledger.Producer{Type: " connector ", ID: " producer "}
	sum := sha256.Sum256(req.Content)
	expected := hex.EncodeToString(sum[:])
	req.ExpectedSHA256 = "  " + strings.ToUpper(expected) + "  "

	got, err := Build(req)
	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}
	if got.ArtifactID != "art_1" || got.MissionID != "mis_1" || got.MediaType != "text/plain" || got.Filename != "source.txt" {
		t.Fatalf("Build() did not trim metadata: %#v", got)
	}
	if got.Producer.Type != "connector" || got.Producer.ID != "producer" {
		t.Fatalf("Build() did not trim producer: %#v", got.Producer)
	}
	if got.SHA256 != expected || got.ByteSize != int64(len(req.Content)) {
		t.Fatalf("Build() metadata = %#v", got)
	}
	wantURI := "plasma-artifact://mis_1/" + expected[:2] + "/" + expected
	if got.StorageURI != wantURI {
		t.Fatalf("StorageURI = %q, want %q", got.StorageURI, wantURI)
	}
}

func TestBuildRejectsMismatchedAndWhitespaceOnlyHash(t *testing.T) {
	for _, expected := range []string{"bad", "   "} {
		t.Run(expected, func(t *testing.T) {
			req := validCreateRequest("art_1", "mis_1")
			req.ExpectedSHA256 = expected
			_, err := Build(req)
			if err == nil || err.Error() != "invalid input: artifact sha256 mismatch" {
				t.Fatalf("Build() error = %v, want hash mismatch", err)
			}
			if !errors.Is(err, producterror.ErrInvalidInput) {
				t.Fatalf("Build() error does not preserve producterror.ErrInvalidInput: %v", err)
			}
		})
	}
}

func TestBuildCopiesContentAndCapturesUTCWithinBounds(t *testing.T) {
	content := []byte("hello")
	req := validCreateRequest("art_1", "mis_1")
	req.Content = content
	before := time.Now().UTC()
	got, err := Build(req)
	after := time.Now().UTC()
	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}
	if got.CreatedAt.Before(before) || got.CreatedAt.After(after) || got.CreatedAt.Location() != time.UTC {
		t.Fatalf("CreatedAt = %v, want UTC time between %v and %v", got.CreatedAt, before, after)
	}
	if &got.Content[0] == &content[0] {
		t.Fatal("Build() aliased request content")
	}
	content[0] = 'X'
	if string(got.Content) != "hello" {
		t.Fatalf("Build() content changed after request mutation: %q", got.Content)
	}
}

func validCreateRequest(artifactID, missionID string) CreateRequest {
	return CreateRequest{
		ArtifactID: artifactID,
		MissionID:  missionID,
		MediaType:  "text/plain",
		Producer:   ledger.Producer{Type: "connector", ID: "liquid2"},
		Content:    []byte("hello"),
	}
}
