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

const (
	reportILDocumentDefaultReadBytes = 32 * 1024
	ReportILDocumentMaxReadBytes     = 64 * 1024
	reportILDocumentMaxReplaceBytes  = 8 * 1024
)

type DocumentWorkspace struct {
	WorkspaceID      string
	MissionID        string
	SessionID        string
	Stage            string
	CatalogSHA256    string
	EditorialMemory  reportilcontract.EditorialMemory
	LongFormPlan     reportilcontract.LongFormPlan
	Document         reportilcontract.AuthorDocument
	Revision         int
	Replacements     int
	ReviewRevision   int
	ReviewNextOffset int
	ReviewComplete   bool
	Finalizing       bool
	Finalized        bool
	BaseArtifactID   string
	BaseSHA256       string
	BaseByteSize     int
	ArtifactID       string
	SHA256           string
	ByteSize         int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (server *SourceHandler) CallReportILDocumentStart(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	_ = ctx
	binding, err := server.reportILAuthorDocumentBinding()
	if err != nil {
		return server.ErrorResult(call.Name, server.MissionID(), "binding", err.Error(), false, nil)
	}
	var input ReportILDocumentStartInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, nil)
	}
	if strings.TrimSpace(input.Title) == "" || !utf8.ValidString(input.Title) {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "report IL document title is invalid", false, nil)
	}
	if !reportILDocumentLanguageValid(input.Language) {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "report IL document language is invalid", false, nil)
	}

	directSourceAuthor := binding.MaxReadBytes > 0
	server.Mu.Lock()
	memoryRead := server.Editorial.ReviewComplete
	sourceRead := server.reportILCompleteCatalogRead(binding)
	server.Mu.Unlock()
	if directSourceAuthor {
		if !sourceRead {
			return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "report IL document workspace requires the complete selected source read", false, nil)
		}
	} else if !memoryRead {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "report IL document workspace requires the complete editorial memory read", false, []string{binding.EditorialMemoryArtifactID})
	}
	var memory reportilcontract.EditorialMemory
	if !directSourceAuthor {
		memory, err = server.reportILBoundEditorialMemory(ctx, binding)
		if err != nil {
			return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, []string{binding.EditorialMemoryArtifactID})
		}
	}
	server.Mu.Lock()
	defer server.Mu.Unlock()
	if len(server.Documents.Workspaces) != 0 {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "conflict", "report IL document workspace already exists", false, nil)
	}
	now := time.Now().UTC()
	workspace := &DocumentWorkspace{
		WorkspaceID:     server.NewID("ilw"),
		MissionID:       binding.Catalog.MissionID,
		SessionID:       server.SessionID(),
		Stage:           binding.Stage,
		CatalogSHA256:   binding.Catalog.SHA256,
		EditorialMemory: memory,
		Document: reportilcontract.AuthorDocument{
			SchemaVersion: reportilcontract.AuthorDocumentSchemaVersion,
			Title:         strings.TrimSpace(input.Title),
			Language:      input.Language,
		},
		Revision:  1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	server.Documents.Workspaces[workspace.WorkspaceID] = workspace
	return wire.ToolResult{ToolName: call.Name, MissionID: workspace.MissionID, Content: reportILDocumentState(*workspace)}
}

func (server *SourceHandler) CallReportILDocumentOpen(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	binding, err := server.reportILPublicationDocumentBinding()
	if err != nil {
		return server.ErrorResult(call.Name, server.MissionID(), "binding", err.Error(), false, nil)
	}
	var input ReportILDocumentOpenInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, nil)
	}

	var memory reportilcontract.EditorialMemory
	if binding.EditorialMemoryArtifactID != "" {
		server.Mu.Lock()
		memoryRead := server.Editorial.ReviewComplete
		server.Mu.Unlock()
		if !memoryRead {
			return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "report IL document workspace requires the complete editorial memory read", false, []string{binding.EditorialMemoryArtifactID})
		}
		memory, err = server.reportILBoundEditorialMemory(ctx, binding)
		if err != nil {
			return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, []string{binding.EditorialMemoryArtifactID})
		}
	}
	server.Mu.Lock()
	if len(server.Documents.Workspaces) != 0 {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "conflict", "report IL document workspace already exists", false, nil)
	}
	server.Mu.Unlock()

	artifact, err := server.Artifacts.GetRawArtifact(ctx, binding.BaseAuthorArtifactID)
	if err != nil {
		return server.ErrorFromErr(call.Name, binding.Catalog.MissionID, err, []string{binding.BaseAuthorArtifactID})
	}
	if artifact.MissionID != binding.Catalog.MissionID ||
		artifact.MediaType != reportilcontract.AuthorDocumentMediaType ||
		artifact.SHA256 != binding.BaseAuthorSHA256 || server.SHA256(artifact.Content) != binding.BaseAuthorSHA256 ||
		int64(len(artifact.Content)) != artifact.ByteSize {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "report IL publication base document binding is invalid", false, []string{binding.BaseAuthorArtifactID})
	}
	decoder := json.NewDecoder(bytes.NewReader(artifact.Content))
	decoder.DisallowUnknownFields()
	var document reportilcontract.AuthorDocument
	if err := decoder.Decode(&document); err != nil {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "report IL publication base document is invalid", false, []string{binding.BaseAuthorArtifactID})
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "report IL publication base document is invalid", false, []string{binding.BaseAuthorArtifactID})
	}
	if validationErr := reportilcontract.ValidateAuthorDocumentWithMemory(document, memory, binding.Catalog); validationErr != nil {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", validationErr.Error(), false, []string{binding.BaseAuthorArtifactID, binding.EditorialMemoryArtifactID})
	}

	now := time.Now().UTC()
	workspace := &DocumentWorkspace{
		WorkspaceID: server.NewID("ilw"), MissionID: binding.Catalog.MissionID,
		SessionID: server.SessionID(), Stage: binding.Stage, CatalogSHA256: binding.Catalog.SHA256,
		EditorialMemory: memory, Document: document, Revision: 1,
		BaseArtifactID: artifact.ArtifactID, BaseSHA256: artifact.SHA256, BaseByteSize: int(artifact.ByteSize),
		CreatedAt: now, UpdatedAt: now,
	}
	server.Mu.Lock()
	defer server.Mu.Unlock()
	if len(server.Documents.Workspaces) != 0 {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "conflict", "report IL document workspace already exists", false, nil)
	}
	server.Documents.Workspaces[workspace.WorkspaceID] = workspace
	return wire.ToolResult{ToolName: call.Name, MissionID: workspace.MissionID, Content: reportILDocumentState(*workspace)}
}

