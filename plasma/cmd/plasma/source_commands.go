package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/config"
	confluenceconnector "github.com/c86j224s/liquid2/plasma/internal/connectors/confluence"
	"github.com/c86j224s/liquid2/plasma/internal/sources/localpath"
	"github.com/c86j224s/liquid2/plasma/internal/sources/pdftext"
)

const (
	cliDefaultReadBytes = int64(20 * 1024)
	cliMaxReadBytes     = int64(256 * 1024)
)

type repeatedStringFlag []string

func (flagValue *repeatedStringFlag) Set(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed != "" {
		*flagValue = append(*flagValue, trimmed)
	}
	return nil
}

func (flagValue repeatedStringFlag) String() string {
	return strings.Join(flagValue, ",")
}

func confluenceOAuthUnsupportedCLIError(command string, stderr io.Writer) int {
	writeSourceCommandError(stderr, command, fmt.Errorf("%w: Confluence OAuth is disabled in Plasma 0.0. Use an API token connection instead.", app.ErrInvalidInput))
	return 2
}

func runSources(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printSourcesUsage(stderr)
		return 2
	}
	switch args[0] {
	case "roots":
		return runSourcesRoots(ctx, args[1:], stdout, stderr)
	case "tree":
		return runSourcesTree(ctx, args[1:], stdout, stderr)
	case "attach-local":
		return runSourcesAttachLocal(ctx, args[1:], stdout, stderr)
	case "upload":
		return runSourcesUpload(ctx, args[1:], stdout, stderr)
	case "list":
		return runSourcesList(ctx, args[1:], stdout, stderr)
	case "show":
		return runSourcesShow(ctx, args[1:], stdout, stderr)
	case "read":
		return runSourcesRead(ctx, args[1:], stdout, stderr)
	case "grep":
		return runSourcesGrep(ctx, args[1:], stdout, stderr)
	case "remove":
		return runSourcesRemove(ctx, args[1:], stdout, stderr)
	case "restore":
		return runSourcesRestore(ctx, args[1:], stdout, stderr)
	case "confluence":
		return runSourcesConfluence(ctx, args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown sources command %q\n", args[0])
		printSourcesUsage(stderr)
		return 2
	}
}

func runSourcesConfluence(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printSourcesConfluenceUsage(stderr)
		return 2
	}
	switch args[0] {
	case "oauth-url":
		return runSourcesConfluenceOAuthURL(ctx, args[1:], stdout, stderr)
	case "oauth-exchange":
		return runSourcesConfluenceOAuthExchange(ctx, args[1:], stdout, stderr)
	case "connect-token", "connect-oauth-token":
		return runSourcesConfluenceConnectToken(ctx, args[1:], stdout, stderr)
	case "connections":
		return runSourcesConfluenceConnections(ctx, args[1:], stdout, stderr)
	case "rename-connection":
		return runSourcesConfluenceRenameConnection(ctx, args[1:], stdout, stderr)
	case "revoke-connection":
		return runSourcesConfluenceRevokeConnection(ctx, args[1:], stdout, stderr)
	case "delete-connection":
		return runSourcesConfluenceDeleteConnection(ctx, args[1:], stdout, stderr)
	case "sites":
		return runSourcesConfluenceSites(ctx, args[1:], stdout, stderr)
	case "spaces":
		return runSourcesConfluenceSpaces(ctx, args[1:], stdout, stderr)
	case "pages":
		return runSourcesConfluencePages(ctx, args[1:], stdout, stderr)
	case "children":
		return runSourcesConfluenceChildren(ctx, args[1:], stdout, stderr)
	case "search":
		return runSourcesConfluenceSearch(ctx, args[1:], stdout, stderr)
	case "preview":
		return runSourcesConfluencePreview(ctx, args[1:], stdout, stderr)
	case "snapshot":
		return runSourcesConfluenceSnapshot(ctx, args[1:], stdout, stderr)
	case "check-update":
		return runSourcesConfluenceCheckUpdate(ctx, args[1:], stdout, stderr)
	case "update-preview":
		return runSourcesConfluenceUpdatePreview(ctx, args[1:], stdout, stderr)
	case "update":
		return runSourcesConfluenceUpdate(ctx, args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown sources confluence command %q\n", args[0])
		printSourcesConfluenceUsage(stderr)
		return 2
	}
}

func runSourcesConfluenceOAuthURL(_ context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources confluence oauth-url", flag.ContinueOnError)
	fs.SetOutput(stderr)
	clientID := fs.String("client-id", "", "Atlassian OAuth 3LO client id")
	redirectURI := fs.String("redirect-uri", "", "Atlassian OAuth 3LO callback URL")
	authorizeURL := fs.String("authorize-url", "", "override Atlassian OAuth authorize URL")
	state := fs.String("state", "", "OAuth state; defaults to a generated value")
	allowOAuthOverrides := fs.Bool("unsafe-allow-oauth-overrides", false, "allow OAuth endpoint overrides for local test environments")
	jsonOut := fs.Bool("json", false, "write JSON")
	scopes := repeatedStringFlag{}
	fs.Var(&scopes, "scope", "OAuth scope; repeatable")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	return confluenceOAuthUnsupportedCLIError("sources confluence oauth-url", stderr)
	cfg, err := cliConfluenceOAuthConfig(confluenceconnector.OAuthConfig{
		ClientID:     *clientID,
		RedirectURI:  *redirectURI,
		Scopes:       []string(scopes),
		AuthorizeURL: *authorizeURL,
	}, *allowOAuthOverrides)
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence oauth-url", err)
		return cliErrorCode(err)
	}
	oauthClient, err := confluenceconnector.NewOAuthClient(cfg)
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence oauth-url", err)
		return cliErrorCode(err)
	}
	oauthState := strings.TrimSpace(*state)
	if oauthState == "" {
		oauthState = cliNewID("oauth")
	}
	authorizationURL, err := oauthClient.AuthorizationURL(confluenceconnector.OAuthAuthorizationRequest{State: oauthState})
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence oauth-url", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{"authorization_url": authorizationURL, "state": oauthState})
		return 0
	}
	fmt.Fprintf(stdout, "state=%s\n%s\n", oauthState, authorizationURL)
	return 0
}

func runSourcesConfluenceOAuthExchange(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources confluence oauth-exchange", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	connectionID := fs.String("connection", "", "connection id; defaults to a generated cnf_ id")
	name := fs.String("name", "Confluence", "connection display name")
	accountID := fs.String("account-id", "", "Atlassian account id metadata")
	accountName := fs.String("account-name", "", "Atlassian account display name metadata")
	clientID := fs.String("client-id", "", "Atlassian OAuth 3LO client id")
	clientSecret := fs.String("client-secret", "", "Atlassian OAuth 3LO client secret")
	redirectURI := fs.String("redirect-uri", "", "Atlassian OAuth 3LO callback URL")
	tokenURL := fs.String("token-url", "", "override Atlassian OAuth token URL")
	discoveryURL := fs.String("discovery-url", "", "override Atlassian accessible-resources base URL")
	code := fs.String("code", "", "authorization code returned by Atlassian")
	allowOAuthOverrides := fs.Bool("unsafe-allow-oauth-overrides", false, "allow OAuth endpoint overrides for local test environments")
	jsonOut := fs.Bool("json", false, "write JSON")
	scopes := repeatedStringFlag{}
	fs.Var(&scopes, "scope", "OAuth scope; repeatable")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	return confluenceOAuthUnsupportedCLIError("sources confluence oauth-exchange", stderr)
	discoveryBaseURL, err := cliConfluenceOAuthDiscoveryURL(*discoveryURL, *allowOAuthOverrides)
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence oauth-exchange", err)
		return cliErrorCode(err)
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	cfg, err := cliConfluenceOAuthConfig(confluenceconnector.OAuthConfig{
		ClientID:     *clientID,
		ClientSecret: *clientSecret,
		RedirectURI:  *redirectURI,
		Scopes:       []string(scopes),
		TokenURL:     *tokenURL,
	}, *allowOAuthOverrides)
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence oauth-exchange", err)
		return cliErrorCode(err)
	}
	oauthClient, err := confluenceconnector.NewOAuthClient(cfg)
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence oauth-exchange", err)
		return cliErrorCode(err)
	}
	token, err := oauthClient.ExchangeCode(ctx, confluenceconnector.OAuthCodeExchangeRequest{Code: *code})
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence oauth-exchange", err)
		return cliErrorCode(err)
	}
	lister, err := confluenceconnector.NewDiscoveryClient(
		confluenceconnector.WithDiscoveryBearerToken(token.AccessToken),
		confluenceconnector.WithDiscoveryBaseURL(discoveryBaseURL),
	)
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence oauth-exchange", err)
		return cliErrorCode(err)
	}
	siteResult, err := lister.ListConfluenceSites(ctx)
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence oauth-exchange", err)
		return cliErrorCode(err)
	}
	id := strings.TrimSpace(*connectionID)
	if id == "" {
		id = cliNewID("cnf")
	}
	connectionScopes := token.Scopes
	if len(connectionScopes) == 0 {
		connectionScopes = cfg.Scopes
	}
	connection, err := svc.UpsertConfluenceConnection(ctx, app.UpsertConfluenceConnectionRequest{
		ConnectionID:   id,
		DisplayName:    *name,
		AuthType:       app.ConfluenceAuthTypeOAuth,
		AccountID:      *accountID,
		AccountName:    *accountName,
		AccessToken:    token.AccessToken,
		RefreshToken:   token.RefreshToken,
		TokenExpiresAt: token.TokenExpiresAt,
		Scopes:         connectionScopes,
		Sites:          siteResult.Sites,
	})
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence oauth-exchange", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{"connection": connection, "sites": connection.Sites})
		return 0
	}
	fmt.Fprintf(stdout, "connected confluence %s auth=%s sites=%d\n", connection.ConnectionID, connection.AuthType, len(connection.Sites))
	return 0
}

