package reporthumanize

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func finalizedArtifact(ctx context.Context, service Service, missionID string, finalized ledger.Event) (artifact.Raw, error) {
	var payload struct {
		ArtifactID string `json:"artifact_id"`
	}
	if err := json.Unmarshal(finalized.Payload, &payload); err != nil {
		return artifact.Raw{}, fmt.Errorf("%w: invalid H5 patch finalized payload", producterror.ErrInvalidInput)
	}
	artifactID := strings.TrimSpace(payload.ArtifactID)
	if artifactID == "" {
		return artifact.Raw{}, fmt.Errorf("%w: H5 patch finalized payload is missing artifact_id", producterror.ErrInvalidInput)
	}
	raw, err := service.GetRawArtifact(ctx, artifactID)
	if err != nil {
		return artifact.Raw{}, err
	}
	if raw.MissionID != missionID {
		return artifact.Raw{}, fmt.Errorf("%w: H5 patch artifact belongs to another mission", producterror.ErrInvalidInput)
	}
	if !isMarkdownMediaType(raw.MediaType) {
		return artifact.Raw{}, fmt.Errorf("%w: H5 patch artifact must be Markdown", producterror.ErrInvalidInput)
	}
	return raw, nil
}
