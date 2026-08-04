package app

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/confluenceaccess"
)

const (
	// ConfluenceConnectorID와 관련 상수는 Confluence source의 connector identity와
	// snapshot media/schema version을 정의한다.
	ConfluenceConnectorID          = confluenceaccess.ConnectorID
	ConfluenceConnectorType        = "confluence_cloud"
	ConfluenceHTTPConnectorV1      = "confluence-cloud-http.v1"
	ConfluenceSnapshotMediaType    = "application/vnd.plasma.confluence.snapshot+json"
	ConfluenceSnapshotSchemaV1     = "plasma.confluence.snapshot.v1"
	ConfluenceAuthTypeOAuth        = confluenceaccess.AuthOAuth
	ConfluenceAuthTypeAPIToken     = confluenceaccess.AuthAPIToken
	ConfluenceUpdateCurrentEvent   = "source.update.current"
	ConfluenceUpdateAvailableEvent = "source.update.available"
	ConfluenceUpdateFailedEvent    = "source.update.check_failed"
	ConfluenceUpdatedEvent         = "source.updated"
	defaultConfluenceSearchLimit   = 10
	maxConfluenceSearchLimit       = 100
	DefaultConfluenceMaxBodyBytes  = int64(1024 * 1024)
)

// ConfluenceSourceConnector는 Confluence page 검색/읽기를 제공하는 source connector port다.
//
// 구현체는 외부 API 호출을 맡고, 미션 접근 권한과 source 승인 정책은 app service가
// 별도로 적용한다.
type ConfluenceSourceConnector interface {
	SearchConfluenceSources(context.Context, ConfluenceSourceSearchRequest) (ConfluenceSourceSearchResult, error)
	ReadConfluenceSource(context.Context, ConfluenceSourceReadRequest) (ConfluenceSourcePage, error)
}

// ConfluenceBrowserConnector는 설정 화면/소스 선택 UI가 space와 page tree를 탐색할
// 때 쓰는 connector port다.
type ConfluenceBrowserConnector interface {
	ListConfluenceSpaces(context.Context, ConfluenceSpaceListRequest) (ConfluenceSpaceListResult, error)
	ListConfluenceSpacePages(context.Context, ConfluenceSpacePagesRequest) (ConfluencePageListResult, error)
	ListConfluencePageChildren(context.Context, ConfluencePageChildrenRequest) (ConfluencePageListResult, error)
}

// ConfluenceSiteLister는 OAuth/API token 연결에서 접근 가능한 site 목록을 조회하는 port다.
type ConfluenceSiteLister = confluenceaccess.SiteLister

// ConfluenceSourceVersionConnector는 기존 Confluence snapshot의 외부 version을 확인하는 port다.
type ConfluenceSourceVersionConnector interface {
	GetConfluenceSourceVersion(context.Context, ConfluenceSourceReadRequest) (ConfluenceSourceVersion, error)
}

// ConfluenceSite는 connection이 접근 가능한 Confluence site 하나를 나타낸다.
type ConfluenceSite = confluenceaccess.Site

// ConfluenceSiteListResult는 접근 가능한 site 목록 조회 결과다.
type ConfluenceSiteListResult = confluenceaccess.SiteListResult

// ConfluenceConnection은 사용자가 등록한 Confluence credential과 site 목록의 저장 레코드다.
//
// AccessToken과 RefreshToken은 JSON 응답에서 제외되어야 하며, 로그에도 직접 노출하면
// 안 된다.
type ConfluenceConnection = confluenceaccess.Connection

// UpsertConfluenceConnectionRequest는 Confluence connection을 생성하거나 갱신하는 입력이다.
type UpsertConfluenceConnectionRequest = confluenceaccess.UpsertRequest

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

// ConfluenceSourceReadRequest는 애플리케이션 서비스 계층에 전달되는 요청 값이다.
type ConfluenceSourceReadRequest struct {
	CloudID string
	PageID  string
}

// ConfluenceSourceVersion는 Confluence page snapshot의 외부 version metadata다.
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

// ConfluenceSourcePage는 snapshot으로 저장할 Confluence page 본문과 version metadata다.
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

// SnapshotConfluenceSourceRequest는 애플리케이션 서비스 계층에 전달되는 요청 값이다.
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

// SnapshotConfluenceSourceWithEventRequest는 애플리케이션 서비스 계층에 전달되는 요청 값이다.
type SnapshotConfluenceSourceWithEventRequest struct {
	Snapshot SnapshotConfluenceSourceRequest
	EventID  string
	Producer Producer
}

// ConfluenceSnapshotResult는 Confluence page snapshot과 원문 artifact를 함께 반환한다.
type ConfluenceSnapshotResult struct {
	Artifact RawArtifact
	Snapshot SourceSnapshot
}

// ConfluenceSnapshotWithEventResult는 Confluence snapshot 결과와 기록된 이벤트를 함께 반환한다.
type ConfluenceSnapshotWithEventResult struct {
	Artifact RawArtifact
	Snapshot SourceSnapshot
	Event    LedgerEvent
}

// ConfluenceRangeSelection는 Confluence page 본문 중 source로 삼을 범위를 나타낸다.
type ConfluenceRangeSelection struct {
	ContentID string `json:"content_id,omitempty"`
	Start     int    `json:"start"`
	End       int    `json:"end"`
}

// ConfluenceRangeOption는 애플리케이션 서비스 계층 실행 옵션이다. 0 값과 누락 값의 의미는 생성자나 Normalize 경계가 정한다.
type ConfluenceRangeOption struct {
	ContentID string `json:"content_id"`
	Label     string `json:"label"`
	Start     int    `json:"start"`
	End       int    `json:"end"`
	RuneCount int    `json:"rune_count"`
}

// ConfluenceSourcePreviewRequest는 애플리케이션 서비스 계층에 전달되는 요청 값이다.
type ConfluenceSourcePreviewRequest struct {
	MissionID       string
	CloudID         string
	PageID          string
	ExpectedVersion int
	MaxBodyBytes    int64
	PreviewRunes    int
}

// ConfluenceSourcePreviewResult는 source 추가 전 preview page와 선택 범위를 반환한다.
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

// ConfluenceSourcePreviewPage는 source 추가 전 사용자에게 보여줄 Confluence page preview다.
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

// CheckConfluenceSourceUpdateRequest는 애플리케이션 서비스 계층에 전달되는 요청 값이다.
type CheckConfluenceSourceUpdateRequest struct {
	MissionID  string
	SnapshotID string
	EventID    string
	Producer   Producer
}

// ConfluenceUpdateCheckResult는 현재 snapshot과 외부 page version 비교 결과다.
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

// ConfluenceUpdatePreviewRequest는 애플리케이션 서비스 계층에 전달되는 요청 값이다.
type ConfluenceUpdatePreviewRequest struct {
	MissionID       string
	SnapshotID      string
	ExpectedVersion int
	MaxBodyBytes    int64
	PreviewRunes    int
}

// ConfluenceUpdatePreviewResult는 update 전 사용자가 볼 page preview와 변경 여부다.
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

// UpdateConfluenceSourceRequest는 애플리케이션 서비스 계층에 전달되는 요청 값이다.
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

// ConfluenceUpdateResult는 update snapshot과 기록된 이벤트를 함께 반환한다.
type ConfluenceUpdateResult struct {
	PreviousSnapshot SourceSnapshot
	Artifact         RawArtifact
	Snapshot         SourceSnapshot
	SnapshotEvent    LedgerEvent
	UpdateEvent      LedgerEvent
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