func runSourcesConfluenceConnectToken(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources confluence connect-token", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	connectionID := fs.String("connection", "", "connection id; defaults to a generated cnf_ id")
	name := fs.String("name", "Confluence", "connection display name")
	authType := fs.String("auth-type", app.ConfluenceAuthTypeAPIToken, "auth type: api_token")
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
		normalizedAuthType = app.ConfluenceAuthTypeAPIToken
		token = strings.TrimSpace(*apiToken)
	}
	if normalizedAuthType == app.ConfluenceAuthTypeOAuth {
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
	connection, err := svc.UpsertConfluenceConnection(ctx, app.UpsertConfluenceConnectionRequest{
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

func runSourcesConfluenceConnections(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources confluence connections", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	jsonOut := fs.Bool("json", false, "write JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	connections, err := svc.ListConfluenceConnections(ctx)
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence connections", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{"connections": connections})
		return 0
	}
	for _, connection := range connections {
		fmt.Fprintf(stdout, "%s\t%s\t%s\tsites=%d\trevoked=%v\n",
			connection.ConnectionID, connection.AuthType, connection.DisplayName, len(connection.Sites), connection.Revoked)
	}
	return 0
}

func runSourcesConfluenceRenameConnection(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources confluence rename-connection", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	name := fs.String("name", "", "new display name")
	jsonOut := fs.Bool("json", false, "write JSON")
	positionals, parseArgs := leadingPositionals(args, 1)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 1 {
		fmt.Fprintln(stderr, "usage: plasma sources confluence rename-connection <connection_id> --name <display_name>")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	connection, err := svc.RenameConfluenceConnection(ctx, positionals[0], *name)
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence rename-connection", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{"connection": connection})
		return 0
	}
	fmt.Fprintf(stdout, "renamed confluence connection %s name=%q\n", connection.ConnectionID, connection.DisplayName)
	return 0
}

func runSourcesConfluenceRevokeConnection(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources confluence revoke-connection", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	jsonOut := fs.Bool("json", false, "write JSON")
	positionals, parseArgs := leadingPositionals(args, 1)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 1 {
		fmt.Fprintln(stderr, "usage: plasma sources confluence revoke-connection <connection_id>")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	connection, err := svc.RevokeConfluenceConnection(ctx, positionals[0])
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence revoke-connection", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{"connection": connection, "revoked": true})
		return 0
	}
	fmt.Fprintf(stdout, "revoked confluence connection %s\n", connection.ConnectionID)
	return 0
}

func runSourcesConfluenceDeleteConnection(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources confluence delete-connection", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	jsonOut := fs.Bool("json", false, "write JSON")
	positionals, parseArgs := leadingPositionals(args, 1)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 1 {
		fmt.Fprintln(stderr, "usage: plasma sources confluence delete-connection <connection_id>")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	if err := svc.DeleteConfluenceConnection(ctx, positionals[0]); err != nil {
		writeSourceCommandError(stderr, "sources confluence delete-connection", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{"connection_id": positionals[0], "deleted": true})
		return 0
	}
	fmt.Fprintf(stdout, "deleted confluence connection %s\n", positionals[0])
	return 0
}

func runSourcesConfluenceSites(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources confluence sites", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	connectionID := fs.String("connection", "", "confluence connection id")
	refresh := fs.Bool("refresh", false, "refresh sites from Atlassian accessible-resources")
	discoveryURL := fs.String("discovery-url", "", "override Atlassian discovery base URL")
	allowOAuthOverrides := fs.Bool("unsafe-allow-oauth-overrides", false, "allow OAuth endpoint overrides for local test environments")
	jsonOut := fs.Bool("json", false, "write JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	connection, err := svc.GetConfluenceConnection(ctx, *connectionID)
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence sites", err)
		return cliErrorCode(err)
	}
	if *refresh {
		lister, err := cliConfluenceDiscoveryClient(connection, *discoveryURL, *allowOAuthOverrides)
		if err != nil {
			writeSourceCommandError(stderr, "sources confluence sites", err)
			return cliErrorCode(err)
		}
		connection, err = svc.RefreshConfluenceConnectionSites(ctx, connection.ConnectionID, lister)
		if err != nil {
			writeSourceCommandError(stderr, "sources confluence sites", err)
			return cliErrorCode(err)
		}
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{"connection": connection, "sites": connection.Sites})
		return 0
	}
	for _, site := range connection.Sites {
		fmt.Fprintf(stdout, "%s\t%s\t%s\n", site.CloudID, site.Name, site.URL)
	}
	return 0
}

func runSourcesConfluenceSpaces(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources confluence spaces", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	connectionID := fs.String("connection", "", "confluence connection id")
	cloudID := fs.String("cloud-id", "", "Atlassian cloud id")
	limit := fs.Int("limit", 10, "maximum spaces")
	cursor := fs.String("cursor", "", "Confluence cursor")
	apiBaseURL := fs.String("api-base-url", "", "override Confluence API base URL")
	siteURL := fs.String("site-url", "", "override Confluence site URL")
	allowOAuthOverrides := fs.Bool("unsafe-allow-oauth-overrides", false, "allow OAuth endpoint overrides for local test environments")
	jsonOut := fs.Bool("json", false, "write JSON")
	positionals, parseArgs := leadingPositionals(args, 1)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 1 {
		fmt.Fprintln(stderr, "usage: plasma sources confluence spaces <mission_id> --connection <id> --cloud-id <cloud_id>")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	connector, err := cliConfluenceBrowserConnector(ctx, svc, *connectionID, *cloudID, *apiBaseURL, *siteURL, *allowOAuthOverrides)
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence spaces", err)
		return cliErrorCode(err)
	}
	result, err := svc.ListConfluenceSpaces(ctx, connector, app.ConfluenceSpaceListRequest{MissionID: positionals[0], CloudID: *cloudID, Limit: *limit, Cursor: *cursor})
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence spaces", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, result)
		return 0
	}
	for _, space := range result.Spaces {
		fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\n", space.SpaceID, space.SpaceKey, space.Name, space.WebURL)
	}
	if result.NextCursor != "" {
		fmt.Fprintf(stdout, "next_cursor=%s\n", result.NextCursor)
	}
	return 0
}

func runSourcesConfluencePages(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources confluence pages", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	connectionID := fs.String("connection", "", "confluence connection id")
	cloudID := fs.String("cloud-id", "", "Atlassian cloud id")
	spaceID := fs.String("space-id", "", "Confluence space id")
	limit := fs.Int("limit", 10, "maximum pages")
	cursor := fs.String("cursor", "", "Confluence cursor")
	apiBaseURL := fs.String("api-base-url", "", "override Confluence API base URL")
	siteURL := fs.String("site-url", "", "override Confluence site URL")
	allowOAuthOverrides := fs.Bool("unsafe-allow-oauth-overrides", false, "allow OAuth endpoint overrides for local test environments")
	jsonOut := fs.Bool("json", false, "write JSON")
	positionals, parseArgs := leadingPositionals(args, 1)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 1 {
		fmt.Fprintln(stderr, "usage: plasma sources confluence pages <mission_id> --connection <id> --cloud-id <cloud_id> --space-id <space_id>")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	connector, err := cliConfluenceBrowserConnector(ctx, svc, *connectionID, *cloudID, *apiBaseURL, *siteURL, *allowOAuthOverrides)
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence pages", err)
		return cliErrorCode(err)
	}
	result, err := svc.ListConfluenceSpacePages(ctx, connector, app.ConfluenceSpacePagesRequest{MissionID: positionals[0], CloudID: *cloudID, SpaceID: *spaceID, Limit: *limit, Cursor: *cursor})
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence pages", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, result)
		return 0
	}
	writeConfluencePages(stdout, result)
	return 0
}

