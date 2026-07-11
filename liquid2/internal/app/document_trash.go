package app

import (
	"context"
	"log/slog"
)

func (s *Service) MoveDocumentToTrash(ctx context.Context, id string) (DocumentDetail, error) {
	return withUpdate(ctx, s, func(tx RepositoryTx) (DocumentDetail, error) {
		doc, ok := tx.Document(id)
		if !ok || doc.meta.DeletedAt != nil {
			return DocumentDetail{}, notFound("document")
		}
		if doc.meta.FolderID != nil && folderHasSystemRoleAncestor(tx, *doc.meta.FolderID, FolderSystemRoleTrash) {
			s.logger.DebugContext(ctx, "document already in trash",
				slog.String("operation", "document_move_to_trash"),
				slog.String("document_id", id),
			)
			return documentDetail(tx, id), nil
		}
		trashID := ensureTrashFolder(tx)
		doc.meta.FolderID = &trashID
		doc.meta.UpdatedAt = tx.Now()
		tx.PutDocument(doc)
		s.logger.DebugContext(ctx, "document moved to trash",
			slog.String("operation", "document_move_to_trash"),
			slog.String("document_id", id),
		)
		return documentDetail(tx, id), nil
	})
}
