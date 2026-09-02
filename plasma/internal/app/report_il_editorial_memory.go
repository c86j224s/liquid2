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

func (s *Service) ReadReportILEditorialMemory(
	ctx context.Context,
	missionID,
	toolSessionID string,
	catalog reportilcontract.SourceCatalog,
) (reportilcontract.EditorialMemory, reportilcontract.EditorialMemoryReceipt, error) {
	missionID = strings.TrimSpace(missionID)
	toolSessionID = strings.TrimSpace(toolSessionID)
	if !strings.HasPrefix(missionID, "mis_") || !strings.HasPrefix(toolSessionID, "ses_") || catalog.MissionID != missionID {
		return reportilcontract.EditorialMemory{}, reportilcontract.EditorialMemoryReceipt{}, fmt.Errorf("editorial memory binding is invalid")
	}
	if err := reportilcontract.ValidateSourceCatalog(catalog); err != nil {
		return reportilcontract.EditorialMemory{}, reportilcontract.EditorialMemoryReceipt{}, err
	}
	events, err := s.ListEvents(ctx, missionID)
	if err != nil {
		return reportilcontract.EditorialMemory{}, reportilcontract.EditorialMemoryReceipt{}, err
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
			Accounts      int    `json:"accounts"`
			Finalized     bool   `json:"finalized"`
		} `json:"io_metrics"`
	}
	var receipt reportilcontract.EditorialMemoryReceipt
	finalizeCount := 0
	for _, event := range events {
		if event.EventType != "mcp.tool.called" || event.CorrelationID != toolSessionID {
			continue
		}
		var payload toolPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return reportilcontract.EditorialMemory{}, reportilcontract.EditorialMemoryReceipt{}, fmt.Errorf("editorial memory trace is malformed")
		}
		if payload.ToolSessionID != toolSessionID || !payload.Success ||
			payload.ToolName != reportilcontract.EditorialMemoryFinalizeTool ||
			payload.IOMetrics.ReportILStage != "il_editorial_memory" {
			continue
		}
		finalizeCount++
		if finalizeCount > 1 || !payload.IOMetrics.Finalized ||
			!strings.HasPrefix(payload.IOMetrics.WorkspaceID, "ilm_") ||
			!strings.HasPrefix(payload.IOMetrics.ArtifactID, "art_") ||
			len(payload.IOMetrics.SHA256) != 64 || payload.IOMetrics.ByteSize < 1 ||
			payload.IOMetrics.Revision < 1 || payload.IOMetrics.Accounts < 1 {
			return reportilcontract.EditorialMemory{}, reportilcontract.EditorialMemoryReceipt{}, fmt.Errorf("editorial memory trace is invalid")
		}
		receipt = reportilcontract.EditorialMemoryReceipt{
			WorkspaceID: payload.IOMetrics.WorkspaceID, ArtifactID: payload.IOMetrics.ArtifactID,
			SHA256: payload.IOMetrics.SHA256, ByteSize: payload.IOMetrics.ByteSize,
			Revision: payload.IOMetrics.Revision, Accounts: payload.IOMetrics.Accounts,
		}
	}
	if finalizeCount != 1 {
		return reportilcontract.EditorialMemory{}, reportilcontract.EditorialMemoryReceipt{}, fmt.Errorf("editorial memory was not finalized exactly once")
	}
	artifact, err := s.GetRawArtifact(ctx, receipt.ArtifactID)
	if err != nil {
		return reportilcontract.EditorialMemory{}, reportilcontract.EditorialMemoryReceipt{}, err
	}
	contentSum := sha256.Sum256(artifact.Content)
	if artifact.MissionID != missionID || artifact.MediaType != reportilcontract.EditorialMemoryMediaType ||
		artifact.SHA256 != receipt.SHA256 || hex.EncodeToString(contentSum[:]) != receipt.SHA256 ||
		int(artifact.ByteSize) != receipt.ByteSize || len(artifact.Content) != receipt.ByteSize {
		return reportilcontract.EditorialMemory{}, reportilcontract.EditorialMemoryReceipt{}, fmt.Errorf("editorial memory artifact binding is invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(artifact.Content))
	decoder.DisallowUnknownFields()
	var memoryArtifact reportilcontract.EditorialMemoryArtifact
	if err := decoder.Decode(&memoryArtifact); err != nil {
		return reportilcontract.EditorialMemory{}, reportilcontract.EditorialMemoryReceipt{}, fmt.Errorf("editorial memory decode failed: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return reportilcontract.EditorialMemory{}, reportilcontract.EditorialMemoryReceipt{}, fmt.Errorf("editorial memory contains multiple JSON values")
	}
	if err := reportilcontract.ValidateEditorialMemoryArtifact(memoryArtifact, catalog); err != nil {
		return reportilcontract.EditorialMemory{}, reportilcontract.EditorialMemoryReceipt{}, err
	}
	return reportilcontract.EditorialMemoryFromArtifact(memoryArtifact), receipt, nil
}