func runSourcesConfluenceChildren(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources confluence children", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	connectionID := fs.String("connection", "", "confluence connection id")
	cloudID := fs.String("cloud-id", "", "Atlassian cloud id")
	pageID := fs.String("page-id", "", "Confluence parent page id")
	limit := fs.Int("limit", 10, "maximum pages")
	cursor := fs.String("cursor", "", "Confluence cursor")
	apiBaseURL := fs.String("api-base-url", "", "override Confluence API base URL")
	siteURL := fs.String("site-url", "", "override Confluence site URL")
	allowOAuthOverrides := fs.Bool("unsafe-allow-oauth-overrides", false, "allow OAuth endpoint overrides for local test environments")
	jsonOut := fs.Bool("json", false, "write JSON")
	positionals, parseArgs := leadingPositionals(args, 1)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 1 {
		fmt.Fprintln(stderr, "usage: plasma sources confluence children <mission_id> --connection <id> --cloud-id <cloud_id> --page-id <page_id>")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	connector, err := cliConfluenceBrowserConnector(ctx, svc, *connectionID, *cloudID, *apiBaseURL, *siteURL, *allowOAuthOverrides)
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence children", err)
		return cliErrorCode(err)
	}
	result, err := svc.ListConfluencePageChildren(ctx, connector, app.ConfluencePageChildrenRequest{MissionID: positionals[0], CloudID: *cloudID, PageID: *pageID, Limit: *limit, Cursor: *cursor})
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence children", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, result)
		return 0
	}
	writeConfluencePages(stdout, result)
	return 0
}

func writeConfluencePages(stdout io.Writer, result app.ConfluencePageListResult) {
	for _, page := range result.Pages {
		fmt.Fprintf(stdout, "%s\tv%d\t%s\t%s\n", page.PageID, page.Version, page.Title, page.WebURL)
	}
	if result.NextCursor != "" {
		fmt.Fprintf(stdout, "next_cursor=%s\n", result.NextCursor)
	}
}

func runSourcesConfluenceSearch(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources confluence search", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	connectionID := fs.String("connection", "", "confluence connection id")
	cloudID := fs.String("cloud-id", "", "Atlassian cloud id")
	query := fs.String("query", "", "Confluence search text")
	spaceKey := fs.String("space-key", "", "Confluence space key")
	limit := fs.Int("limit", 10, "maximum candidates")
	cursor := fs.String("cursor", "", "Confluence cursor")
	apiBaseURL := fs.String("api-base-url", "", "override Confluence API base URL")
	siteURL := fs.String("site-url", "", "override Confluence site URL")
	allowOAuthOverrides := fs.Bool("unsafe-allow-oauth-overrides", false, "allow OAuth endpoint overrides for local test environments")
	jsonOut := fs.Bool("json", false, "write JSON")
	positionals, parseArgs := leadingPositionals(args, 1)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 1 {
		fmt.Fprintln(stderr, "usage: plasma sources confluence search <mission_id> --connection <id> --cloud-id <cloud_id> --query <text>")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	connector, err := cliConfluenceClient(ctx, svc, *connectionID, *cloudID, *apiBaseURL, *siteURL, *allowOAuthOverrides)
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence search", err)
		return cliErrorCode(err)
	}
	result, err := svc.SearchConfluenceSources(ctx, connector, app.ConfluenceSourceSearchRequest{
		MissionID: positionals[0],
		CloudID:   *cloudID,
		Query:     *query,
		Limit:     *limit,
		Cursor:    *cursor,
		SpaceKey:  *spaceKey,
	})
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence search", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, result)
		return 0
	}
	for _, candidate := range result.Candidates {
		fmt.Fprintf(stdout, "%s\tv%d\t%s\t%s\n", candidate.Connector.ExternalSourceID, candidate.Version, candidate.Title, candidate.SourceURI)
	}
	if result.NextCursor != "" {
		fmt.Fprintf(stdout, "next_cursor=%s\n", result.NextCursor)
	}
	return 0
}

func runSourcesConfluencePreview(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources confluence preview", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	connectionID := fs.String("connection", "", "confluence connection id")
	cloudID := fs.String("cloud-id", "", "Atlassian cloud id")
	pageID := fs.String("page-id", "", "Confluence page id")
	version := fs.Int("version", 0, "expected Confluence page version")
	maxBodyBytes := fs.Int64("max-body-bytes", app.DefaultConfluenceMaxBodyBytes, "maximum storage body bytes")
	apiBaseURL := fs.String("api-base-url", "", "override Confluence API base URL")
	siteURL := fs.String("site-url", "", "override Confluence site URL")
	allowOAuthOverrides := fs.Bool("unsafe-allow-oauth-overrides", false, "allow OAuth endpoint overrides for local test environments")
	jsonOut := fs.Bool("json", false, "write JSON")
	positionals, parseArgs := leadingPositionals(args, 1)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 1 {
		fmt.Fprintln(stderr, "usage: plasma sources confluence preview <mission_id> --connection <id> --cloud-id <cloud_id> --page-id <page_id>")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	connector, err := cliConfluenceClient(ctx, svc, *connectionID, *cloudID, *apiBaseURL, *siteURL, *allowOAuthOverrides)
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence preview", err)
		return cliErrorCode(err)
	}
	result, err := svc.PreviewConfluenceSource(ctx, connector, app.ConfluenceSourcePreviewRequest{
		MissionID:       positionals[0],
		CloudID:         *cloudID,
		PageID:          *pageID,
		ExpectedVersion: *version,
		MaxBodyBytes:    *maxBodyBytes,
	})
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence preview", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, result)
		return 0
	}
	fmt.Fprintf(stdout, "candidate confluence page=%s version=%d too_large=%v bytes=%d/%d title=%q\n",
		result.Page.PageID, result.Page.Version, result.FullBodyTooLarge, result.BodyBytes, result.MaxBodyBytes, result.Page.Title)
	for _, option := range result.RangeOptions {
		fmt.Fprintf(stdout, "range\t%s\t%d\t%d\n", option.ContentID, option.Start, option.End)
	}
	return 0
}

func runSourcesConfluenceSnapshot(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources confluence snapshot", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	connectionID := fs.String("connection", "", "confluence connection id")
	cloudID := fs.String("cloud-id", "", "Atlassian cloud id")
	pageID := fs.String("page-id", "", "Confluence page id")
	version := fs.Int("version", 0, "expected Confluence page version")
	reason := fs.String("reason", "", "snapshot reason")
	maxBodyBytes := fs.Int64("max-body-bytes", app.DefaultConfluenceMaxBodyBytes, "maximum storage body bytes")
	rangeContentID := fs.String("range-content-id", "", "range content id, normally plain_text")
	rangeStart := fs.Int("range-start", 0, "range start rune offset")
	rangeEnd := fs.Int("range-end", 0, "range end rune offset")
	apiBaseURL := fs.String("api-base-url", "", "override Confluence API base URL")
	siteURL := fs.String("site-url", "", "override Confluence site URL")
	allowOAuthOverrides := fs.Bool("unsafe-allow-oauth-overrides", false, "allow OAuth endpoint overrides for local test environments")
	jsonOut := fs.Bool("json", false, "write JSON")
	positionals, parseArgs := leadingPositionals(args, 1)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 1 {
		fmt.Fprintln(stderr, "usage: plasma sources confluence snapshot <mission_id> --connection <id> --cloud-id <cloud_id> --page-id <page_id> --version <version>")
		return 2
	}
	if *version <= 0 {
		fmt.Fprintln(stderr, "usage: plasma sources confluence snapshot <mission_id> --connection <id> --cloud-id <cloud_id> --page-id <page_id> --version <version>")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	connector, err := cliConfluenceClient(ctx, svc, *connectionID, *cloudID, *apiBaseURL, *siteURL, *allowOAuthOverrides)
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence snapshot", err)
		return cliErrorCode(err)
	}
	result, err := svc.SnapshotConfluenceSourceWithEvent(ctx, connector, app.SnapshotConfluenceSourceWithEventRequest{
		Snapshot: app.SnapshotConfluenceSourceRequest{
			MissionID:       positionals[0],
			ArtifactID:      cliNewID("art"),
			SnapshotID:      cliNewID("src"),
			CloudID:         *cloudID,
			PageID:          *pageID,
			ExpectedVersion: *version,
			MaxBodyBytes:    *maxBodyBytes,
			Range: app.ConfluenceRangeSelection{
				ContentID: *rangeContentID,
				Start:     *rangeStart,
				End:       *rangeEnd,
			},
			Reason: *reason,
		},
		EventID:  cliNewID("evt"),
		Producer: app.Producer{Type: "user", ID: "plasma-cli"},
	})
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence snapshot", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, result)
		return 0
	}
	fmt.Fprintf(stdout, "snapshotted confluence source %s artifact=%s event=%s\n",
		result.Snapshot.SnapshotID, result.Artifact.ArtifactID, cliLedgerEventID(&result.Event))
	return 0
}