func (server *SourceHandler) CallReportILDocumentAppend(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	_ = ctx
	binding, err := server.reportILAuthorDocumentBinding()
	if err != nil {
		return server.ErrorResult(call.Name, server.MissionID(), "binding", err.Error(), false, nil)
	}
	var input ReportILDocumentAppendInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, nil)
	}
	server.Mu.Lock()
	defer server.Mu.Unlock()
	workspace, err := server.DocumentWorkspace(input.WorkspaceID, binding)
	if err != nil {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, nil)
	}
	if workspace.Finalized || workspace.Finalizing {
		return server.ErrorResult(call.Name, workspace.MissionID, "conflict", "report IL document is already finalizing or finalized", false, []string{workspace.WorkspaceID, workspace.ArtifactID})
	}
	sourceKeys, err := reportilcontract.EditorialAccountSourceKeys(workspace.EditorialMemory, input.EditorialAccountKeys)
	if err != nil {
		return server.ErrorResult(call.Name, workspace.MissionID, "validation", err.Error(), false, []string{workspace.WorkspaceID})
	}
	block := reportilcontract.AuthorBlock{
		Kind: input.Kind, Prose: input.Prose, Items: append([]string(nil), input.Items...),
		Code: input.Code, Language: input.Language,
		EditorialAccountKeys: append([]string(nil), input.EditorialAccountKeys...),
		EvidenceSourceKeys:   sourceKeys,
	}
	if input.Table != nil {
		block.Table = &reportilcontract.AuthorTable{Caption: input.Table.Caption, Columns: append([]string(nil), input.Table.Columns...)}
		for _, row := range input.Table.Rows {
			block.Table.Rows = append(block.Table.Rows, reportilcontract.AuthorTableRow{Cells: append([]string(nil), row.Cells...)})
		}
	}
	if err := server.appendReportILDocumentBlock(workspace, binding, input.SectionTitle, block); err != nil {
		return server.ErrorResult(call.Name, workspace.MissionID, "validation", err.Error(), false, []string{workspace.WorkspaceID})
	}
	return wire.ToolResult{ToolName: call.Name, MissionID: workspace.MissionID, Content: reportILDocumentState(*workspace)}
}

func (server *SourceHandler) CallReportILDocumentAppendSource(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	_ = ctx
	binding, err := server.reportILAuthorDocumentBinding()
	if err != nil {
		return server.ErrorResult(call.Name, server.MissionID(), "binding", err.Error(), false, nil)
	}
	if binding.MaxReadBytes == 0 {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "binding", "report IL direct source authoring is unavailable for this stage", false, nil)
	}
	var input ReportILDocumentAppendSourceInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, nil)
	}
	server.Mu.Lock()
	defer server.Mu.Unlock()
	workspace, err := server.DocumentWorkspace(input.WorkspaceID, binding)
	if err != nil {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, nil)
	}
	if workspace.Finalized || workspace.Finalizing {
		return server.ErrorResult(call.Name, workspace.MissionID, "conflict", "report IL document is already finalizing or finalized", false, []string{workspace.WorkspaceID, workspace.ArtifactID})
	}
	block := reportilcontract.AuthorBlock{
		Kind: input.Kind, Prose: input.Prose, Items: append([]string(nil), input.Items...),
		Code: input.Code, Language: input.Language,
		EvidenceSourceKeys: append([]string(nil), input.EvidenceSourceKeys...),
	}
	if input.Table != nil {
		block.Table = &reportilcontract.AuthorTable{Caption: input.Table.Caption, Columns: append([]string(nil), input.Table.Columns...)}
		for _, row := range input.Table.Rows {
			block.Table.Rows = append(block.Table.Rows, reportilcontract.AuthorTableRow{Cells: append([]string(nil), row.Cells...)})
		}
	}
	if err := server.appendReportILDocumentBlock(workspace, binding, input.SectionTitle, block); err != nil {
		return server.ErrorResult(call.Name, workspace.MissionID, "validation", err.Error(), false, []string{workspace.WorkspaceID})
	}
	return wire.ToolResult{ToolName: call.Name, MissionID: workspace.MissionID, Content: reportILDocumentState(*workspace)}
}

