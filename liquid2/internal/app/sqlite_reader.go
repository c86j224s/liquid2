package app

type sqliteReader struct {
	tx *sqliteTx
}

func (reader sqliteReader) Document(id string) (documentRecord, bool) {
	return reader.tx.Document(id)
}

func (reader sqliteReader) Documents() []documentRecord {
	return reader.tx.Documents()
}

func (reader sqliteReader) DocumentVersions(documentID string) []DocumentVersion {
	return reader.tx.DocumentVersions(documentID)
}

func (reader sqliteReader) Folder(id string) (Folder, bool) {
	return reader.tx.Folder(id)
}

func (reader sqliteReader) Folders() []Folder {
	return reader.tx.Folders()
}

func (reader sqliteReader) Tag(id string) (Tag, bool) {
	return reader.tx.Tag(id)
}

func (reader sqliteReader) TagBySlug(slug string) (Tag, bool) {
	return reader.tx.TagBySlug(slug)
}

func (reader sqliteReader) TagHasDocuments(id string) bool {
	return reader.tx.TagHasDocuments(id)
}

func (reader sqliteReader) Tags() []Tag {
	return reader.tx.Tags()
}

func (reader sqliteReader) Feed(id string) (Feed, bool) {
	return reader.tx.Feed(id)
}

func (reader sqliteReader) FeedByURL(url string) (Feed, bool) {
	return reader.tx.FeedByURL(url)
}

func (reader sqliteReader) Feeds() []Feed {
	return reader.tx.Feeds()
}

func (reader sqliteReader) FeedItemByDocumentID(documentID string) (FeedItem, bool) {
	return reader.tx.FeedItemByDocumentID(documentID)
}

func (reader sqliteReader) FeedItems(feedID string) []FeedItem {
	return reader.tx.FeedItems(feedID)
}

func (reader sqliteReader) Job(id string) (Job, bool) {
	return reader.tx.Job(id)
}

func (reader sqliteReader) Jobs(filters JobFilters) []Job {
	return reader.tx.Jobs(filters)
}

func (reader sqliteReader) DocumentNotes(documentID string) []DocumentNote {
	return reader.tx.DocumentNotes(documentID)
}

func (reader sqliteReader) Note(documentID string, noteID string) (DocumentNote, bool) {
	return reader.tx.Note(documentID, noteID)
}

func (reader sqliteReader) Settings() AppSettings {
	return reader.tx.Settings()
}
