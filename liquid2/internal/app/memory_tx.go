package app

type memoryReader struct {
	state *repositoryState
}

type memoryTx struct {
	memoryReader
}

func (reader memoryReader) Document(id string) (documentRecord, bool) {
	record, ok := reader.state.docs[id]
	if !ok {
		return documentRecord{}, false
	}
	return cloneDocumentRecord(*record), true
}

func (reader memoryReader) Documents() []documentRecord {
	records := make([]documentRecord, 0, len(reader.state.docs))
	for _, record := range reader.state.docs {
		records = append(records, cloneDocumentRecord(*record))
	}
	return records
}

func (reader memoryReader) Folder(id string) (Folder, bool) {
	folder, ok := reader.state.folders[id]
	if !ok {
		return Folder{}, false
	}
	return cloneFolder(*folder), true
}

func (reader memoryReader) Folders() []Folder {
	folders := make([]Folder, 0, len(reader.state.folders))
	for _, folder := range reader.state.folders {
		folders = append(folders, cloneFolder(*folder))
	}
	return folders
}

func (reader memoryReader) Tag(id string) (Tag, bool) {
	tag, ok := reader.state.tags[id]
	if !ok {
		return Tag{}, false
	}
	return *tag, true
}

func (reader memoryReader) TagBySlug(slug string) (Tag, bool) {
	id, ok := reader.state.tagSlugs[slug]
	if !ok {
		return Tag{}, false
	}
	return reader.Tag(id)
}

func (reader memoryReader) TagHasDocuments(id string) bool {
	for _, doc := range reader.state.docs {
		if hasValueInSlice(doc.tagIDs, id) {
			return true
		}
	}
	return false
}

func (reader memoryReader) Tags() []Tag {
	tags := make([]Tag, 0, len(reader.state.tags))
	for _, tag := range reader.state.tags {
		tags = append(tags, *tag)
	}
	return tags
}

func (reader memoryReader) DocumentNotes(documentID string) []DocumentNote {
	notes := make([]DocumentNote, 0, len(reader.state.notes[documentID]))
	for _, note := range reader.state.notes[documentID] {
		notes = append(notes, cloneNote(*note))
	}
	return notes
}

func (reader memoryReader) Note(documentID string, noteID string) (DocumentNote, bool) {
	note, ok := reader.state.notes[documentID][noteID]
	if !ok {
		return DocumentNote{}, false
	}
	return cloneNote(*note), true
}

func (tx memoryTx) Now() int64 {
	return tx.state.now()
}

func (tx memoryTx) NextID(prefix string) string {
	return tx.state.nextID(prefix)
}

func (tx memoryTx) PutDocument(record documentRecord) {
	cloned := cloneDocumentRecord(record)
	tx.state.docs[record.meta.ID] = &cloned
}

func (tx memoryTx) PutFolder(folder Folder) {
	cloned := cloneFolder(folder)
	tx.state.folders[folder.ID] = &cloned
}

func (tx memoryTx) DeleteFolder(id string) {
	delete(tx.state.folders, id)
}

func (tx memoryTx) PutTag(tag Tag) {
	cloned := tag
	tx.state.tags[tag.ID] = &cloned
	tx.state.tagSlugs[tag.Slug] = tag.ID
}

func (tx memoryTx) DeleteTag(id string) {
	tag, ok := tx.state.tags[id]
	if ok {
		delete(tx.state.tagSlugs, tag.Slug)
	}
	delete(tx.state.tags, id)
	for _, doc := range tx.state.docs {
		doc.tagIDs = removeStringValue(doc.tagIDs, id)
	}
}

func (tx memoryTx) PutNote(note DocumentNote) {
	cloned := cloneNote(note)
	if tx.state.notes[note.DocumentID] == nil {
		tx.state.notes[note.DocumentID] = map[string]*DocumentNote{}
	}
	tx.state.notes[note.DocumentID][note.ID] = &cloned
}
