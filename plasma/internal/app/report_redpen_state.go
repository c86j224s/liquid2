package app

import (
	"encoding/json"
	"fmt"
	"mime"
	"path/filepath"
	"strings"
)

func latestReportRedpenEvent(events []LedgerEvent, sourceArtifactID string) (reportRedpenEventPayload, LedgerEvent, bool, error) {
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].EventType != ReportRedpenSavedEvent {
			continue
		}
		payload, err := decodeReportRedpenPayload(events[i])
		if err != nil {
			return reportRedpenEventPayload{}, LedgerEvent{}, false, err
		}
		if payload.SourceArtifactID == sourceArtifactID {
			return payload, events[i], true, nil
		}
	}
	return reportRedpenEventPayload{}, LedgerEvent{}, false, nil
}

func decodeReportRedpenPayload(event LedgerEvent) (reportRedpenEventPayload, error) {
	var payload reportRedpenEventPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return payload, fmt.Errorf("%w: invalid report redpen event payload", ErrInvalidInput)
	}
	payload.Kind = strings.TrimSpace(payload.Kind)
	payload.WorkcopyID = strings.TrimSpace(payload.WorkcopyID)
	payload.SourceArtifactID = strings.TrimSpace(payload.SourceArtifactID)
	payload.ArtifactID = strings.TrimSpace(payload.ArtifactID)
	payload.PreviousArtifactID = strings.TrimSpace(payload.PreviousArtifactID)
	payload.SHA256 = strings.TrimSpace(payload.SHA256)
	payload.MediaType = strings.TrimSpace(payload.MediaType)
	payload.Filename = strings.TrimSpace(payload.Filename)
	if payload.Kind != ReportRedpenArtifactKind || payload.Revision < 1 || len(payload.SHA256) != 64 ||
		!isMarkdownArtifactMediaType(payload.MediaType) || payload.Filename == "" {
		return payload, fmt.Errorf("%w: invalid report redpen event payload", ErrInvalidInput)
	}
	if err := validateID("rwc_", payload.WorkcopyID); err != nil {
		return payload, err
	}
	if err := validateID("art_", payload.SourceArtifactID); err != nil {
		return payload, err
	}
	if err := validateID("art_", payload.ArtifactID); err != nil {
		return payload, err
	}
	if err := validateID("art_", payload.PreviousArtifactID); err != nil {
		return payload, err
	}
	return payload, nil
}

func hasReportRedpenSourceEvent(events []LedgerEvent, artifactID string) bool {
	for _, event := range events {
		if event.EventType != "report.artifact.created" && event.EventType != "report.artifact.exported" {
			continue
		}
		var payload struct {
			ArtifactID string `json:"artifact_id"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err == nil && strings.TrimSpace(payload.ArtifactID) == artifactID {
			return true
		}
	}
	return false
}

func reportRedpenWorkcopy(payload reportRedpenEventPayload, artifact RawArtifact, event LedgerEvent, changed bool) ReportRedpenWorkcopy {
	return ReportRedpenWorkcopy{
		Exists:             true,
		WorkcopyID:         payload.WorkcopyID,
		SourceArtifactID:   payload.SourceArtifactID,
		PreviousArtifactID: payload.PreviousArtifactID,
		Revision:           payload.Revision,
		MediaType:          payload.MediaType,
		Filename:           payload.Filename,
		Artifact:           artifact,
		Event:              event,
		Changed:            changed,
	}
}

func isMarkdownArtifactMediaType(mediaType string) bool {
	base, _, err := mime.ParseMediaType(strings.TrimSpace(mediaType))
	if err != nil {
		base = mediaType
	}
	base = strings.ToLower(strings.TrimSpace(base))
	return base == "text/markdown" || base == "text/x-markdown"
}

func reportRedpenFilename(source RawArtifact) string {
	name := filepath.Base(strings.TrimSpace(source.Filename))
	base := strings.TrimSuffix(name, filepath.Ext(name))
	if base == "" || base == "." {
		base = source.ArtifactID
	}
	return base + "-redpen.md"
}
