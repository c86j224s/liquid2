package artifact

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

// Build validates and assembles a raw artifact without persisting it.
func Build(req CreateRequest) (Raw, error) {
	artifactID := strings.TrimSpace(req.ArtifactID)
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("art_", artifactID); err != nil {
		return Raw{}, err
	}
	if err := validateID("mis_", missionID); err != nil {
		return Raw{}, err
	}
	if strings.TrimSpace(req.MediaType) == "" {
		return Raw{}, fmt.Errorf("%w: media type is required", producterror.ErrInvalidInput)
	}
	if strings.TrimSpace(req.Producer.Type) == "" || strings.TrimSpace(req.Producer.ID) == "" {
		return Raw{}, fmt.Errorf("%w: producer type and id are required", producterror.ErrInvalidInput)
	}
	if len(req.Content) == 0 {
		return Raw{}, fmt.Errorf("%w: artifact content is required", producterror.ErrInvalidInput)
	}

	sum := sha256.Sum256(req.Content)
	sha := hex.EncodeToString(sum[:])
	if req.ExpectedSHA256 != "" && !strings.EqualFold(strings.TrimSpace(req.ExpectedSHA256), sha) {
		return Raw{}, fmt.Errorf("%w: artifact sha256 mismatch", producterror.ErrInvalidInput)
	}

	artifact := Raw{
		ArtifactID: artifactID,
		MissionID:  missionID,
		MediaType:  strings.TrimSpace(req.MediaType),
		ByteSize:   int64(len(req.Content)),
		SHA256:     sha,
		StorageURI: artifactStorageURI(missionID, sha),
		Filename:   strings.TrimSpace(req.Filename),
		Producer: ledger.Producer{
			Type: strings.TrimSpace(req.Producer.Type),
			ID:   strings.TrimSpace(req.Producer.ID),
		},
		CreatedAt: time.Now().UTC(),
		Content:   append([]byte(nil), req.Content...),
	}
	return artifact, nil
}

func validateID(prefix, id string) error {
	trimmed := strings.TrimSpace(id)
	if !strings.HasPrefix(trimmed, prefix) || len(trimmed) <= len(prefix) {
		return fmt.Errorf("%w: id must start with %s", producterror.ErrInvalidInput, prefix)
	}
	return nil
}

func artifactStorageURI(missionID, sha string) string {
	prefix := sha
	if len(prefix) > 2 {
		prefix = prefix[:2]
	}
	return fmt.Sprintf("plasma-artifact://%s/%s/%s", missionID, prefix, sha)
}
