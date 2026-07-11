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
)

type DiscoveryClient struct {
	baseURL      *url.URL
	httpClient   *http.Client
	authProvider AuthorizationProvider
	optionErr    error
}

type DiscoveryOption func(*DiscoveryClient)

func WithDiscoveryHTTPClient(httpClient *http.Client) DiscoveryOption {
	return func(client *DiscoveryClient) {
		if httpClient != nil {
			client.httpClient = httpClient
		}
	}
}

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

func WithDiscoveryBearerToken(token string) DiscoveryOption {
	return WithDiscoveryAccessTokenProvider(AccessTokenProviderFunc(func(context.Context) (string, error) {
		return strings.TrimSpace(token), nil
	}))
}

func WithDiscoveryAuthorizationProvider(provider AuthorizationProvider) DiscoveryOption {
	return func(client *DiscoveryClient) {
		if provider != nil {
			client.authProvider = provider
		}
	}
}

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

func (client *DiscoveryClient) ListConfluenceSites(ctx context.Context) (app.ConfluenceSiteListResult, error) {
	var response []accessibleResource
	if err := client.getJSON(ctx, "/oauth/token/accessible-resources", &response); err != nil {
		return app.ConfluenceSiteListResult{}, err
	}
	sites := make([]app.ConfluenceSite, 0, len(response))
	for _, resource := range response {
		cloudID := strings.TrimSpace(resource.ID)
		if cloudID == "" || !resource.hasConfluenceScope() {
			continue
		}
		sites = append(sites, app.ConfluenceSite{
			CloudID: cloudID,
			Name:    strings.TrimSpace(resource.Name),
			URL:     strings.TrimRight(strings.TrimSpace(resource.URL), "/"),
			Scopes:  resource.Scopes,
		})
	}
	return app.ConfluenceSiteListResult{Sites: sites}, nil
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
