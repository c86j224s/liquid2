package reportil

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
)

type EditorialMemoryWorkspace struct {
	WorkspaceID      string
	MissionID        string
	SessionID        string
	CatalogSHA256    string
	Memory           reportilcontract.EditorialMemory
	Anchors          []reportilcontract.EditorialAnchor
	Revision         int
	ReviewRevision   int
	ReviewNextOffset int
	ReviewComplete   bool
	Finalizing       bool
	Finalized        bool
	ArtifactID       string
	SHA256           string
	ByteSize         int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (server *SourceHandler) CallReportILEditorialMemoryStart(_ context.Context, call wire.ToolCall) wire.ToolResult {
	binding, err := server.AccessBinding()
	if err != nil {
		return server.ErrorResult(call.Name, server.MissionID(), "binding", err.Error(), false, nil)
	}
	if binding.Stage != "il_editorial_memory" {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "binding", "editorial memory is unavailable for this stage", false, nil)
	}
	var input ReportILEditorialMemoryStartInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, nil)
	}
	if !reportILDocumentLanguageValid(input.Language) {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "editorial memory language is invalid", false, nil)
	}
	server.Mu.Lock()
	defer server.Mu.Unlock()
	if !server.reportILCompleteCatalogRead(binding) {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "editorial memory requires the complete frozen catalog read", false, nil)
	}
	if len(server.Editorial.Workspaces) != 0 {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "conflict", "editorial memory workspace already exists", false, nil)
	}
	now := time.Now().UTC()
	workspace := &EditorialMemoryWorkspace{
		WorkspaceID: server.NewID("ilm"), MissionID: binding.Catalog.MissionID,
		SessionID: server.SessionID(), CatalogSHA256: binding.Catalog.SHA256,
		Memory: reportilcontract.EditorialMemory{
			SchemaVersion: reportilcontract.EditorialMemorySchemaVersion,
			Language:      input.Language,
		},
		Revision: 1, CreatedAt: now, UpdatedAt: now,
	}
	server.Editorial.Workspaces[workspace.WorkspaceID] = workspace
	return wire.ToolResult{ToolName: call.Name, MissionID: workspace.MissionID, Content: reportILEditorialMemoryState(*workspace)}
}

