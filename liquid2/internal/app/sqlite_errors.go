package app

import (
	"database/sql"
	"errors"
	"strings"

	modernsqlite "modernc.org/sqlite"
)

const sqliteConstraint = 19 // SQLITE_CONSTRAINT primary result code.

func mapSQLiteError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	var sqliteErr *modernsqlite.Error
	if errors.As(err, &sqliteErr) {
		return mapSQLiteCode(sqliteErr.Code(), err)
	}
	return err
}

func mapSQLiteCode(code int, err error) error {
	if code&0xff != sqliteConstraint {
		return err
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "check constraint") {
		return validation("sqlite constraint")
	}
	if strings.Contains(message, "foreign key constraint") {
		return notFound("referenced record")
	}
	return conflict("sqlite constraint")
}
