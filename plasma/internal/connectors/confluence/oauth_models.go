package confluence

import "net/http"

// OAuthConfig preserves legacy configuration parsing only; no OAuth client remains.
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       []string
	AuthorizeURL string
	TokenURL     string
	HTTPClient   *http.Client
}

func DefaultOAuthScopes() []string {
	return []string{"read:confluence-content.all", "read:confluence-space.summary", "offline_access"}
}
