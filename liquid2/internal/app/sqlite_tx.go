package app

import (
	"context"
	"database/sql"
	"errors"
	"time"

	sqlitedb "github.com/c86j224s/liquid2/internal/storage/sqlite/sqlc"
)

type sqliteTx struct {
	ctx context.Context
	q   *sqlitedb.Queries
	now func() int64
}

func (tx *sqliteTx) Now() int64 {
	return tx.now()
}

func (tx *sqliteTx) NextID(prefix string) string {
	seq, err := tx.q.NextSequence(tx.ctx, prefix)
	tx.abort(err)
	return prefix + "_" + time.UnixMilli(tx.now()).Format("20060102150405") + "_" + formatSeq(seq)
}

func (tx *sqliteTx) abort(err error) {
	if err != nil {
		panic(sqliteAbort{err: mapSQLiteError(err)})
	}
}

func (tx *sqliteTx) missing(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, sql.ErrNoRows) {
		return true
	}
	tx.abort(err)
	return false
}