func (server *SourceHandler) CallReportILEditorialMemoryAppend(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	binding, err := server.AccessBinding()
	if err != nil {
		return server.ErrorResult(call.Name, server.MissionID(), "binding", err.Error(), false, nil)
	}
	if binding.Stage != "il_editorial_memory" {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "binding", "editorial memory is unavailable for this stage", false, nil)
	}
	var input ReportILEditorialMemoryAppendInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, nil)
	}
	if (input.Importance != "essential" && input.Importance != "supporting") ||
		strings.TrimSpace(input.Account) == "" || !utf8.ValidString(input.Account) || len([]byte(input.Account)) > 8192 ||
		len(input.SourceAnchors) == 0 || len(input.SourceAnchors) > reportilcontract.MaxEditorialAnchorsPerAccount {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "editorial memory account is invalid", false, nil)
	}
	server.Mu.Lock()
	defer server.Mu.Unlock()
	workspace, err := server.EditorialMemoryWorkspace(input.WorkspaceID, binding)
	if err != nil {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, nil)
	}
	if workspace.Finalizing || workspace.Finalized {
		return server.ErrorResult(call.Name, workspace.MissionID, "conflict", "editorial memory is already finalizing or finalized", false, []string{workspace.WorkspaceID, workspace.ArtifactID})
	}
	declaredSources := make(map[string]bool, len(input.SourceKeys))
	for _, sourceKey := range input.SourceKeys {
		declaredSources[sourceKey] = true
	}
	anchors := make([]reportilcontract.EditorialAnchor, 0, len(input.SourceAnchors))
	seenReceipts := make(map[string]bool, len(input.SourceAnchors))
	for _, sourceReceipt := range input.SourceAnchors {
		registered, ok := server.State.Quotes[sourceReceipt]
		if !ok || seenReceipts[sourceReceipt] || server.Editorial.UsedAnchors[sourceReceipt] || !declaredSources[registered.SourceKey] {
			return server.ErrorResult(call.Name, workspace.MissionID, "validation", "editorial memory source anchor is invalid", false, []string{workspace.WorkspaceID})
		}
		entry, ok := binding.Catalog.Entry(registered.SourceKey)
		if !ok {
			return server.ErrorResult(call.Name, workspace.MissionID, "validation", "editorial memory source anchor is invalid", false, []string{workspace.WorkspaceID})
		}
		readable, readErr := server.ReadableSource(ctx, binding, entry)
		if readErr != nil || registered.Offset < 0 || registered.Offset > len(readable.Text)-registered.ByteSize {
			return server.ErrorResult(call.Name, workspace.MissionID, "validation", "frozen editorial memory source anchor validation failed", false, []string{workspace.WorkspaceID})
		}
		excerpt := readable.Text[registered.Offset : registered.Offset+registered.ByteSize]
		if server.SHA256([]byte(excerpt)) != registered.SHA256 {
			return server.ErrorResult(call.Name, workspace.MissionID, "validation", "frozen editorial memory source anchor validation failed", false, []string{workspace.WorkspaceID})
		}
		anchors = append(anchors, reportilcontract.EditorialAnchor{
			AccountKey: fmt.Sprintf("account_%03d", len(workspace.Memory.Accounts)+1),
			SourceKey:  registered.SourceKey, Excerpt: excerpt, Offset: registered.Offset,
			ByteSize: registered.ByteSize, SHA256: registered.SHA256,
		})
		seenReceipts[sourceReceipt] = true
	}
	for sourceKey := range declaredSources {
		anchored := false
		for _, anchor := range anchors {
			if anchor.SourceKey == sourceKey {
				anchored = true
				break
			}
		}
		if !anchored {
			return server.ErrorResult(call.Name, workspace.MissionID, "validation", "editorial memory account source lacks an exact anchor", false, []string{workspace.WorkspaceID})
		}
	}
	account := reportilcontract.EditorialAccount{
		AccountKey: fmt.Sprintf("account_%03d", len(workspace.Memory.Accounts)+1),
		Importance: input.Importance,
		Account:    strings.TrimSpace(input.Account), SourceKeys: append([]string(nil), input.SourceKeys...),
	}
	workspace.Memory.Accounts = append(workspace.Memory.Accounts, account)
	workspace.Anchors = append(workspace.Anchors, anchors...)
	if err := reportilcontract.ValidateEditorialMemoryArtifact(reportilcontract.EditorialMemoryArtifact{
		SchemaVersion: workspace.Memory.SchemaVersion, Language: workspace.Memory.Language,
		Accounts: workspace.Memory.Accounts, Anchors: workspace.Anchors,
	}, binding.Catalog); err != nil {
		workspace.Memory.Accounts = workspace.Memory.Accounts[:len(workspace.Memory.Accounts)-1]
		workspace.Anchors = workspace.Anchors[:len(workspace.Anchors)-len(anchors)]
		return server.ErrorResult(call.Name, workspace.MissionID, "validation", err.Error(), false, []string{workspace.WorkspaceID})
	}
	for sourceReceipt := range seenReceipts {
		server.Editorial.UsedAnchors[sourceReceipt] = true
	}
	reportILEditorialMemoryMutated(workspace)
	return wire.ToolResult{ToolName: call.Name, MissionID: workspace.MissionID, Content: reportILEditorialMemoryState(*workspace)}
}

