package liquid2

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

type Client struct {
	baseURL          *url.URL
	httpClient       *http.Client
	connectorVersion string
}

type Option func(*Client)

func WithHTTPClient(httpClient *http.Client) Option {
	return func(client *Client) {
		if httpClient != nil {
			client.httpClient = httpClient
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

func NewClient(baseURL string, options ...Option) (*Client, error) {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		return nil, fmt.Errorf("%w: liquid2 base URL is required", app.ErrInvalidInput)
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid liquid2 base URL", app.ErrInvalidInput)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("%w: liquid2 base URL must include scheme and host", app.ErrInvalidInput)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("%w: liquid2 base URL must use http or https", app.ErrInvalidInput)
	}
	client := &Client{
		baseURL:          parsed,
		httpClient:       http.DefaultClient,
		connectorVersion: app.Liquid2HTTPConnectorV1,
	}
	for _, option := range options {
		option(client)
	}
	return client, nil
}

func (client *Client) SearchLiquid2Sources(
	ctx context.Context,
	req app.Liquid2SourceSearchRequest,
) (app.Liquid2SourceSearchResult, error) {
	query := url.Values{}
	if req.Query != "" {
		query.Set("q", req.Query)
		query.Set("sort", "relevance")
	} else {
		query.Set("sort", "recent")
	}
	if req.Limit > 0 {
		query.Set("limit", strconv.Itoa(req.Limit))
	}
	if req.Cursor != "" {
		query.Set("cursor", req.Cursor)
	}
	if req.Filters.Status != "" {
		query.Set("status", req.Filters.Status)
	}
	if req.Filters.Tag != "" {
		query.Set("tag", req.Filters.Tag)
	}
	if req.Filters.Kind != "" {
		query.Set("kind", req.Filters.Kind)
	}
	if req.Filters.RatingMin > 0 {
		query.Set("ratingMin", strconv.Itoa(req.Filters.RatingMin))
	}
	if req.Filters.IncludeDeleted {
		query.Set("includeDeleted", "true")
	}
	if req.Filters.IncludeTrash {
		query.Set("includeTrash", "true")
	}

	var response liquid2DocumentList
	if err := client.getJSON(ctx, "/api/v1/documents", query, &response); err != nil {
		return app.Liquid2SourceSearchResult{}, err
	}
	candidates := make([]app.Liquid2SourceCandidate, 0, len(response.Items))
	for _, item := range response.Items {
		candidates = append(candidates, client.candidate(item))
	}
	nextCursor := ""
	if response.NextCursor != nil {
		nextCursor = *response.NextCursor
	}
	return app.Liquid2SourceSearchResult{
		MissionID:  req.MissionID,
		Candidates: candidates,
		NextCursor: nextCursor,
	}, nil
}

func (client *Client) ReadLiquid2Source(
	ctx context.Context,
	req app.Liquid2SourceReadRequest,
) (app.Liquid2SourceDocument, error) {
	externalSourceID := strings.TrimSpace(req.ExternalSourceID)
	if externalSourceID == "" {
		return app.Liquid2SourceDocument{}, fmt.Errorf("%w: liquid2 external source id is required", app.ErrInvalidInput)
	}
	var response liquid2DocumentDetail
	if err := client.getJSON(ctx, "/api/v1/documents/"+url.PathEscape(externalSourceID), nil, &response); err != nil {
		return app.Liquid2SourceDocument{}, err
	}
	metadata, err := json.Marshal(liquid2DocumentMetadataEnvelope{
		Document:   response.Document,
		FolderPath: response.FolderPath,
		Tags:       response.Tags,
		Blobs:      response.Blobs,
	})
	if err != nil {
		return app.Liquid2SourceDocument{}, err
	}
	contents := make([]app.Liquid2SourceContent, 0, len(response.Contents))
	for _, content := range response.Contents {
		contents = append(contents, app.Liquid2SourceContent{
			ContentID: content.ID,
			Role:      content.Role,
			Format:    content.Format,
			Language:  stringValue(content.Language),
			Content:   content.Content,
		})
	}
	documentID := strings.TrimSpace(response.Document.ID)
	if documentID == "" {
		documentID = externalSourceID
	} else if documentID != externalSourceID {
		return app.Liquid2SourceDocument{}, fmt.Errorf("%w: liquid2 document id mismatch", app.ErrInvalidInput)
	}
	return app.Liquid2SourceDocument{
		Connector: app.ConnectorRef{
			ConnectorID:      app.Liquid2ConnectorID,
			ConnectorType:    app.Liquid2ConnectorType,
			ExternalSourceID: documentID,
			ExternalURI:      liquid2DocumentURI(documentID),
			ExternalVersion:  strconv.FormatInt(response.Document.UpdatedAt, 10),
			ConnectorVersion: client.connectorVersion,
		},
		Title:     response.Document.Title,
		SourceURI: sourceURI(response.Document.CanonicalURL, response.Document.SourceURL),
		UpdatedAt: unixMillisTime(response.Document.UpdatedAt),
		Contents:  contents,
		Metadata:  metadata,
	}, nil
}

func (client *Client) getJSON(ctx context.Context, endpoint string, query url.Values, target any) error {
	requestURL := client.endpoint(endpoint, query)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")

	response, err := client.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("liquid2 connector GET %s returned %d: %s", endpoint, response.StatusCode, strings.TrimSpace(string(body)))
	}
	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode liquid2 response: %w", err)
	}
	return nil
}