func runSourcesUpload(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources upload", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	title := fs.String("title", "", "source title")
	jsonOut := fs.Bool("json", false, "write JSON")
	positionals, parseArgs := leadingPositionals(args, 2)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 2 {
		fmt.Fprintln(stderr, "usage: plasma sources upload <mission_id> <path> [--title title] [--json]")
		return 2
	}
	path := strings.TrimSpace(positionals[1])
	info, err := os.Stat(path)
	if err != nil {
		fmt.Fprintf(stderr, "stat upload file: %v\n", err)
		return 1
	}
	if info.IsDir() {
		fmt.Fprintln(stderr, "upload path must be a file")
		return 2
	}
	if info.Size() > app.UploadedFileMaxBytes {
		fmt.Fprintln(stderr, "upload file exceeds 100 MiB limit")
		return 2
	}
	content, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "read upload file: %v\n", err)
		return 1
	}
	if int64(len(content)) > app.UploadedFileMaxBytes {
		fmt.Fprintln(stderr, "upload file exceeds 100 MiB limit")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	result, err := svc.CreateUploadedFileSourceWithEvent(ctx, app.CreateUploadedFileSourceRequest{
		MissionID:        positionals[0],
		ArtifactID:       cliNewID("art"),
		SnapshotID:       cliNewID("src"),
		EventID:          cliNewID("evt"),
		Title:            *title,
		OriginalFilename: filepath.Base(path),
		Content:          content,
		Producer:         app.Producer{Type: "user", ID: "plasma-cli"},
	})
	if err != nil {
		writeSourceCommandError(stderr, "sources upload", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{
			"artifact": cliRawArtifactResponse(result.Artifact),
			"snapshot": result.Snapshot,
			"event":    result.Event,
			"existing": result.Existing,
		})
		return 0
	}
	status := "uploaded"
	if result.Existing {
		status = "existing"
	}
	fmt.Fprintf(stdout, "%s file source %s artifact=%s sha256=%s event=%s\n",
		status, result.Snapshot.SnapshotID, result.Artifact.ArtifactID, result.Artifact.SHA256, cliLedgerEventID(&result.Event))
	return 0
}

func runSourcesConfluenceCheckUpdate(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources confluence check-update", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	connectionID := fs.String("connection", "", "confluence connection id")
	apiBaseURL := fs.String("api-base-url", "", "override Confluence API base URL")
	siteURL := fs.String("site-url", "", "override Confluence site URL")
	allowOAuthOverrides := fs.Bool("unsafe-allow-oauth-overrides", false, "allow OAuth endpoint overrides for local test environments")
	jsonOut := fs.Bool("json", false, "write JSON")
	positionals, parseArgs := leadingPositionals(args, 2)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 2 {
		fmt.Fprintln(stderr, "usage: plasma sources confluence check-update <mission_id> <source_id> --connection <id>")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	cloudID, err := cliConfluenceCloudID(ctx, svc, positionals[1])
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence check-update", err)
		return cliErrorCode(err)
	}
	connector, err := cliConfluenceClient(ctx, svc, *connectionID, cloudID, *apiBaseURL, *siteURL, *allowOAuthOverrides)
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence check-update", err)
		return cliErrorCode(err)
	}
	result, err := svc.CheckConfluenceSourceUpdateWithEvent(ctx, connector, app.CheckConfluenceSourceUpdateRequest{
		MissionID:  positionals[0],
		SnapshotID: positionals[1],
		EventID:    cliNewID("evt"),
		Producer:   app.Producer{Type: "user", ID: "plasma-cli"},
	})
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence check-update", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, result)
		return 0
	}
	fmt.Fprintf(stdout, "confluence update_available=%v source=%s old_version=%d new_version=%d event=%s\n",
		result.UpdateAvailable, result.Snapshot.SnapshotID, result.CurrentVersion, result.LatestVersion, cliLedgerEventID(&result.Event))
	return 0
}

func runSourcesConfluenceUpdatePreview(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources confluence update-preview", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	connectionID := fs.String("connection", "", "confluence connection id")
	version := fs.Int("version", 0, "expected new Confluence page version")
	maxBodyBytes := fs.Int64("max-body-bytes", app.DefaultConfluenceMaxBodyBytes, "maximum storage body bytes")
	apiBaseURL := fs.String("api-base-url", "", "override Confluence API base URL")
	siteURL := fs.String("site-url", "", "override Confluence site URL")
	allowOAuthOverrides := fs.Bool("unsafe-allow-oauth-overrides", false, "allow OAuth endpoint overrides for local test environments")
	jsonOut := fs.Bool("json", false, "write JSON")
	positionals, parseArgs := leadingPositionals(args, 2)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 2 {
		fmt.Fprintln(stderr, "usage: plasma sources confluence update-preview <mission_id> <source_id> --connection <id>")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	cloudID, err := cliConfluenceCloudID(ctx, svc, positionals[1])
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence update-preview", err)
		return cliErrorCode(err)
	}
	connector, err := cliConfluenceClient(ctx, svc, *connectionID, cloudID, *apiBaseURL, *siteURL, *allowOAuthOverrides)
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence update-preview", err)
		return cliErrorCode(err)
	}
	result, err := svc.PreviewConfluenceSourceUpdate(ctx, connector, app.ConfluenceUpdatePreviewRequest{
		MissionID:       positionals[0],
		SnapshotID:      positionals[1],
		ExpectedVersion: *version,
		MaxBodyBytes:    *maxBodyBytes,
	})
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence update-preview", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, result)
		return 0
	}
	fmt.Fprintf(stdout, "confluence update_preview source=%s old_version=%d new_version=%d available=%v requires_range_reselect=%v\n",
		result.Snapshot.SnapshotID, result.OldPage.Version, result.NewPage.Version, result.UpdateAvailable, result.RequiresRangeReselect)
	for _, option := range result.RangeOptions {
		fmt.Fprintf(stdout, "range\t%s\t%d\t%d\n", option.ContentID, option.Start, option.End)
	}
	return 0
}

func runSourcesConfluenceUpdate(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources confluence update", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	connectionID := fs.String("connection", "", "confluence connection id")
	version := fs.Int("version", 0, "expected new Confluence page version")
	reason := fs.String("reason", "Confluence source update", "update reason")
	maxBodyBytes := fs.Int64("max-body-bytes", app.DefaultConfluenceMaxBodyBytes, "maximum storage body bytes")
	rangeContentID := fs.String("range-content-id", "", "range content id, normally plain_text")
	rangeStart := fs.Int("range-start", 0, "range start rune offset")
	rangeEnd := fs.Int("range-end", 0, "range end rune offset")
	apiBaseURL := fs.String("api-base-url", "", "override Confluence API base URL")
	siteURL := fs.String("site-url", "", "override Confluence site URL")
	allowOAuthOverrides := fs.Bool("unsafe-allow-oauth-overrides", false, "allow OAuth endpoint overrides for local test environments")
	jsonOut := fs.Bool("json", false, "write JSON")
	positionals, parseArgs := leadingPositionals(args, 2)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 2 {
		fmt.Fprintln(stderr, "usage: plasma sources confluence update <mission_id> <source_id> --connection <id> --version <new_version>")
		return 2
	}
	if *version <= 0 {
		fmt.Fprintln(stderr, "usage: plasma sources confluence update <mission_id> <source_id> --connection <id> --version <new_version>")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	cloudID, err := cliConfluenceCloudID(ctx, svc, positionals[1])
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence update", err)
		return cliErrorCode(err)
	}
	connector, err := cliConfluenceClient(ctx, svc, *connectionID, cloudID, *apiBaseURL, *siteURL, *allowOAuthOverrides)
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence update", err)
		return cliErrorCode(err)
	}
	result, err := svc.UpdateConfluenceSourceWithEvent(ctx, connector, app.UpdateConfluenceSourceRequest{
		MissionID:          positionals[0],
		PreviousSnapshotID: positionals[1],
		ArtifactID:         cliNewID("art"),
		SnapshotID:         cliNewID("src"),
		ExpectedVersion:    *version,
		MaxBodyBytes:       *maxBodyBytes,
		Range: app.ConfluenceRangeSelection{
			ContentID: *rangeContentID,
			Start:     *rangeStart,
			End:       *rangeEnd,
		},
		Reason:          *reason,
		SnapshotEventID: cliNewID("evt"),
		UpdateEventID:   cliNewID("evt"),
		Producer:        app.Producer{Type: "user", ID: "plasma-cli"},
	})
	if err != nil {
		writeSourceCommandError(stderr, "sources confluence update", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, result)
		return 0
	}
	fmt.Fprintf(stdout, "updated confluence source old=%s new=%s event=%s\n",
		result.PreviousSnapshot.SnapshotID, result.Snapshot.SnapshotID, cliLedgerEventID(&result.UpdateEvent))
	return 0
}

