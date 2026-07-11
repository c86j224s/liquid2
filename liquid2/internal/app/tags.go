package app

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
)

var slugClean = regexp.MustCompile(`[^a-z0-9]+`)

func (s *Service) ListTags(ctx context.Context) ([]Tag, error) {
	return withView(ctx, s, func(tx RepositoryReader) ([]Tag, error) {
		return tx.Tags(), nil
	})
}

func (s *Service) CreateTag(ctx context.Context, name string) (Tag, error) {
	return withUpdate(ctx, s, func(tx RepositoryTx) (Tag, error) {
		name = strings.TrimSpace(name)
		if name == "" {
			return Tag{}, validation("tag name is required")
		}
		slug := slugify(name)
		if slug == "" {
			return Tag{}, validation("tag slug is required")
		}
		if _, ok := tx.TagBySlug(slug); ok {
			return Tag{}, conflict("tag slug already exists")
		}
		tag := Tag{ID: tx.NextID("tag"), Name: name, Slug: slug, CreatedAt: tx.Now()}
		tx.PutTag(tag)
		s.logger.DebugContext(ctx, "tag created", slog.String("operation", "tag_create"), slog.String("tag_id", tag.ID))
		return tag, nil
	})
}

func slugify(value string) string {
	slug := strings.ToLower(strings.TrimSpace(value))
	slug = slugClean.ReplaceAllString(slug, "-")
	return strings.Trim(slug, "-")
}
