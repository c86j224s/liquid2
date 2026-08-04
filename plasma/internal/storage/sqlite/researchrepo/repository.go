package researchrepo

import "database/sql"

// Repository executes research-record SQL against a caller-owned DB.
//
// It does not own migrations, DB lifecycle, ledger writes, or cross-capability
// transaction policy.
type Repository struct {
	db *sql.DB
}

// New binds research persistence to an existing SQLite connection pool.
func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}
