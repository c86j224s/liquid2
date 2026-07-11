package app

func (reader memoryReader) DocumentVersions(documentID string) []DocumentVersion {
	versions := reader.state.versions[documentID]
	items := make([]DocumentVersion, 0, len(versions))
	for _, version := range versions {
		items = append(items, cloneDocumentVersion(*version))
	}
	return items
}

func (tx memoryTx) PutDocumentVersion(version DocumentVersion) {
	cloned := cloneDocumentVersion(version)
	tx.state.versions[version.DocumentID] = append(tx.state.versions[version.DocumentID], &cloned)
}
