package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func (s *Store) UpsertConfluenceConnection(ctx context.Context, connection app.ConfluenceConnection) error {
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
	_, err = s.db.ExecContext(ctx, `
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
		formatOptionalTime(connection.TokenExpiresAt),
		string(scopesJSON),
		string(sitesJSON),
		boolInt(connection.Revoked),
		formatTime(createdAt),
		formatTime(updatedAt))
	return err
}

func (s *Store) GetConfluenceConnection(ctx context.Context, connectionID string) (app.ConfluenceConnection, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT connection_id, display_name, auth_type, account_id, account_name, access_token,
       refresh_token, token_expires_at, scopes_json, sites_json, revoked, created_at, updated_at
FROM plasma_confluence_connections
WHERE connection_id = ?`, connectionID)
	return scanConfluenceConnection(row)
}

func (s *Store) ListConfluenceConnections(ctx context.Context) ([]app.ConfluenceConnection, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT connection_id, display_name, auth_type, account_id, account_name, access_token,
       refresh_token, token_expires_at, scopes_json, sites_json, revoked, created_at, updated_at
FROM plasma_confluence_connections
ORDER BY updated_at DESC, connection_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var connections []app.ConfluenceConnection
	for rows.Next() {
		connection, err := scanConfluenceConnection(rows)
		if err != nil {
			return nil, err
		}
		connections = append(connections, connection)
	}
	return connections, rows.Err()
}

func (s *Store) DeleteConfluenceConnection(ctx context.Context, connectionID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM plasma_confluence_connections WHERE connection_id = ?`, connectionID)
	return err
}

type confluenceConnectionScanner interface {
	Scan(dest ...any) error
}

func scanConfluenceConnection(scanner confluenceConnectionScanner) (app.ConfluenceConnection, error) {
	var connection app.ConfluenceConnection
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
		return app.ConfluenceConnection{}, err
	}
	if tokenExpiresAt != "" {
		parsed, err := time.Parse(time.RFC3339Nano, tokenExpiresAt)
		if err != nil {
			return app.ConfluenceConnection{}, err
		}
		connection.TokenExpiresAt = parsed
	}
	if err := json.Unmarshal([]byte(scopesJSON), &connection.Scopes); err != nil {
		return app.ConfluenceConnection{}, err
	}
	if err := json.Unmarshal([]byte(sitesJSON), &connection.Sites); err != nil {
		return app.ConfluenceConnection{}, err
	}
	connection.Revoked = revoked != 0
	parsedCreated, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return app.ConfluenceConnection{}, err
	}
	connection.CreatedAt = parsedCreated
	parsedUpdated, err := time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return app.ConfluenceConnection{}, err
	}
	connection.UpdatedAt = parsedUpdated
	return connection, nil
}

var _ confluenceConnectionScanner = (*sql.Row)(nil)
var _ confluenceConnectionScanner = (*sql.Rows)(nil)