func (server *SourceHandler) CallReportILDocumentRead(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	_ = ctx
	binding, err := server.reportILDocumentBinding()
	if err != nil {
		return server.ErrorResult(call.Name, server.MissionID(), "binding", err.Error(), false, nil)
	}
	var input ReportILDocumentReadInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, nil)
	}
	if input.Offset < 0 || input.MaxBytes < 1 || input.MaxBytes > ReportILDocumentMaxReadBytes {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "report IL document read range is invalid", false, []string{input.WorkspaceID})
	}
	server.Mu.Lock()
	defer server.Mu.Unlock()
	workspace, err := server.DocumentWorkspace(input.WorkspaceID, binding)
	if err != nil {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, nil)
	}
	if workspace.ReviewRevision != workspace.Revision {
		if input.Offset != 0 {
			return server.ErrorResult(call.Name, workspace.MissionID, "validation", "report IL document reread must start at offset 0", false, []string{workspace.WorkspaceID})
		}
		workspace.ReviewRevision = workspace.Revision
		workspace.ReviewNextOffset = 0
		workspace.ReviewComplete = false
	}
	if workspace.ReviewComplete || input.Offset != workspace.ReviewNextOffset {
		return server.ErrorResult(call.Name, workspace.MissionID, "validation", fmt.Sprintf("report IL document reads must continue at exact next_offset %d", workspace.ReviewNextOffset), false, []string{workspace.WorkspaceID})
	}
	review := reportILDocumentReviewText(workspace.Document)
	content, offset, nextOffset, truncated, err := server.BoundedRead([]byte(review), input.Offset, input.MaxBytes, input.MaxBytes)
	if err != nil {
		return server.ErrorResult(call.Name, workspace.MissionID, "validation", err.Error(), false, []string{workspace.WorkspaceID})
	}
	if truncated {
		workspace.ReviewNextOffset = nextOffset
	} else {
		workspace.ReviewNextOffset = 0
		workspace.ReviewComplete = true
	}
	return wire.ToolResult{ToolName: call.Name, MissionID: workspace.MissionID, Content: ReportILDocumentReadOutput{
		WorkspaceID: workspace.WorkspaceID, ReportILStage: workspace.Stage, Revision: workspace.Revision, Content: content,
		Offset: offset, NextOffset: nextOffset, ContentLength: len([]byte(review)), Truncated: truncated,
	}}
}

func (server *SourceHandler) CallReportILDocumentReplace(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	_ = ctx
	binding, err := server.reportILDocumentBinding()
	if err != nil {
		return server.ErrorResult(call.Name, server.MissionID(), "binding", err.Error(), false, nil)
	}
	var input ReportILDocumentReplaceInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, nil)
	}
	if strings.TrimSpace(input.OldText) == "" || input.OldText == input.NewText ||
		!utf8.ValidString(input.OldText) || !utf8.ValidString(input.NewText) ||
		len([]byte(input.OldText)) > reportILDocumentMaxReplaceBytes || len([]byte(input.NewText)) > reportILDocumentMaxReplaceBytes {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "report IL document replacement is invalid", false, []string{input.WorkspaceID})
	}
	server.Mu.Lock()
	defer server.Mu.Unlock()
	workspace, err := server.DocumentWorkspace(input.WorkspaceID, binding)
	if err != nil {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, nil)
	}
	if workspace.Finalized || workspace.Finalizing {
		return server.ErrorResult(call.Name, workspace.MissionID, "conflict", "report IL document is already finalizing or finalized", false, []string{workspace.WorkspaceID, workspace.ArtifactID})
	}
	candidate := cloneReportILAuthorDocument(workspace.Document)
	if !replaceUniqueReportILDocumentText(&candidate, input.OldText, input.NewText) {
		return server.ErrorResult(call.Name, workspace.MissionID, "validation", "old_text must occur exactly once in one reader-facing document value", false, []string{workspace.WorkspaceID})
	}
	if err := reportilcontract.ValidateAuthorDocumentPartial(candidate, binding.Catalog); err != nil {
		return server.ErrorResult(call.Name, workspace.MissionID, "validation", err.Error(), false, []string{workspace.WorkspaceID})
	}
	workspace.Document = candidate
	workspace.Replacements++
	server.reportILDocumentMutated(workspace)
	return wire.ToolResult{ToolName: call.Name, MissionID: workspace.MissionID, Content: reportILDocumentState(*workspace)}
}

