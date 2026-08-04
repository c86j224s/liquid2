package sqlitevalue

import (
	"encoding/json"
	"fmt"
	"time"
)

// FormatTime returns the SQLite timestamp string used by Plasma persistence.
//
// A zero time is replaced with the current UTC time, matching the legacy store
// helper used for user-created records and ledger events.
func FormatTime(t time.Time) string {
	if t.IsZero() {
		t = time.Now().UTC()
	}
	return t.UTC().Format(time.RFC3339Nano)
}

// FormatOptionalTime returns an empty SQLite value for zero times.
func FormatOptionalTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

// MarshalJSON encodes a structured field with the legacy projection error text.
func MarshalJSON(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal projection field: %w", err)
	}
	return string(encoded), nil
}

// UnmarshalJSON decodes a structured field with the legacy projection error text.
func UnmarshalJSON(text string, target any) error {
	if err := json.Unmarshal([]byte(text), target); err != nil {
		return fmt.Errorf("unmarshal projection field: %w", err)
	}
	return nil
}

// BoolInt maps booleans to the integer representation used in SQLite rows.
func BoolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
