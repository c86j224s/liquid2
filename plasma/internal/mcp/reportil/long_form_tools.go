package reportil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"io"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

type ReportILLongFormPlanSubmitInput struct {
	Title   string                          `json:"title"`
	Summary string                          `json:"summary"`
	Parts   []ReportILLongFormPlanPartInput `json:"parts"`
}

type ReportILLongFormPlanPartInput struct {
	Title    string                             `json:"title"`
	Purpose  string                             `json:"purpose"`
	Sections []ReportILLongFormPlanSectionInput `json:"sections"`
}

type ReportILLongFormPlanSectionInput struct {
	Title                string   `json:"title"`
	Purpose              string   `json:"purpose"`
	Representations      []string `json:"representations"`
	EditorialAccountKeys []string `json:"editorial_account_keys"`
	EvidenceSourceKeys   []string `json:"evidence_source_keys"`
}

type ReportILLongFormPlanReadInput struct {
	Offset   int `json:"offset"`
	MaxBytes int `json:"max_bytes"`
}

type ReportILLongFormPlanStateOutput struct {
	ReportILStage string `json:"report_il_stage"`
	ArtifactID    string `json:"artifact_id"`
	SHA256        string `json:"sha256"`
	ByteSize      int    `json:"byte_size"`
	Parts         int    `json:"parts"`
	Sections      int    `json:"sections"`
}

type ReportILLongFormPlanReadOutput struct {
	ReportILStage string `json:"report_il_stage"`
	Content       string `json:"content"`
	Offset        int    `json:"offset"`
	NextOffset    int    `json:"next_offset,omitempty"`
	ContentLength int    `json:"content_length"`
	Truncated     bool   `json:"truncated"`
}

func (server *SourceHandler) CallReportILLongFormPlanSubmit(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	binding, err := server.AccessBinding()
	if err != nil {
		return server.ErrorResult(call.Name, server.MissionID(), "binding", err.Error(), false, nil)
	}
	if binding.Stage != "il_long_form_plan" {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "binding", "long-form plan submission is unavailable for this stage", false, nil)
	}
	var input ReportILLongFormPlanSubmitInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, nil)
	}
	var memory *reportilcontract.EditorialMemory
	if binding.MaxReadBytes > 0 {
		if !server.reportILBoundSourcesComplete(binding.LongFormSourceKeys) {
			return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "direct long-form plan requires the complete frozen source read", false, nil)
		}
	} else {
		if !server.Editorial.ReviewComplete {
			return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "long-form plan requires the complete editorial memory read", false, []string{binding.EditorialMemoryArtifactID})
		}
		boundMemory, memoryErr := server.reportILBoundEditorialMemory(ctx, binding)
		if memoryErr != nil {
			return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", memoryErr.Error(), false, []string{binding.EditorialMemoryArtifactID})
		}
		memory = &boundMemory
	}
	plan := reportilcontract.LongFormPlan{
		SchemaVersion: reportilcontract.LongFormPlanSchemaVersion,
		Title:         strings.TrimSpace(input.Title), Language: binding.LongFormTargetLanguage, Summary: strings.TrimSpace(input.Summary),
	}
	sectionCount := 0
	for _, sourcePart := range input.Parts {
		sectionCount += len(sourcePart.Sections)
	}
	currentSection := 0
	for partIndex, sourcePart := range input.Parts {
		part := reportilcontract.LongFormPart{PartKey: fmt.Sprintf("part_%03d", partIndex+1), Title: strings.TrimSpace(sourcePart.Title), Purpose: strings.TrimSpace(sourcePart.Purpose)}
		for sectionIndex, sourceSection := range sourcePart.Sections {
			currentSection++
			role := reportilcontract.LongFormSectionRoleBody
			if currentSection == sectionCount {
				role = reportilcontract.LongFormSectionRoleConclusion
			}
			representations := append([]string{}, sourceSection.Representations...)
			part.Sections = append(part.Sections, reportilcontract.LongFormSection{
				SectionKey: fmt.Sprintf("%s.section_%03d", part.PartKey, sectionIndex+1),
				Title:      strings.TrimSpace(sourceSection.Title), Purpose: strings.TrimSpace(sourceSection.Purpose),
				Role:                 role,
				Representations:      representations,
				EditorialAccountKeys: append([]string(nil), sourceSection.EditorialAccountKeys...),
				EvidenceSourceKeys:   append([]string(nil), sourceSection.EvidenceSourceKeys...),
			})
		}
		plan.Parts = append(plan.Parts, part)
	}
	if err := reportilcontract.ValidateLongFormPlan(plan, memory, binding.Catalog); err != nil {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, nil)
	}
	if plan.Language != binding.LongFormTargetLanguage {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "long-form plan target language differs from the request binding", false, nil)
	}
	if memory == nil && !reportILLongFormPlanUsesBoundSources(plan, binding.LongFormSourceKeys) {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "long-form plan contains a source outside the bound inventory", false, nil)
	}
	content, _ := json.Marshal(plan)
	content = append(content, '\n')
	stored, err := server.Artifacts.CreateRawArtifact(ctx, artifact.CreateRequest{
		ArtifactID: server.NewID("art"), MissionID: binding.Catalog.MissionID,
		MediaType: reportilcontract.LongFormPlanMediaType, Filename: "report-il-long-form-plan.json",
		Producer: ledger.Producer{Type: "mcp_tool", ID: reportilcontract.LongFormPlanSubmitTool}, Content: content,
	})
	if err != nil {
		return server.ErrorFromErr(call.Name, binding.Catalog.MissionID, err, nil)
	}
	return wire.ToolResult{ToolName: call.Name, MissionID: binding.Catalog.MissionID, Content: ReportILLongFormPlanStateOutput{
		ReportILStage: binding.Stage, ArtifactID: stored.ArtifactID, SHA256: stored.SHA256,
		ByteSize: int(stored.ByteSize), Parts: len(plan.Parts), Sections: reportilcontract.LongFormPlanSectionCount(plan),
	}}
}

