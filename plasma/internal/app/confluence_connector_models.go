package app

import (
	"github.com/c86j224s/liquid2/plasma/internal/source/confluencesource"
	"time"

	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
)

const (
	ConfluenceUpdateCurrentEvent   = "source.update.current"
	ConfluenceUpdateAvailableEvent = "source.update.available"
	ConfluenceUpdateFailedEvent    = "source.update.check_failed"
	ConfluenceUpdatedEvent         = "source.updated"
	DefaultConfluenceMaxBodyBytes  = int64(1024 * 1024)
)

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
	Range               confluencesource.ConfluenceRangeSelection
	Producer            ledger.Producer
	Reason              string
	ExpectedContentHash sourcecontract.ContentHash
}

// SnapshotConfluenceSourceWithEventRequest는 애플리케이션 서비스 계층에 전달되는 요청 값이다.
type SnapshotConfluenceSourceWithEventRequest struct {
	Snapshot                       SnapshotConfluenceSourceRequest
	EventID                        string
	Producer                       ledger.Producer
	SourceCandidateProposalEventID string
	SourceCandidateURL             string
}

// ConfluenceSnapshotResult는 Confluence page snapshot과 원문 artifact를 함께 반환한다.
type ConfluenceSnapshotResult struct {
	Artifact artifactcontract.Raw
	Snapshot sourcecontract.Snapshot
}

// ConfluenceSnapshotWithEventResult는 Confluence snapshot 결과와 기록된 이벤트를 함께 반환한다.
type ConfluenceSnapshotWithEventResult struct {
	Artifact artifactcontract.Raw
	Snapshot sourcecontract.Snapshot
	Event    ledger.Event
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
	MissionID        string                                   `json:"mission_id"`
	CandidateKind    string                                   `json:"candidate_kind"`
	Page             ConfluenceSourcePreviewPage              `json:"page"`
	PreviewText      string                                   `json:"preview_text,omitempty"`
	PreviewTruncated bool                                     `json:"preview_truncated"`
	BodyBytes        int64                                    `json:"body_bytes"`
	MaxBodyBytes     int64                                    `json:"max_body_bytes"`
	FullBodyTooLarge bool                                     `json:"full_body_too_large"`
	RangeOptions     []confluencesource.ConfluenceRangeOption `json:"range_options,omitempty"`
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
	Producer   ledger.Producer
}

// ConfluenceUpdateCheckResult는 현재 snapshot과 외부 page version 비교 결과다.
type ConfluenceUpdateCheckResult struct {
	Snapshot         sourcecontract.Snapshot
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
	Event            ledger.Event
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
	Snapshot               sourcecontract.Snapshot                   `json:"snapshot"`
	OldPage                ConfluenceSourcePreviewPage               `json:"old_page"`
	NewPage                ConfluenceSourcePreviewPage               `json:"new_page"`
	UpdateAvailable        bool                                      `json:"update_available"`
	PreviewText            string                                    `json:"preview_text,omitempty"`
	PreviewTruncated       bool                                      `json:"preview_truncated"`
	BodyBytes              int64                                     `json:"body_bytes"`
	MaxBodyBytes           int64                                     `json:"max_body_bytes"`
	FullBodyTooLarge       bool                                      `json:"full_body_too_large"`
	RangeOptions           []confluencesource.ConfluenceRangeOption  `json:"range_options,omitempty"`
	RequiresRangeReselect  bool                                      `json:"requires_range_reselect"`
	PreviousRangeSelection confluencesource.ConfluenceRangeSelection `json:"previous_range_selection,omitempty"`
}

// UpdateConfluenceSourceRequest는 애플리케이션 서비스 계층에 전달되는 요청 값이다.
type UpdateConfluenceSourceRequest struct {
	MissionID          string
	PreviousSnapshotID string
	ArtifactID         string
	SnapshotID         string
	ExpectedVersion    int
	MaxBodyBytes       int64
	Range              confluencesource.ConfluenceRangeSelection
	Reason             string
	SnapshotEventID    string
	UpdateEventID      string
	Producer           ledger.Producer
}

// ConfluenceUpdateResult는 update snapshot과 기록된 이벤트를 함께 반환한다.
type ConfluenceUpdateResult struct {
	PreviousSnapshot sourcecontract.Snapshot
	Artifact         artifactcontract.Raw
	Snapshot         sourcecontract.Snapshot
	SnapshotEvent    ledger.Event
	UpdateEvent      ledger.Event
}
