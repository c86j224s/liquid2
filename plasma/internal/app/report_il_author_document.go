package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

// ReadReportILAuthorDocument verifies that one request-local provider session
// finalized exactly one server-owned authoring workspace, then returns the
// finalized artifact. Tool traces contain only workspace identity and byte
// metadata; manuscript content stays in the artifact store.
func (s *Service) ReadReportILAuthorDocument(
	ctx context.Context,
	missionID,
	toolSessionID string,
	catalog reportilcontract.SourceCatalog,
) (reportilcontract.AuthorDocument, reportilcontract.AuthorWorkspaceReceipt, error) {
	return s.readReportILDocument(ctx, missionID, toolSessionID, "il_narrative", catalog)
}

// ReadReportILContinuityDocument verifies the finalized workspace emitted by
// the source-account continuity editor and returns the revised manuscript.
func (s *Service) ReadReportILContinuityDocument(
	ctx context.Context,
	missionID,
	toolSessionID string,
	catalog reportilcontract.SourceCatalog,
) (reportilcontract.AuthorDocument, reportilcontract.AuthorWorkspaceReceipt, error) {
	return s.readReportILDocument(ctx, missionID, toolSessionID, "il_continuity", catalog)
}

// ReadReportILPublicationDocument verifies the finalized workspace emitted by
// the publication reader and returns the revised authoritative manuscript.
func (s *Service) ReadReportILPublicationDocument(
	ctx context.Context,
	missionID,
	toolSessionID string,
	catalog reportilcontract.SourceCatalog,
) (reportilcontract.AuthorDocument, reportilcontract.AuthorWorkspaceReceipt, error) {
	return s.readReportILDocument(ctx, missionID, toolSessionID, "il_reader", catalog)
}

// ReadReportILLongFormStageDocument binds one Section, Part, or final authoring
// session to exactly one server-owned hierarchical manuscript artifact.
func (s *Service) ReadReportILLongFormStageDocument(
	ctx context.Context,
	missionID,
	toolSessionID,
	stage string,
	catalog reportilcontract.SourceCatalog,
) (reportilcontract.AuthorDocument, reportilcontract.AuthorWorkspaceReceipt, error) {
	return s.readReportILDocument(ctx, missionID, toolSessionID, stage, catalog)
}