func (server *SourceHandler) CallReportILEditorialMemoryRead(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	binding, err := server.AccessBinding()
	if err != nil {
		return server.ErrorResult(call.Name, server.MissionID(), "binding", err.Error(), false, nil)
	}
	var input ReportILEditorialMemoryReadInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, nil)
	}
	if input.Offset < 0 || input.MaxBytes < 1 || input.MaxBytes > ReportILDocumentMaxReadBytes {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "editorial memory read range is invalid", false, []string{input.WorkspaceID})
	}
	if binding.Stage == "il_editorial_memory" {
		server.Mu.Lock()
		defer server.Mu.Unlock()
		workspace, workspaceErr := server.EditorialMemoryWorkspace(input.WorkspaceID, binding)
		if workspaceErr != nil {
			return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", workspaceErr.Error(), false, nil)
		}
		if workspace.ReviewRevision != workspace.Revision {
			if input.Offset != 0 {
				return server.ErrorResult(call.Name, workspace.MissionID, "validation", "editorial memory reread must start at offset 0", false, []string{workspace.WorkspaceID})
			}
			workspace.ReviewRevision = workspace.Revision
			workspace.ReviewNextOffset = 0
			workspace.ReviewComplete = false
		}
		if workspace.ReviewComplete || input.Offset != workspace.ReviewNextOffset {
			return server.ErrorResult(call.Name, workspace.MissionID, "validation", fmt.Sprintf("editorial memory reads must continue at exact next_offset %d", workspace.ReviewNextOffset), false, []string{workspace.WorkspaceID})
		}
		encoded, _ := json.MarshalIndent(reportilcontract.EditorialMemoryArtifact{
			SchemaVersion: workspace.Memory.SchemaVersion, Language: workspace.Memory.Language,
			Accounts: workspace.Memory.Accounts, Anchors: workspace.Anchors,
		}, "", "  ")
		content, offset, nextOffset, truncated, rangeErr := server.BoundedRead(encoded, input.Offset, input.MaxBytes, input.MaxBytes)
		if rangeErr != nil {
			return server.ErrorResult(call.Name, workspace.MissionID, "validation", rangeErr.Error(), false, []string{workspace.WorkspaceID})
		}
		if truncated {
			workspace.ReviewNextOffset = nextOffset
		} else {
			workspace.ReviewNextOffset = 0
			workspace.ReviewComplete = true
		}
		return wire.ToolResult{ToolName: call.Name, MissionID: workspace.MissionID, Content: ReportILEditorialMemoryReadOutput{
			WorkspaceID: workspace.WorkspaceID, ReportILStage: binding.Stage,
			Revision: workspace.Revision, Content: content,
			Offset: offset, NextOffset: nextOffset, ContentLength: len(encoded), Truncated: truncated,
		}}
	}
	if input.WorkspaceID != "" || (binding.Stage != "il_narrative" && binding.Stage != "il_continuity" && binding.Stage != "il_reader" && !strings.HasPrefix(binding.Stage, "il_long_form_")) {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "editorial memory read binding is invalid", false, nil)
	}
	artifactValue, err := server.Artifacts.GetRawArtifact(ctx, binding.EditorialMemoryArtifactID)
	if err != nil {
		return server.ErrorFromErr(call.Name, binding.Catalog.MissionID, err, []string{binding.EditorialMemoryArtifactID})
	}
	if artifactValue.MissionID != binding.Catalog.MissionID || artifactValue.MediaType != reportilcontract.EditorialMemoryMediaType ||
		artifactValue.SHA256 != binding.EditorialMemorySHA256 || server.SHA256(artifactValue.Content) != binding.EditorialMemorySHA256 ||
		int64(len(artifactValue.Content)) != artifactValue.ByteSize {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", "editorial memory artifact binding is invalid", false, []string{binding.EditorialMemoryArtifactID})
	}
	server.Mu.Lock()
	defer server.Mu.Unlock()
	if server.Editorial.ReviewComplete || input.Offset != server.Editorial.ReviewNextOffset {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", fmt.Sprintf("editorial memory reads must start at offset 0 and continue at exact next_offset %d", server.Editorial.ReviewNextOffset), false, []string{binding.EditorialMemoryArtifactID})
	}
	content, offset, nextOffset, truncated, err := server.BoundedRead(artifactValue.Content, input.Offset, input.MaxBytes, input.MaxBytes)
	if err != nil {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, []string{binding.EditorialMemoryArtifactID})
	}
	if truncated {
		server.Editorial.ReviewNextOffset = nextOffset
	} else {
		server.Editorial.ReviewNextOffset = 0
		server.Editorial.ReviewComplete = true
	}
	return wire.ToolResult{ToolName: call.Name, MissionID: binding.Catalog.MissionID, Content: ReportILEditorialMemoryReadOutput{
		ReportILStage: binding.Stage, Revision: 1, Content: content,
		Offset: offset, NextOffset: nextOffset,
		ContentLength: len(artifactValue.Content), Truncated: truncated,
	}}
}

