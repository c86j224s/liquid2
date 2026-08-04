package confluence

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/confluenceaccess"
)

// DiscoveryClient는 Atlassian discovery API 호출에 필요한 base URL, HTTP client,
// 인증 provider를 보관한다.
type DiscoveryClient struct {
	baseURL      *url.URL
	httpClient   *http.Client
	authProvider AuthorizationProvider
	optionErr    error
}

// DiscoveryOption는 Confluence 커넥터 실행 옵션이다. 0 값과 누락 값의 의미는 생성자나 Normalize 경계가 정한다.
type DiscoveryOption func(*DiscoveryClient)

// WithDiscoveryHTTPClient는 discovery 요청에 사용할 HTTP client를 주입한다.
func WithDiscoveryHTTPClient(httpClient *http.Client) DiscoveryOption {
	return func(client *DiscoveryClient) {
		if httpClient != nil {
			client.httpClient = httpClient
		}
	}
}

// WithDiscoveryBaseURL는 Atlassian discovery endpoint의 base URL을 검증해 교체한다.
func WithDiscoveryBaseURL(baseURL string) DiscoveryOption {
	return func(client *DiscoveryClient) {
		if strings.TrimSpace(baseURL) == "" {
			return
		}
		parsed, err := parseHTTPURL(baseURL, "confluence discovery base URL")
		if err != nil {
			client.optionErr = err
			return
		}
		if err := rejectSensitiveURLParts(parsed, "confluence discovery base URL"); err != nil {
			client.optionErr = err
			return
		}
		client.baseURL = parsed
	}
}

// WithDiscoveryAccessTokenProvider는 access token provider를 Authorization header provider로 감싼다.
func WithDiscoveryAccessTokenProvider(provider AccessTokenProvider) DiscoveryOption {
	return func(client *DiscoveryClient) {
		if provider != nil {
			client.authProvider = AuthorizationProviderFunc(func(ctx context.Context) (string, error) {
				token, err := provider.AccessToken(ctx)
				if err != nil {
					return "", err
				}
				token = strings.TrimSpace(token)
				if token == "" {
					return "", nil
				}
				return "Bearer " + token, nil
			})
		}
	}
}

// WithDiscoveryBearerToken는 고정 bearer token을 쓰는 discovery 인증 provider를 만든다.
func WithDiscoveryBearerToken(token string) DiscoveryOption {
	return WithDiscoveryAccessTokenProvider(AccessTokenProviderFunc(func(context.Context) (string, error) {
		return strings.TrimSpace(token), nil
	}))
}

// WithDiscoveryAuthorizationProvider는 완성된 Authorization header provider를 주입한다.
func WithDiscoveryAuthorizationProvider(provider AuthorizationProvider) DiscoveryOption {
	return func(client *DiscoveryClient) {
		if provider != nil {
			client.authProvider = provider
		}
	}
}

// NewDiscoveryClient는 discovery 기본값과 option을 합쳐 호출 가능한 client를 만든다.
func NewDiscoveryClient(options ...DiscoveryOption) (*DiscoveryClient, error) {
	parsed, err := parseHTTPURL("https://api.atlassian.com", "confluence discovery base URL")
	if err != nil {
		return nil, err
	}
	client := &DiscoveryClient{
		baseURL:    parsed,
		httpClient: http.DefaultClient,
	}
	for _, option := range options {
		option(client)
	}
	if client.optionErr != nil {
		return nil, client.optionErr
	}
	return client, nil
}

// ListConfluenceSites는 Confluence 커넥터의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
func (client *DiscoveryClient) ListConfluenceSites(ctx context.Context) (confluenceaccess.SiteListResult, error) {
	var response []accessibleResource
	if err := client.getJSON(ctx, "/oauth/token/accessible-resources", &response); err != nil {
		return confluenceaccess.SiteListResult{}, err
	}
	sites := make([]confluenceaccess.Site, 0, len(response))
	for _, resource := range response {
		cloudID := strings.TrimSpace(resource.ID)
		if cloudID == "" || !resource.hasConfluenceScope() {
			continue
		}
		sites = append(sites, confluenceaccess.Site{
			CloudID: cloudID,
			Name:    strings.TrimSpace(resource.Name),
			URL:     strings.TrimRight(strings.TrimSpace(resource.URL), "/"),
			Scopes:  resource.Scopes,
		})
	}
	return confluenceaccess.SiteListResult{Sites: sites}, nil
}

func (client *DiscoveryClient) getJSON(ctx context.Context, endpoint string, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, client.endpoint(endpoint), nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")
	if client.authProvider != nil {
		header, err := client.authProvider.AuthorizationHeader(ctx)
		if err != nil {
			return err
		}
		if strings.TrimSpace(header) != "" {
			request.Header.Set("Authorization", strings.TrimSpace(header))
		}
	}
	response, err := client.httpClient.Do(request)
	if err != nil {
		return app.NewConfluenceTransportError(confluenceDiscoveryOperation, err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return app.NewConfluenceHTTPError(response.StatusCode, response.Header.Get("Retry-After"), confluenceDiscoveryOperation)
	}
	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode confluence discovery response: %w", err)
	}
	return nil
}

func (client *DiscoveryClient) endpoint(endpoint string) string {
	u := *client.baseURL
	u.Path = strings.TrimRight(u.Path, "/") + endpoint
	return u.String()
}

type accessibleResource struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	URL    string   `json:"url"`
	Scopes []string `json:"scopes"`
}

func (resource accessibleResource) hasConfluenceScope() bool {
	if len(resource.Scopes) == 0 {
		return true
	}
	for _, scope := range resource.Scopes {
		if strings.Contains(strings.ToLower(scope), "confluence") {
			return true
		}
	}
	return false
}
