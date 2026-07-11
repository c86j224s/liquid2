package app

func (reader memoryReader) ListDocumentIDs(filters DocumentFilters) ([]string, *string, error) {
	return listDocumentIDsFromRecords(reader, reader.Documents(), filters)
}

func (reader memoryReader) CountDocuments(filters DocumentFilters) (int, error) {
	return countDocumentRecords(reader, reader.Documents(), filters)
}
