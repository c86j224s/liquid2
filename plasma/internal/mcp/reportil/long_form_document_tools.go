package reportil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

func (server *SourceHandler) CallReportILLongFormDocumentStart(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	binding, err := server.reportILLongFormDocumentBinding()
	if err != nil {
		return server.ErrorResult(call.Name, server.MissionID(), "binding", err.Error(), false, nil)
	}
	server.Mu.Lock()
	planRead := server.Documents.PlanReviewComplete
	server.Mu.Unlock()
	if !planRead {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "long-form document requires the complete bound plan read", false, []string{binding.LongFormPlanArtifactID})
	}
	var input ReportILLongFormDocumentStartInput
	if err := server.Decode(call.Arguments, &input); err != nil || strings.TrimSpace(input.Title) == "" ||
		!utf8.ValidString(input.Title) || !reportILDocumentLanguageValid(input.Language) {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "long-form document start input is invalid", false, nil)
	}

	var memory reportilcontract.EditorialMemory
	if binding.Stage == "il_long_form_section" && binding.MaxReadBytes > 0 {
		if !server.reportILBoundSourcesComplete(binding.LongFormSourceKeys) {
			return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "long-form Section requires the complete bound source read", false, nil)
		}
	} else if binding.EditorialMemoryArtifactID != "" {
		server.Mu.Lock()
		memoryRead := server.Editorial.ReviewComplete
		server.Mu.Unlock()
		if !memoryRead {
			return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "long-form document requires the complete editorial memory read", false, []string{binding.EditorialMemoryArtifactID})
		}
		memory, err = server.reportILBoundEditorialMemory(ctx, binding)
		if err != nil {
			return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, []string{binding.EditorialMemoryArtifactID})
		}
	} else if binding.Stage == "il_long_form_section" {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "binding", "long-form Section source binding is unavailable", false, nil)
	}

	document, inputs, err := server.reportILLongFormDocumentSeed(ctx, binding, input)
	if err != nil {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, inputs)
	}
	plan, err := server.reportILBoundLongFormPlan(ctx, binding)
	if err != nil {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, []string{binding.LongFormPlanArtifactID})
	}
	now := time.Now().UTC()
	workspace := &DocumentWorkspace{
		WorkspaceID: server.NewID("ilw"), MissionID: binding.Catalog.MissionID,
		SessionID: server.SessionID(), Stage: binding.Stage, CatalogSHA256: binding.Catalog.SHA256,
		EditorialMemory: memory, LongFormPlan: plan, Document: document, Revision: 1, CreatedAt: now, UpdatedAt: now,
	}
	server.Mu.Lock()
	defer server.Mu.Unlock()
	if len(server.Documents.Workspaces) != 0 {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "conflict", "long-form document workspace already exists", false, inputs)
	}
	server.Documents.Workspaces[workspace.WorkspaceID] = workspace
	return wire.ToolResult{ToolName: call.Name, MissionID: workspace.MissionID, Content: reportILDocumentState(*workspace)}
}

