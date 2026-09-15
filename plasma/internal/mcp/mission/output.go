package mission

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/source"
)

// SnapshotOutput preserves the existing source snapshot projection fields.
type SnapshotOutput struct {
	SnapshotID        string             `json:"snapshot_id"`
	MissionID         string             `json:"mission_id"`
	Connector         any                `json:"connector"`
	Title             string             `json:"title"`
	CapturedAt        string             `json:"captured_at,omitempty"`
	ExternalUpdatedAt string             `json:"external_updated_at,omitempty"`
	ArtifactIDs       []string           `json:"artifact_ids"`
	ContentHash       source.ContentHash `json:"content_hash"`
	Locators          json.RawMessage    `json:"locators"`
	Access            source.Access      `json:"access"`
	RetrievalPolicy   string             `json:"retrieval_policy"`
	State             source.State       `json:"state"`
}

// SnapshotMapper keeps the source adapter's exact projection at this boundary.
type SnapshotMapper func(source.Snapshot) any

func mapSnapshot(snapshot source.Snapshot) SnapshotOutput {
	output := SnapshotOutput{
		SnapshotID: snapshot.SnapshotID, MissionID: snapshot.MissionID, Connector: snapshot.Connector,
		Title: snapshot.Title, ArtifactIDs: append([]string(nil), snapshot.ArtifactIDs...), ContentHash: snapshot.ContentHash,
		Locators: append(json.RawMessage(nil), snapshot.Locators...), Access: snapshot.Access,
		RetrievalPolicy: strings.TrimSpace(snapshot.Access.RetrievalPolicy), State: snapshot.State,
	}
	if output.RetrievalPolicy == "" {
		output.RetrievalPolicy = "snapshot_only"
	}
	if !snapshot.CapturedAt.IsZero() {
		output.CapturedAt = snapshot.CapturedAt.UTC().Format(time.RFC3339)
	}
	if !snapshot.ExternalUpdatedAt.IsZero() {
		output.ExternalUpdatedAt = snapshot.ExternalUpdatedAt.UTC().Format(time.RFC3339)
	}
	return output
}