func (server *SourceHandler) CallReportILDocumentEditText(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	_ = ctx
	binding, err := server.reportILPublicationReaderDocumentBinding()
	if err != nil {
		return server.ErrorResult(call.Name, server.MissionID(), "binding", err.Error(), false, nil)
	}
	var input ReportILDocumentEditTextInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, nil)
	}
	if strings.TrimSpace(input.TargetKey) == "" || strings.TrimSpace(input.OldText) == "" ||
		input.OldText == input.NewText || !utf8.ValidString(input.OldText) || !utf8.ValidString(input.NewText) ||
		len([]byte(input.OldText)) > reportILDocumentMaxReplaceBytes ||
		len([]byte(input.NewText)) > reportILDocumentMaxReplaceBytes {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "report IL publication text edit is invalid", false, []string{input.WorkspaceID})
	}
	server.Mu.Lock()
	defer server.Mu.Unlock()
	workspace, err := server.DocumentWorkspace(input.WorkspaceID, binding)
	if err != nil {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, nil)
	}
	if workspace.Finalized || workspace.Finalizing {
		return server.ErrorResult(call.Name, workspace.MissionID, "conflict", "report IL document is already finalizing or finalized", false, []string{workspace.WorkspaceID, workspace.ArtifactID})
	}
	candidate := cloneReportILAuthorDocument(workspace.Document)
	if !EditUniqueReportILDocumentReaderText(&candidate, input.TargetKind, strings.TrimSpace(input.TargetKey), input.OldText, input.NewText) {
		return server.ErrorResult(call.Name, workspace.MissionID, "validation", "target_kind and target_key must identify one editable reader-facing value where old_text occurs exactly once", false, []string{workspace.WorkspaceID})
	}
	if err := reportilcontract.ValidateAuthorDocumentPartial(candidate, binding.Catalog); err != nil {
		return server.ErrorResult(call.Name, workspace.MissionID, "validation", err.Error(), false, []string{workspace.WorkspaceID})
	}
	workspace.Document = candidate
	workspace.Replacements++
	server.reportILDocumentMutated(workspace)
	return wire.ToolResult{ToolName: call.Name, MissionID: workspace.MissionID, Content: reportILDocumentState(*workspace)}
}

func (server *SourceHandler) CallReportILDocumentReviseBlock(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	_ = ctx
	binding, err := server.reportILContinuityDocumentBinding()
	if err != nil {
		return server.ErrorResult(call.Name, server.MissionID(), "binding", err.Error(), false, nil)
	}
	var input ReportILDocumentReviseBlockInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, nil)
	}
	if strings.TrimSpace(input.BlockKey) == "" || strings.TrimSpace(input.OldText) == "" ||
		strings.TrimSpace(input.NewText) == "" || input.OldText == input.NewText ||
		!utf8.ValidString(input.OldText) || !utf8.ValidString(input.NewText) ||
		len([]byte(input.OldText)) > reportILDocumentMaxReplaceBytes ||
		len([]byte(input.NewText)) > reportILDocumentMaxReplaceBytes {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "report IL document block revision is invalid", false, []string{input.WorkspaceID})
	}
	server.Mu.Lock()
	defer server.Mu.Unlock()
	workspace, err := server.DocumentWorkspace(input.WorkspaceID, binding)
	if err != nil {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, nil)
	}
	if workspace.Finalized || workspace.Finalizing {
		return server.ErrorResult(call.Name, workspace.MissionID, "conflict", "report IL document is already finalizing or finalized", false, []string{workspace.WorkspaceID, workspace.ArtifactID})
	}
	candidate := cloneReportILAuthorDocument(workspace.Document)
	sourceKeys, err := reportilcontract.EditorialAccountSourceKeys(workspace.EditorialMemory, input.EditorialAccountKeys)
	if err != nil {
		return server.ErrorResult(call.Name, workspace.MissionID, "validation", err.Error(), false, []string{workspace.WorkspaceID})
	}
	if !reviseReportILDocumentBlock(&candidate, strings.TrimSpace(input.BlockKey), input.OldText, input.NewText, input.EditorialAccountKeys, sourceKeys) {
		return server.ErrorResult(call.Name, workspace.MissionID, "validation", "block_key must identify one content block where old_text occurs exactly once", false, []string{workspace.WorkspaceID})
	}
	if err := reportilcontract.ValidateAuthorDocumentPartial(candidate, binding.Catalog); err != nil {
		return server.ErrorResult(call.Name, workspace.MissionID, "validation", err.Error(), false, []string{workspace.WorkspaceID})
	}
	workspace.Document = candidate
	workspace.Replacements++
	server.reportILDocumentMutated(workspace)
	return wire.ToolResult{ToolName: call.Name, MissionID: workspace.MissionID, Content: reportILDocumentState(*workspace)}
}