func (server *SourceHandler) reportILLongFormDocumentSeed(
	ctx context.Context,
	binding reportilcontract.SourceAccessBinding,
	input ReportILLongFormDocumentStartInput,
) (reportilcontract.AuthorDocument, []string, error) {
	document := reportilcontract.AuthorDocument{
		SchemaVersion: reportilcontract.LongFormAuthorDocumentSchemaVersion,
		Title:         strings.TrimSpace(input.Title), Language: input.Language,
	}
	if binding.Stage == "il_long_form_section" {
		plan, err := server.reportILBoundLongFormPlan(ctx, binding)
		if err != nil {
			return reportilcontract.AuthorDocument{}, []string{binding.LongFormPlanArtifactID}, err
		}
		part, section, ok := plan.Find(binding.LongFormPartKey, binding.LongFormSectionKey)
		if !ok {
			return reportilcontract.AuthorDocument{}, []string{binding.LongFormPlanArtifactID}, fmt.Errorf("bound long-form Section is unavailable")
		}
		document.Title = plan.Title
		document.Language = plan.Language
		document.Parts = []reportilcontract.AuthorPart{{
			PartKey: part.PartKey, Title: part.Title,
			Sections: []reportilcontract.AuthorSection{{SectionKey: section.SectionKey, Title: section.Title}},
		}}
		return document, []string{binding.LongFormPlanArtifactID}, nil
	}

	if len(binding.LongFormInputArtifactIDs) == 0 {
		return reportilcontract.AuthorDocument{}, nil, fmt.Errorf("long-form document input artifact binding is empty")
	}
	plan, err := server.reportILBoundLongFormPlan(ctx, binding)
	if err != nil {
		return reportilcontract.AuthorDocument{}, []string{binding.LongFormPlanArtifactID}, err
	}
	fragments := make([]reportilcontract.AuthorDocument, 0, len(binding.LongFormInputArtifactIDs))
	for index, artifactID := range binding.LongFormInputArtifactIDs {
		artifactValue, err := server.Artifacts.GetRawArtifact(ctx, artifactID)
		if err != nil {
			return reportilcontract.AuthorDocument{}, binding.LongFormInputArtifactIDs, err
		}
		expectedProducer := reportilcontract.LongFormDocumentFinalizeTool
		if artifactValue.MissionID != binding.Catalog.MissionID ||
			artifactValue.MediaType != reportilcontract.AuthorDocumentMediaType ||
			artifactValue.Producer.Type != "mcp_tool" || artifactValue.Producer.ID != expectedProducer ||
			artifactValue.SHA256 != binding.LongFormInputSHA256s[index] ||
			server.SHA256(artifactValue.Content) != binding.LongFormInputSHA256s[index] ||
			int64(len(artifactValue.Content)) != artifactValue.ByteSize {
			return reportilcontract.AuthorDocument{}, binding.LongFormInputArtifactIDs, fmt.Errorf("long-form input artifact binding is invalid")
		}
		if err := server.reportILValidateLongFormInputStage(ctx, binding, artifactID); err != nil {
			return reportilcontract.AuthorDocument{}, binding.LongFormInputArtifactIDs, err
		}
		decoder := json.NewDecoder(bytes.NewReader(artifactValue.Content))
		decoder.DisallowUnknownFields()
		var fragment reportilcontract.AuthorDocument
		if err := decoder.Decode(&fragment); err != nil {
			return reportilcontract.AuthorDocument{}, binding.LongFormInputArtifactIDs, fmt.Errorf("long-form input artifact is invalid")
		}
		var extra any
		if err := decoder.Decode(&extra); err != io.EOF {
			return reportilcontract.AuthorDocument{}, binding.LongFormInputArtifactIDs, fmt.Errorf("long-form input artifact is invalid")
		}
		if err := reportilcontract.ValidateLongFormAuthorFragment(fragment, binding.Catalog); err != nil {
			return reportilcontract.AuthorDocument{}, binding.LongFormInputArtifactIDs, err
		}
		fragments = append(fragments, fragment)
	}
	parts, err := ReportILAssembleLongFormInputs(binding, plan, fragments)
	if err != nil {
		return reportilcontract.AuthorDocument{}, binding.LongFormInputArtifactIDs, err
	}
	document.Title = plan.Title
	document.Language = plan.Language
	document.Parts = parts
	return document, append([]string(nil), binding.LongFormInputArtifactIDs...), nil
}

