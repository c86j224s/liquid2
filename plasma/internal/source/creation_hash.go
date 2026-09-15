package source

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

func verifiedSnapshotHash(requested ContentHash, artifacts []artifact.Raw) (ContentHash, error) {
	algorithm := strings.TrimSpace(requested.Algorithm)
	if algorithm == "" {
		algorithm = "sha256"
	}
	if !strings.EqualFold(algorithm, "sha256") {
		return ContentHash{}, fmt.Errorf("%w: snapshot content hash algorithm must be sha256", producterror.ErrInvalidInput)
	}

	value := snapshotHashValue(artifacts)
	if requested.Value != "" && !strings.EqualFold(strings.TrimSpace(requested.Value), value) {
		return ContentHash{}, fmt.Errorf("%w: snapshot content hash mismatch", producterror.ErrInvalidInput)
	}
	return ContentHash{Algorithm: "sha256", Value: value}, nil
}

func snapshotHashValue(artifacts []artifact.Raw) string {
	if len(artifacts) == 1 {
		return artifacts[0].SHA256
	}
	hash := sha256.New()
	for _, artifact := range artifacts {
		hash.Write([]byte(artifact.ArtifactID))
		hash.Write([]byte{0})
		hash.Write([]byte(artifact.SHA256))
		hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}
