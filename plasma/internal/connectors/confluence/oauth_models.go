package confluence

import (
	"net/http"
	"net/url"
	"time"
)

const (
	defaultOAuthAuthorizeURL = "https://auth.atlassian.com/authorize"
	defaultOAuthTokenURL     = "https://auth.atlassian.com/oauth/token"
	defaultOAuthAudience     = "api.atlassian.com"
	defaultOAuthPrompt       = "consent"
)

type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       []string
	AuthorizeURL string
	TokenURL     string
	HTTPClient   *http.Client
}

type OAuthClient struct {
	clientID     string
	clientSecret string
	redirectURI  string
	scopes       []string
	authorizeURL *url.URL
	tokenURL     *url.URL
	httpClient   *http.Client
}

type OAuthAuthorizationRequest struct {
	State       string
	RedirectURI string
	Scopes      []string
}

type OAuthCodeExchangeRequest struct {
	Code        string
	RedirectURI string
}

type OAuthTokenResult struct {
	AccessToken    string
	RefreshToken   string
	TokenType      string
	Scopes         []string
	TokenExpiresAt time.Time
}

type oauthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

func DefaultOAuthScopes() []string {
	return []string{"read:confluence-content.all", "read:confluence-space.summary", "offline_access"}
}