func (server *SourceHandler) CallReportILLongFormDocumentAppend(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	_ = ctx
	binding, err := server.reportILLongFormSectionDocumentBinding()
	if err != nil {
		return server.ErrorResult(call.Name, server.MissionID(), "binding", err.Error(), false, nil)
	}
	var input ReportILLongFormDocumentAppendInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, nil)
	}
	server.Mu.Lock()
	defer server.Mu.Unlock()
	workspace, err := server.DocumentWorkspace(input.WorkspaceID, binding)
	if err != nil || workspace.Finalized || workspace.Finalizing {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "long-form Section workspace is unavailable", false, []string{input.WorkspaceID})
	}
	if len(workspace.Document.Parts) != 1 || len(workspace.Document.Parts[0].Sections) != 1 {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "long-form Section workspace inventory is invalid", false, []string{input.WorkspaceID})
	}
	section := &workspace.Document.Parts[0].Sections[0]
	if input.SectionKey != "" && input.SectionKey != section.SectionKey ||
		input.SectionTitle != "" && strings.TrimSpace(input.SectionTitle) != section.Title {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "long-form Section append target is invalid", false, []string{input.WorkspaceID})
	}
	block := reportilcontract.AuthorBlock{
		BlockKey: fmt.Sprintf("%s.block_%03d", section.SectionKey, len(section.Blocks)+1),
		Kind:     input.Kind, Prose: input.Prose, Items: append([]string(nil), input.Items...),
		Code: input.Code, Language: input.Language,
		EditorialAccountKeys: append([]string(nil), input.EditorialAccountKeys...),
		EvidenceSourceKeys:   append([]string(nil), input.EvidenceSourceKeys...),
	}
	if input.Table != nil {
		block.Table = &reportilcontract.AuthorTable{Caption: input.Table.Caption, Columns: append([]string(nil), input.Table.Columns...)}
		for _, row := range input.Table.Rows {
			block.Table.Rows = append(block.Table.Rows, reportilcontract.AuthorTableRow{Cells: append([]string(nil), row.Cells...)})
		}
	}
	if input.Equation != nil {
		block.Equation = &reportilcontract.AuthorEquation{
			Expression: input.Equation.Expression,
			Notation:   input.Equation.Notation,
		}
	}
	if err := validateReportILLongFormBlockBinding(workspace, binding, block); err != nil {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, []string{input.WorkspaceID})
	}
	section.Blocks = append(section.Blocks, block)
	if err := reportilcontract.ValidateAuthorDocumentPartial(workspace.Document, binding.Catalog); err != nil {
		section.Blocks = section.Blocks[:len(section.Blocks)-1]
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, []string{input.WorkspaceID})
	}
	server.reportILDocumentMutated(workspace)
	return wire.ToolResult{ToolName: call.Name, MissionID: workspace.MissionID, Content: reportILDocumentState(*workspace)}
}

func validateReportILLongFormBlockBinding(
	workspace *DocumentWorkspace,
	binding reportilcontract.SourceAccessBinding,
	block reportilcontract.AuthorBlock,
) error {
	if binding.EditorialMemoryArtifactID != "" {
		_, plannedSection, ok := workspace.LongFormPlan.Find(binding.LongFormPartKey, binding.LongFormSectionKey)
		sourceKeys, err := reportilcontract.EditorialAccountSourceKeys(workspace.EditorialMemory, block.EditorialAccountKeys)
		if !ok || !equalReportILStringInventory(binding.LongFormSectionAccountKeys, plannedSection.EditorialAccountKeys) ||
			err != nil || !equalReportILStringInventory(sourceKeys, block.EvidenceSourceKeys) ||
			!reportILSubset(block.EditorialAccountKeys, binding.LongFormSectionAccountKeys) ||
			!reportILSubset(block.EvidenceSourceKeys, plannedSection.EvidenceSourceKeys) {
			return fmt.Errorf("long-form block source binding differs from the bound plan")
		}
		return nil
	}
	if len(block.EditorialAccountKeys) != 0 || !reportILSubset(block.EvidenceSourceKeys, binding.LongFormSourceKeys) {
		return fmt.Errorf("direct long-form block source binding is invalid")
	}
	return nil
}

func (server *SourceHandler) CallReportILLongFormDocumentRead(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	return server.CallReportILDocumentRead(ctx, call)
}

func (server *SourceHandler) CallReportILLongFormDocumentReplace(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	return server.CallReportILDocumentReplace(ctx, call)
}

