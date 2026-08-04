package artifactrepo

import "database/sql"

// Repository executes raw artifact and source snapshot SQL against a caller-owned DB.
//
// It does not own migrations, DB lifecycle, ledger updates, or product-level
// orchestration.
type Repository struct {
	db *sql.DB
}

// New binds artifact persistence to an existing SQLite connection pool.
func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}
