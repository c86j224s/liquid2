package confluencerepo

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/confluenceaccess"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite/internal/sqlitevalue"
)

// Repository executes Confluence connection SQL against a caller-owned DB.
type Repository struct {
	db *sql.DB
}

// New binds Confluence connection persistence to an existing SQLite connection pool.
func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// UpsertConfluenceConnection stores or updates a Confluence connection.
func (r *Repository) UpsertConfluenceConnection(ctx context.Context, connection confluenceaccess.Connection) error {
	scopesJSON, err := json.Marshal(connection.Scopes)
	if err != nil {
		return err
	}
	sitesJSON, err := json.Marshal(connection.Sites)
	if err != nil {
		return err
	}
	createdAt := connection.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	updatedAt := connection.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}
	_, err = r.db.ExecContext(ctx, `
INSERT INTO plasma_confluence_connections (
  connection_id, display_name, auth_type, account_id, account_name, access_token,
  refresh_token, token_expires_at, scopes_json, sites_json, revoked, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(connection_id) DO UPDATE SET
  display_name = excluded.display_name,
  auth_type = excluded.auth_type,
  account_id = excluded.account_id,
  account_name = excluded.account_name,
  access_token = excluded.access_token,
  refresh_token = excluded.refresh_token,
  token_expires_at = excluded.token_expires_at,
  scopes_json = excluded.scopes_json,
  sites_json = excluded.sites_json,
  revoked = excluded.revoked,
  updated_at = excluded.updated_at`,
		connection.ConnectionID,
		connection.DisplayName,
		connection.AuthType,
		connection.AccountID,
		connection.AccountName,
		connection.AccessToken,
		connection.RefreshToken,
		sqlitevalue.FormatOptionalTime(connection.TokenExpiresAt),
		string(scopesJSON),
		string(sitesJSON),
		sqlitevalue.BoolInt(connection.Revoked),
		sqlitevalue.FormatTime(createdAt),
		sqlitevalue.FormatTime(updatedAt))
	return err
}

// GetConfluenceConnection reads one Confluence connection by stable ID.
func (r *Repository) GetConfluenceConnection(ctx context.Context, connectionID string) (confluenceaccess.Connection, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT connection_id, display_name, auth_type, account_id, account_name, access_token,
       refresh_token, token_expires_at, scopes_json, sites_json, revoked, created_at, updated_at
FROM plasma_confluence_connections
WHERE connection_id = ?`, connectionID)
	return scanConfluenceConnection(row)
}

// ListConfluenceConnections reads all Confluence connections ordered by update time.
func (r *Repository) ListConfluenceConnections(ctx context.Context) ([]confluenceaccess.Connection, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT connection_id, display_name, auth_type, account_id, account_name, access_token,
       refresh_token, token_expires_at, scopes_json, sites_json, revoked, created_at, updated_at
FROM plasma_confluence_connections
ORDER BY updated_at DESC, connection_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var connections []confluenceaccess.Connection
	for rows.Next() {
		connection, err := scanConfluenceConnection(rows)
		if err != nil {
			return nil, err
		}
		connections = append(connections, connection)
	}
	return connections, rows.Err()
}

// DeleteConfluenceConnection deletes one Confluence connection row.
func (r *Repository) DeleteConfluenceConnection(ctx context.Context, connectionID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM plasma_confluence_connections WHERE connection_id = ?`, connectionID)
	return err
}

type confluenceConnectionScanner interface {
	Scan(dest ...any) error
}

func scanConfluenceConnection(scanner confluenceConnectionScanner) (confluenceaccess.Connection, error) {
	var connection confluenceaccess.Connection
	var tokenExpiresAt string
	var scopesJSON string
	var sitesJSON string
	var revoked int
	var createdAt string
	var updatedAt string
	if err := scanner.Scan(
		&connection.ConnectionID,
		&connection.DisplayName,
		&connection.AuthType,
		&connection.AccountID,
		&connection.AccountName,
		&connection.AccessToken,
		&connection.RefreshToken,
		&tokenExpiresAt,
		&scopesJSON,
		&sitesJSON,
		&revoked,
		&createdAt,
		&updatedAt); err != nil {
		return confluenceaccess.Connection{}, err
	}
	if tokenExpiresAt != "" {
		parsed, err := time.Parse(time.RFC3339Nano, tokenExpiresAt)
		if err != nil {
			return confluenceaccess.Connection{}, err
		}
		connection.TokenExpiresAt = parsed
	}
	if err := json.Unmarshal([]byte(scopesJSON), &connection.Scopes); err != nil {
		return confluenceaccess.Connection{}, err
	}
	if err := json.Unmarshal([]byte(sitesJSON), &connection.Sites); err != nil {
		return confluenceaccess.Connection{}, err
	}
	connection.Revoked = revoked != 0
	parsedCreated, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return confluenceaccess.Connection{}, err
	}
	connection.CreatedAt = parsedCreated
	parsedUpdated, err := time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return confluenceaccess.Connection{}, err
	}
	connection.UpdatedAt = parsedUpdated
	return connection, nil
}

var _ confluenceConnectionScanner = (*sql.Row)(nil)
var _ confluenceConnectionScanner = (*sql.Rows)(nil)
