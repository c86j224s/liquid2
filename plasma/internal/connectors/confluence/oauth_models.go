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

// OAuthConfig는 Confluence OAuth client를 만들기 위한 정적 설정이다.
//
// HTTPClient가 nil이면 기본 client를 사용하며, scopes가 비어 있으면 호출자가
// DefaultOAuthScopes를 적용할 수 있다.
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       []string
	AuthorizeURL string
	TokenURL     string
	HTTPClient   *http.Client
}

// OAuthClient는 Atlassian OAuth authorize URL 생성과 code exchange를 수행한다.
//
// token 저장, refresh scheduling, 미션별 연결 정책은 이 타입의 책임이 아니다.
type OAuthClient struct {
	clientID     string
	clientSecret string
	redirectURI  string
	scopes       []string
	authorizeURL *url.URL
	tokenURL     *url.URL
	httpClient   *http.Client
}

// OAuthAuthorizationRequest는 사용자 브라우저를 보낼 authorize URL 입력이다.
type OAuthAuthorizationRequest struct {
	State       string
	RedirectURI string
	Scopes      []string
}

// OAuthCodeExchangeRequest는 OAuth callback code를 token으로 교환할 때의 입력이다.
type OAuthCodeExchangeRequest struct {
	Code        string
	RedirectURI string
}

// OAuthTokenResult는 Atlassian token endpoint 응답을 저장 가능한 형태로 정리한 값이다.
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

// DefaultOAuthScopes는 Plasma가 Confluence source 검색/읽기에 필요한 기본 scope다.
func DefaultOAuthScopes() []string {
	return []string{"read:confluence-content.all", "read:confluence-space.summary", "offline_access"}
}
