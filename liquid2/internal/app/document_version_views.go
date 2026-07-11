package app

func cloneDocumentVersion(version DocumentVersion) DocumentVersion {
	version.Contents = cloneDocumentContents(version.Contents)
	version.Metadata = cloneDocumentMetadata(version.Metadata)
	return version
}

func cloneDocumentVersions(versions []DocumentVersion) []DocumentVersion {
	if versions == nil {
		return nil
	}
	cloned := make([]DocumentVersion, len(versions))
	for i, version := range versions {
		cloned[i] = cloneDocumentVersion(version)
	}
	return cloned
}

func cloneDocumentVersionPointers(versions []*DocumentVersion) []*DocumentVersion {
	if versions == nil {
		return nil
	}
	cloned := make([]*DocumentVersion, len(versions))
	for i, version := range versions {
		copied := cloneDocumentVersion(*version)
		cloned[i] = &copied
	}
	return cloned
}