func (server *SourceHandler) CallReportILDocumentFinalize(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	binding, err := server.reportILDocumentBinding()
	if err != nil {
		return server.ErrorResult(call.Name, server.MissionID(), "binding", err.Error(), false, nil)
	}
	var input ReportILDocumentFinalizeInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, nil)
	}
	server.Mu.Lock()
	workspace, err := server.DocumentWorkspace(input.WorkspaceID, binding)
	if err != nil {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, nil)
	}
	if workspace.Finalized || workspace.Finalizing {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, workspace.MissionID, "conflict", "report IL document is already finalizing or finalized", false, []string{workspace.WorkspaceID, workspace.ArtifactID})
	}
	if !workspace.ReviewComplete || workspace.ReviewRevision != workspace.Revision {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, workspace.MissionID, "validation", "report IL document must be reread completely after its last edit", false, []string{workspace.WorkspaceID})
	}
	var validationErr error
	if len(workspace.EditorialMemory.Accounts) == 0 {
		validationErr = reportilcontract.ValidateAuthorDocument(workspace.Document, binding.Catalog)
	} else {
		validationErr = reportilcontract.ValidateAuthorDocumentWithMemory(workspace.Document, workspace.EditorialMemory, binding.Catalog)
	}
	if validationErr != nil {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, workspace.MissionID, "validation", validationErr.Error(), false, []string{workspace.WorkspaceID})
	}
	content, err := json.Marshal(workspace.Document)
	if err != nil {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, workspace.MissionID, "internal", "report IL document encoding failed", false, []string{workspace.WorkspaceID})
	}
	content = append(content, '\n')
	workspace.Finalizing = true
	workspaceCopy := *workspace
	server.Mu.Unlock()

	storedArtifact := artifact.Raw{}
	if (workspaceCopy.Stage == "il_continuity" || workspaceCopy.Stage == "il_reader") && workspaceCopy.Replacements == 0 {
		storedArtifact, err = server.Artifacts.GetRawArtifact(ctx, workspaceCopy.BaseArtifactID)
		if err == nil && (storedArtifact.MissionID != workspaceCopy.MissionID ||
			storedArtifact.MediaType != reportilcontract.AuthorDocumentMediaType ||
			storedArtifact.SHA256 != workspaceCopy.BaseSHA256 || int(storedArtifact.ByteSize) != workspaceCopy.BaseByteSize ||
			!bytes.Equal(storedArtifact.Content, content)) {
			err = fmt.Errorf("report IL bound base document changed")
		}
	} else {
		artifactID := server.NewID("art")
		storedArtifact, err = server.Artifacts.CreateRawArtifact(ctx, artifact.CreateRequest{
			ArtifactID: artifactID, MissionID: workspaceCopy.MissionID,
			MediaType: reportilcontract.AuthorDocumentMediaType,
			Filename:  "report-il-author-document.json",
			Producer:  ledger.Producer{Type: "mcp_tool", ID: reportilcontract.AuthorDocumentFinalizeTool},
			Content:   content,
		})
	}
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
		return server.ErrorResult(call.Name, workspaceCopy.MissionID, "conflict", "report IL document changed during finalization", false, []string{workspaceCopy.WorkspaceID, storedArtifact.ArtifactID})
	}
	workspace.Finalizing = false
	workspace.Finalized = true
	workspace.ArtifactID = storedArtifact.ArtifactID
	workspace.SHA256 = storedArtifact.SHA256
	workspace.ByteSize = int(storedArtifact.ByteSize)
	workspace.UpdatedAt = time.Now().UTC()
	return wire.ToolResult{ToolName: call.Name, MissionID: workspace.MissionID, Content: reportILDocumentState(*workspace)}
}

func (server *SourceHandler) reportILAuthorDocumentBinding() (reportilcontract.SourceAccessBinding, error) {
	binding, err := server.AccessBinding()
	if err != nil {
		return reportilcontract.SourceAccessBinding{}, err
	}
	if binding.Stage != "il_narrative" {
		return reportilcontract.SourceAccessBinding{}, fmt.Errorf("report IL author document workspace is unavailable for this stage")
	}
	return binding, nil
}

func (server *SourceHandler) reportILPublicationDocumentBinding() (reportilcontract.SourceAccessBinding, error) {
	binding, err := server.AccessBinding()
	if err != nil {
		return reportilcontract.SourceAccessBinding{}, err
	}
	if binding.Stage != "il_continuity" && binding.Stage != "il_reader" {
		return reportilcontract.SourceAccessBinding{}, fmt.Errorf("report IL publication document workspace is unavailable for this stage")
	}
	return binding, nil
}

func (server *SourceHandler) reportILContinuityDocumentBinding() (reportilcontract.SourceAccessBinding, error) {
	binding, err := server.AccessBinding()
	if err != nil {
		return reportilcontract.SourceAccessBinding{}, err
	}
	if binding.Stage != "il_continuity" {
		return reportilcontract.SourceAccessBinding{}, fmt.Errorf("report IL continuity document workspace is unavailable for this stage")
	}
	return binding, nil
}

