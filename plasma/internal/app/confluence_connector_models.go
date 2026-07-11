package app

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

const (
	ConfluenceConnectorID          = "confluence"
	ConfluenceConnectorType        = "confluence_cloud"
	ConfluenceHTTPConnectorV1      = "confluence-cloud-http.v1"
	ConfluenceSnapshotMediaType    = "application/vnd.plasma.confluence.snapshot+json"
	ConfluenceSnapshotSchemaV1     = "plasma.confluence.snapshot.v1"
	ConfluenceAuthTypeOAuth        = "oauth"
	ConfluenceAuthTypeAPIToken     = "api_token"
	ConfluenceUpdateCurrentEvent   = "source.update.current"
	ConfluenceUpdateAvailableEvent = "source.update.available"
	ConfluenceUpdatedEvent         = "source.updated"
	defaultConfluenceSearchLimit   = 10
	maxConfluenceSearchLimit       = 100
	DefaultConfluenceMaxBodyBytes  = int64(1024 * 1024)
)

type ConfluenceSourceConnector interface {
	SearchConfluenceSources(context.Context, ConfluenceSourceSearchRequest) (ConfluenceSourceSearchResult, error)
	ReadConfluenceSource(context.Context, ConfluenceSourceReadRequest) (ConfluenceSourcePage, error)
}

type ConfluenceBrowserConnector interface {
	ListConfluenceSpaces(context.Context, ConfluenceSpaceListRequest) (ConfluenceSpaceListResult, error)
	ListConfluenceSpacePages(context.Context, ConfluenceSpacePagesRequest) (ConfluencePageListResult, error)
	ListConfluencePageChildren(context.Context, ConfluencePageChildrenRequest) (ConfluencePageListResult, error)
}

type ConfluenceSiteLister interface {
	ListConfluenceSites(context.Context) (ConfluenceSiteListResult, error)
}

type ConfluenceSourceVersionConnector interface {
	GetConfluenceSourceVersion(context.Context, ConfluenceSourceReadRequest) (ConfluenceSourceVersion, error)
}

type ConfluenceSite struct {
	CloudID string   `json:"cloud_id"`
	Name    string   `json:"name"`
	URL     string   `json:"url"`
	Scopes  []string `json:"scopes,omitempty"`
}

type ConfluenceSiteListResult struct {
	Sites []ConfluenceSite `json:"sites"`
}

type ConfluenceConnection struct {
	ConnectionID   string           `json:"connection_id"`
	DisplayName    string           `json:"display_name"`
	AuthType       string           `json:"auth_type"`
	AccountID      string           `json:"account_id,omitempty"`
	AccountName    string           `json:"account_name,omitempty"`
	AccessToken    string           `json:"-"`
	RefreshToken   string           `json:"-"`
	TokenExpiresAt time.Time        `json:"token_expires_at,omitempty"`
	Scopes         []string         `json:"scopes,omitempty"`
	Sites          []ConfluenceSite `json:"sites,omitempty"`
	Revoked        bool             `json:"revoked"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

type UpsertConfluenceConnectionRequest struct {
	ConnectionID   string
	DisplayName    string
	AuthType       string
	AccountID      string
	AccountName    string
	AccessToken    string
	RefreshToken   string
	TokenExpiresAt time.Time
	Scopes         []string
	Sites          []ConfluenceSite
	Revoked        bool
}

type ConfluenceSourceSearchRequest struct {
	MissionID string
	CloudID   string
	SiteURL   string
	Query     string
	Limit     int
	Cursor    string
	SpaceID   string
	SpaceKey  string
}

type ConfluenceSourceSearchResult struct {
	MissionID  string
	CloudID    string
	Candidates []ConfluenceSourceCandidate
	NextCursor string
}

type ConfluenceSpaceListRequest struct {
	MissionID string
	CloudID   string
	Limit     int
	Cursor    string
}

type ConfluenceSpaceListResult struct {
	MissionID  string
	CloudID    string
	Spaces     []ConfluenceSpaceSummary
	NextCursor string
}

type ConfluenceSpaceSummary struct {
	CloudID  string `json:"cloud_id"`
	SpaceID  string `json:"space_id"`
	SpaceKey string `json:"space_key,omitempty"`
	Name     string `json:"name"`
	Type     string `json:"type,omitempty"`
	Status   string `json:"status,omitempty"`
	WebURL   string `json:"web_url,omitempty"`
}

type ConfluenceSpacePagesRequest struct {
	MissionID string
	CloudID   string
	SpaceID   string
	Limit     int
	Cursor    string
}

type ConfluencePageChildrenRequest struct {
	MissionID string
	CloudID   string
	PageID    string
	Limit     int
	Cursor    string
}

type ConfluencePageListResult struct {
	MissionID  string
	CloudID    string
	Pages      []ConfluencePageSummary
	NextCursor string
}

type ConfluencePageSummary struct {
	CloudID     string    `json:"cloud_id"`
	PageID      string    `json:"page_id"`
	SpaceID     string    `json:"space_id,omitempty"`
	ParentID    string    `json:"parent_id,omitempty"`
	Title       string    `json:"title"`
	WebURL      string    `json:"web_url,omitempty"`
	Version     int       `json:"version,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
	HasChildren bool      `json:"has_children,omitempty"`
}

