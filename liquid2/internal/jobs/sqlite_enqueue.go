package jobs

import (
	"context"
	"errors"
	"strconv"
	"time"

	sqlitedb "github.com/c86j224s/liquid2/internal/storage/sqlite/sqlc"
	modernsqlite "modernc.org/sqlite"
)

const sqliteConstraint = 19 // SQLITE_CONSTRAINT primary result code.

func mapEnqueueError(err error) error {
	var sqliteErr *modernsqlite.Error
	if errors.As(err, &sqliteErr) && sqliteErr.Code()&0xff == sqliteConstraint {
		return ErrJobConflict
	}
	return err
}

func (queue *SQLiteQueue) jobID(ctx context.Context, q *sqlitedb.Queries, requested string) (string, error) {
	if requested != "" {
		return requested, nil
	}
	seq, err := q.NextSequence(ctx, "job")
	if err != nil {
		return "", err
	}
	return "job_" + time.UnixMilli(queue.now()).Format("20060102150405") + "_" + strconv.FormatInt(seq, 10), nil
}
