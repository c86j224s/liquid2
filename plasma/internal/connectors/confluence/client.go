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

// AccessTokenProvider는 Confluence OAuth token을 호출 시점에 공급하는 port다.
//
// 구현체는 token 저장/갱신 정책을 소유할 수 있지만, Client는 받은 token을 인증
// header로 바꾸는 일만 한다.
type AccessTokenProvider interface {
	AccessToken(context.Context) (string, error)
}

// AccessTokenProviderFunc는 함수를 AccessTokenProvider로 쓰게 하는 adapter다.
type AccessTokenProviderFunc func(context.Context) (string, error)

// AccessToken은 요청 시점의 bearer token을 반환한다.
func (f AccessTokenProviderFunc) AccessToken(ctx context.Context) (string, error) {
	return f(ctx)
}

// AuthorizationProvider는 완성된 Authorization header 값을 공급한다.
//
// Basic auth처럼 token 형식이 bearer가 아닌 경우 이 port를 통해 Client에 주입한다.
type AuthorizationProvider interface {
	AuthorizationHeader(context.Context) (string, error)
}

// AuthorizationProviderFunc는 함수를 AuthorizationProvider로 쓰게 하는 adapter다.
type AuthorizationProviderFunc func(context.Context) (string, error)

// AuthorizationHeader는 Confluence REST 호출에 사용할 Authorization header 값을 만든다.
func (f AuthorizationProviderFunc) AuthorizationHeader(ctx context.Context) (string, error) {
	return f(ctx)
}

// Client는 Confluence Cloud API를 Plasma connector port로 변환하는 HTTP adapter다.
//
// Client는 cloudID와 baseURL mismatch를 거부하고, API 응답을 app 계층의 source
// 후보/페이지 모델로 변환한다. 미션 접근 권한과 source 승인 정책은 포함하지 않는다.
type Client struct {
	baseURL          *url.URL
	siteURL          *url.URL
	cloudID          string
	httpClient       *http.Client
	authProvider     AuthorizationProvider
	connectorVersion string
	optionErr        error
}

// Option은 Client 생성 시 transport와 인증 방식을 주입하는 함수다.
type Option func(*Client)

// WithHTTPClient는 테스트 또는 embedding 환경에서 HTTP transport를 교체한다.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(client *Client) {
		if httpClient != nil {
			client.httpClient = httpClient
		}
	}
}

// WithAccessTokenProvider는 bearer token provider를 Client 인증 방식으로 설정한다.
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

// WithBearerToken은 고정 bearer token으로 Client를 구성한다.
func WithBearerToken(token string) Option {
	return WithAccessTokenProvider(AccessTokenProviderFunc(func(context.Context) (string, error) {
		return strings.TrimSpace(token), nil
	}))
}

// WithBasicAuth는 Confluence API token 인증용 Basic Authorization header를 구성한다.
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

// WithAuthorizationProvider는 완성된 Authorization header provider를 주입한다.
func WithAuthorizationProvider(provider AuthorizationProvider) Option {
	return func(client *Client) {
		if provider != nil {
			client.authProvider = provider
		}
	}
}

// WithConnectorVersion은 snapshot metadata에 기록할 adapter version을 바꾼다.
func WithConnectorVersion(version string) Option {
	return func(client *Client) {
		if strings.TrimSpace(version) != "" {
			client.connectorVersion = strings.TrimSpace(version)
		}
	}
}

// WithSiteURL은 API 응답의 상대 web URL을 사람이 여는 site URL로 해석하게 한다.
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

// NewClient는 Confluence Cloud API client를 생성하고 base URL/cloud ID를 검증한다.
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

// APIBaseURLForCloud는 Atlassian cloud ID를 Confluence REST API base URL로 변환한다.
func APIBaseURLForCloud(cloudID string) string {
	cloudID = strings.TrimSpace(cloudID)
	if cloudID == "" {
		return ""
	}
	return "https://api.atlassian.com/ex/confluence/" + url.PathEscape(cloudID) + "/wiki"
}

// SearchConfluenceSources는 Confluence CQL 검색 결과를 source 후보로 변환한다.
//
// 반환값은 아직 승인된 source가 아니며, 호출자는 별도 user review/staging 흐름을
// 거쳐야 한다.
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

// ReadConfluenceSource는 단일 Confluence page를 snapshot 가능한 page model로 읽는다.
//
// pageID 불일치는 잘못된 source가 등록되지 않도록 invalid input으로 거부한다.
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

// GetConfluenceSourceVersion는 Confluence 커넥터의 읽기 경계다. 제품 상태를 바꾸지 않고 필요한 projection이나 외부 자료만 반환한다.
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