func (s *Service) readReportILDocument(
	ctx context.Context,
	missionID,
	toolSessionID,
	stage string,
	catalog reportilcontract.SourceCatalog,
) (reportilcontract.AuthorDocument, reportilcontract.AuthorWorkspaceReceipt, error) {
	missionID = strings.TrimSpace(missionID)
	toolSessionID = strings.TrimSpace(toolSessionID)
	if !strings.HasPrefix(missionID, "mis_") || !strings.HasPrefix(toolSessionID, "ses_") ||
		(stage != "il_narrative" && stage != "il_continuity" && stage != "il_reader" &&
			stage != "il_long_form_section" && stage != "il_long_form_part" && stage != "il_long_form_final") ||
		catalog.MissionID != missionID {
		return reportilcontract.AuthorDocument{}, reportilcontract.AuthorWorkspaceReceipt{}, fmt.Errorf("report IL document binding is invalid")
	}
	if err := reportilcontract.ValidateSourceCatalog(catalog); err != nil {
		return reportilcontract.AuthorDocument{}, reportilcontract.AuthorWorkspaceReceipt{}, err
	}
	events, err := s.ListEvents(ctx, missionID)
	if err != nil {
		return reportilcontract.AuthorDocument{}, reportilcontract.AuthorWorkspaceReceipt{}, err
	}
	type toolPayload struct {
		ToolName      string `json:"tool_name"`
		ToolSessionID string `json:"tool_session_id"`
		Success       bool   `json:"success"`
		IOMetrics     struct {
			ReportILStage string `json:"report_il_stage"`
			WorkspaceID   string `json:"workspace_id"`
			ArtifactID    string `json:"artifact_id"`
			SHA256        string `json:"sha256"`
			ByteSize      int    `json:"byte_size"`
			Revision      int    `json:"revision"`
			Replacements  int    `json:"replacements"`
			Finalized     bool   `json:"finalized"`
		} `json:"io_metrics"`
	}
	var receipt reportilcontract.AuthorWorkspaceReceipt
	finalizeCount := 0
	for _, event := range events {
		if event.EventType != "mcp.tool.called" || event.CorrelationID != toolSessionID {
			continue
		}
		var payload toolPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return reportilcontract.AuthorDocument{}, reportilcontract.AuthorWorkspaceReceipt{}, fmt.Errorf("report IL author document trace is malformed")
		}
		if payload.ToolSessionID != toolSessionID || !payload.Success ||
			!reportILDocumentFinalizeToolForStage(payload.ToolName, stage) ||
			payload.IOMetrics.ReportILStage != stage {
			continue
		}
		finalizeCount++
		if finalizeCount > 1 || !payload.IOMetrics.Finalized ||
			!strings.HasPrefix(payload.IOMetrics.WorkspaceID, "ilw_") ||
			!strings.HasPrefix(payload.IOMetrics.ArtifactID, "art_") ||
			len(payload.IOMetrics.SHA256) != 64 || payload.IOMetrics.ByteSize < 1 || payload.IOMetrics.Revision < 1 {
			return reportilcontract.AuthorDocument{}, reportilcontract.AuthorWorkspaceReceipt{}, fmt.Errorf("report IL author document trace is invalid")
		}
		receipt = reportilcontract.AuthorWorkspaceReceipt{
			WorkspaceID:  payload.IOMetrics.WorkspaceID,
			ArtifactID:   payload.IOMetrics.ArtifactID,
			SHA256:       payload.IOMetrics.SHA256,
			ByteSize:     payload.IOMetrics.ByteSize,
			Revision:     payload.IOMetrics.Revision,
			Stage:        stage,
			Replacements: payload.IOMetrics.Replacements,
		}
	}
	if finalizeCount != 1 {
		return reportilcontract.AuthorDocument{}, reportilcontract.AuthorWorkspaceReceipt{}, fmt.Errorf("report IL author document was not finalized exactly once")
	}
	artifact, err := s.GetRawArtifact(ctx, receipt.ArtifactID)
	if err != nil {
		return reportilcontract.AuthorDocument{}, reportilcontract.AuthorWorkspaceReceipt{}, err
	}
	contentSum := sha256.Sum256(artifact.Content)
	validProducer := artifact.Producer.Type == "mcp_tool" &&
		artifact.Producer.ID == reportilcontract.AuthorDocumentFinalizeTool
	if strings.HasPrefix(stage, "il_long_form_") ||
		(stage == "il_reader" || stage == "il_continuity") && receipt.Replacements == 0 {
		validProducer = validProducer || artifact.Producer.Type == "mcp_tool" &&
			artifact.Producer.ID == reportilcontract.LongFormDocumentFinalizeTool
	}
	if artifact.MissionID != missionID || artifact.MediaType != reportilcontract.AuthorDocumentMediaType ||
		!validProducer ||
		artifact.SHA256 != receipt.SHA256 || hex.EncodeToString(contentSum[:]) != receipt.SHA256 ||
		int(artifact.ByteSize) != receipt.ByteSize || len(artifact.Content) != receipt.ByteSize {
		return reportilcontract.AuthorDocument{}, reportilcontract.AuthorWorkspaceReceipt{}, fmt.Errorf("report IL author document artifact binding is invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(artifact.Content))
	decoder.DisallowUnknownFields()
	var document reportilcontract.AuthorDocument
	if err := decoder.Decode(&document); err != nil {
		return reportilcontract.AuthorDocument{}, reportilcontract.AuthorWorkspaceReceipt{}, fmt.Errorf("report IL author document decode failed: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return reportilcontract.AuthorDocument{}, reportilcontract.AuthorWorkspaceReceipt{}, fmt.Errorf("report IL author document contains multiple JSON values")
	}
	var validationErr error
	if stage == "il_long_form_final" {
		validationErr = reportilcontract.ValidateLongFormAuthorDocument(document, catalog)
	} else if stage == "il_long_form_section" || stage == "il_long_form_part" {
		validationErr = reportilcontract.ValidateLongFormAuthorFragment(document, catalog)
	} else {
		validationErr = reportilcontract.ValidateAuthorDocument(document, catalog)
	}
	if validationErr != nil {
		return reportilcontract.AuthorDocument{}, reportilcontract.AuthorWorkspaceReceipt{}, validationErr
	}
	return document, receipt, nil
}

func reportILDocumentFinalizeToolForStage(toolName, stage string) bool {
	if strings.HasPrefix(stage, "il_long_form_") {
		return toolName == reportilcontract.LongFormDocumentFinalizeTool
	}
	return toolName == reportilcontract.AuthorDocumentFinalizeTool
}
