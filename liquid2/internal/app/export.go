package app

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"path"
	"sort"
)

func (s *Service) ExportMarkdown(
	ctx context.Context,
	input MarkdownExportInput,
	writer ExportWriter,
) (MarkdownExportResult, error) {
	if writer == nil {
		return MarkdownExportResult{}, validation("export writer is required")
	}
	if input.ExportID == "" {
		return MarkdownExportResult{}, validation("export id is required")
	}
	snapshot, err := withView(ctx, s, func(tx RepositoryReader) (markdownExportSnapshot, error) {
		records, err := exportRecords(tx, input.DocumentIDs)
		if err != nil {
			return markdownExportSnapshot{}, err
		}
		return buildMarkdownExport(input, tx, records)
	})
	if err != nil {
		return MarkdownExportResult{}, err
	}
	if err = snapshot.Write(ctx, writer); err != nil {
		return MarkdownExportResult{}, err
	}
	result := snapshot.Result
	s.logger.DebugContext(ctx, "markdown export created",
		slog.String("operation", "markdown_export"),
		slog.String("export_id", result.Manifest.ExportID),
		slog.Int("document_count", result.DocumentCount),
		slog.Int("blob_count", result.BlobCount),
	)
	return result, nil
}

type markdownExportSnapshot struct {
	Result MarkdownExportResult
	files  []markdownExportFile
}

type markdownExportFile struct {
	path string
	data []byte
}

func (snapshot markdownExportSnapshot) Write(ctx context.Context, writer ExportWriter) error {
	for _, file := range snapshot.files {
		if err := writer.WriteFile(ctx, file.path, file.data); err != nil {
			return err
		}
	}
	return nil
}

func exportRecords(tx RepositoryReader, documentIDs []string) ([]documentRecord, error) {
	if len(documentIDs) == 0 {
		records := tx.Documents()
		sort.Slice(records, func(i int, j int) bool { return records[i].meta.ID < records[j].meta.ID })
		return filterExportRecords(records), nil
	}
	records := make([]documentRecord, 0, len(documentIDs))
	seen := map[string]struct{}{}
	for _, id := range documentIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		record, ok := tx.Document(id)
		if !ok || record.meta.DeletedAt != nil {
			return nil, notFound("document")
		}
		records = append(records, record)
	}
	sort.Slice(records, func(i int, j int) bool { return records[i].meta.ID < records[j].meta.ID })
	return records, nil
}

func filterExportRecords(records []documentRecord) []documentRecord {
	filtered := records[:0]
	for _, record := range records {
		if record.meta.DeletedAt == nil {
			filtered = append(filtered, record)
		}
	}
	return filtered
}

func buildMarkdownExport(
	input MarkdownExportInput,
	tx RepositoryReader,
	records []documentRecord,
) (markdownExportSnapshot, error) {
	manifest := exportManifest(input, tx, records)
	blobCount := 0
	files := []markdownExportFile{}
	for i, record := range records {
		files = append(files, markdownExportFile{
			path: manifest.Documents[i].MarkdownPath, data: renderMarkdownDocument(tx, record),
		})
		for j, blob := range record.blobs {
			data, ok := record.blobData[blob.ID]
			if !ok {
				return markdownExportSnapshot{}, fmt.Errorf("blob data missing: %s", blob.ID)
			}
			manifest.Documents[i].Blobs[j].SHA256 = fmt.Sprintf("%x", sha256.Sum256(data))
			manifest.Documents[i].Blobs[j].SizeBytes = int64(len(data))
			files = append(files, markdownExportFile{
				path: manifest.Documents[i].Blobs[j].Path, data: append([]byte(nil), data...),
			})
			blobCount++
		}
	}
	manifest.Counts.Blobs = blobCount
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return markdownExportSnapshot{}, err
	}
	body = append(body, '\n')
	files = append(files, markdownExportFile{path: "manifest.json", data: body})
	result := MarkdownExportResult{Manifest: manifest, DocumentCount: len(records), BlobCount: blobCount}
	return markdownExportSnapshot{Result: result, files: files}, nil
}

func exportManifest(input MarkdownExportInput, tx RepositoryReader, records []documentRecord) ExportManifest {
	manifest := ExportManifest{
		ManifestVersion: ExportManifestVersion, ExportID: input.ExportID, CreatedAt: input.CreatedAt,
		Source: ExportSource{App: exportAppName, AppVersion: cloneString(input.AppVersion), SchemaVersion: cloneInt64(input.SchemaVersion)},
		Counts: ExportCounts{Documents: len(records)}, Documents: []ExportDocument{},
	}
	for _, record := range records {
		manifest.Documents = append(manifest.Documents, exportDocument(tx, record))
	}
	return manifest
}

func exportDocument(tx RepositoryReader, record documentRecord) ExportDocument {
	meta := record.meta
	doc := ExportDocument{
		ID: meta.ID, MarkdownPath: path.Join("documents", sanitizeExportName(meta.ID)+".md"),
		Title: meta.Title, Kind: meta.Kind, FolderID: cloneString(meta.FolderID),
		SourceURL: cloneString(meta.SourceURL), CanonicalURL: cloneString(meta.CanonicalURL),
		Language: cloneString(meta.Language), Contents: []ExportContent{}, Tags: []ExportTag{}, Blobs: []ExportBlob{},
	}
	for _, content := range record.contents {
		doc.Contents = append(doc.Contents, ExportContent{
			ID: content.ID, Role: content.Role, Format: content.Format,
			Language: cloneString(content.Language), SourceContentID: cloneString(content.SourceContentID),
		})
	}
	for _, tag := range documentTags(tx, record) {
		doc.Tags = append(doc.Tags, ExportTag{ID: tag.ID, Name: tag.Name, Slug: tag.Slug})
	}
	for _, blob := range record.blobs {
		doc.Blobs = append(doc.Blobs, ExportBlob{
			ID: blob.ID, Path: path.Join("blobs", exportBlobName(blob)),
			Filename: blob.Filename, MimeType: blob.MimeType,
		})
	}
	return doc
}
