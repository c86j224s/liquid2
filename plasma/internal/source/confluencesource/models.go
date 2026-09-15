package confluencesource

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
)

const (
	ConfluenceConnectorID        = "confluence"
	ConfluenceConnectorType      = "confluence_cloud"
	ConfluenceHTTPConnectorV1    = "confluence-cloud-http.v1"
	ConfluenceSnapshotMediaType  = "application/vnd.plasma.confluence.snapshot+json"
	ConfluenceSnapshotSchemaV1   = "plasma.confluence.snapshot.v1"
	defaultConfluenceSearchLimit = 10
	maxConfluenceSearchLimit     = 100
)

// ConfluenceSourceConnector는 Confluence page 검색/읽기를 제공하는 source connector port다.
type ConfluenceSourceConnector interface {
	SearchConfluenceSources(context.Context, ConfluenceSourceSearchRequest) (ConfluenceSourceSearchResult, error)
	ReadConfluenceSource(context.Context, ConfluenceSourceReadRequest) (ConfluenceSourcePage, error)
}

// ConfluenceBrowserConnector는 설정 화면/소스 선택 UI가 space와 page tree를 탐색할 때 쓰는 connector port다.
type ConfluenceBrowserConnector interface {
	ListConfluenceSpaces(context.Context, ConfluenceSpaceListRequest) (ConfluenceSpaceListResult, error)
	ListConfluenceSpacePages(context.Context, ConfluenceSpacePagesRequest) (ConfluencePageListResult, error)
	ListConfluencePageChildren(context.Context, ConfluencePageChildrenRequest) (ConfluencePageListResult, error)
}

// ConfluenceSourceVersionConnector는 기존 Confluence snapshot의 외부 version을 확인하는 port다.
type ConfluenceSourceVersionConnector interface {
	GetConfluenceSourceVersion(context.Context, ConfluenceSourceReadRequest) (ConfluenceSourceVersion, error)
}

// ConfluenceSourceSearchRequest는 Confluence source 후보 검색 범위와 cursor를 지정한다.
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

// ConfluenceSourceSearchResult는 아직 승인되지 않은 Confluence source 후보 목록이다.
type ConfluenceSourceSearchResult struct {
	MissionID  string
	CloudID    string
	Candidates []ConfluenceSourceCandidate
	NextCursor string
}

// ConfluenceSpaceListRequest는 Confluence space 목록 조회 입력이다.
type ConfluenceSpaceListRequest struct {
	MissionID string
	CloudID   string
	Limit     int
	Cursor    string
}

// ConfluenceSpaceListResult는 Confluence space 목록과 다음 cursor를 담는다.
type ConfluenceSpaceListResult struct {
	MissionID  string
	CloudID    string
	Spaces     []ConfluenceSpaceSummary
	NextCursor string
}

// ConfluenceSpaceSummary는 source 선택 UI에 필요한 space metadata다.
type ConfluenceSpaceSummary struct {
	CloudID  string `json:"cloud_id"`
	SpaceID  string `json:"space_id"`
	SpaceKey string `json:"space_key,omitempty"`
	Name     string `json:"name"`
	Type     string `json:"type,omitempty"`
	Status   string `json:"status,omitempty"`
	WebURL   string `json:"web_url,omitempty"`
}

// ConfluenceSpacePagesRequest는 특정 space의 page 목록 조회 입력이다.
type ConfluenceSpacePagesRequest struct {
	MissionID string
	CloudID   string
	SpaceID   string
	Limit     int
	Cursor    string
}

// ConfluencePageChildrenRequest는 특정 page 아래 children 조회 입력이다.
type ConfluencePageChildrenRequest struct {
	MissionID string
	CloudID   string
	PageID    string
	Limit     int
	Cursor    string
}

// ConfluencePageListResult는 Confluence page 목록과 paging 정보를 담는다.
type ConfluencePageListResult struct {
	MissionID  string
	CloudID    string
	Pages      []ConfluencePageSummary
	NextCursor string
}

// ConfluencePageSummary는 검색이나 URL 해석 결과로 받은 Confluence page의 표시 요약이다.
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

// ConfluenceSourceCandidate는 source로 추가하기 전 검증된 Confluence page 후보다.
type ConfluenceSourceCandidate struct {
	Connector   sourcecontract.ConnectorRef
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

// ConfluenceSourceReadRequest는 애플리케이션 서비스 계층에 전달되는 요청 값이다.
type ConfluenceSourceReadRequest struct {
	CloudID string
	PageID  string
}

// ConfluenceSourceVersion는 Confluence page snapshot의 외부 version metadata다.
type ConfluenceSourceVersion struct {
	Connector sourcecontract.ConnectorRef
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

// ConfluenceSourcePage는 snapshot으로 저장할 Confluence page 본문과 version metadata다.
type ConfluenceSourcePage struct {
	Connector   sourcecontract.ConnectorRef
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

// ConfluenceExternalSourceID는 Confluence page identity를 Plasma source external ID로 정규화한다.
func ConfluenceExternalSourceID(cloudID string, pageID string) string {
	cloudID = strings.TrimSpace(cloudID)
	pageID = strings.TrimSpace(pageID)
	if cloudID == "" || pageID == "" {
		return ""
	}
	return cloudID + ":" + pageID
}

// ConfluenceExternalURI는 Confluence page를 다시 찾을 수 있는 canonical URI를 만든다.
func ConfluenceExternalURI(cloudID string, pageID string) string {
	cloudID = strings.TrimSpace(cloudID)
	pageID = strings.TrimSpace(pageID)
	if cloudID == "" || pageID == "" {
		return ""
	}
	return "confluence://cloud/" + cloudID + "/pages/" + pageID
}