func runSourcesRoots(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources roots", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	jsonOut := fs.Bool("json", false, "write JSON")
	localRoots := repeatedStringFlag{}
	fs.Var(&localRoots, "local-source-root", "allowlisted local source root root_id=path; repeatable")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath, []string(localRoots)...)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	roots, err := svc.ListLocalPathRoots(ctx)
	if err != nil {
		writeSourceCommandError(stderr, "sources roots", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{"roots": roots})
		return 0
	}
	for _, root := range roots {
		fmt.Fprintf(stdout, "%s\talias=%s\n", root.RootID, root.Alias)
	}
	return 0
}

func runSourcesTree(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources tree", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	rootID := fs.String("root", "", "local source root id")
	relativePath := fs.String("path", ".", "root-relative path")
	depth := fs.Int("depth", 1, "tree depth")
	limit := fs.Int("limit", 0, "maximum entries")
	jsonOut := fs.Bool("json", false, "write JSON")
	localRoots := repeatedStringFlag{}
	fs.Var(&localRoots, "local-source-root", "allowlisted local source root root_id=path; repeatable")
	positionals, parseArgs := leadingPositionals(args, 1)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 1 {
		fmt.Fprintln(stderr, "usage: plasma sources tree <mission_id> --root <root_id> --path <relative_path>")
		return 2
	}
	if err := validateCLIRelativePath(*relativePath); err != nil {
		writeSourceCommandError(stderr, "sources tree", err)
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath, []string(localRoots)...)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	tree, err := svc.BrowseLocalPathRoot(ctx, app.BrowseLocalPathRootRequest{
		RootID:       *rootID,
		RelativePath: *relativePath,
		Depth:        *depth,
		Limit:        *limit,
	})
	if err != nil {
		writeSourceCommandError(stderr, "sources tree", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{"mission_id": positionals[0], "tree": tree})
		return 0
	}
	fmt.Fprintf(stdout, "root=%s alias=%s path=%s truncated=%v\n", tree.RootID, tree.RootAlias, tree.RelativePath, tree.Truncated)
	for _, entry := range tree.Entries {
		status := entry.PathKind
		if entry.Denied {
			status = "denied:" + entry.Reason
		}
		fmt.Fprintf(stdout, "%s\t%s\t%s\n", entry.RelativePath, entry.Name, status)
	}
	return 0
}

func runSourcesAttachLocal(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources attach-local", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	rootID := fs.String("root", "", "local source root id")
	relativePath := fs.String("path", "", "root-relative path")
	title := fs.String("title", "", "source title")
	restore := fs.Bool("restore", false, "restore an exact removed local path source")
	jsonOut := fs.Bool("json", false, "write JSON")
	localRoots := repeatedStringFlag{}
	fs.Var(&localRoots, "local-source-root", "allowlisted local source root root_id=path; repeatable")
	positionals, parseArgs := leadingPositionals(args, 1)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 1 {
		fmt.Fprintln(stderr, "usage: plasma sources attach-local <mission_id> --root <root_id> --path <relative_path>")
		return 2
	}
	if err := validateCLIRelativePath(*relativePath); err != nil {
		writeSourceCommandError(stderr, "sources attach-local", err)
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath, []string(localRoots)...)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	result, err := svc.AttachLocalPathSource(ctx, app.AttachLocalPathSourceRequest{
		MissionID:    positionals[0],
		RootID:       *rootID,
		RelativePath: *relativePath,
		Title:        *title,
		Restore:      *restore,
		Producer:     app.Producer{Type: "user", ID: "plasma-cli"},
	})
	if err != nil {
		writeSourceCommandError(stderr, "sources attach-local", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{
			"snapshot":         result.Snapshot,
			"event":            result.Event,
			"event_id":         cliLedgerEventID(result.Event),
			"existing":         result.Existing,
			"restored":         result.Restored,
			"restore_required": result.RestoreRequired,
		})
		return 0
	}
	status := "attached"
	if result.Restored {
		status = "restored"
	} else if result.Existing {
		status = "existing"
	}
	fmt.Fprintf(stdout, "%s source %s %s root=%s path=%s event=%s\n",
		status, result.Snapshot.SnapshotID, result.Snapshot.Access.RetrievalPolicy, *rootID, cliSourceRelativePath(result.Snapshot), cliLedgerEventID(result.Event))
	return 0
}

func runSourcesList(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	includeRemoved := fs.Bool("include-removed", false, "include soft-removed sources")
	includeSuperseded := fs.Bool("include-superseded", false, "include superseded source snapshots")
	jsonOut := fs.Bool("json", false, "write JSON")
	positionals, parseArgs := leadingPositionals(args, 1)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 1 {
		fmt.Fprintln(stderr, "usage: plasma sources list <mission_id> [--include-removed] [--include-superseded]")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	sources, err := svc.ListSourceSnapshotsWithState(ctx, app.ListSourceSnapshotsRequest{
		MissionID:         positionals[0],
		IncludeRemoved:    *includeRemoved,
		IncludeSuperseded: *includeSuperseded,
	})
	if err != nil {
		writeSourceCommandError(stderr, "sources list", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{"sources": sources})
		return 0
	}
	for _, source := range sources {
		fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\t%s\n",
			source.SnapshotID, cliSourceState(source), source.Access.RetrievalPolicy, source.Connector.ConnectorType, cliSourceLocatorSummary(source))
	}
	return 0
}

func runSourcesShow(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources show", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	includeRemoved := fs.Bool("include-removed", false, "show soft-removed sources")
	jsonOut := fs.Bool("json", false, "write JSON")
	positionals, parseArgs := leadingPositionals(args, 2)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 2 {
		fmt.Fprintln(stderr, "usage: plasma sources show <mission_id> <source_id> [--include-removed]")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	source, err := svc.GetSourceSnapshot(ctx, positionals[1])
	if err != nil {
		writeSourceCommandError(stderr, "sources show", err)
		return cliErrorCode(err)
	}
	if source.MissionID != positionals[0] {
		writeSourceCommandError(stderr, "sources show", fmt.Errorf("%w: source belongs to another mission", app.ErrInvalidInput))
		return 2
	}
	if source.State.Removed && !*includeRemoved {
		writeSourceCommandError(stderr, "sources show", fmt.Errorf("%w: source is removed; pass --include-removed for audit visibility", app.ErrInvalidInput))
		return 2
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{"source": source})
		return 0
	}
	fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\t%s\n",
		source.SnapshotID, cliSourceState(source), source.Access.RetrievalPolicy, source.Connector.ConnectorType, cliSourceLocatorSummary(source))
	return 0
}

func runSourcesRead(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources read", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	artifactID := fs.String("artifact", "", "artifact id for snapshot sources")
	offset := fs.Int64("offset", 0, "read offset")
	maxBytes := fs.Int64("max-bytes", 0, "maximum bytes")
	depth := fs.Int("depth", 1, "directory tree depth for live directory sources")
	limit := fs.Int("limit", 0, "directory tree entry limit for live directory sources")
	jsonOut := fs.Bool("json", false, "write JSON")
	localRoots := repeatedStringFlag{}
	fs.Var(&localRoots, "local-source-root", "allowlisted local source root root_id=path; repeatable")
	positionals, parseArgs := leadingPositionals(args, 2)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 2 {
		fmt.Fprintln(stderr, "usage: plasma sources read <mission_id> <source_id> [--offset N --max-bytes N]")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath, []string(localRoots)...)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	snapshot, err := svc.GetSourceSnapshot(ctx, positionals[1])
	if err != nil {
		writeSourceCommandError(stderr, "sources read", err)
		return cliErrorCode(err)
	}
	if snapshot.MissionID != positionals[0] {
		writeSourceCommandError(stderr, "sources read", fmt.Errorf("%w: source belongs to another mission", app.ErrInvalidInput))
		return 2
	}
	if snapshot.State.Removed {
		writeSourceCommandError(stderr, "sources read", fmt.Errorf("%w: source is removed", app.ErrInvalidInput))
		return 2
	}
	if snapshot.Access.RetrievalPolicy == app.SourceRetrievalPolicyLiveReference && snapshot.Connector.ConnectorType == app.SourceConnectorTypeLocalPath {
		return runSourcesReadLive(ctx, svc, positionals[0], snapshot, *offset, *maxBytes, *depth, *limit, *jsonOut, stdout, stderr)
	}
	return runSourcesReadSnapshot(ctx, svc, positionals[0], snapshot, *artifactID, *offset, *maxBytes, *jsonOut, stdout, stderr)
}

