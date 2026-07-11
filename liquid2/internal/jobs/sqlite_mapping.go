package jobs

import (
	"database/sql"
	"time"

	sqlitedb "github.com/c86j224s/liquid2/internal/storage/sqlite/sqlc"
)

func sqliteJob(row sqlitedb.Job) Job {
	return Job{
		ID: row.ID, Kind: row.Kind, Status: row.Status, PayloadJSON: row.PayloadJson,
		Error: sqliteStringPtr(row.Error), Attempts: row.Attempts, CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt, StartedAt: sqliteInt64Ptr(row.StartedAt),
		FinishedAt: sqliteInt64Ptr(row.FinishedAt),
	}
}

func sqliteStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	cloned := value.String
	return &cloned
}

func sqliteInt64Ptr(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	cloned := value.Int64
	return &cloned
}

func nullString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func nullInt64(value int64) sql.NullInt64 {
	return sql.NullInt64{Int64: value, Valid: true}
}

func filterString(value string) string {
	return value
}

func unixMillis() int64 {
	return time.Now().UnixMilli()
}