func (server *SourceHandler) CallReportILLongFormPlanRead(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	binding, err := server.AccessBinding()
	if err != nil {
		return server.ErrorResult(call.Name, server.MissionID(), "binding", err.Error(), false, nil)
	}
	if binding.Stage != "il_long_form_section" && binding.Stage != "il_long_form_part" && binding.Stage != "il_long_form_final" {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "binding", "long-form plan read is unavailable for this stage", false, nil)
	}
	var input ReportILLongFormPlanReadInput
	if err := server.Decode(call.Arguments, &input); err != nil || input.Offset < 0 || input.MaxBytes < 1 || input.MaxBytes > ReportILDocumentMaxReadBytes {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "long-form plan read range is invalid", false, nil)
	}
	artifactValue, err := server.Artifacts.GetRawArtifact(ctx, binding.LongFormPlanArtifactID)
	if err != nil {
		return server.ErrorFromErr(call.Name, binding.Catalog.MissionID, err, []string{binding.LongFormPlanArtifactID})
	}
	if artifactValue.MissionID != binding.Catalog.MissionID || artifactValue.MediaType != reportilcontract.LongFormPlanMediaType ||
		artifactValue.Producer.Type != "mcp_tool" || artifactValue.Producer.ID != reportilcontract.LongFormPlanSubmitTool ||
		artifactValue.SHA256 != binding.LongFormPlanSHA256 || server.SHA256(artifactValue.Content) != binding.LongFormPlanSHA256 ||
		int64(len(artifactValue.Content)) != artifactValue.ByteSize {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "long-form plan artifact binding is invalid", false, []string{binding.LongFormPlanArtifactID})
	}
	decoder := json.NewDecoder(bytes.NewReader(artifactValue.Content))
	decoder.DisallowUnknownFields()
	var plan reportilcontract.LongFormPlan
	if err := decoder.Decode(&plan); err != nil {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "long-form plan artifact is invalid", false, []string{binding.BaseAuthorArtifactID})
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "long-form plan artifact is invalid", false, []string{binding.BaseAuthorArtifactID})
	}
	if plan.Language != binding.LongFormTargetLanguage {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "long-form plan target language differs from the request binding", false, []string{binding.LongFormPlanArtifactID})
	}
	scoped, err := reportILLongFormPlanScope(plan, binding)
	if err != nil {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, []string{binding.LongFormPlanArtifactID})
	}
	server.Mu.Lock()
	defer server.Mu.Unlock()
	if server.Documents.PlanReviewComplete || input.Offset != server.Documents.PlanReviewNextOffset {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", fmt.Sprintf("long-form plan reads must start at offset 0 and continue at exact next_offset %d", server.Documents.PlanReviewNextOffset), false, []string{binding.LongFormPlanArtifactID})
	}
	content, offset, nextOffset, truncated, err := server.BoundedRead(scoped, input.Offset, input.MaxBytes, input.MaxBytes)
	if err != nil {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, []string{binding.LongFormPlanArtifactID})
	}
	if truncated {
		server.Documents.PlanReviewNextOffset = nextOffset
	} else {
		server.Documents.PlanReviewNextOffset = 0
		server.Documents.PlanReviewComplete = true
	}
	return wire.ToolResult{ToolName: call.Name, MissionID: binding.Catalog.MissionID, Content: ReportILLongFormPlanReadOutput{
		ReportILStage: binding.Stage, Content: content, Offset: offset, NextOffset: nextOffset,
		ContentLength: len(scoped), Truncated: truncated,
	}}
}