func (client *Client) endpoint(endpoint string, query url.Values) string {
	u := *client.baseURL
	basePath := strings.TrimRight(u.Path, "/")
	u.Path = basePath + endpoint
	u.RawQuery = query.Encode()
	return u.String()
}

func (client *Client) candidate(item liquid2DocumentSummary) app.Liquid2SourceCandidate {
	return app.Liquid2SourceCandidate{
		Connector: app.ConnectorRef{
			ConnectorID:      app.Liquid2ConnectorID,
			ConnectorType:    app.Liquid2ConnectorType,
			ExternalSourceID: item.ID,
			ExternalURI:      liquid2DocumentURI(item.ID),
			ExternalVersion:  strconv.FormatInt(item.UpdatedAt, 10),
			ConnectorVersion: client.connectorVersion,
		},
		Title:       item.Title,
		SourceURI:   sourceURI(item.CanonicalURL, item.SourceURL),
		Summary:     candidateSummary(item),
		UpdatedAt:   unixMillisTime(item.UpdatedAt),
		CanSnapshot: true,
	}
}

func liquid2DocumentURI(externalSourceID string) string {
	return "liquid2://documents/" + strings.TrimSpace(externalSourceID)
}

func sourceURI(canonicalURL *string, sourceURL *string) string {
	if value := strings.TrimSpace(stringValue(canonicalURL)); value != "" {
		return value
	}
	return strings.TrimSpace(stringValue(sourceURL))
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func unixMillisTime(value int64) time.Time {
	if value <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(value).UTC()
}

func candidateSummary(item liquid2DocumentSummary) string {
	parts := []string{}
	if item.Kind != "" {
		parts = append(parts, item.Kind)
	}
	if item.Status != "" {
		parts = append(parts, item.Status)
	}
	if len(item.Tags) > 0 {
		parts = append(parts, "tags: "+strings.Join(item.Tags, ", "))
	}
	return strings.Join(parts, " | ")
}

type liquid2DocumentList struct {
	Items      []liquid2DocumentSummary `json:"items"`
	NextCursor *string                  `json:"nextCursor"`
	TotalCount int                      `json:"totalCount"`
}

type liquid2DocumentSummary struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Kind         string   `json:"kind"`
	CanonicalURL *string  `json:"canonicalUrl"`
	SourceURL    *string  `json:"sourceUrl"`
	Language     *string  `json:"language"`
	Status       string   `json:"status"`
	UpdatedAt    int64    `json:"updatedAt"`
	Tags         []string `json:"tags"`
}

type liquid2DocumentDetail struct {
	Document   liquid2DocumentMetadata   `json:"document"`
	FolderPath []liquid2FolderBreadcrumb `json:"folderPath"`
	Contents   []liquid2DocumentContent  `json:"contents"`
	Tags       []liquid2Tag              `json:"tags"`
	Blobs      []liquid2Blob             `json:"blobs"`
}

type liquid2DocumentMetadata struct {
	ID           string  `json:"id"`
	Title        string  `json:"title"`
	Kind         string  `json:"kind"`
	FolderID     *string `json:"folderId"`
	CanonicalURL *string `json:"canonicalUrl"`
	SourceURL    *string `json:"sourceUrl"`
	Language     *string `json:"language"`
	Status       string  `json:"status"`
	Rating       *int    `json:"rating"`
	CreatedAt    int64   `json:"createdAt"`
	UpdatedAt    int64   `json:"updatedAt"`
	PublishedAt  *int64  `json:"publishedAt"`
	ReadAt       *int64  `json:"readAt"`
	DeletedAt    *int64  `json:"deletedAt"`
}

type liquid2DocumentContent struct {
	ID       string  `json:"id"`
	Role     string  `json:"role"`
	Format   string  `json:"format"`
	Language *string `json:"language"`
	Content  string  `json:"content"`
}

type liquid2Tag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type liquid2FolderBreadcrumb struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type liquid2Blob struct {
	ID        string `json:"id"`
	Filename  string `json:"filename"`
	MimeType  string `json:"mimeType"`
	Size      int64  `json:"size"`
	CreatedAt int64  `json:"createdAt"`
}

type liquid2DocumentMetadataEnvelope struct {
	Document   liquid2DocumentMetadata   `json:"document"`
	FolderPath []liquid2FolderBreadcrumb `json:"folderPath"`
	Tags       []liquid2Tag              `json:"tags"`
	Blobs      []liquid2Blob             `json:"blobs"`
}