func (server *SourceHandler) reportILPublicationReaderDocumentBinding() (reportilcontract.SourceAccessBinding, error) {
	binding, err := server.AccessBinding()
	if err != nil {
		return reportilcontract.SourceAccessBinding{}, err
	}
	if binding.Stage != "il_reader" {
		return reportilcontract.SourceAccessBinding{}, fmt.Errorf("report IL publication reader document workspace is unavailable for this stage")
	}
	return binding, nil
}

func (server *SourceHandler) reportILDocumentBinding() (reportilcontract.SourceAccessBinding, error) {
	binding, err := server.AccessBinding()
	if err != nil {
		return reportilcontract.SourceAccessBinding{}, err
	}
	if binding.Stage != "il_narrative" && binding.Stage != "il_continuity" && binding.Stage != "il_reader" &&
		binding.Stage != "il_long_form_section" && binding.Stage != "il_long_form_part" && binding.Stage != "il_long_form_final" {
		return reportilcontract.SourceAccessBinding{}, fmt.Errorf("report IL document workspace is unavailable for this stage")
	}
	return binding, nil
}

func (server *SourceHandler) reportILCompleteCatalogRead(binding reportilcontract.SourceAccessBinding) bool {
	for _, entry := range binding.Catalog.Sources {
		if !server.State.Complete[entry.SourceKey] {
			return false
		}
	}
	return true
}

func (server *SourceHandler) DocumentWorkspace(workspaceID string, binding reportilcontract.SourceAccessBinding) (*DocumentWorkspace, error) {
	workspace := server.Documents.Workspaces[strings.TrimSpace(workspaceID)]
	if workspace == nil || workspace.MissionID != binding.Catalog.MissionID ||
		workspace.SessionID != server.SessionID() || workspace.CatalogSHA256 != binding.Catalog.SHA256 {
		return nil, fmt.Errorf("report IL document workspace is unavailable")
	}
	return workspace, nil
}

func (server *SourceHandler) reportILDocumentMutated(workspace *DocumentWorkspace) {
	workspace.Revision++
	workspace.ReviewRevision = 0
	workspace.ReviewNextOffset = 0
	workspace.ReviewComplete = false
	workspace.UpdatedAt = time.Now().UTC()
}

func reportILDocumentState(workspace DocumentWorkspace) ReportILDocumentStateOutput {
	content, _ := json.Marshal(workspace.Document)
	byteSize := len(content)
	if workspace.Finalized {
		byteSize = workspace.ByteSize
	}
	return ReportILDocumentStateOutput{
		WorkspaceID: workspace.WorkspaceID, ReportILStage: workspace.Stage, Revision: workspace.Revision,
		Replacements: workspace.Replacements,
		Sections:     reportILDocumentSectionCount(workspace.Document), Blocks: reportILDocumentBlockCount(workspace.Document),
		ByteSize: byteSize, Finalized: workspace.Finalized,
		ArtifactID: workspace.ArtifactID, SHA256: workspace.SHA256,
	}
}

func (server *SourceHandler) appendReportILDocumentBlock(
	workspace *DocumentWorkspace,
	binding reportilcontract.SourceAccessBinding,
	sectionTitle string,
	block reportilcontract.AuthorBlock,
) error {
	sectionTitle = strings.TrimSpace(sectionTitle)
	if sectionTitle == "" || !utf8.ValidString(sectionTitle) {
		return fmt.Errorf("report IL document section title is invalid")
	}
	sections := workspace.Document.Sections
	if len(sections) == 0 || sections[len(sections)-1].Title != sectionTitle {
		if len(sections) >= reportilcontract.MaxAuthorSections {
			return fmt.Errorf("report IL document section inventory exceeds the ceiling")
		}
		sections = append(sections, reportilcontract.AuthorSection{
			SectionKey: fmt.Sprintf("section_%03d", len(sections)+1),
			Title:      sectionTitle,
		})
	} else if len(sections[len(sections)-1].Blocks) == 0 {
		return fmt.Errorf("report IL document section state is invalid")
	}
	section := &sections[len(sections)-1]
	block.BlockKey = fmt.Sprintf("%s.block_%03d", section.SectionKey, len(section.Blocks)+1)
	section.Blocks = append(section.Blocks, block)
	workspace.Document.Sections = sections
	if err := reportilcontract.ValidateAuthorDocumentPartial(workspace.Document, binding.Catalog); err != nil {
		section.Blocks = section.Blocks[:len(section.Blocks)-1]
		if len(section.Blocks) == 0 {
			workspace.Document.Sections = workspace.Document.Sections[:len(workspace.Document.Sections)-1]
		}
		return err
	}
	server.reportILDocumentMutated(workspace)
	return nil
}

func reportILDocumentSectionCount(document reportilcontract.AuthorDocument) int {
	count := len(document.Sections)
	for _, part := range document.Parts {
		count += len(part.Sections)
	}
	return count
}