func (server *SourceHandler) CallReportILEditorialMemoryFinalize(ctx context.Context, call wire.ToolCall) wire.ToolResult {
	binding, err := server.AccessBinding()
	if err != nil {
		return server.ErrorResult(call.Name, server.MissionID(), "binding", err.Error(), false, nil)
	}
	if binding.Stage != "il_editorial_memory" {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "binding", "editorial memory is unavailable for this stage", false, nil)
	}
	var input ReportILEditorialMemoryFinalizeInput
	if err := server.Decode(call.Arguments, &input); err != nil {
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, nil)
	}
	server.Mu.Lock()
	workspace, err := server.EditorialMemoryWorkspace(input.WorkspaceID, binding)
	if err != nil {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, binding.Catalog.MissionID, "validation", err.Error(), false, nil)
	}
	if workspace.Finalizing || workspace.Finalized {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, workspace.MissionID, "conflict", "editorial memory is already finalizing or finalized", false, []string{workspace.WorkspaceID, workspace.ArtifactID})
	}
	if !workspace.ReviewComplete || workspace.ReviewRevision != workspace.Revision {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, workspace.MissionID, "validation", "editorial memory must be reread completely after its last append", false, []string{workspace.WorkspaceID})
	}
	memoryArtifact := reportilcontract.EditorialMemoryArtifact{
		SchemaVersion: workspace.Memory.SchemaVersion, Language: workspace.Memory.Language,
		Accounts: workspace.Memory.Accounts, Anchors: workspace.Anchors,
	}
	if err := reportilcontract.ValidateEditorialMemoryArtifact(memoryArtifact, binding.Catalog); err != nil {
		server.Mu.Unlock()
		return server.ErrorResult(call.Name, workspace.MissionID, "validation", err.Error(), false, []string{workspace.WorkspaceID})
	}
	content, _ := json.Marshal(memoryArtifact)
	content = append(content, '\n')
	workspace.Finalizing = true
	workspaceCopy := *workspace
	server.Mu.Unlock()

	stored, err := server.Artifacts.CreateRawArtifact(ctx, artifact.CreateRequest{
		ArtifactID: server.NewID("art"), MissionID: workspaceCopy.MissionID,
		MediaType: reportilcontract.EditorialMemoryMediaType,
		Filename:  "report-il-editorial-memory.json",
		Producer:  ledger.Producer{Type: "mcp_tool", ID: reportilcontract.EditorialMemoryFinalizeTool},
		Content:   content,
	})
	if err != nil {
		server.Mu.Lock()
		if current, currentErr := server.EditorialMemoryWorkspace(input.WorkspaceID, binding); currentErr == nil && current.Revision == workspaceCopy.Revision {
			current.Finalizing = false
		}
		server.Mu.Unlock()
		return server.ErrorFromErr(call.Name, workspaceCopy.MissionID, err, []string{workspaceCopy.WorkspaceID})
	}
	server.Mu.Lock()
	defer server.Mu.Unlock()
	workspace, err = server.EditorialMemoryWorkspace(input.WorkspaceID, binding)
	if err != nil || workspace.Finalized || !workspace.Finalizing || workspace.Revision != workspaceCopy.Revision {
		return server.ErrorResult(call.Name, workspaceCopy.MissionID, "conflict", "editorial memory changed during finalization", false, []string{workspaceCopy.WorkspaceID, stored.ArtifactID})
	}
	workspace.Finalizing = false
	workspace.Finalized = true
	workspace.ArtifactID = stored.ArtifactID
	workspace.SHA256 = stored.SHA256
	workspace.ByteSize = int(stored.ByteSize)
	workspace.UpdatedAt = time.Now().UTC()
	return wire.ToolResult{ToolName: call.Name, MissionID: workspace.MissionID, Content: reportILEditorialMemoryState(*workspace)}
}

func (server *SourceHandler) EditorialMemoryWorkspace(workspaceID string, binding reportilcontract.SourceAccessBinding) (*EditorialMemoryWorkspace, error) {
	workspace := server.Editorial.Workspaces[strings.TrimSpace(workspaceID)]
	if workspace == nil || workspace.MissionID != binding.Catalog.MissionID ||
		workspace.SessionID != server.SessionID() || workspace.CatalogSHA256 != binding.Catalog.SHA256 {
		return nil, fmt.Errorf("editorial memory workspace is unavailable")
	}
	return workspace, nil
}

func reportILEditorialMemoryMutated(workspace *EditorialMemoryWorkspace) {
	workspace.Revision++
	workspace.ReviewRevision = 0
	workspace.ReviewNextOffset = 0
	workspace.ReviewComplete = false
	workspace.UpdatedAt = time.Now().UTC()
}

func reportILEditorialMemoryState(workspace EditorialMemoryWorkspace) ReportILEditorialMemoryStateOutput {
	encoded, _ := json.Marshal(reportilcontract.EditorialMemoryArtifact{
		SchemaVersion: workspace.Memory.SchemaVersion, Language: workspace.Memory.Language,
		Accounts: workspace.Memory.Accounts, Anchors: workspace.Anchors,
	})
	byteSize := len(encoded)
	if workspace.Finalized {
		byteSize = workspace.ByteSize
	}
	return ReportILEditorialMemoryStateOutput{
		WorkspaceID: workspace.WorkspaceID, Revision: workspace.Revision,
		Accounts: len(workspace.Memory.Accounts), ByteSize: byteSize,
		Finalized: workspace.Finalized, ArtifactID: workspace.ArtifactID, SHA256: workspace.SHA256,
	}
}