func (server *SourceHandler) CallReportILLongFormDocumentCorrectBlock(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	_ = ctx
	binding, err := server.reportILLongFormSectionDocumentBinding()
	if err != nil {
		return server.ErrorResult(call.Name, server.MissionID(), "binding", err.Error(), false, nil)
	}
	var input ReportILLongFormDocumentCorrectBlockInput
	if err := server.Decode(call.Arguments, &input); err != nil || strings.TrimSpace(input.BlockKey) == "" ||
		input.Operation != "delete" && input.Operation != "replace" ||
		input.Operation == "delete" && input.Replacement != nil ||
		input.Operation == "replace" && input.Replacement == nil {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "long-form Section block correction is invalid", false, []string{input.WorkspaceID})
	}
	server.Mu.Lock()
	defer server.Mu.Unlock()
	workspace, err := server.DocumentWorkspace(input.WorkspaceID, binding)
	if err != nil || workspace.Finalized || workspace.Finalizing ||
		len(workspace.Document.Parts) != 1 || len(workspace.Document.Parts[0].Sections) != 1 {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "long-form Section workspace is unavailable", false, []string{input.WorkspaceID})
	}
	if !workspace.ReviewComplete || workspace.ReviewRevision != workspace.Revision {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "long-form Section must be read completely before correcting a block", false, []string{input.WorkspaceID})
	}
	candidate := cloneReportILAuthorDocument(workspace.Document)
	section := &candidate.Parts[0].Sections[0]
	blockIndex := -1
	for index := range section.Blocks {
		if section.Blocks[index].BlockKey == input.BlockKey {
			if blockIndex != -1 {
				return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "long-form Section block identity is invalid", false, []string{input.WorkspaceID})
			}
			blockIndex = index
		}
	}
	if blockIndex == -1 {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "block_key must identify one draft block in the bound Section", false, []string{input.WorkspaceID})
	}
	if input.Operation == "delete" {
		section.Blocks = append(section.Blocks[:blockIndex], section.Blocks[blockIndex+1:]...)
		for index := range section.Blocks {
			section.Blocks[index].BlockKey = fmt.Sprintf("%s.block_%03d", section.SectionKey, index+1)
		}
	} else {
		replacement := input.Replacement
		block := reportilcontract.AuthorBlock{
			BlockKey: section.Blocks[blockIndex].BlockKey,
			Kind:     replacement.Kind, Prose: replacement.Prose,
			Items: append([]string(nil), replacement.Items...),
			Code:  replacement.Code, Language: replacement.Language,
			EditorialAccountKeys: append([]string(nil), replacement.EditorialAccountKeys...),
			EvidenceSourceKeys:   append([]string(nil), replacement.EvidenceSourceKeys...),
		}
		if replacement.Table != nil {
			block.Table = &reportilcontract.AuthorTable{
				Caption: replacement.Table.Caption,
				Columns: append([]string(nil), replacement.Table.Columns...),
			}
			for _, row := range replacement.Table.Rows {
				block.Table.Rows = append(block.Table.Rows, reportilcontract.AuthorTableRow{Cells: append([]string(nil), row.Cells...)})
			}
		}
		if replacement.Equation != nil {
			block.Equation = &reportilcontract.AuthorEquation{
				Expression: replacement.Equation.Expression,
				Notation:   replacement.Equation.Notation,
			}
		}
		if err := validateReportILLongFormBlockBinding(workspace, binding, block); err != nil {
			return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, []string{input.WorkspaceID})
		}
		section.Blocks[blockIndex] = block
	}
	if err := reportilcontract.ValidateAuthorDocumentPartial(candidate, binding.Catalog); err != nil {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, []string{input.WorkspaceID})
	}
	workspace.Document = candidate
	workspace.Replacements++
	server.reportILDocumentMutated(workspace)
	return wire.ToolResult{ToolName: call.Name, MissionID: workspace.MissionID, Content: reportILDocumentState(*workspace)}
}

