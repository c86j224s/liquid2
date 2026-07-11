package app

func countDocumentRecords(
	tx RepositoryReader,
	records []documentRecord,
	filters DocumentFilters,
) (int, error) {
	filters, err := normalizeDocumentFilters(filters)
	if err != nil {
		return 0, err
	}
	folderIDs := documentFolderFilterIDs(tx, filters)
	tokens := queryTokens(filters.Query)
	count := 0
	for _, record := range records {
		if !matchesDocumentFiltersWithFolders(tx, record, filters, folderIDs) {
			continue
		}
		if _, ok := documentSearchScore(record, tokens); ok {
			count++
		}
	}
	return count, nil
}
