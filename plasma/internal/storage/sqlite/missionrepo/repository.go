package missionrepo

import "database/sql"

// Repository executes mission, ledger, and projection SQL against a caller-owned DB.
//
// Repository does not own migrations, DB lifecycle, or cross-capability transaction
// policy. Those remain with the parent sqlite.Store.
type Repository struct {
	db *sql.DB
}

// New binds mission persistence to an existing SQLite connection pool.
func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}