func (server *SourceHandler) CallReportILLongFormDocumentFinalize(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	binding, err := server.reportILLongFormDocumentBinding()
	if err != nil {
		return server.ErrorResult(call.Name, server.MissionID(), "binding", err.Error(), false, nil)
	}
	var input ReportILDocumentFinalizeInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, nil)
	}
	server.Mu.Lock()
	workspace, err := server.DocumentWorkspace(input.WorkspaceID, binding)
	if err != nil || workspace.Finalized || workspace.Finalizing || !workspace.ReviewComplete || workspace.ReviewRevision != workspace.Revision {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "long-form document must be read completely after its final edit", false, []string{input.WorkspaceID})
	}
	if err := server.validateReportILLongFormWorkspace(workspace, binding); err != nil {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, []string{input.WorkspaceID})
	}
	content, err := json.Marshal(workspace.Document)
	if err != nil {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "internal", "long-form document encoding failed", false, []string{input.WorkspaceID})
	}
	content = append(content, '\n')
	workspace.Finalizing = true
	workspaceCopy := *workspace
	server.Mu.Unlock()

	stored, err := server.Artifacts.CreateRawArtifact(ctx, artifact.CreateRequest{
		ArtifactID: server.NewID("art"), MissionID: workspaceCopy.MissionID,
		MediaType: reportilcontract.AuthorDocumentMediaType, Filename: "report-il-long-form-author-document.json",
		Producer: ledger.Producer{Type: "mcp_tool", ID: reportilcontract.LongFormDocumentFinalizeTool}, Content: content,
	})
	if err != nil {
		server.Mu.Lock()
		if current, currentErr := server.DocumentWorkspace(input.WorkspaceID, binding); currentErr == nil && current.Revision == workspaceCopy.Revision {
			current.Finalizing = false
		}
		server.Mu.Unlock()
		return server.ErrorFromErr(call.Name, workspaceCopy.MissionID, err, []string{workspaceCopy.WorkspaceID})
	}
	server.Mu.Lock()
	defer server.Mu.Unlock()
	workspace, err = server.DocumentWorkspace(input.WorkspaceID, binding)
	if err != nil || workspace.Finalized || !workspace.Finalizing || workspace.Revision != workspaceCopy.Revision {
		return server.ErrorResult(call.Name, workspaceCopy.MissionID, "conflict", "long-form document changed during finalization", false, []string{workspaceCopy.WorkspaceID, stored.ArtifactID})
	}
	workspace.Finalizing = false
	workspace.Finalized = true
	workspace.ArtifactID = stored.ArtifactID
	workspace.SHA256 = stored.SHA256
	workspace.ByteSize = int(stored.ByteSize)
	workspace.UpdatedAt = time.Now().UTC()
	return wire.ToolResult{ToolName: call.Name, MissionID: workspace.MissionID, Content: reportILDocumentState(*workspace)}
}

func (server *SourceHandler) validateReportILLongFormWorkspace(workspace *DocumentWorkspace, binding reportilcontract.SourceAccessBinding) error {
	if err := validateReportILLongFormPlanIdentity(workspace.Document, workspace.LongFormPlan, binding); err != nil {
		return err
	}
	if binding.Stage == "il_long_form_final" {
		if err := reportilcontract.ValidateLongFormAuthorDocument(workspace.Document, binding.Catalog); err != nil {
			return err
		}
	} else if err := reportilcontract.ValidateLongFormAuthorFragment(workspace.Document, binding.Catalog); err != nil {
		return err
	}
	if len(workspace.EditorialMemory.Accounts) > 0 {
		if binding.Stage == "il_long_form_final" {
			return reportilcontract.ValidateAuthorDocumentWithMemory(workspace.Document, workspace.EditorialMemory, binding.Catalog)
		}
		return reportilcontract.ValidateLongFormAuthorFragmentWithMemory(workspace.Document, workspace.EditorialMemory, binding.Catalog)
	}
	return nil
}

func validateReportILLongFormPlanIdentity(
	document reportilcontract.AuthorDocument,
	plan reportilcontract.LongFormPlan,
	binding reportilcontract.SourceAccessBinding,
) error {
	if document.SchemaVersion != reportilcontract.LongFormAuthorDocumentSchemaVersion || document.Title != plan.Title || document.Language != plan.Language {
		return fmt.Errorf("long-form document identity differs from the bound plan")
	}
	plannedParts := plan.Parts
	if binding.Stage == "il_long_form_section" || binding.Stage == "il_long_form_part" {
		plannedParts = nil
		for _, part := range plan.Parts {
			if part.PartKey == binding.LongFormPartKey {
				plannedParts = []reportilcontract.LongFormPart{part}
				break
			}
		}
	}
	if len(document.Parts) != len(plannedParts) {
		return fmt.Errorf("long-form document inventory differs from the bound plan")
	}
	for partIndex, plannedPart := range plannedParts {
		part := document.Parts[partIndex]
		plannedSections := plannedPart.Sections
		if binding.Stage == "il_long_form_section" {
			_, section, ok := plan.Find(binding.LongFormPartKey, binding.LongFormSectionKey)
			if !ok {
				return fmt.Errorf("long-form document Section is unavailable in the bound plan")
			}
			plannedSections = []reportilcontract.LongFormSection{section}
		}
		if part.PartKey != plannedPart.PartKey || part.Title != plannedPart.Title || len(part.Sections) != len(plannedSections) {
			return fmt.Errorf("long-form document Part identity differs from the bound plan")
		}
		for sectionIndex, plannedSection := range plannedSections {
			section := part.Sections[sectionIndex]
			if section.SectionKey != plannedSection.SectionKey || section.Title != plannedSection.Title {
				return fmt.Errorf("long-form document Section identity differs from the bound plan")
			}
			if err := reportilcontract.ValidateLongFormRepresentationObligations(plannedSection, section); err != nil {
				return err
			}
		}
	}
	return nil
}

