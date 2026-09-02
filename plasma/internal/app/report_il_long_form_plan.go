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

// ReadReportILLongFormPlan binds one planner session to exactly one finalized
// server-owned plan artifact. The plan body is never taken from provider output.
func (s *Service) ReadReportILLongFormPlan(
	ctx context.Context,
	missionID,
	toolSessionID string,
	catalog reportilcontract.SourceCatalog,
) (reportilcontract.LongFormPlan, reportilcontract.LongFormPlanReceipt, error) {
	missionID = strings.TrimSpace(missionID)
	toolSessionID = strings.TrimSpace(toolSessionID)
	if !strings.HasPrefix(missionID, "mis_") || !strings.HasPrefix(toolSessionID, "ses_") || catalog.MissionID != missionID {
		return reportilcontract.LongFormPlan{}, reportilcontract.LongFormPlanReceipt{}, fmt.Errorf("long-form plan binding is invalid")
	}
	if err := reportilcontract.ValidateSourceCatalog(catalog); err != nil {
		return reportilcontract.LongFormPlan{}, reportilcontract.LongFormPlanReceipt{}, err
	}
	events, err := s.ListEvents(ctx, missionID)
	if err != nil {
		return reportilcontract.LongFormPlan{}, reportilcontract.LongFormPlanReceipt{}, err
	}
	type payload struct {
		ToolName      string `json:"tool_name"`
		ToolSessionID string `json:"tool_session_id"`
		Success       bool   `json:"success"`
		IOMetrics     struct {
			ReportILStage string `json:"report_il_stage"`
			ArtifactID    string `json:"artifact_id"`
			SHA256        string `json:"sha256"`
			ByteSize      int    `json:"byte_size"`
			Parts         int    `json:"parts"`
			Sections      int    `json:"sections"`
		} `json:"io_metrics"`
	}
	var receipt reportilcontract.LongFormPlanReceipt
	count := 0
	for _, event := range events {
		if event.EventType != "mcp.tool.called" || event.CorrelationID != toolSessionID {
			continue
		}
		var value payload
		if json.Unmarshal(event.Payload, &value) != nil || !value.Success || value.ToolName != reportilcontract.LongFormPlanSubmitTool || value.IOMetrics.ReportILStage != "il_long_form_plan" {
			continue
		}
		count++
		receipt = reportilcontract.LongFormPlanReceipt{
			ArtifactID: value.IOMetrics.ArtifactID, SHA256: value.IOMetrics.SHA256,
			ByteSize: value.IOMetrics.ByteSize, Parts: value.IOMetrics.Parts, Sections: value.IOMetrics.Sections,
		}
	}
	if count != 1 || !strings.HasPrefix(receipt.ArtifactID, "art_") || len(receipt.SHA256) != 64 || receipt.ByteSize < 1 {
		return reportilcontract.LongFormPlan{}, reportilcontract.LongFormPlanReceipt{}, fmt.Errorf("long-form plan was not finalized exactly once")
	}
	artifact, err := s.GetRawArtifact(ctx, receipt.ArtifactID)
	if err != nil {
		return reportilcontract.LongFormPlan{}, reportilcontract.LongFormPlanReceipt{}, err
	}
	contentSum := sha256.Sum256(artifact.Content)
	if artifact.MissionID != missionID || artifact.MediaType != reportilcontract.LongFormPlanMediaType ||
		artifact.Producer.Type != "mcp_tool" || artifact.Producer.ID != reportilcontract.LongFormPlanSubmitTool ||
		artifact.SHA256 != receipt.SHA256 || hex.EncodeToString(contentSum[:]) != receipt.SHA256 ||
		int(artifact.ByteSize) != receipt.ByteSize || len(artifact.Content) != receipt.ByteSize {
		return reportilcontract.LongFormPlan{}, reportilcontract.LongFormPlanReceipt{}, fmt.Errorf("long-form plan artifact binding is invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(artifact.Content))
	decoder.DisallowUnknownFields()
	var plan reportilcontract.LongFormPlan
	if err := decoder.Decode(&plan); err != nil {
		return reportilcontract.LongFormPlan{}, reportilcontract.LongFormPlanReceipt{}, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return reportilcontract.LongFormPlan{}, reportilcontract.LongFormPlanReceipt{}, fmt.Errorf("long-form plan contains multiple JSON values")
	}
	if planParts, planSections := len(plan.Parts), reportilcontract.LongFormPlanSectionCount(plan); planParts != receipt.Parts || planSections != receipt.Sections {
		return reportilcontract.LongFormPlan{}, reportilcontract.LongFormPlanReceipt{}, fmt.Errorf("long-form plan inventory binding is invalid")
	}
	return plan, receipt, nil
}
