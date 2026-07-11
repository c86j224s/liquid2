package confluence

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

type AccessTokenProvider interface {
	AccessToken(context.Context) (string, error)
}

type AccessTokenProviderFunc func(context.Context) (string, error)

func (f AccessTokenProviderFunc) AccessToken(ctx context.Context) (string, error) {
	return f(ctx)
}

type AuthorizationProvider interface {
	AuthorizationHeader(context.Context) (string, error)
}

type AuthorizationProviderFunc func(context.Context) (string, error)

func (f AuthorizationProviderFunc) AuthorizationHeader(ctx context.Context) (string, error) {
	return f(ctx)
}

type Client struct {
	baseURL          *url.URL
	siteURL          *url.URL
	cloudID          string
	httpClient       *http.Client
	authProvider     AuthorizationProvider
	connectorVersion string
	optionErr        error
}

type Option func(*Client)

func WithHTTPClient(httpClient *http.Client) Option {
	return func(client *Client) {
		if httpClient != nil {
			client.httpClient = httpClient
		}
	}
}

func WithAccessTokenProvider(provider AccessTokenProvider) Option {
	return func(client *Client) {
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

func WithBearerToken(token string) Option {
	return WithAccessTokenProvider(AccessTokenProviderFunc(func(context.Context) (string, error) {
		return strings.TrimSpace(token), nil
	}))
}

func WithBasicAuth(email string, token string) Option {
	return func(client *Client) {
		email = strings.TrimSpace(email)
		token = strings.TrimSpace(token)
		if email == "" || token == "" {
			return
		}
		encoded := base64.StdEncoding.EncodeToString([]byte(email + ":" + token))
		client.authProvider = AuthorizationProviderFunc(func(context.Context) (string, error) {
			return "Basic " + encoded, nil
		})
	}
}

func WithAuthorizationProvider(provider AuthorizationProvider) Option {
	return func(client *Client) {
		if provider != nil {
			client.authProvider = provider
		}
	}
}

func WithConnectorVersion(version string) Option {
	return func(client *Client) {
		if strings.TrimSpace(version) != "" {
			client.connectorVersion = strings.TrimSpace(version)
		}
	}
}

func WithSiteURL(siteURL string) Option {
	return func(client *Client) {
		parsed, err := parseHTTPURL(siteURL, "confluence site URL")
		if err != nil {
			client.optionErr = err
			return
		}
		client.siteURL = parsed
	}
}

func NewClient(baseURL string, cloudID string, options ...Option) (*Client, error) {
	parsedBase, err := parseHTTPURL(baseURL, "confluence base URL")
	if err != nil {
		return nil, err
	}
	trimmedCloudID := strings.TrimSpace(cloudID)
	if trimmedCloudID == "" {
		return nil, fmt.Errorf("%w: confluence cloud id is required", app.ErrInvalidInput)
	}
	client := &Client{
		baseURL:          parsedBase,
		cloudID:          trimmedCloudID,
		httpClient:       http.DefaultClient,
		connectorVersion: app.ConfluenceHTTPConnectorV1,
	}
	for _, option := range options {
		option(client)
	}
	if client.optionErr != nil {
		return nil, client.optionErr
	}
	return client, nil
}

func APIBaseURLForCloud(cloudID string) string {
	cloudID = strings.TrimSpace(cloudID)
	if cloudID == "" {
		return ""
	}
	return "https://api.atlassian.com/ex/confluence/" + url.PathEscape(cloudID) + "/wiki"
}

func (client *Client) SearchConfluenceSources(
	ctx context.Context,
	req app.ConfluenceSourceSearchRequest,
) (app.ConfluenceSourceSearchResult, error) {
	if err := client.validateCloudID(req.CloudID); err != nil {
		return app.ConfluenceSourceSearchResult{}, err
	}
	query := url.Values{}
	query.Set("cql", confluenceCQL(req.Query, req.SpaceKey))
	if req.Limit > 0 {
		query.Set("limit", strconv.Itoa(req.Limit))
	}
	if req.Cursor != "" {
		query.Set("cursor", req.Cursor)
	}

	var response confluenceSearchResponse
	if err := client.getJSON(ctx, "/rest/api/search", query, &response); err != nil {
		return app.ConfluenceSourceSearchResult{}, err
	}
	candidates := make([]app.ConfluenceSourceCandidate, 0, len(response.Results))
	for _, item := range response.Results {
		candidates = append(candidates, client.candidate(item, response.Links.Base))
	}
	return app.ConfluenceSourceSearchResult{
		MissionID:  req.MissionID,
		CloudID:    client.cloudID,
		Candidates: candidates,
		NextCursor: cursorFromNextLink(response.Links.Next),
	}, nil
}

func (client *Client) ReadConfluenceSource(
	ctx context.Context,
	req app.ConfluenceSourceReadRequest,
) (app.ConfluenceSourcePage, error) {
	if err := client.validateCloudID(req.CloudID); err != nil {
		return app.ConfluenceSourcePage{}, err
	}
	pageID := strings.TrimSpace(req.PageID)
	if pageID == "" {
		return app.ConfluenceSourcePage{}, fmt.Errorf("%w: confluence page id is required", app.ErrInvalidInput)
	}
	query := url.Values{"body-format": []string{"storage"}}
	var response confluencePageResponse
	if err := client.getJSON(ctx, "/api/v2/pages/"+url.PathEscape(pageID), query, &response); err != nil {
		return app.ConfluenceSourcePage{}, err
	}
	if response.ID == "" {
		response.ID = pageID
	} else if response.ID != pageID {
		return app.ConfluenceSourcePage{}, fmt.Errorf("%w: confluence page id mismatch", app.ErrInvalidInput)
	}
	metadata, err := json.Marshal(response.metadata(client.cloudID, client.siteURLString()))
	if err != nil {
		return app.ConfluenceSourcePage{}, err
	}
	bodyStorage := response.Body.Storage.Value
	return app.ConfluenceSourcePage{
		Connector: app.ConnectorRef{
			ConnectorID:      app.ConfluenceConnectorID,
			ConnectorType:    app.ConfluenceConnectorType,
			ExternalSourceID: app.ConfluenceExternalSourceID(client.cloudID, response.ID),
			ExternalURI:      app.ConfluenceExternalURI(client.cloudID, response.ID),
			ExternalVersion:  confluenceExternalVersion(response.Version.Number, response.Version.CreatedAt),
			ConnectorVersion: client.connectorVersion,
		},
		CloudID:     client.cloudID,
		SiteURL:     client.siteURLString(),
		PageID:      response.ID,
		SpaceID:     response.SpaceID,
		Title:       response.Title,
		WebURL:      client.absoluteURL(response.Links.Base, response.Links.WebUI),
		Version:     response.Version.Number,
		UpdatedAt:   parseConfluenceTime(response.Version.CreatedAt),
		BodyStorage: bodyStorage,
		PlainText:   plainTextFromStorage(bodyStorage),
		Metadata:    metadata,
	}, nil
}

func (client *Client) GetConfluenceSourceVersion(
	ctx context.Context,
	req app.ConfluenceSourceReadRequest,
) (app.ConfluenceSourceVersion, error) {
	if err := client.validateCloudID(req.CloudID); err != nil {
		return app.ConfluenceSourceVersion{}, err
	}
	pageID := strings.TrimSpace(req.PageID)
	if pageID == "" {
		return app.ConfluenceSourceVersion{}, fmt.Errorf("%w: confluence page id is required", app.ErrInvalidInput)
	}
	var response confluencePageResponse
	if err := client.getJSON(ctx, "/api/v2/pages/"+url.PathEscape(pageID), nil, &response); err != nil {
		return app.ConfluenceSourceVersion{}, err
	}
	if response.ID == "" {
		response.ID = pageID
	} else if response.ID != pageID {
		return app.ConfluenceSourceVersion{}, fmt.Errorf("%w: confluence page id mismatch", app.ErrInvalidInput)
	}
	webURL := client.absoluteURL(response.Links.Base, response.Links.WebUI)
	return app.ConfluenceSourceVersion{
		Connector: app.ConnectorRef{
			ConnectorID:      app.ConfluenceConnectorID,
			ConnectorType:    app.ConfluenceConnectorType,
			ExternalSourceID: app.ConfluenceExternalSourceID(client.cloudID, response.ID),
			ExternalURI:      app.ConfluenceExternalURI(client.cloudID, response.ID),
			ExternalVersion:  confluenceExternalVersion(response.Version.Number, response.Version.CreatedAt),
			ConnectorVersion: client.connectorVersion,
		},
		CloudID:   client.cloudID,
		SiteURL:   client.siteURLString(),
		PageID:    response.ID,
		SpaceID:   response.SpaceID,
		Title:     response.Title,
		WebURL:    webURL,
		Version:   response.Version.Number,
		UpdatedAt: parseConfluenceTime(response.Version.CreatedAt),
	}, nil
}

func (client *Client) validateCloudID(requestCloudID string) error {
	if trimmed := strings.TrimSpace(requestCloudID); trimmed != "" && trimmed != client.cloudID {
		return app.NewConfluenceValidationError(
			app.ConfluenceErrorCodeCloudMismatch,
			"Confluence cloud id가 연결된 site와 일치하지 않습니다. site 선택을 확인하세요.",
		)
	}
	return nil
}