func (server *SourceHandler) reportILLongFormDocumentBinding() (reportilcontract.SourceAccessBinding, error) {
	binding, err := server.AccessBinding()
	if err != nil {
		return reportilcontract.SourceAccessBinding{}, err
	}
	if binding.Stage != "il_long_form_section" && binding.Stage != "il_long_form_part" && binding.Stage != "il_long_form_final" {
		return reportilcontract.SourceAccessBinding{}, fmt.Errorf("long-form document workspace is unavailable for this stage")
	}
	return binding, nil
}

func (server *SourceHandler) reportILLongFormSectionDocumentBinding() (reportilcontract.SourceAccessBinding, error) {
	binding, err := server.reportILLongFormDocumentBinding()
	if err != nil {
		return reportilcontract.SourceAccessBinding{}, err
	}
	if binding.Stage != "il_long_form_section" {
		return reportilcontract.SourceAccessBinding{}, fmt.Errorf("long-form document append is unavailable for this stage")
	}
	return binding, nil
}

func (server *SourceHandler) reportILValidateLongFormInputStage(
	ctx context.Context,
	binding reportilcontract.SourceAccessBinding,
	artifactID string,
) error {
	events, err := server.Artifacts.ListEvents(ctx, binding.Catalog.MissionID)
	if err != nil {
		return err
	}
	expectedStage := "il_long_form_section"
	if binding.Stage == "il_long_form_final" {
		expectedStage = "il_long_form_part"
	}
	matches := 0
	for _, event := range events {
		if event.EventType != "mcp.tool.called" {
			continue
		}
		var payload struct {
			ToolName  string `json:"tool_name"`
			Success   bool   `json:"success"`
			IOMetrics struct {
				ReportILStage string `json:"report_il_stage"`
				ArtifactID    string `json:"artifact_id"`
				Finalized     bool   `json:"finalized"`
			} `json:"io_metrics"`
		}
		if json.Unmarshal(event.Payload, &payload) != nil || !payload.Success ||
			payload.ToolName != reportilcontract.LongFormDocumentFinalizeTool ||
			payload.IOMetrics.ReportILStage != expectedStage ||
			payload.IOMetrics.ArtifactID != artifactID || !payload.IOMetrics.Finalized {
			continue
		}
		matches++
	}
	if matches != 1 {
		return fmt.Errorf("long-form input artifact stage lineage is invalid")
	}
	return nil
}

func (server *SourceHandler) reportILBoundLongFormPlanSection(ctx context.Context, binding reportilcontract.SourceAccessBinding) (reportilcontract.LongFormPart, reportilcontract.LongFormSection, error) {
	plan, err := server.reportILBoundLongFormPlan(ctx, binding)
	if err != nil {
		return reportilcontract.LongFormPart{}, reportilcontract.LongFormSection{}, err
	}
	part, section, ok := plan.Find(binding.LongFormPartKey, binding.LongFormSectionKey)
	if !ok {
		return reportilcontract.LongFormPart{}, reportilcontract.LongFormSection{}, fmt.Errorf("bound long-form Section is unavailable")
	}
	return part, section, nil
}

