package app

import "encoding/base64"

func documentCursorStart(entries []documentListEntry, cursor string) (int, error) {
	if cursor == "" {
		return 0, nil
	}
	id, err := decodeDocumentCursor(cursor)
	if err != nil {
		return 0, err
	}
	for index, entry := range entries {
		if entry.record.meta.ID == id {
			return index + 1, nil
		}
	}
	return 0, validation("cursor is invalid")
}

func sliceDocumentIDsByCursor(ids []string, filters DocumentFilters) ([]string, *string, error) {
	start := 0
	if filters.Cursor != "" {
		id, err := decodeDocumentCursor(filters.Cursor)
		if err != nil {
			return nil, nil, err
		}
		for index, item := range ids {
			if item == id {
				start = index + 1
				break
			}
		}
		if start == 0 {
			return nil, nil, validation("cursor is invalid")
		}
	}
	end := start + filters.Limit
	if end > len(ids) {
		end = len(ids)
	}
	page := append([]string(nil), ids[start:end]...)
	var nextCursor *string
	if end < len(ids) {
		nextCursor = encodeDocumentCursor(ids[end-1])
	}
	return page, nextCursor, nil
}

func sliceDocumentIDPage(ids []string, limit int) ([]string, *string) {
	end := limit
	if end > len(ids) {
		end = len(ids)
	}
	page := append([]string(nil), ids[:end]...)
	var nextCursor *string
	if end < len(ids) {
		nextCursor = encodeDocumentCursor(ids[end-1])
	}
	return page, nextCursor
}

func encodeDocumentCursor(id string) *string {
	encoded := base64.RawURLEncoding.EncodeToString([]byte(id))
	return &encoded
}

func decodeDocumentCursor(cursor string) (string, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil || len(decoded) == 0 {
		return "", validation("cursor is invalid")
	}
	return string(decoded), nil
}
