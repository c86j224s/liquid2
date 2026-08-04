package artifact

import (
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

// Raw is metadata for an original or derived stored byte blob. Content is only
// populated at immediate create/read boundaries; ArtifactID is its identity.
type Raw struct {
	ArtifactID string
	MissionID  string
	MediaType  string
	ByteSize   int64
	SHA256     string
	StorageURI string
	Filename   string
	Producer   ledger.Producer
	CreatedAt  time.Time
	Content    []byte
}

// CreateRequest contains a raw artifact and optional expected content hash.
type CreateRequest struct {
	ArtifactID     string
	MissionID      string
	MediaType      string
	Filename       string
	Producer       ledger.Producer
	Content        []byte
	ExpectedSHA256 string
}
