package app

import (
	"context"
	"encoding/json"
)

const (
	ReportRedpenSavedEvent   = "report.redpen.saved"
	ReportRedpenArtifactKind = "redpen_markdown_report_artifact"
)

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
}

type reportRedpenRevisionStore interface {
	CommitReportRedpenRevision(
		context.Context,
		RawArtifact,
		func([]LedgerEvent, RawArtifact) (LedgerEvent, bool, error),
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
