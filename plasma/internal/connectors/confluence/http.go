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

const (
	confluenceDiscoveryOperation  = "GET /oauth/token/accessible-resources"
	confluenceOAuthTokenOperation = "POST /oauth/token"
)

func (client *Client) getJSON(ctx context.Context, endpoint string, query url.Values, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, client.endpoint(endpoint, query), nil)
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
		return app.NewConfluenceTransportError(safeOperation(http.MethodGet, endpoint), err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return app.NewConfluenceHTTPError(response.StatusCode, response.Header.Get("Retry-After"), safeOperation(http.MethodGet, endpoint))
	}
	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode confluence response: %w", err)
	}
	return nil
}

func safeOperation(method string, endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return strings.TrimSpace(method)
	}
	if index := strings.Index(endpoint, "?"); index >= 0 {
		endpoint = endpoint[:index]
	}
	parts := strings.Split(endpoint, "/")
	for index := 0; index < len(parts)-1; index++ {
		if parts[index] == "pages" && hasConfluenceV2Prefix(parts, index) {
			parts[index+1] = "{page_id}"
		}
		if parts[index] == "spaces" && hasConfluenceV2Prefix(parts, index) {
			parts[index+1] = "{space_id}"
		}
	}
	return strings.TrimSpace(method + " " + strings.Join(parts, "/"))
}

func hasConfluenceV2Prefix(parts []string, before int) bool {
	for index := 0; index < before-1; index++ {
		if parts[index] == "api" && parts[index+1] == "v2" {
			return true
		}
	}
	return false
}

func (client *Client) endpoint(endpoint string, query url.Values) string {
	u := *client.baseURL
	basePath := strings.TrimRight(u.Path, "/")
	u.Path = basePath + endpoint
	if query != nil {
		u.RawQuery = query.Encode()
	}
	return u.String()
}

func parseHTTPURL(value string, label string) (*url.URL, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, fmt.Errorf("%w: %s is required", app.ErrInvalidInput, label)
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid %s", app.ErrInvalidInput, label)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("%w: %s must include scheme and host", app.ErrInvalidInput, label)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("%w: %s must use http or https", app.ErrInvalidInput, label)
	}
	return parsed, nil
}

func rejectSensitiveURLParts(parsed *url.URL, label string) error {
	if parsed == nil {
		return fmt.Errorf("%w: %s is required", app.ErrInvalidInput, label)
	}
	if parsed.User != nil {
		return fmt.Errorf("%w: %s must not include credentials", app.ErrInvalidInput, label)
	}
	if strings.TrimSpace(parsed.RawQuery) != "" {
		return fmt.Errorf("%w: %s must not include query parameters", app.ErrInvalidInput, label)
	}
	if strings.TrimSpace(parsed.Fragment) != "" {
		return fmt.Errorf("%w: %s must not include a fragment", app.ErrInvalidInput, label)
	}
	return nil
}