func reportILLongFormPlanScope(plan reportilcontract.LongFormPlan, binding reportilcontract.SourceAccessBinding) ([]byte, error) {
	var value any = plan
	switch binding.Stage {
	case "il_long_form_section":
		part, section, ok := plan.Find(binding.LongFormPartKey, binding.LongFormSectionKey)
		if !ok {
			return nil, fmt.Errorf("bound long-form Section is unavailable")
		}
		value = struct {
			Title   string                           `json:"title"`
			Summary string                           `json:"summary"`
			Part    reportilcontract.LongFormPart    `json:"part"`
			Section reportilcontract.LongFormSection `json:"section"`
		}{Title: plan.Title, Summary: plan.Summary, Part: reportilcontract.LongFormPart{PartKey: part.PartKey, Title: part.Title, Purpose: part.Purpose}, Section: section}
	case "il_long_form_part":
		for _, part := range plan.Parts {
			if part.PartKey == binding.LongFormPartKey {
				value = struct {
					Title   string                        `json:"title"`
					Summary string                        `json:"summary"`
					Part    reportilcontract.LongFormPart `json:"part"`
				}{Title: plan.Title, Summary: plan.Summary, Part: part}
				break
			}
		}
		if scoped, ok := value.(reportilcontract.LongFormPlan); ok && len(scoped.Parts) > 0 {
			return nil, fmt.Errorf("bound long-form Part is unavailable")
		}
	}
	content, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return append(content, '\n'), nil
}

func reportILLongFormPlanUsesBoundSources(plan reportilcontract.LongFormPlan, sourceKeys []string) bool {
	allowed := make(map[string]bool, len(sourceKeys))
	for _, sourceKey := range sourceKeys {
		allowed[sourceKey] = true
	}
	for _, part := range plan.Parts {
		for _, section := range part.Sections {
			for _, sourceKey := range section.EvidenceSourceKeys {
				if !allowed[sourceKey] {
					return false
				}
			}
		}
	}
	return len(allowed) > 0
}

func (server *SourceHandler) reportILBoundSourcesComplete(sourceKeys []string) bool {
	server.Mu.Lock()
	defer server.Mu.Unlock()
	for _, sourceKey := range sourceKeys {
		if !server.State.Complete[sourceKey] {
			return false
		}
	}
	return len(sourceKeys) > 0
}
