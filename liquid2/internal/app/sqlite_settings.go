package app

import sqlitedb "github.com/c86j224s/liquid2/internal/storage/sqlite/sqlc"

func (tx *sqliteTx) Settings() AppSettings {
	row, err := tx.q.GetAppSettings(tx.ctx)
	if tx.missing(err) {
		return DefaultAppSettings()
	}
	return AppSettings{
		FeedSchedulerEnabled:    row.FeedSchedulerEnabled != 0,
		FeedPollIntervalSeconds: int(row.FeedPollIntervalSeconds),
		FeedNextPollAt:          sqliteInt64Ptr(row.FeedNextPollAt),
		UpdatedAt:               row.UpdatedAt,
	}
}

func (tx *sqliteTx) PutSettings(settings AppSettings) {
	_, err := tx.q.UpsertAppSettings(tx.ctx, sqlitedb.UpsertAppSettingsParams{
		FeedSchedulerEnabled:    sqliteBool(settings.FeedSchedulerEnabled),
		FeedPollIntervalSeconds: int64(settings.FeedPollIntervalSeconds),
		FeedNextPollAt:          sqliteNullInt64(settings.FeedNextPollAt),
		UpdatedAt:               settings.UpdatedAt,
	})
	tx.abort(err)
}
