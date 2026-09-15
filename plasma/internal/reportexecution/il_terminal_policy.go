package reportexecution

import (
	"bytes"
	"encoding/json"
	"fmt"
	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
	"io"
	"strings"
)

func ValidateReportILEventBindings(req ledger.AppendRequest, missionID, pendingID string) error {
	if err := validateID("evt_", strings.TrimSpace(req.EventID)); err != nil {
		return fmt.Errorf("%w: report IL event id is invalid", producterror.ErrInvalidInput)
	}
	if strings.TrimSpace(req.MissionID) != missionID || strings.TrimSpace(req.CausationEventID) != pendingID || strings.TrimSpace(req.CorrelationID) != pendingID {
		return fmt.Errorf("%w: report IL event lineage binding is invalid", producterror.ErrInvalidInput)
	}
	return nil
}

func ValidateReportILStoreCompletedPayload(raw []byte, pendingID string) error {
	var payload struct {
		Kind           string `json:"kind"`
		PendingEventID string `json:"pending_event_id"`
		PipelineFamily string `json:"pipeline_family"`
		Stage          string `json:"stage"`
		Status         string `json:"status"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return fmt.Errorf("%w: invalid report IL store receipt", producterror.ErrInvalidInput)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("%w: invalid report IL store receipt", producterror.ErrInvalidInput)
	}
	if payload.Kind != "report_il_stage_progress" || payload.PendingEventID != pendingID || payload.PipelineFamily != reportilcontract.PipelineFamily || payload.Stage != "il_store" || payload.Status != "completed" {
		return fmt.Errorf("%w: invalid report IL store receipt", producterror.ErrInvalidInput)
	}
	return nil
}

func ValidateReportILTerminalLineage(event ledger.AppendRequest, pendingID string, artifacts []artifactcontract.Raw) error {
	payload, err := reportilcontract.DecodeTerminalPayload(event.Payload)
	if err != nil {
		return fmt.Errorf("%w: invalid report IL terminal lineage: %v", producterror.ErrInvalidInput, err)
	}
	if payload.PendingEventID != pendingID {
		return fmt.Errorf("%w: report IL terminal pending binding is invalid", producterror.ErrInvalidInput)
	}
	actual := make([]reportilcontract.Artifact, 0, len(artifacts))
	for _, item := range artifacts {
		actual = append(actual, reportilcontract.Artifact{ArtifactID: item.ArtifactID, MissionID: item.MissionID, MediaType: item.MediaType, Filename: item.Filename, ByteSize: item.ByteSize, SHA256: item.SHA256, Content: item.Content})
	}
	if err := reportilcontract.ValidateActual(payload.Bundle, actual); err != nil {
		return fmt.Errorf("%w: invalid report IL terminal lineage: %v", producterror.ErrInvalidInput, err)
	}
	return nil
}

func ReportILPendingEvent(events []ledger.Event, missionID, pendingID string) (ledger.Event, bool) {
	for _, event := range events {
		if event.EventID == pendingID && event.MissionID == missionID && event.EventType == "report.draft.pending" {
			return event, true
		}
	}
	return ledger.Event{}, false
}

func ValidateReportILPendingFamily(event ledger.Event) error {
	var payload struct {
		PipelineFamily string `json:"pipeline_family"`
	}
	decoder := json.NewDecoder(bytes.NewReader(event.Payload))
	if err := decoder.Decode(&payload); err != nil || payload.PipelineFamily != reportilcontract.PipelineFamily {
		return fmt.Errorf("%w: report IL pending event is not experimental", producterror.ErrInvalidInput)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("%w: report IL pending event is malformed", producterror.ErrInvalidInput)
	}
	return nil
}