func runSourcesReadLive(ctx context.Context, svc *app.Service, missionID string, snapshot app.SourceSnapshot, offset int64, maxBytes int64, depth int, limit int, jsonOut bool, stdout, stderr io.Writer) int {
	locator, err := cliLocalPathLocator(snapshot)
	if err != nil {
		writeSourceCommandError(stderr, "sources read", err)
		return cliErrorCode(err)
	}
	if locator.PathKind == "directory" {
		result, err := svc.TreeLocalPathSource(ctx, app.TreeLocalPathSourceRequest{
			MissionID:     missionID,
			SnapshotID:    snapshot.SnapshotID,
			Depth:         depth,
			Limit:         limit,
			Producer:      app.Producer{Type: "user", ID: "plasma-cli"},
			ToolSessionID: "plasma-cli",
		})
		if err != nil {
			writeSourceCommandError(stderr, "sources read", err)
			return cliErrorCode(err)
		}
		if jsonOut {
			writeCLIJSON(stdout, map[string]any{
				"snapshot":             result.Snapshot,
				"tree":                 result.Tree,
				"observation_metadata": result.Tree.Metadata,
				"observation_event":    result.ObservationEvent,
				"observation_event_id": cliLedgerEventID(result.ObservationEvent),
			})
			return 0
		}
		fmt.Fprintf(stdout, "observation_event=%s root=%s path=%s truncated=%v\n",
			cliLedgerEventID(result.ObservationEvent), result.Tree.RootID, result.Tree.RelativePath, result.Tree.Truncated)
		for _, entry := range result.Tree.Entries {
			fmt.Fprintf(stdout, "%s\t%s\n", entry.RelativePath, entry.PathKind)
		}
		return 0
	}
	result, err := svc.ReadLocalPathSource(ctx, app.ReadLocalPathSourceRequest{
		MissionID:     missionID,
		SnapshotID:    snapshot.SnapshotID,
		Offset:        offset,
		MaxBytes:      maxBytes,
		Producer:      app.Producer{Type: "user", ID: "plasma-cli"},
		ToolSessionID: "plasma-cli",
	})
	if err != nil {
		writeSourceCommandError(stderr, "sources read", err)
		return cliErrorCode(err)
	}
	if jsonOut {
		writeCLIJSON(stdout, map[string]any{
			"snapshot":             result.Snapshot,
			"content":              result.Read.Content,
			"observation_metadata": result.Read.Metadata,
			"observation_event":    result.ObservationEvent,
			"observation_event_id": cliLedgerEventID(result.ObservationEvent),
		})
		return 0
	}
	fmt.Fprint(stdout, result.Read.Content)
	if result.Read.Metadata.Truncated {
		fmt.Fprintf(stdout, "\n[next_offset=%d]\n", result.Read.Metadata.NextOffset)
	}
	fmt.Fprintf(stdout, "\nobservation_event=%s\n", cliLedgerEventID(result.ObservationEvent))
	return 0
}

func runSourcesReadSnapshot(ctx context.Context, svc *app.Service, missionID string, snapshot app.SourceSnapshot, artifactID string, offset int64, maxBytes int64, jsonOut bool, stdout, stderr io.Writer) int {
	artifactID = strings.TrimSpace(artifactID)
	if artifactID == "" && len(snapshot.ArtifactIDs) == 1 {
		artifactID = snapshot.ArtifactIDs[0]
	}
	if artifactID == "" {
		writeSourceCommandError(stderr, "sources read", fmt.Errorf("%w: artifact id is required for snapshot sources", app.ErrInvalidInput))
		return 2
	}
	artifact, err := svc.GetRawArtifact(ctx, artifactID)
	if err != nil {
		writeSourceCommandError(stderr, "sources read", err)
		return cliErrorCode(err)
	}
	if artifact.MissionID != missionID {
		writeSourceCommandError(stderr, "sources read", fmt.Errorf("%w: artifact belongs to another mission", app.ErrInvalidInput))
		return 2
	}
	read, err := cliReadSourceArtifact(artifact, offset, maxBytes)
	if err != nil {
		writeSourceCommandError(stderr, "sources read", err)
		return 2
	}
	if jsonOut {
		response := map[string]any{
			"snapshot":    snapshot,
			"artifact":    cliRawArtifactMetadata(artifact),
			"content":     read.Content,
			"offset":      read.Offset,
			"next_offset": read.NextOffset,
			"truncated":   read.Truncated,
		}
		if read.ExtractionType != "" {
			response["content_length"] = read.ContentLength
			response["extraction"] = map[string]any{"type": read.ExtractionType, "page_count": read.PageCount}
		}
		if read.MetadataOnly {
			response["content_length"] = artifact.ByteSize
			response["metadata_only"] = true
		}
		writeCLIJSON(stdout, response)
		return 0
	}
	if read.MetadataOnly {
		fmt.Fprintf(stdout, "metadata-only source artifact %s media_type=%s byte_size=%d sha256=%s\n",
			artifact.ArtifactID, artifact.MediaType, artifact.ByteSize, artifact.SHA256)
		return 0
	}
	fmt.Fprint(stdout, read.Content)
	if read.Truncated {
		fmt.Fprintf(stdout, "\n[next_offset=%d]\n", read.NextOffset)
	}
	return 0
}

func runSourcesGrep(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources grep", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	query := fs.String("query", "", "grep query")
	maxSnippets := fs.Int("max-snippets", 0, "maximum snippets")
	jsonOut := fs.Bool("json", false, "write JSON")
	localRoots := repeatedStringFlag{}
	fs.Var(&localRoots, "local-source-root", "allowlisted local source root root_id=path; repeatable")
	positionals, parseArgs := leadingPositionals(args, 2)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 2 || strings.TrimSpace(*query) == "" {
		fmt.Fprintln(stderr, "usage: plasma sources grep <mission_id> <source_id> --query <text>")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath, []string(localRoots)...)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	result, err := svc.GrepLocalPathSource(ctx, app.GrepLocalPathSourceRequest{
		MissionID:     positionals[0],
		SnapshotID:    positionals[1],
		Query:         *query,
		MaxSnippets:   *maxSnippets,
		Producer:      app.Producer{Type: "user", ID: "plasma-cli"},
		ToolSessionID: "plasma-cli",
	})
	if err != nil {
		writeSourceCommandError(stderr, "sources grep", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{
			"snapshot":             result.Snapshot,
			"grep":                 result.Grep,
			"observation_metadata": result.Grep.Metadata,
			"observation_event":    result.ObservationEvent,
			"observation_event_id": cliLedgerEventID(result.ObservationEvent),
		})
		return 0
	}
	for _, match := range result.Grep.Matches {
		fmt.Fprintf(stdout, "%s:%d:%d\t%s\n", match.RelativePath, match.Line, match.Column, match.Snippet)
	}
	fmt.Fprintf(stdout, "observation_event=%s truncated=%v\n", cliLedgerEventID(result.ObservationEvent), result.Grep.Truncated)
	return 0
}

func runSourcesRemove(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources remove", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	reason := fs.String("reason", "", "source removal reason")
	jsonOut := fs.Bool("json", false, "write JSON")
	positionals, parseArgs := leadingPositionals(args, 2)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 2 {
		fmt.Fprintln(stderr, "usage: plasma sources remove <mission_id> <source_id> --reason <reason>")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	result, err := svc.RemoveSource(ctx, app.RemoveSourceRequest{
		MissionID:  positionals[0],
		SnapshotID: positionals[1],
		Reason:     *reason,
		Producer:   app.Producer{Type: "user", ID: "plasma-cli"},
	})
	if err != nil {
		writeSourceCommandError(stderr, "sources remove", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{"snapshot": result.Snapshot, "event": result.Event, "event_id": cliLedgerEventID(result.Event), "idempotent": result.Idempotent})
		return 0
	}
	fmt.Fprintf(stdout, "removed source %s event=%s idempotent=%v\n", result.Snapshot.SnapshotID, cliLedgerEventID(result.Event), result.Idempotent)
	return 0
}

func runSourcesRestore(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sources restore", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "Plasma SQLite database path")
	jsonOut := fs.Bool("json", false, "write JSON")
	positionals, parseArgs := leadingPositionals(args, 2)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	positionals = append(positionals, fs.Args()...)
	if len(positionals) != 2 {
		fmt.Fprintln(stderr, "usage: plasma sources restore <mission_id> <source_id>")
		return 2
	}
	svc, closeStore, _, err := openCLIService(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open storage: %v\n", err)
		return 1
	}
	defer closeStore()
	result, err := svc.RestoreSource(ctx, app.RestoreSourceRequest{
		MissionID:  positionals[0],
		SnapshotID: positionals[1],
		Producer:   app.Producer{Type: "user", ID: "plasma-cli"},
	})
	if err != nil {
		writeSourceCommandError(stderr, "sources restore", err)
		return cliErrorCode(err)
	}
	if *jsonOut {
		writeCLIJSON(stdout, map[string]any{"snapshot": result.Snapshot, "event": result.Event, "event_id": cliLedgerEventID(result.Event), "idempotent": result.Idempotent})
		return 0
	}
	fmt.Fprintf(stdout, "restored source %s event=%s idempotent=%v\n", result.Snapshot.SnapshotID, cliLedgerEventID(result.Event), result.Idempotent)
	return 0
}

func newCLIService(store app.Store, cfg config.Config, extraRootSpecs []string) (*app.Service, error) {
	engine, err := localPathEngineFromSpecs(effectiveLocalSourceRootSpecs(cfg, extraRootSpecs))
	if err != nil {
		return nil, err
	}
	if engine == nil {
		return app.NewService(store), nil
	}
	return app.NewServiceWithLocalPathEngine(store, engine), nil
}

func effectiveLocalSourceRootSpecs(cfg config.Config, extraRootSpecs []string) []string {
	specs := make([]string, 0, len(cfg.LocalSourceRoots)+len(extraRootSpecs))
	seen := map[string]struct{}{}
	appendSpec := func(spec string) {
		spec = strings.TrimSpace(spec)
		if spec == "" {
			return
		}
		if _, exists := seen[spec]; exists {
			return
		}
		seen[spec] = struct{}{}
		specs = append(specs, spec)
	}
	for _, spec := range cfg.LocalSourceRoots {
		appendSpec(spec)
	}
	for _, spec := range extraRootSpecs {
		appendSpec(spec)
	}
	return specs
}

func localPathEngineFromSpecs(specs []string) (*localpath.Engine, error) {
	roots := make([]localpath.RootConfig, 0, len(specs))
	for _, spec := range specs {
		spec = strings.TrimSpace(spec)
		if spec == "" {
			continue
		}
		parts := strings.SplitN(spec, "=", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			return nil, fmt.Errorf("%w: local source root must be root_id=path", app.ErrInvalidInput)
		}
		rootID := strings.TrimSpace(parts[0])
		roots = append(roots, localpath.RootConfig{RootID: rootID, Alias: rootID, Path: strings.TrimSpace(parts[1])})
	}
	if len(roots) == 0 {
		return nil, nil
	}
	engine, err := localpath.New(localpath.Config{Roots: roots})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", app.ErrInvalidInput, err)
	}
	return engine, nil
}

