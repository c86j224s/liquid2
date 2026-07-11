package app

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
)

var languageTagPattern = regexp.MustCompile(`^[a-zA-Z]{2,8}(-[a-zA-Z0-9]{2,8})*$`)

// NormalizeContentLanguage applies app-owned content language rules.
func NormalizeContentLanguage(value string) (string, error) {
	return normalizeContentLanguage(value)
}

// PreparedTranslation contains source content approved for provider work.
type PreparedTranslation struct {
	Document       DocumentMetadata
	SourceContent  DocumentContent
	TargetLanguage string
}

// PrepareTranslation validates source ownership and duplicate policy before provider work.
func (s *Service) PrepareTranslation(
	ctx context.Context,
	documentID string,
	input PrepareTranslationInput,
) (PreparedTranslation, error) {
	return withView(ctx, s, func(tx RepositoryReader) (PreparedTranslation, error) {
		_, prepared, err := prepareTranslation(tx, documentID, input)
		if err != nil {
			return PreparedTranslation{}, err
		}
		return PreparedTranslation{
			Document:       prepared.document,
			SourceContent:  prepared.source,
			TargetLanguage: prepared.targetLanguage,
		}, nil
	})
}

func (s *Service) AppendTranslatedContent(
	ctx context.Context,
	documentID string,
	input AppendTranslationInput,
) (DocumentDetail, error) {
	return withUpdate(ctx, s, func(tx RepositoryTx) (DocumentDetail, error) {
		doc, prepared, err := prepareTranslation(tx, documentID, PrepareTranslationInput{
			SourceContentID: input.SourceContentID,
			TargetLanguage:  input.TargetLanguage,
		})
		if err != nil {
			return DocumentDetail{}, err
		}
		content := strings.TrimSpace(input.Content)
		if content == "" {
			return DocumentDetail{}, validation("translated content is required")
		}
		format := strings.TrimSpace(input.Format)
		if format == "" {
			format = ContentFormatText
		}
		if !validContentFormat(format) {
			return DocumentDetail{}, validation("content format is invalid")
		}
		now := tx.Now()
		recordDocumentVersion(tx, doc, DocumentMutationContent, now)
		doc.contents = append(doc.contents, DocumentContent{
			ID: tx.NextID("content"), Role: ContentRoleTranslation,
			Format: format, Language: &prepared.targetLanguage, Content: content,
			SourceContentID: &prepared.sourceContentID,
		})
		doc.meta.UpdatedAt = now
		tx.PutDocument(doc)
		s.logger.DebugContext(ctx, "translation content appended",
			slog.String("operation", "document_translation_append"),
			slog.String("document_id", documentID),
			slog.String("source_content_id", prepared.sourceContentID),
			slog.String("target_language", prepared.targetLanguage),
		)
		return documentDetail(tx, documentID), nil
	})
}

type translationPreparation struct {
	document        DocumentMetadata
	source          DocumentContent
	sourceContentID string
	targetLanguage  string
}

func prepareTranslation(
	tx RepositoryReader,
	documentID string,
	input PrepareTranslationInput,
) (documentRecord, translationPreparation, error) {
	doc, ok := tx.Document(documentID)
	if !ok || doc.meta.DeletedAt != nil {
		return documentRecord{}, translationPreparation{}, notFound("document")
	}
	sourceContentID := strings.TrimSpace(input.SourceContentID)
	if sourceContentID == "" {
		return documentRecord{}, translationPreparation{}, validation("source content is required")
	}
	source, ok := findDocumentContent(doc.contents, sourceContentID)
	if !ok {
		return documentRecord{}, translationPreparation{}, notFound("document content")
	}
	targetLanguage, err := normalizeContentLanguage(input.TargetLanguage)
	if err != nil {
		return documentRecord{}, translationPreparation{}, err
	}
	if hasTranslationContent(doc.contents, sourceContentID, targetLanguage) {
		return documentRecord{}, translationPreparation{}, conflict("translation already exists")
	}
	return doc, translationPreparation{
		document: doc.meta, source: source, sourceContentID: sourceContentID,
		targetLanguage: targetLanguage,
	}, nil
}

func validContentFormat(format string) bool {
	switch format {
	case ContentFormatHTML, ContentFormatMarkdown, ContentFormatText:
		return true
	default:
		return false
	}
}

func findDocumentContent(contents []DocumentContent, contentID string) (DocumentContent, bool) {
	for _, content := range contents {
		if content.ID == contentID {
			return content, true
		}
	}
	return DocumentContent{}, false
}

func hasTranslationContent(contents []DocumentContent, sourceContentID string, language string) bool {
	for _, content := range contents {
		if content.Role != ContentRoleTranslation || content.SourceContentID == nil {
			continue
		}
		if *content.SourceContentID == sourceContentID &&
			content.Language != nil &&
			*content.Language == language {
			return true
		}
	}
	return false
}

func normalizeContentLanguage(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "", validation("target language is required")
	}
	if !languageTagPattern.MatchString(value) {
		return "", validation("target language is invalid")
	}
	return value, nil
}
