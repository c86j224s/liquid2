package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/confluenceaccess"
	"io"
	"strings"
	"time"
)

func confluenceOAuthUnsupportedCLIError(command string, stderr io.Writer) int {
	writeSourceCommandError(stderr, command, fmt.Errorf("%w: Confluence OAuth is disabled in Plasma 0.0. Use an API token connection instead.", app.ErrInvalidInput))
	return 2
}

func runSourcesConfluenceOAuthURL(_ context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources confluence oauth-url", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.String("client-id", "", "Atlassian OAuth 3LO client id")
	fs.String("redirect-uri", "", "Atlassian OAuth 3LO callback URL")
	fs.String("authorize-url", "", "override Atlassian OAuth authorize URL")
	fs.String("state", "", "OAuth state; defaults to a generated value")
	fs.Bool("unsafe-allow-oauth-overrides", false, "allow OAuth endpoint overrides for local test environments")
	fs.Bool("json", false, "write JSON")
	scopes := repeatedStringFlag{}
	fs.Var(&scopes, "scope", "OAuth scope; repeatable")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	return confluenceOAuthUnsupportedCLIError("sources confluence oauth-url", stderr)
}

func runSourcesConfluenceOAuthExchange(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources confluence oauth-exchange", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.String("db", "", "Plasma SQLite database path")
	fs.String("connection", "", "connection id; defaults to a generated cnf_ id")
	fs.String("name", "Confluence", "connection display name")
	fs.String("account-id", "", "Atlassian account id metadata")
	fs.String("account-name", "", "Atlassian account display name metadata")
	fs.String("client-id", "", "Atlassian OAuth 3LO client id")
	fs.String("client-secret", "", "Atlassian OAuth 3LO client secret")
	fs.String("redirect-uri", "", "Atlassian OAuth 3LO callback URL")
	fs.String("token-url", "", "override Atlassian OAuth token URL")
	fs.String("discovery-url", "", "override Atlassian accessible-resources base URL")
	fs.String("code", "", "authorization code returned by Atlassian")
	fs.Bool("unsafe-allow-oauth-overrides", false, "allow OAuth endpoint overrides for local test environments")
	fs.Bool("json", false, "write JSON")
	scopes := repeatedStringFlag{}
	fs.Var(&scopes, "scope", "OAuth scope; repeatable")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	return confluenceOAuthUnsupportedCLIError("sources confluence oauth-exchange", stderr)
}

func runSourcesConfluenceConnectToken(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources confluence connect-token", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	connectionID := fs.String("connection", "", "connection id; defaults to a generated cnf_ id")
	name := fs.String("name", "Confluence", "connection display name")
	authType := fs.String("auth-type", confluenceaccess.AuthAPIToken, "auth type: api_token")
	email := fs.String("email", "", "Atlassian email")
	accessToken := fs.String("access-token", "", "API token value")
	apiToken := fs.String("api-token", "", "Atlassian API token")
	refreshToken := fs.String("refresh-token", "", "deprecated OAuth refresh token; unsupported in Plasma 0.0")
	expiresAt := fs.String("expires-at", "", "deprecated OAuth token expiry; unsupported in Plasma 0.0")
	jsonOut := fs.Bool("json", false, "write JSON")
	scopes := repeatedStringFlag{}
	fs.Var(&scopes, "scope", "OAuth scope; repeatable")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	id := strings.TrimSpace(*connectionID)
	if id == "" {
		id = cliNewID("cnf")
	}
	token := strings.TrimSpace(*accessToken)
	normalizedAuthType := strings.TrimSpace(*authType)
	if strings.TrimSpace(*apiToken) != "" {
		normalizedAuthType = confluenceaccess.AuthAPIToken
		token = strings.TrimSpace(*apiToken)
	}
	if normalizedAuthType == confluenceaccess.AuthOAuth {
		return confluenceOAuthUnsupportedCLIError("sources confluence connect-token", stderr)
	}
	var expires time.Time
	if strings.TrimSpace(*expiresAt) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*expiresAt))
		if err != nil {
			writeSourceCommandError(stderr, "sources confluence connect-token", fmt.Errorf("%w: expires-at must be RFC3339", app.ErrInvalidInput))
			return 2
		}
		expires = parsed
	}
	connection, err := svc.UpsertConfluenceConnection(ctx, confluenceaccess.UpsertRequest{
		ConnectionID:   id,
		DisplayName:    *name,
		AuthType:       normalizedAuthType,
		AccountName:    *email,
		AccessToken:    token,
		RefreshToken:   *refreshToken,
		TokenExpiresAt: expires,
		Scopes:         []string(scopes),
	})
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence connect-token", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{"connection": connection})
		return 0
	}
	fmt.Fprintf(stdout, "connected confluence %s auth=%s name=%q\n", connection.ConnectionID, connection.AuthType, connection.DisplayName)
	return 0
}

func rejectConfluenceOAuthEndpointOverride(value string, name string, allowed bool) error {
	if strings.TrimSpace(value) == "" || allowed {
		return nil
	}
	return fmt.Errorf("%w: %s override requires --unsafe-allow-oauth-overrides", app.ErrInvalidInput, name)
}