type ConfluenceSourceCandidate struct {
	Connector   ConnectorRef
	CloudID     string
	SiteURL     string
	SpaceID     string
	SpaceKey    string
	Title       string
	SourceURI   string
	Summary     string
	Version     int
	UpdatedAt   time.Time
	CanSnapshot bool
}

type ConfluenceSourceReadRequest struct {
	CloudID string
	PageID  string
}

type ConfluenceSourceVersion struct {
	Connector ConnectorRef
	CloudID   string
	SiteURL   string
	PageID    string
	SpaceID   string
	SpaceKey  string
	Title     string
	WebURL    string
	Version   int
	UpdatedAt time.Time
}

type ConfluenceSourcePage struct {
	Connector   ConnectorRef
	CloudID     string
	SiteURL     string
	PageID      string
	SpaceID     string
	SpaceKey    string
	Title       string
	WebURL      string
	Version     int
	UpdatedAt   time.Time
	BodyStorage string
	PlainText   string
	Metadata    json.RawMessage
}

type SnapshotConfluenceSourceRequest struct {
	MissionID           string
	ArtifactID          string
	SnapshotID          string
	CloudID             string
	PageID              string
	Title               string
	ExpectedVersion     int
	MaxBodyBytes        int64
	Range               ConfluenceRangeSelection
	Producer            Producer
	Reason              string
	ExpectedContentHash ContentHash
}

type SnapshotConfluenceSourceWithEventRequest struct {
	Snapshot SnapshotConfluenceSourceRequest
	EventID  string
	Producer Producer
}

type ConfluenceSnapshotResult struct {
	Artifact RawArtifact
	Snapshot SourceSnapshot
}

type ConfluenceSnapshotWithEventResult struct {
	Artifact RawArtifact
	Snapshot SourceSnapshot
	Event    LedgerEvent
}

type ConfluenceRangeSelection struct {
	ContentID string `json:"content_id,omitempty"`
	Start     int    `json:"start"`
	End       int    `json:"end"`
}

type ConfluenceRangeOption struct {
	ContentID string `json:"content_id"`
	Label     string `json:"label"`
	Start     int    `json:"start"`
	End       int    `json:"end"`
	RuneCount int    `json:"rune_count"`
}

type ConfluenceSourcePreviewRequest struct {
	MissionID       string
	CloudID         string
	PageID          string
	ExpectedVersion int
	MaxBodyBytes    int64
	PreviewRunes    int
}