func reportILDocumentBlockCount(document reportilcontract.AuthorDocument) int {
	count := 0
	for _, section := range document.Sections {
		count += len(section.Blocks)
	}
	for _, part := range document.Parts {
		for _, section := range part.Sections {
			count += len(section.Blocks)
		}
	}
	return count
}

func reportILDocumentLanguageValid(language string) bool {
	if language == "" || len(language) > 32 {
		return false
	}
	for index, r := range language {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (index > 0 && ((r >= '0' && r <= '9') || r == '-')) {
			continue
		}
		return false
	}
	return true
}

func reportILDocumentReviewText(document reportilcontract.AuthorDocument) string {
	var out strings.Builder
	out.WriteString("TITLE document [target_kind=title]\n")
	out.WriteString(document.Title)
	out.WriteString("\n\nLANGUAGE\n")
	out.WriteString(document.Language)
	writeSection := func(section reportilcontract.AuthorSection) {
		out.WriteString("\n\nSECTION ")
		out.WriteString(section.SectionKey)
		out.WriteString(" [target_kind=section_title]\n")
		out.WriteString(section.Title)
		for _, block := range section.Blocks {
			out.WriteString("\n\nBLOCK ")
			out.WriteString(block.BlockKey)
			out.WriteString(" (")
			out.WriteString(block.Kind)
			out.WriteString(")")
			if block.Kind == "prose" || block.Kind == "quote" || block.Kind == "callout" {
				out.WriteString(" [target_kind=block_text]")
			}
			out.WriteString("\n")
			switch block.Kind {
			case "prose", "quote", "callout":
				out.WriteString(block.Prose)
			case "list":
				for _, item := range block.Items {
					out.WriteString("- ")
					out.WriteString(item)
					out.WriteString("\n")
				}
			case "code":
				out.WriteString(block.Code)
			case "equation":
				if block.Equation != nil {
					out.WriteString(block.Equation.Expression)
				}
			case "table":
				if block.Table.Caption != nil {
					out.WriteString(*block.Table.Caption)
					out.WriteString("\n")
				}
				out.WriteString(strings.Join(block.Table.Columns, " | "))
				out.WriteString("\n")
				for _, row := range block.Table.Rows {
					out.WriteString(strings.Join(row.Cells, " | "))
					out.WriteString("\n")
				}
			}
			out.WriteString("\nACCOUNTS: ")
			out.WriteString(strings.Join(block.EditorialAccountKeys, ", "))
			out.WriteString("\nSOURCES: ")
			out.WriteString(strings.Join(block.EvidenceSourceKeys, ", "))
		}
	}
	for _, section := range document.Sections {
		writeSection(section)
	}
	for _, part := range document.Parts {
		out.WriteString("\n\nPART ")
		out.WriteString(part.PartKey)
		out.WriteString(" [target_kind=part_title]\n")
		out.WriteString(part.Title)
		for _, section := range part.Sections {
			writeSection(section)
		}
	}
	return out.String()
}

func cloneReportILAuthorDocument(document reportilcontract.AuthorDocument) reportilcontract.AuthorDocument {
	encoded, _ := json.Marshal(document)
	var clone reportilcontract.AuthorDocument
	_ = json.Unmarshal(encoded, &clone)
	return clone
}

func replaceUniqueReportILDocumentText(document *reportilcontract.AuthorDocument, oldText, newText string) bool {
	matches := 0
	var target *string
	consider := func(value *string) {
		count := strings.Count(*value, oldText)
		matches += count
		if count == 1 {
			target = value
		}
	}
	considerSection := func(section *reportilcontract.AuthorSection) {
		consider(&section.Title)
		for blockIndex := range section.Blocks {
			considerReportILDocumentBlockText(&section.Blocks[blockIndex], consider)
		}
	}
	consider(&document.Title)
	for sectionIndex := range document.Sections {
		considerSection(&document.Sections[sectionIndex])
	}
	for partIndex := range document.Parts {
		part := &document.Parts[partIndex]
		consider(&part.Title)
		for sectionIndex := range part.Sections {
			considerSection(&part.Sections[sectionIndex])
		}
	}
	if matches != 1 || target == nil {
		return false
	}
	*target = strings.Replace(*target, oldText, newText, 1)
	return true
}

