package artifactrepo

import (
	"context"

	artifactmodel "github.com/c86j224s/liquid2/plasma/internal/artifact"
)

// CreateRawArtifact stores a raw artifact row.
func (r *Repository) CreateRawArtifact(ctx context.Context, artifact artifactmodel.Raw) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO plasma_raw_artifacts (
  artifact_id, mission_id, media_type, byte_size, sha256, storage_uri, filename,
  producer_type, producer_id, created_at, content_blob
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rawArtifactArgs(artifact)...)
	return err
}

// GetRawArtifact reads a raw artifact by stable artifact ID.
func (r *Repository) GetRawArtifact(ctx context.Context, artifactID string) (artifactmodel.Raw, error) {
	return GetRawArtifactTx(ctx, r.db, artifactID)
}

// ListRawArtifacts reads mission raw artifacts ordered by creation time descending.
func (r *Repository) ListRawArtifacts(ctx context.Context, missionID string) ([]artifactmodel.Raw, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT artifact_id
FROM plasma_raw_artifacts
WHERE mission_id = ?
ORDER BY created_at DESC, artifact_id`, missionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artifacts []artifactmodel.Raw
	for rows.Next() {
		var artifactID string
		if err := rows.Scan(&artifactID); err != nil {
			return nil, err
		}
		artifact, err := r.GetRawArtifact(ctx, artifactID)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, artifact)
	}
	return artifacts, rows.Err()
}