type ConfluenceSourcePreviewResult struct {
	MissionID        string                      `json:"mission_id"`
	CandidateKind    string                      `json:"candidate_kind"`
	Page             ConfluenceSourcePreviewPage `json:"page"`
	PreviewText      string                      `json:"preview_text,omitempty"`
	PreviewTruncated bool                        `json:"preview_truncated"`
	BodyBytes        int64                       `json:"body_bytes"`
	MaxBodyBytes     int64                       `json:"max_body_bytes"`
	FullBodyTooLarge bool                        `json:"full_body_too_large"`
	RangeOptions     []ConfluenceRangeOption     `json:"range_options,omitempty"`
}

type ConfluenceSourcePreviewPage struct {
	CloudID   string    `json:"cloud_id"`
	SiteURL   string    `json:"site_url,omitempty"`
	PageID    string    `json:"page_id"`
	SpaceID   string    `json:"space_id,omitempty"`
	SpaceKey  string    `json:"space_key,omitempty"`
	Title     string    `json:"title"`
	WebURL    string    `json:"web_url,omitempty"`
	Version   int       `json:"version,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type CheckConfluenceSourceUpdateRequest struct {
	MissionID  string
	SnapshotID string
	EventID    string
	Producer   Producer
}

type ConfluenceUpdateCheckResult struct {
	Snapshot         SourceSnapshot
	CurrentVersion   int
	CurrentTitle     string
	CurrentUpdatedAt time.Time
	LatestPageID     string
	LatestSpaceID    string
	LatestSpaceKey   string
	LatestWebURL     string
	LatestVersion    int
	LatestTitle      string
	LatestUpdatedAt  time.Time
	UpdateAvailable  bool
	Event            LedgerEvent
}

type ConfluenceUpdatePreviewRequest struct {
	MissionID       string
	SnapshotID      string
	ExpectedVersion int
	MaxBodyBytes    int64
	PreviewRunes    int
}

type ConfluenceUpdatePreviewResult struct {
	Snapshot               SourceSnapshot              `json:"snapshot"`
	OldPage                ConfluenceSourcePreviewPage `json:"old_page"`
	NewPage                ConfluenceSourcePreviewPage `json:"new_page"`
	UpdateAvailable        bool                        `json:"update_available"`
	PreviewText            string                      `json:"preview_text,omitempty"`
	PreviewTruncated       bool                        `json:"preview_truncated"`
	BodyBytes              int64                       `json:"body_bytes"`
	MaxBodyBytes           int64                       `json:"max_body_bytes"`
	FullBodyTooLarge       bool                        `json:"full_body_too_large"`
	RangeOptions           []ConfluenceRangeOption     `json:"range_options,omitempty"`
	RequiresRangeReselect  bool                        `json:"requires_range_reselect"`
	PreviousRangeSelection ConfluenceRangeSelection    `json:"previous_range_selection,omitempty"`
}

type UpdateConfluenceSourceRequest struct {
	MissionID          string
	PreviousSnapshotID string
	ArtifactID         string
	SnapshotID         string
	ExpectedVersion    int
	MaxBodyBytes       int64
	Range              ConfluenceRangeSelection
	Reason             string
	SnapshotEventID    string
	UpdateEventID      string
	Producer           Producer
}

type ConfluenceUpdateResult struct {
	PreviousSnapshot SourceSnapshot
	Artifact         RawArtifact
	Snapshot         SourceSnapshot
	SnapshotEvent    LedgerEvent
	UpdateEvent      LedgerEvent
}

func ConfluenceExternalSourceID(cloudID string, pageID string) string {
	cloudID = strings.TrimSpace(cloudID)
	pageID = strings.TrimSpace(pageID)
	if cloudID == "" || pageID == "" {
		return ""
	}
	return cloudID + ":" + pageID
}

func ConfluenceExternalURI(cloudID string, pageID string) string {
	cloudID = strings.TrimSpace(cloudID)
	pageID = strings.TrimSpace(pageID)
	if cloudID == "" || pageID == "" {
		return ""
	}
	return "confluence://cloud/" + cloudID + "/pages/" + pageID
}