func EditUniqueReportILDocumentReaderText(
	document *reportilcontract.AuthorDocument,
	targetKind,
	targetKey,
	oldText,
	newText string,
) bool {
	var target *string
	switch targetKind {
	case "title":
		if targetKey == "document" {
			target = &document.Title
		}
	case "part_title":
		for partIndex := range document.Parts {
			part := &document.Parts[partIndex]
			if part.PartKey == targetKey {
				if target != nil {
					return false
				}
				target = &part.Title
			}
		}
	case "section_title":
		visit := func(section *reportilcontract.AuthorSection) bool {
			if section.SectionKey != targetKey {
				return true
			}
			if target != nil {
				return false
			}
			target = &section.Title
			return true
		}
		for sectionIndex := range document.Sections {
			if !visit(&document.Sections[sectionIndex]) {
				return false
			}
		}
		for partIndex := range document.Parts {
			for sectionIndex := range document.Parts[partIndex].Sections {
				if !visit(&document.Parts[partIndex].Sections[sectionIndex]) {
					return false
				}
			}
		}
	case "block_text":
		visit := func(section *reportilcontract.AuthorSection) bool {
			for blockIndex := range section.Blocks {
				block := &section.Blocks[blockIndex]
				if block.BlockKey != targetKey {
					continue
				}
				if target != nil || block.Kind != "prose" && block.Kind != "quote" && block.Kind != "callout" {
					return false
				}
				target = &block.Prose
			}
			return true
		}
		for sectionIndex := range document.Sections {
			if !visit(&document.Sections[sectionIndex]) {
				return false
			}
		}
		for partIndex := range document.Parts {
			for sectionIndex := range document.Parts[partIndex].Sections {
				if !visit(&document.Parts[partIndex].Sections[sectionIndex]) {
					return false
				}
			}
		}
	default:
		return false
	}
	if target == nil || strings.Count(*target, oldText) != 1 {
		return false
	}
	*target = strings.Replace(*target, oldText, newText, 1)
	return true
}

func reviseReportILDocumentBlock(
	document *reportilcontract.AuthorDocument,
	blockKey,
	oldText,
	newText string,
	editorialAccountKeys,
	evidenceSourceKeys []string,
) bool {
	var block *reportilcontract.AuthorBlock
	considerSection := func(section *reportilcontract.AuthorSection) bool {
		for blockIndex := range section.Blocks {
			candidate := &section.Blocks[blockIndex]
			if candidate.BlockKey != blockKey {
				continue
			}
			if block != nil {
				return false
			}
			block = candidate
		}
		return true
	}
	for sectionIndex := range document.Sections {
		if !considerSection(&document.Sections[sectionIndex]) {
			return false
		}
	}
	for partIndex := range document.Parts {
		for sectionIndex := range document.Parts[partIndex].Sections {
			if !considerSection(&document.Parts[partIndex].Sections[sectionIndex]) {
				return false
			}
		}
	}
	if block == nil {
		return false
	}
	matches := 0
	var target *string
	considerReportILDocumentBlockText(block, func(value *string) {
		count := strings.Count(*value, oldText)
		matches += count
		if count == 1 {
			target = value
		}
	})
	if matches != 1 || target == nil {
		return false
	}
	*target = strings.Replace(*target, oldText, newText, 1)
	block.EditorialAccountKeys = append([]string(nil), editorialAccountKeys...)
	block.EvidenceSourceKeys = append([]string(nil), evidenceSourceKeys...)
	return true
}

func (server *SourceHandler) reportILBoundEditorialMemory(
	ctx context.Context,
	binding reportilcontract.SourceAccessBinding,
) (reportilcontract.EditorialMemory, error) {
	artifactValue, err := server.Artifacts.GetRawArtifact(ctx, binding.EditorialMemoryArtifactID)
	if err != nil {
		return reportilcontract.EditorialMemory{}, err
	}
	if artifactValue.MissionID != binding.Catalog.MissionID ||
		artifactValue.MediaType != reportilcontract.EditorialMemoryMediaType ||
		artifactValue.SHA256 != binding.EditorialMemorySHA256 ||
		server.SHA256(artifactValue.Content) != binding.EditorialMemorySHA256 ||
		int64(len(artifactValue.Content)) != artifactValue.ByteSize {
		return reportilcontract.EditorialMemory{}, fmt.Errorf("editorial memory artifact binding is invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(artifactValue.Content))
	decoder.DisallowUnknownFields()
	var memoryArtifact reportilcontract.EditorialMemoryArtifact
	if err := decoder.Decode(&memoryArtifact); err != nil {
		return reportilcontract.EditorialMemory{}, fmt.Errorf("editorial memory decode failed")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return reportilcontract.EditorialMemory{}, fmt.Errorf("editorial memory contains multiple JSON values")
	}
	if err := reportilcontract.ValidateEditorialMemoryArtifact(memoryArtifact, binding.Catalog); err != nil {
		return reportilcontract.EditorialMemory{}, err
	}
	return reportilcontract.EditorialMemoryFromArtifact(memoryArtifact), nil
}

func considerReportILDocumentBlockText(block *reportilcontract.AuthorBlock, consider func(*string)) {
	switch block.Kind {
	case "prose", "quote", "callout":
		consider(&block.Prose)
	case "list":
		for index := range block.Items {
			consider(&block.Items[index])
		}
	case "code":
		consider(&block.Code)
	case "equation":
		if block.Equation != nil {
			consider(&block.Equation.Expression)
		}
	case "table":
		if block.Table == nil {
			return
		}
		if block.Table.Caption != nil {
			consider(block.Table.Caption)
		}
		for index := range block.Table.Columns {
			consider(&block.Table.Columns[index])
		}
		for rowIndex := range block.Table.Rows {
			for cellIndex := range block.Table.Rows[rowIndex].Cells {
				consider(&block.Table.Rows[rowIndex].Cells[cellIndex])
			}
		}
	}
}