func validateCLIRelativePath(relativePath string) error {
	trimmed := strings.TrimSpace(relativePath)
	if trimmed == "" {
		return nil
	}
	if strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, `\`) || strings.HasPrefix(trimmed, "~") {
		return fmt.Errorf("%w: relative_path must be root-relative", app.ErrInvalidInput)
	}
	firstSegment := trimmed
	if slash := strings.IndexAny(firstSegment, `/\`); slash >= 0 {
		firstSegment = firstSegment[:slash]
	}
	if strings.Contains(firstSegment, ":") {
		return fmt.Errorf("%w: relative_path must be root-relative", app.ErrInvalidInput)
	}
	if _, err := localpath.NormalizeRelativePath(trimmed); err != nil {
		return fmt.Errorf("%w: %v", app.ErrInvalidInput, err)
	}
	return nil
}

func cliLocalPathLocator(snapshot app.SourceSnapshot) (app.LocalPathLocator, error) {
	var locators []app.LocalPathLocator
	if err := json.Unmarshal(snapshot.Locators, &locators); err != nil {
		return app.LocalPathLocator{}, fmt.Errorf("%w: invalid local path locator", app.ErrInvalidInput)
	}
	for _, locator := range locators {
		if cliLocatorType(locator.LocatorType, locator.Kind) == app.SourceLocatorTypeLocalPath {
			locator.LocatorType = app.SourceLocatorTypeLocalPath
			locator.Kind = ""
			return locator, nil
		}
	}
	return app.LocalPathLocator{}, fmt.Errorf("%w: local path locator is required", app.ErrInvalidInput)
}

func cliLocatorType(locatorType, legacyKind string) string {
	if strings.TrimSpace(locatorType) != "" {
		return strings.TrimSpace(locatorType)
	}
	return strings.TrimSpace(legacyKind)
}

func cliSourceRelativePath(snapshot app.SourceSnapshot) string {
	locator, err := cliLocalPathLocator(snapshot)
	if err == nil {
		return locator.RelativePath
	}
	return ""
}

func cliSourceLocatorSummary(snapshot app.SourceSnapshot) string {
	if snapshot.Connector.ConnectorType == app.SourceConnectorTypeLocalPath {
		locator, err := cliLocalPathLocator(snapshot)
		if err == nil {
			return fmt.Sprintf("root=%s path=%s kind=%s", locator.RootID, locator.RelativePath, locator.PathKind)
		}
	}
	if strings.TrimSpace(snapshot.Connector.ExternalURI) != "" {
		return snapshot.Connector.ExternalURI
	}
	return snapshot.Connector.ExternalSourceID
}

func cliSourceState(snapshot app.SourceSnapshot) string {
	if snapshot.State.Removed || snapshot.State.State == app.SourceStateRemoved {
		return app.SourceStateRemoved
	}
	if snapshot.State.Superseded {
		return "superseded"
	}
	if strings.TrimSpace(snapshot.State.State) != "" {
		return strings.TrimSpace(snapshot.State.State)
	}
	return app.SourceStateActive
}

func rejectConfluenceOAuthEndpointOverride(value string, name string, allowed bool) error {
	if strings.TrimSpace(value) == "" || allowed {
		return nil
	}
	return fmt.Errorf("%w: %s override requires --unsafe-allow-oauth-overrides", app.ErrInvalidInput, name)
}

func cliConfluenceOAuthConfig(overrides confluenceconnector.OAuthConfig, allowEndpointOverrides bool) (confluenceconnector.OAuthConfig, error) {
	cfg, err := config.Load(config.Args{
		ConfluenceOAuthClientID:     overrides.ClientID,
		ConfluenceOAuthClientSecret: overrides.ClientSecret,
		ConfluenceOAuthRedirectURI:  overrides.RedirectURI,
		ConfluenceOAuthScopes:       overrides.Scopes,
		ConfluenceOAuthAuthorizeURL: overrides.AuthorizeURL,
		ConfluenceOAuthTokenURL:     overrides.TokenURL,
	})
	if err != nil {
		return confluenceconnector.OAuthConfig{}, err
	}
	if err := rejectConfluenceOAuthEndpointOverride(cfg.ConfluenceOAuthAuthorizeURL, "Confluence OAuth authorize URL", allowEndpointOverrides); err != nil {
		return confluenceconnector.OAuthConfig{}, err
	}
	if err := rejectConfluenceOAuthEndpointOverride(cfg.ConfluenceOAuthTokenURL, "Confluence OAuth token URL", allowEndpointOverrides); err != nil {
		return confluenceconnector.OAuthConfig{}, err
	}
	return confluenceconnector.OAuthConfig{
		ClientID:     strings.TrimSpace(cfg.ConfluenceOAuthClientID),
		ClientSecret: strings.TrimSpace(cfg.ConfluenceOAuthClientSecret),
		RedirectURI:  strings.TrimSpace(cfg.ConfluenceOAuthRedirectURI),
		Scopes:       cfg.ConfluenceOAuthScopes,
		AuthorizeURL: strings.TrimSpace(cfg.ConfluenceOAuthAuthorizeURL),
		TokenURL:     strings.TrimSpace(cfg.ConfluenceOAuthTokenURL),
	}, nil
}

func cliConfluenceOAuthDiscoveryURL(override string, allowEndpointOverrides bool) (string, error) {
	cfg, err := config.Load(config.Args{ConfluenceOAuthDiscoveryURL: override})
	if err != nil {
		return "", err
	}
	if err := rejectConfluenceOAuthEndpointOverride(cfg.ConfluenceOAuthDiscoveryURL, "Confluence OAuth discovery URL", allowEndpointOverrides); err != nil {
		return "", err
	}
	return strings.TrimSpace(cfg.ConfluenceOAuthDiscoveryURL), nil
}

func cliConfluenceDiscoveryClient(connection app.ConfluenceConnection, discoveryURL string, allowEndpointOverrides bool) (*confluenceconnector.DiscoveryClient, error) {
	if err := rejectConfluenceOAuthEndpointOverride(discoveryURL, "Confluence OAuth discovery URL", allowEndpointOverrides); err != nil {
		return nil, err
	}
	options := []confluenceconnector.DiscoveryOption{}
	if strings.TrimSpace(discoveryURL) != "" {
		options = append(options, confluenceconnector.WithDiscoveryBaseURL(discoveryURL))
	}
	switch connection.AuthType {
	case app.ConfluenceAuthTypeOAuth:
		options = append(options, confluenceconnector.WithDiscoveryBearerToken(connection.AccessToken))
	default:
		return nil, fmt.Errorf("%w: confluence site discovery requires an oauth connection", app.ErrInvalidInput)
	}
	return confluenceconnector.NewDiscoveryClient(options...)
}

func cliConfluenceClient(ctx context.Context, svc *app.Service, connectionID string, cloudID string, apiBaseURL string, siteURL string, allowEndpointOverrides bool) (app.ConfluenceSourceConnector, error) {
	connection, err := svc.GetConfluenceConnection(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	if connection.Revoked {
		return nil, app.NewConfluenceValidationError(
			app.ConfluenceErrorCodeRevoked,
			"Confluence 연결이 로컬에서 해제되었습니다. 다시 연결하거나 다른 연결을 선택하세요.",
		)
	}
	cloudID = strings.TrimSpace(cloudID)
	if cloudID == "" {
		return nil, fmt.Errorf("%w: confluence cloud id is required", app.ErrInvalidInput)
	}
	baseURL := strings.TrimSpace(apiBaseURL)
	effectiveSiteURL := strings.TrimSpace(siteURL)
	if effectiveSiteURL == "" {
		effectiveSiteURL = cliConfluenceCachedSiteURL(connection, cloudID)
	}
	options := []confluenceconnector.Option{}
	switch connection.AuthType {
	case app.ConfluenceAuthTypeOAuth:
		return nil, fmt.Errorf("%w: Confluence OAuth is disabled in Plasma 0.0. Use an API token connection instead.", app.ErrInvalidInput)
	case app.ConfluenceAuthTypeAPIToken:
		if effectiveSiteURL == "" {
			return nil, fmt.Errorf("%w: confluence site URL is required for api_token connections", app.ErrInvalidInput)
		}
		normalizedSiteURL, err := app.NormalizeConfluenceAPITokenSiteURL(effectiveSiteURL)
		if err != nil {
			return nil, err
		}
		effectiveSiteURL = normalizedSiteURL
		if baseURL == "" {
			baseURL = cliConfluenceWikiBaseURL(effectiveSiteURL)
		} else {
			if err := rejectConfluenceOAuthEndpointOverride(baseURL, "Confluence API token API base URL", allowEndpointOverrides); err != nil {
				return nil, err
			}
			if allowEndpointOverrides {
				baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
			} else {
				normalizedBaseURL, err := app.NormalizeConfluenceAPITokenAPIBaseURLForSite(baseURL, effectiveSiteURL)
				if err != nil {
					return nil, err
				}
				baseURL = normalizedBaseURL
			}
		}
		options = append(options, confluenceconnector.WithSiteURL(effectiveSiteURL))
		options = append(options, confluenceconnector.WithBasicAuth(connection.AccountName, connection.AccessToken))
	default:
		return nil, fmt.Errorf("%w: unsupported confluence auth type", app.ErrInvalidInput)
	}
	return confluenceconnector.NewClient(baseURL, cloudID, options...)
}

func cliConfluenceBrowserConnector(ctx context.Context, svc *app.Service, connectionID string, cloudID string, apiBaseURL string, siteURL string, allowEndpointOverrides bool) (app.ConfluenceBrowserConnector, error) {
	connector, err := cliConfluenceClient(ctx, svc, connectionID, cloudID, apiBaseURL, siteURL, allowEndpointOverrides)
	if err != nil {
		return nil, err
	}
	browser, ok := connector.(app.ConfluenceBrowserConnector)
	if !ok {
		return nil, fmt.Errorf("%w: confluence browser connector is required", app.ErrInvalidInput)
	}
	return browser, nil
}

func cliConfluenceCachedSiteURL(connection app.ConfluenceConnection, cloudID string) string {
	cloudID = strings.TrimSpace(cloudID)
	for _, site := range connection.Sites {
		if strings.TrimSpace(site.CloudID) == cloudID {
			return strings.TrimSpace(site.URL)
		}
	}
	return ""
}

func cliConfluenceWikiBaseURL(siteURL string) string {
	siteURL = strings.TrimRight(strings.TrimSpace(siteURL), "/")
	if strings.HasSuffix(siteURL, "/wiki") {
		return siteURL
	}
	return siteURL + "/wiki"
}

func cliConfluenceCloudID(ctx context.Context, svc *app.Service, snapshotID string) (string, error) {
	snapshot, err := svc.GetSourceSnapshot(ctx, snapshotID)
	if err != nil {
		return "", err
	}
	var locators []struct {
		CloudID string `json:"cloud_id"`
	}
	if len(snapshot.Locators) > 0 && json.Unmarshal(snapshot.Locators, &locators) == nil {
		for _, locator := range locators {
			if cloudID := strings.TrimSpace(locator.CloudID); cloudID != "" {
				return cloudID, nil
			}
		}
	}
	parts := strings.Split(strings.TrimSpace(snapshot.Connector.ExternalSourceID), ":")
	if len(parts) == 2 && strings.TrimSpace(parts[0]) != "" {
		return strings.TrimSpace(parts[0]), nil
	}
	return "", fmt.Errorf("%w: confluence cloud id is required", app.ErrInvalidInput)
}

type cliArtifactRead struct {
	Content        string
	Offset         int64
	NextOffset     int64
	ContentLength  int
	Truncated      bool
	ExtractionType string
	PageCount      int
	MetadataOnly   bool
}

func cliReadSourceArtifact(artifact app.RawArtifact, offset int64, maxBytes int64) (cliArtifactRead, error) {
	if pdftext.IsPDFMediaType(artifact.MediaType) || pdftext.IsPDFBytes(artifact.Content) {
		chunk, err := pdftext.ExtractChunk(artifact.Content, int(offset), int(maxBytes))
		if err != nil {
			return cliArtifactRead{}, fmt.Errorf("%w: PDF text extraction failed: %v", app.ErrInvalidInput, err)
		}
		return cliArtifactRead{
			Content:        chunk.Text,
			Offset:         int64(chunk.Offset),
			NextOffset:     int64(chunk.NextOffset),
			ContentLength:  chunk.ContentLength,
			Truncated:      chunk.Truncated,
			ExtractionType: "pdf_text",
			PageCount:      chunk.PageCount,
		}, nil
	}
	if app.UploadedArtifactReadKind(artifact) == "metadata" {
		return cliArtifactRead{ContentLength: int(artifact.ByteSize), MetadataOnly: true}, nil
	}
	return cliReadArtifactContent(artifact.Content, offset, maxBytes)
}

func cliReadArtifactContent(content []byte, offset int64, maxBytes int64) (cliArtifactRead, error) {
	if !utf8.Valid(content) {
		return cliArtifactRead{}, fmt.Errorf("%w: source artifact is not UTF-8 text", app.ErrInvalidInput)
	}
	if offset < 0 {
		return cliArtifactRead{}, fmt.Errorf("%w: offset must be non-negative", app.ErrInvalidInput)
	}
	if maxBytes < 0 {
		return cliArtifactRead{}, fmt.Errorf("%w: max-bytes must be non-negative", app.ErrInvalidInput)
	}
	if maxBytes == 0 {
		maxBytes = cliDefaultReadBytes
	}
	if maxBytes > cliMaxReadBytes {
		maxBytes = cliMaxReadBytes
	}
	if offset > int64(len(content)) {
		offset = int64(len(content))
	}
	end := offset + maxBytes
	if end > int64(len(content)) {
		end = int64(len(content))
	}
	truncated := end < int64(len(content))
	nextOffset := int64(0)
	if truncated {
		nextOffset = end
	}
	return cliArtifactRead{
		Content:    string(content[offset:end]),
		Offset:     offset,
		NextOffset: nextOffset,
		Truncated:  truncated,
	}, nil
}

func cliRawArtifactMetadata(artifact app.RawArtifact) map[string]any {
	return app.UploadedArtifactMetadata(artifact)
}

type cliRawArtifactAPIResponse struct {
	ArtifactID string
	MissionID  string
	MediaType  string
	ByteSize   int64
	SHA256     string
	StorageURI string
	Filename   string
	Producer   app.Producer
	CreatedAt  time.Time
}

func cliRawArtifactResponse(artifact app.RawArtifact) cliRawArtifactAPIResponse {
	return cliRawArtifactAPIResponse{
		ArtifactID: artifact.ArtifactID,
		MissionID:  artifact.MissionID,
		MediaType:  artifact.MediaType,
		ByteSize:   artifact.ByteSize,
		SHA256:     artifact.SHA256,
		StorageURI: artifact.StorageURI,
		Filename:   artifact.Filename,
		Producer:   artifact.Producer,
		CreatedAt:  artifact.CreatedAt,
	}
}

func cliLedgerEventID(event *app.LedgerEvent) string {
	if event == nil {
		return ""
	}
	return event.EventID
}

func writeSourceCommandError(stderr io.Writer, command string, err error) {
	fmt.Fprintf(stderr, "%s: %v\n", command, err)
}

func cliErrorCode(err error) int {
	if confluenceErr, ok := app.ConfluenceErrorDetails(err); ok {
		if confluenceErr.HTTPStatus >= 400 && confluenceErr.HTTPStatus < 500 {
			return 2
		}
	}
	if errors.Is(err, app.ErrInvalidInput) {
		return 2
	}
	return 1
}

func printSourcesUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: plasma sources <roots|tree|attach-local|upload|list|show|read|grep|remove|restore|confluence> [options]")
}

func printSourcesConfluenceUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: plasma sources confluence <connect-token|connections|rename-connection|revoke-connection|delete-connection|sites|spaces|pages|children|search|preview|snapshot|check-update|update-preview|update> [options]")
	fmt.Fprintln(w, "note: Confluence OAuth commands are disabled in Plasma 0.0; use API token connections.")
}
