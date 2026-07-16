package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func (s *Store) CommitRawArtifactWithEventConditionally(
	ctx context.Context,
	artifact app.RawArtifact,
	build func([]app.LedgerEvent) (app.LedgerEvent, bool, error),
) (app.RawArtifact, app.LedgerEvent, bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return app.RawArtifact{}, app.LedgerEvent{}, false, err
	}
	defer tx.Rollback()
	events, err := listLedgerEventsTx(ctx, tx, artifact.MissionID)
	if err != nil {
		return app.RawArtifact{}, app.LedgerEvent{}, false, err
	}
	event, create, err := build(events)
	if err != nil {
		return app.RawArtifact{}, app.LedgerEvent{}, false, err
	}
	if create {
		event, err = appendLedgerEventTx(ctx, tx, event)
		if err != nil {
			return app.RawArtifact{}, app.LedgerEvent{}, false, err
		}
		if err := insertRawArtifactTx(ctx, tx, artifact); err != nil {
			return app.RawArtifact{}, app.LedgerEvent{}, false, err
		}
	} else if artifact, err = getRawArtifactTx(ctx, tx, artifact.ArtifactID); err != nil {
		return app.RawArtifact{}, app.LedgerEvent{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return app.RawArtifact{}, app.LedgerEvent{}, false, err
	}
	return artifact, event, create, nil
}

func getRawArtifactTx(ctx context.Context, tx *sql.Tx, artifactID string) (app.RawArtifact, error) {
	var artifact app.RawArtifact
	var createdAt string
	err := tx.QueryRowContext(ctx, `
SELECT artifact_id, mission_id, media_type, byte_size, sha256, storage_uri,
       filename, producer_type, producer_id, created_at, content_blob
FROM plasma_raw_artifacts
WHERE artifact_id = ?`, artifactID).Scan(
		&artifact.ArtifactID, &artifact.MissionID, &artifact.MediaType, &artifact.ByteSize,
		&artifact.SHA256, &artifact.StorageURI, &artifact.Filename, &artifact.Producer.Type,
		&artifact.Producer.ID, &createdAt, &artifact.Content)
	if err != nil {
		return app.RawArtifact{}, err
	}
	artifact.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return app.RawArtifact{}, err
	}
	artifact.Content = append([]byte(nil), artifact.Content...)
	return artifact, nil
}
