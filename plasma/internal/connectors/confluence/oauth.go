package confluence

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

func NewOAuthClient(cfg OAuthConfig) (*OAuthClient, error) {
	clientID := strings.TrimSpace(cfg.ClientID)
	clientSecret := strings.TrimSpace(cfg.ClientSecret)
	redirectURI := strings.TrimSpace(cfg.RedirectURI)
	if clientID == "" {
		return nil, fmt.Errorf("%w: confluence OAuth client id is required", app.ErrInvalidInput)
	}
	if redirectURI != "" {
		if _, err := parseHTTPURL(redirectURI, "confluence OAuth redirect URI"); err != nil {
			return nil, err
		}
	}
	authorizeURL, err := parseHTTPURL(firstNonEmpty(cfg.AuthorizeURL, defaultOAuthAuthorizeURL), "confluence OAuth authorize URL")
	if err != nil {
		return nil, err
	}
	tokenURL, err := parseHTTPURL(firstNonEmpty(cfg.TokenURL, defaultOAuthTokenURL), "confluence OAuth token URL")
	if err != nil {
		return nil, err
	}
	if err := rejectSensitiveURLParts(tokenURL, "confluence OAuth token URL"); err != nil {
		return nil, err
	}
	scopes := normalizeScopes(cfg.Scopes)
	if len(scopes) == 0 {
		scopes = DefaultOAuthScopes()
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &OAuthClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
		scopes:       scopes,
		authorizeURL: authorizeURL,
		tokenURL:     tokenURL,
		httpClient:   httpClient,
	}, nil
}

func (client *OAuthClient) AuthorizationURL(req OAuthAuthorizationRequest) (string, error) {
	state := strings.TrimSpace(req.State)
	if state == "" {
		return "", fmt.Errorf("%w: confluence OAuth state is required", app.ErrInvalidInput)
	}
	redirectURI := firstNonEmpty(req.RedirectURI, client.redirectURI)
	if redirectURI == "" {
		return "", fmt.Errorf("%w: confluence OAuth redirect URI is required", app.ErrInvalidInput)
	}
	if _, err := parseHTTPURL(redirectURI, "confluence OAuth redirect URI"); err != nil {
		return "", err
	}
	scopes := normalizeScopes(req.Scopes)
	if len(scopes) == 0 {
		scopes = client.scopes
	}
	u := *client.authorizeURL
	query := u.Query()
	query.Set("audience", defaultOAuthAudience)
	query.Set("client_id", client.clientID)
	query.Set("scope", strings.Join(scopes, " "))
	query.Set("redirect_uri", redirectURI)
	query.Set("state", state)
	query.Set("response_type", "code")
	query.Set("prompt", defaultOAuthPrompt)
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func (client *OAuthClient) ExchangeCode(ctx context.Context, req OAuthCodeExchangeRequest) (OAuthTokenResult, error) {
	code := strings.TrimSpace(req.Code)
	if code == "" {
		return OAuthTokenResult{}, fmt.Errorf("%w: confluence OAuth code is required", app.ErrInvalidInput)
	}
	if client.clientSecret == "" {
		return OAuthTokenResult{}, fmt.Errorf("%w: confluence OAuth client secret is required", app.ErrInvalidInput)
	}
	redirectURI := firstNonEmpty(req.RedirectURI, client.redirectURI)
	if redirectURI == "" {
		return OAuthTokenResult{}, fmt.Errorf("%w: confluence OAuth redirect URI is required", app.ErrInvalidInput)
	}
	if _, err := parseHTTPURL(redirectURI, "confluence OAuth redirect URI"); err != nil {
		return OAuthTokenResult{}, err
	}
	return client.exchangeToken(ctx, map[string]string{
		"grant_type":    "authorization_code",
		"client_id":     client.clientID,
		"client_secret": client.clientSecret,
		"code":          code,
		"redirect_uri":  redirectURI,
	})
}

func (client *OAuthClient) RefreshAccessToken(ctx context.Context, refreshToken string) (OAuthTokenResult, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return OAuthTokenResult{}, fmt.Errorf("%w: confluence OAuth refresh token is required", app.ErrInvalidInput)
	}
	if client.clientSecret == "" {
		return OAuthTokenResult{}, fmt.Errorf("%w: confluence OAuth client secret is required", app.ErrInvalidInput)
	}
	return client.exchangeToken(ctx, map[string]string{
		"grant_type":    "refresh_token",
		"client_id":     client.clientID,
		"client_secret": client.clientSecret,
		"refresh_token": refreshToken,
	})
}

func (client *OAuthClient) exchangeToken(ctx context.Context, payload map[string]string) (OAuthTokenResult, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return OAuthTokenResult{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.tokenURL.String(), bytes.NewReader(body))
	if err != nil {
		return OAuthTokenResult{}, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	response, err := client.httpClient.Do(request)
	if err != nil {
		return OAuthTokenResult{}, app.NewConfluenceTransportError(confluenceOAuthTokenOperation, err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return OAuthTokenResult{}, app.NewConfluenceHTTPError(response.StatusCode, response.Header.Get("Retry-After"), confluenceOAuthTokenOperation)
	}
	var decoded oauthTokenResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return OAuthTokenResult{}, fmt.Errorf("decode confluence OAuth token response: %w", err)
	}
	if strings.TrimSpace(decoded.AccessToken) == "" {
		return OAuthTokenResult{}, fmt.Errorf("%w: confluence OAuth response missing access token", app.ErrInvalidInput)
	}
	expiresAt := time.Time{}
	if decoded.ExpiresIn > 0 {
		expiresAt = time.Now().UTC().Add(time.Duration(decoded.ExpiresIn) * time.Second)
	}
	return OAuthTokenResult{
		AccessToken:    strings.TrimSpace(decoded.AccessToken),
		RefreshToken:   strings.TrimSpace(decoded.RefreshToken),
		TokenType:      strings.TrimSpace(decoded.TokenType),
		Scopes:         normalizeScopeString(decoded.Scope),
		TokenExpiresAt: expiresAt,
	}, nil
}

func (client *OAuthClient) Config() OAuthConfig {
	return OAuthConfig{
		ClientID:     client.clientID,
		ClientSecret: client.clientSecret,
		RedirectURI:  client.redirectURI,
		Scopes:       append([]string(nil), client.scopes...),
		AuthorizeURL: client.authorizeURL.String(),
		TokenURL:     client.tokenURL.String(),
		HTTPClient:   client.httpClient,
	}
}

func normalizeScopes(scopes []string) []string {
	seen := map[string]struct{}{}
	var normalized []string
	for _, scope := range scopes {
		for _, field := range strings.Fields(strings.ReplaceAll(scope, ",", " ")) {
			field = strings.TrimSpace(field)
			if field == "" {
				continue
			}
			if _, ok := seen[field]; ok {
				continue
			}
			seen[field] = struct{}{}
			normalized = append(normalized, field)
		}
	}
	return normalized
}

func normalizeScopeString(scope string) []string {
	return normalizeScopes([]string{scope})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
