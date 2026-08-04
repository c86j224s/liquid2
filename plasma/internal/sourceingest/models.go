package sourceingest

import (
	"context"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

// Store는 소스 수집이 필요로 하는 저장소 포트다. 구현체는 artifact,
// snapshot, event를 하나의 제품 상태 전이로 기록해야 하며, 이 패키지는 SQL
// 형태나 트랜잭션 세부 구현을 직접 알지 않는다.
type Store interface {
	ListSourceSnapshots(context.Context, string) ([]app.SourceSnapshot, error)
	ListEvents(context.Context, string) ([]app.LedgerEvent, error)
	GetRawArtifact(context.Context, string) (app.RawArtifact, error)
	CreateSourceSnapshotWithEvent(context.Context, app.CreateSourceSnapshotWithEventRequest) (app.SourceSnapshotWithEventResult, error)
	CreateExistingArtifactSourceSnapshotWithEvent(context.Context, app.CreateExistingArtifactSourceSnapshotWithEventRequest) (app.ExistingArtifactSourceSnapshotWithEventResult, error)
	CreateLiveSourceSnapshotWithEvent(context.Context, app.CreateLiveSourceSnapshotWithEventRequest) (app.LiveSourceSnapshotWithEventResult, error)
	AppendEvent(context.Context, app.AppendEventRequest) (app.LedgerEvent, error)
}

// Producer는 소스 수집 이벤트에 기록할 생산자 정보를 app 계층과 같은 형태로 공유한다.
type Producer = app.Producer

// RawArtifact는 수집된 원문 본문과 저장 식별자를 함께 전달하는 artifact 계약이다.
type RawArtifact = app.RawArtifact

// SourceSnapshot는 수집 결과가 승인된 source 상태로 남았음을 나타내는 snapshot 계약이다.
type SourceSnapshot = app.SourceSnapshot

// LedgerEvent는 artifact와 snapshot 저장을 설명하는 장부 이벤트 계약이다.
type LedgerEvent = app.LedgerEvent

// SourceSnapshotWithEventResult는 새 artifact로 만든 source snapshot과 기록 이벤트를 함께 반환한다.
type SourceSnapshotWithEventResult = app.SourceSnapshotWithEventResult

// ExistingArtifactSourceSnapshotWithEventResult는 기존 artifact를 재사용한 snapshot 결과와 이벤트를 반환한다.
type ExistingArtifactSourceSnapshotWithEventResult = app.ExistingArtifactSourceSnapshotWithEventResult

// LiveSourceSnapshotWithEventResult는 원문 artifact 없이 live reference snapshot과 이벤트를 반환한다.
type LiveSourceSnapshotWithEventResult = app.LiveSourceSnapshotWithEventResult

// AppendEventRequest는 소스 수집 조립 경계에 전달되는 요청 값이다.
type AppendEventRequest = app.AppendEventRequest

// CreateRawArtifactRequest는 소스 수집 조립 경계에 전달되는 요청 값이다.
type CreateRawArtifactRequest = app.CreateRawArtifactRequest

// CreateSourceSnapshotRequest는 소스 수집 조립 경계에 전달되는 요청 값이다.
type CreateSourceSnapshotRequest = app.CreateSourceSnapshotRequest

// CreateSourceSnapshotWithEventRequest는 소스 수집 조립 경계에 전달되는 요청 값이다.
type CreateSourceSnapshotWithEventRequest = app.CreateSourceSnapshotWithEventRequest

// CreateExistingArtifactSourceSnapshotWithEventRequest는 소스 수집 조립 경계에 전달되는 요청 값이다.
type CreateExistingArtifactSourceSnapshotWithEventRequest = app.CreateExistingArtifactSourceSnapshotWithEventRequest

// CreateLiveSourceSnapshotWithEventRequest는 소스 수집 조립 경계에 전달되는 요청 값이다.
type CreateLiveSourceSnapshotWithEventRequest = app.CreateLiveSourceSnapshotWithEventRequest

// ConnectorRef는 외부 connector와 source를 연결할 때 쓰는 안정 참조다.
type ConnectorRef = app.ConnectorRef

// MediaLocator는 미디어 source 안에서 실제 대상 조각을 가리키는 locator 계약이다.
type MediaLocator = app.MediaLocator

// SourceAccess는 agent가 source 본문을 읽을 수 있는지와 어떤 경로로 읽는지를 나타낸다.
type SourceAccess = app.SourceAccess

const SourceLocatorTypeFullDocument = app.SourceLocatorTypeFullDocument
const SourceLocatorTypePDFDocument = app.SourceLocatorTypePDFDocument
const SourceLocatorTypeMedia = app.SourceLocatorTypeMedia
const SourceConnectorTypePDFURL = app.SourceConnectorTypePDFURL
const SourceConnectorTypeMediaURL = app.SourceConnectorTypeMediaURL
const MediaKindImage = app.MediaKindImage
const MediaKindAudio = app.MediaKindAudio
const MediaKindVideo = app.MediaKindVideo
const SourceRetrievalPolicyLiveReference = app.SourceRetrievalPolicyLiveReference

var ErrInvalidInput = app.ErrInvalidInput

// FetchedURLSource는 URL fetch 어댑터가 이미 검증해 돌려준 문서 본문과
// 메타데이터다. RetrievalMethod와 RawFetch* 필드는 브라우저 렌더링처럼
// 원문 fetch와 최종 저장 본문이 다를 때 추적성을 보존한다.
type FetchedURLSource struct {
	Content            []byte
	MediaType          string
	Title              string
	ExternalVersion    string
	ExternalUpdatedAt  time.Time
	ByteSize           int64
	PageCount          int
	TextLength         int
	TextLengthKnown    bool
	RetrievalMethod    string
	FinalURL           string
	RenderedAt         time.Time
	RawFetchSHA256     string
	RawFetchArtifactID string
}

// FetchedMediaSource는 미디어 URL fetch 결과다. 이미지는 artifact로 저장될 수
// 있지만, 오디오와 비디오는 현재 live reference snapshot으로 남을 수 있다.
type FetchedMediaSource struct {
	Content           []byte
	MediaType         string
	MediaKind         string
	Title             string
	ExternalVersion   string
	ExternalUpdatedAt time.Time
	ByteSize          int64
	Width             int
	Height            int
}

// TextSourceContent는 사용자가 붙여넣은 본문 소스다. ExternalURI가 비어 있으면
// 저장용 외부 ID 대신 manual snapshot ID가 사용된다.
type TextSourceContent struct {
	Title       string
	Content     string
	ExternalURI string
}

// StagedSourceCandidate는 제안 시점에 미리 가져와 둔 후보 artifact를 승인
// 시점에 재사용하기 위한 값이다. Artifact는 같은 mission에 속해야 하며,
// 승인 전에는 source snapshot으로 간주하지 않는다.
type StagedSourceCandidate struct {
	URL               string
	Title             string
	ProposalEventID   string
	Artifact          RawArtifact
	ExternalVersion   string
	ExternalUpdatedAt time.Time
}

// URLSourceSnapshotResult는 URL/PDF 계열 ingest가 만든 artifact, snapshot,
// event를 함께 돌려준다. ReusedSourceCandidate가 true이면 새 fetch 없이 staged
// artifact를 snapshot으로 승격한 것이다.
type URLSourceSnapshotResult struct {
	Artifact              RawArtifact
	Snapshot              SourceSnapshot
	Event                 LedgerEvent
	ReusedSourceCandidate bool
}

// MediaSourceSnapshotResult는 미디어 ingest 결과다. HasArtifact가 false이면
// live reference snapshot만 만들었고 로컬 artifact는 만들지 않았다는 뜻이다.
type MediaSourceSnapshotResult struct {
	Artifact    RawArtifact
	HasArtifact bool
	Snapshot    SourceSnapshot
	Event       LedgerEvent
}

// CreateFetchedURLSourceRequest는 새 URL fetch 결과를 source snapshot으로
// 저장하기 위한 입력이다. ID들은 호출자가 생성해 전달하며, 이 함수군은 그 ID를
// 제품 원장의 안정 식별자로 그대로 사용한다.
type CreateFetchedURLSourceRequest struct {
	MissionID                      string
	URL                            string
	Title                          string
	ArtifactID                     string
	SnapshotID                     string
	EventID                        string
	Producer                       Producer
	Fetched                        FetchedURLSource
	FetchedAt                      time.Time
	SourceCandidateProposalEventID string
}

// SourceSnapshotFailureAppendRequest는 소스 스냅샷 생성 실패를 원장에 남기기
// 위한 입력이다. 실패 이벤트는 디버그 로그가 아니라 사용자가 볼 수 있는 안정
// 상태 전이여야 한다.
type SourceSnapshotFailureAppendRequest struct {
	EventID    string
	MissionID  string
	SourceKind string
	URL        string
	Message    string
	Producer   Producer
}

// CreateTextSourceWithEventRequest는 붙여넣기 텍스트를 artifact와 snapshot으로
// 저장하기 위한 입력이다.
type CreateTextSourceWithEventRequest struct {
	MissionID  string
	ArtifactID string
	SnapshotID string
	EventID    string
	Producer   Producer
	Source     TextSourceContent
}

// CreateStagedURLSourceRequest는 staged URL 후보의 기존 artifact를 승인된
// source snapshot으로 승격하기 위한 입력이다.
type CreateStagedURLSourceRequest struct {
	MissionID  string
	URL        string
	Title      string
	SnapshotID string
	EventID    string
	Producer   Producer
	Staged     StagedSourceCandidate
}

// CreateFetchedPDFURLSourceRequest는 새로 fetch한 PDF URL을 artifact와 PDF
// source snapshot으로 저장하기 위한 입력이다.
type CreateFetchedPDFURLSourceRequest struct {
	MissionID  string
	URL        string
	Title      string
	ArtifactID string
	SnapshotID string
	EventID    string
	Producer   Producer
	Fetched    FetchedURLSource
	FetchedAt  time.Time
}

// CreateFetchedMediaURLSourceRequest는 미디어 URL fetch 결과를 source snapshot
// 으로 저장하기 위한 입력이다. License가 비어 있으면 unknown으로 기록된다.
type CreateFetchedMediaURLSourceRequest struct {
	MissionID   string
	URL         string
	Title       string
	License     string
	Attribution string
	ArtifactID  string
	SnapshotID  string
	EventID     string
	Producer    Producer
	Fetched     FetchedMediaSource
}

// CreateStagedPDFURLSourceRequest는 staged PDF 후보의 기존 artifact를 PDF
// source snapshot으로 승격하기 위한 입력이다.
type CreateStagedPDFURLSourceRequest struct {
	MissionID  string
	URL        string
	Title      string
	SnapshotID string
	EventID    string
	Producer   Producer
	Staged     StagedSourceCandidate
}
