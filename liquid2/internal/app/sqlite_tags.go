package app

import sqlitedb "github.com/c86j224s/liquid2/internal/storage/sqlite/sqlc"

func (tx *sqliteTx) Tag(id string) (Tag, bool) {
	row, err := tx.q.GetTag(tx.ctx, id)
	if tx.missing(err) {
		return Tag{}, false
	}
	return sqliteTag(row), true
}

func (tx *sqliteTx) TagBySlug(slug string) (Tag, bool) {
	row, err := tx.q.GetTagBySlug(tx.ctx, slug)
	if tx.missing(err) {
		return Tag{}, false
	}
	return sqliteTag(row), true
}

func (tx *sqliteTx) TagHasDocuments(id string) bool {
	_, err := tx.q.TagHasDocuments(tx.ctx, id)
	if tx.missing(err) {
		return false
	}
	tx.abort(err)
	return true
}

func (tx *sqliteTx) Tags() []Tag {
	rows, err := tx.q.ListTags(tx.ctx)
	tx.abort(err)
	tags := make([]Tag, 0, len(rows))
	for _, row := range rows {
		tags = append(tags, sqliteTag(row))
	}
	return tags
}

func (tx *sqliteTx) PutTag(tag Tag) {
	_, err := tx.q.UpsertTag(tx.ctx, sqlitedb.UpsertTagParams{
		ID: tag.ID, Name: tag.Name, Slug: tag.Slug, CreatedAt: tag.CreatedAt,
	})
	tx.abort(err)
}

func (tx *sqliteTx) DeleteTag(id string) {
	tx.abort(tx.q.DeleteTag(tx.ctx, id))
}
