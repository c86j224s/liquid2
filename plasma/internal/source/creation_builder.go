package source

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

// BuildSnapshot validates and assembles a source snapshot without persisting it.
// Pending artifacts take precedence over reader lookups by artifact ID. For
// snapshot-only sources it computes the hash in the caller's artifact order,
// while live references keep the no-content hash. CapturedAt is assigned at
// successful assembly time. A reader is needed only for artifact IDs not found
// in pending; callers that commit the result retain responsibility for the
// atomic write.
func BuildSnapshot(
	ctx context.Context,
	artifacts ArtifactReader,
	req CreateRequest,
	pending []artifact.Raw,
) (Snapshot, error) {
	snapshotID := strings.TrimSpace(req.SnapshotID)
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("src_", snapshotID); err != nil {
		return Snapshot{}, err
	}
	if err := validateID("mis_", missionID); err != nil {
		return Snapshot{}, err
	}
	if strings.TrimSpace(req.Connector.ConnectorID) == "" || strings.TrimSpace(req.Connector.ConnectorType) == "" {
		return Snapshot{}, fmt.Errorf("%w: connector id and type are required", producterror.ErrInvalidInput)
	}
	if strings.TrimSpace(req.Connector.ExternalSourceID) == "" && strings.TrimSpace(req.Connector.ExternalURI) == "" {
		return Snapshot{}, fmt.Errorf("%w: external source id or uri is required", producterror.ErrInvalidInput)
	}
	if len(req.Locators) > 0 && !json.Valid(req.Locators) {
		return Snapshot{}, fmt.Errorf("%w: locators must be valid JSON", producterror.ErrInvalidInput)
	}
	access := defaultSourceAccess(req.Access)
	if err := validateSourceRetrievalPolicy(access.RetrievalPolicy); err != nil {
		return Snapshot{}, err
	}
	liveReference := access.RetrievalPolicy == RetrievalPolicyLiveReference
	if liveReference {
		if len(req.ArtifactIDs) != 0 {
			return Snapshot{}, fmt.Errorf("%w: live reference sources must not store raw artifacts", producterror.ErrInvalidInput)
		}
		if err := validateLiveReference(req); err != nil {
			return Snapshot{}, err
		}
	} else if len(req.ArtifactIDs) == 0 {
		return Snapshot{}, fmt.Errorf("%w: snapshot artifact ids are required", producterror.ErrInvalidInput)
	}

	pendingByID := map[string]artifact.Raw{}
	for _, artifact := range pending {
		pendingByID[artifact.ArtifactID] = artifact
	}

	resolvedArtifacts := make([]artifact.Raw, 0, len(req.ArtifactIDs))
	artifactIDs := make([]string, 0, len(req.ArtifactIDs))
	seenArtifactIDs := map[string]struct{}{}
	for _, artifactID := range req.ArtifactIDs {
		trimmedArtifactID := strings.TrimSpace(artifactID)
		if err := validateID("art_", trimmedArtifactID); err != nil {
			return Snapshot{}, err
		}
		if _, ok := seenArtifactIDs[trimmedArtifactID]; ok {
			return Snapshot{}, fmt.Errorf("%w: duplicate snapshot artifact id", producterror.ErrInvalidInput)
		}
		seenArtifactIDs[trimmedArtifactID] = struct{}{}
		resolved, ok := pendingByID[trimmedArtifactID]
		if !ok {
			var err error
			resolved, err = artifacts.GetRawArtifact(ctx, trimmedArtifactID)
			if err != nil {
				return Snapshot{}, err
			}
		}
		if resolved.MissionID != missionID {
			return Snapshot{}, fmt.Errorf("%w: snapshot artifact belongs to another mission", producterror.ErrInvalidInput)
		}
		resolvedArtifacts = append(resolvedArtifacts, resolved)
		artifactIDs = append(artifactIDs, trimmedArtifactID)
	}

	contentHash := ContentHash{Algorithm: "none", Value: ""}
	if !liveReference {
		var err error
		contentHash, err = verifiedSnapshotHash(req.ContentHash, resolvedArtifacts)
		if err != nil {
			return Snapshot{}, err
		}
	}
	capturedAt := time.Now().UTC()
	locators := append(json.RawMessage(nil), req.Locators...)
	if len(locators) == 0 {
		locators = json.RawMessage(`[]`)
	}

	snapshot := Snapshot{
		SnapshotID:        snapshotID,
		MissionID:         missionID,
		Connector:         NormalizeConnector(req.Connector),
		Title:             strings.TrimSpace(req.Title),
		CapturedAt:        capturedAt,
		ExternalUpdatedAt: req.ExternalUpdatedAt,
		ArtifactIDs:       artifactIDs,
		ContentHash:       contentHash,
		Locators:          locators,
		Access:            access,
	}
	return snapshot, nil
}

func validateID(prefix, id string) error {
	trimmed := strings.TrimSpace(id)
	if !strings.HasPrefix(trimmed, prefix) || len(trimmed) <= len(prefix) {
		return fmt.Errorf("%w: id must start with %s", producterror.ErrInvalidInput, prefix)
	}
	return nil
}
