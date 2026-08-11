package reportrunrepo

import "database/sql"

// Repository owns SQLite persistence for the logical report-run boundary.
//
// It stores only run state and membership links. Ledger payloads and raw
// artifact bodies remain in their existing tables.
type Repository struct {
	db *sql.DB
}

// New creates a report-run repository over the shared Plasma SQLite DB.
func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}