func (server *SourceHandler) reportILBoundLongFormPlan(ctx context.Context, binding reportilcontract.SourceAccessBinding) (reportilcontract.LongFormPlan, error) {
	artifactValue, err := server.Artifacts.GetRawArtifact(ctx, binding.LongFormPlanArtifactID)
	if err != nil {
		return reportilcontract.LongFormPlan{}, err
	}
	if artifactValue.MissionID != binding.Catalog.MissionID || artifactValue.MediaType != reportilcontract.LongFormPlanMediaType ||
		artifactValue.Producer.Type != "mcp_tool" || artifactValue.Producer.ID != reportilcontract.LongFormPlanSubmitTool ||
		artifactValue.SHA256 != binding.LongFormPlanSHA256 || server.SHA256(artifactValue.Content) != binding.LongFormPlanSHA256 ||
		int64(len(artifactValue.Content)) != artifactValue.ByteSize {
		return reportilcontract.LongFormPlan{}, fmt.Errorf("long-form plan artifact binding is invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(artifactValue.Content))
	decoder.DisallowUnknownFields()
	var plan reportilcontract.LongFormPlan
	if err := decoder.Decode(&plan); err != nil {
		return reportilcontract.LongFormPlan{}, fmt.Errorf("long-form plan artifact is invalid")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return reportilcontract.LongFormPlan{}, fmt.Errorf("long-form plan artifact is invalid")
	}
	var memory *reportilcontract.EditorialMemory
	if binding.EditorialMemoryArtifactID != "" {
		boundMemory, err := server.reportILBoundEditorialMemory(ctx, binding)
		if err != nil {
			return reportilcontract.LongFormPlan{}, err
		}
		memory = &boundMemory
	}
	if err := reportilcontract.ValidateLongFormPlan(plan, memory, binding.Catalog); err != nil {
		return reportilcontract.LongFormPlan{}, err
	}
	if plan.Language != binding.LongFormTargetLanguage {
		return reportilcontract.LongFormPlan{}, fmt.Errorf("long-form plan target language differs from the request binding")
	}
	return plan, nil
}

func ReportILAssembleLongFormInputs(
	binding reportilcontract.SourceAccessBinding,
	plan reportilcontract.LongFormPlan,
	fragments []reportilcontract.AuthorDocument,
) ([]reportilcontract.AuthorPart, error) {
	if binding.Stage == "il_long_form_part" {
		var planned reportilcontract.LongFormPart
		found := false
		for _, part := range plan.Parts {
			if part.PartKey == binding.LongFormPartKey {
				planned = part
				found = true
				break
			}
		}
		if !found || len(fragments) != len(planned.Sections) {
			return nil, fmt.Errorf("long-form Part input inventory is invalid")
		}
		assembled := reportilcontract.AuthorPart{PartKey: planned.PartKey, Title: planned.Title}
		for index, plannedSection := range planned.Sections {
			fragment := fragments[index]
			if len(fragment.Parts) != 1 || fragment.Parts[0].PartKey != planned.PartKey ||
				len(fragment.Parts[0].Sections) != 1 || fragment.Parts[0].Sections[0].SectionKey != plannedSection.SectionKey {
				return nil, fmt.Errorf("long-form Part input order differs from the bound plan")
			}
			section := fragment.Parts[0].Sections[0]
			if section.Title != plannedSection.Title {
				return nil, fmt.Errorf("long-form Part input identity differs from the bound plan")
			}
			assembled.Sections = append(assembled.Sections, section)
		}
		return []reportilcontract.AuthorPart{assembled}, nil
	}

	if binding.Stage != "il_long_form_final" || len(fragments) != len(plan.Parts) {
		return nil, fmt.Errorf("long-form final input inventory is invalid")
	}
	parts := make([]reportilcontract.AuthorPart, 0, len(plan.Parts))
	for index, plannedPart := range plan.Parts {
		fragment := fragments[index]
		if len(fragment.Parts) != 1 || fragment.Parts[0].PartKey != plannedPart.PartKey ||
			fragment.Parts[0].Title != plannedPart.Title || len(fragment.Parts[0].Sections) != len(plannedPart.Sections) {
			return nil, fmt.Errorf("long-form final input order differs from the bound plan")
		}
		for sectionIndex, plannedSection := range plannedPart.Sections {
			section := fragment.Parts[0].Sections[sectionIndex]
			if section.SectionKey != plannedSection.SectionKey || section.Title != plannedSection.Title {
				return nil, fmt.Errorf("long-form final input identity differs from the bound plan")
			}
		}
		parts = append(parts, fragment.Parts[0])
	}
	return parts, nil
}

func reportILStringInventoryContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func reportILSubset(values, allowed []string) bool {
	if len(values) == 0 {
		return false
	}
	set := make(map[string]bool, len(allowed))
	for _, value := range allowed {
		set[value] = true
	}
	seen := map[string]bool{}
	for _, value := range values {
		if !set[value] || seen[value] {
			return false
		}
		seen[value] = true
	}
	return true
}

func equalReportILStringInventory(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
