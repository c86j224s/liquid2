package app

import (
	"context"
	"encoding/json"
)

const (
	ReportRedpenSavedEvent                  = "report.redpen.saved"
	ReportRedpenArtifactKind                = "redpen_markdown_report_artifact"
	ReportRedpenArtifactOwnershipCreated    = "created"
	ReportRedpenArtifactOwnershipReferenced = "referenced"
)

// SaveReportRedpenRequest는 애플리케이션 서비스 계층에 전달되는 요청 값이다.
type SaveReportRedpenRequest struct {
	EventID                   string
	ArtifactID                string
	NewWorkcopyID             string
	MissionID                 string
	SourceArtifactID          string
	ExpectedCurrentArtifactID string
	Producer                  Producer
	Content                   []byte
}

// ReportRedpenWorkcopy는 report redpen 편집 화면의 현재 작업본 artifact다.
type ReportRedpenWorkcopy struct {
	Exists             bool
	WorkcopyID         string
	SourceArtifactID   string
	PreviousArtifactID string
	Revision           int
	MediaType          string
	Filename           string
	Artifact           RawArtifact
	Event              LedgerEvent
	Changed            bool
}

type reportRedpenEventPayload struct {
	Kind               string `json:"kind"`
	WorkcopyID         string `json:"workcopy_id"`
	SourceArtifactID   string `json:"source_artifact_id"`
	ArtifactID         string `json:"artifact_id"`
	PreviousArtifactID string `json:"previous_artifact_id"`
	Revision           int    `json:"revision"`
	SHA256             string `json:"sha256"`
	MediaType          string `json:"media_type"`
	Filename           string `json:"filename"`
	ArtifactOwnership  string `json:"artifact_ownership"`
}

type reportRedpenRevisionStore interface {
	CommitReportRedpenRevision(
		context.Context,
		RawArtifact,
		func([]LedgerEvent, RawArtifact, string) (LedgerEvent, bool, error),
	) (RawArtifact, LedgerEvent, bool, error)
}

func (payload reportRedpenEventPayload) appendRequest(req SaveReportRedpenRequest, current LedgerEvent) (AppendEventRequest, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return AppendEventRequest{}, err
	}
	return AppendEventRequest{
		EventID:          req.EventID,
		MissionID:        req.MissionID,
		EventType:        ReportRedpenSavedEvent,
		Producer:         req.Producer,
		CausationEventID: current.EventID,
		CorrelationID:    payload.WorkcopyID,
		Payload:          encoded,
	}, nil
}
