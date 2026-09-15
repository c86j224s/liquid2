package mcp

import (
	"context"
	"encoding/json"
	"errors"
	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/reportil"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
	"reflect"
	"strings"
	"testing"
)

func TestReportILAuthorDocumentWorkspaceRequiresRereadAfterEdit(t *testing.T) {
	service, binding, _ := reportILSourceToolFixture(t, "Alpha fact supports a bounded report.", 64, 64)
	binding.Stage = "il_narrative"
	binding.MaxCallBytes = 0
	binding.MaxReadBytes = 0
	bindSingleSourceReportILEditorialMemory(t, service, &binding)
	server := newReportILSourceToolServer(service, binding)
	readBoundReportILEditorialMemory(t, server)
	start := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentStart, Arguments: mustArgs(t, map[string]any{
		"title": "Bounded report", "language": "en",
	})})
	if start.Error != nil {
		t.Fatalf("start = %#v", start)
	}
	workspaceID := start.Content.(reportILDocumentStateOutput).WorkspaceID
	appendBlock := func(sectionTitle, prose string) {
		t.Helper()
		result := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentAppend, Arguments: mustArgs(t, map[string]any{
			"workspace_id": workspaceID, "section_title": sectionTitle, "kind": "prose",
			"prose": prose, "items": []string{}, "code": "", "language": nil, "table": nil,
			"editorial_account_keys": []string{"account_001"},
		})})
		if result.Error != nil {
			t.Fatalf("append = %#v", result)
		}
	}
	appendBlock("Answer", "Alpha fact gives the direct answer.")
	appendBlock("Judgment", "The evidence supports one bounded judgment.")

	beforeRead := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentFinalize, Arguments: mustArgs(t, map[string]any{"workspace_id": workspaceID})})
	if beforeRead.Error == nil || !strings.Contains(beforeRead.Error.Message, "reread completely") {
		t.Fatalf("finalize before read = %#v", beforeRead)
	}
	read := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentRead, Arguments: mustArgs(t, map[string]any{
		"workspace_id": workspaceID, "offset": 0, "max_bytes": reportil.ReportILDocumentMaxReadBytes,
	})})
	if read.Error != nil || read.Content.(reportILDocumentReadOutput).Truncated {
		t.Fatalf("full read = %#v", read)
	}
	replaced := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentReplace, Arguments: mustArgs(t, map[string]any{
		"workspace_id": workspaceID, "old_text": "direct answer", "new_text": "direct historical answer",
	})})
	if replaced.Error != nil || replaced.Content.(reportILDocumentStateOutput).Replacements != 1 {
		t.Fatalf("replace = %#v", replaced)
	}
	staleRead := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentFinalize, Arguments: mustArgs(t, map[string]any{"workspace_id": workspaceID})})
	if staleRead.Error == nil || !strings.Contains(staleRead.Error.Message, "reread completely") {
		t.Fatalf("finalize after edit without reread = %#v", staleRead)
	}
	if reread := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentRead, Arguments: mustArgs(t, map[string]any{
		"workspace_id": workspaceID, "offset": 0, "max_bytes": reportil.ReportILDocumentMaxReadBytes,
	})}); reread.Error != nil {
		t.Fatalf("reread = %#v", reread)
	}
	finalized := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentFinalize, Arguments: mustArgs(t, map[string]any{"workspace_id": workspaceID})})
	state := finalized.Content.(reportILDocumentStateOutput)
	if finalized.Error != nil || !state.Finalized || state.ArtifactID == "" || len(state.SHA256) != 64 || state.Replacements != 1 {
		t.Fatalf("finalize = %#v", finalized)
	}
	finalizeAgain := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentFinalize, Arguments: mustArgs(t, map[string]any{"workspace_id": workspaceID})})
	if finalizeAgain.Error == nil || finalizeAgain.Error.ErrorKind != "conflict" {
		t.Fatalf("duplicate finalize = %#v", finalizeAgain)
	}
	for _, event := range service.events {
		trace := string(event.Payload)
		for _, forbidden := range []string{"Alpha fact", "direct historical answer", "bounded judgment", `"old_text":"direct answer"`, `"new_text":"direct historical answer"`, `"content":"`} {
			if strings.Contains(trace, forbidden) {
				t.Fatalf("document trace leaked %q: %s", forbidden, trace)
			}
		}
	}
	trace := string(service.events[len(service.events)-2].Payload)
	if !strings.Contains(trace, `"report_il_stage":"il_narrative"`) || !strings.Contains(trace, `"replacements":1`) {
		t.Fatalf("document trace lacks stage metrics: %s", trace)
	}
}

func TestReportILDocumentReplaceSchemaAllowsEmptyDeletionReplacement(t *testing.T) {
	var schema map[string]any
	if err := json.Unmarshal(schemaReportILDocumentReplace, &schema); err != nil {
		t.Fatal(err)
	}
	properties := schema["properties"].(map[string]any)
	newText := properties["new_text"].(map[string]any)
	if _, found := newText["minLength"]; found || newText["maxLength"] != float64(8192) {
		t.Fatalf("new_text schema = %#v", newText)
	}
}

func TestReportILAuthorDocumentWorkspaceAllowsBoundedTextDeletion(t *testing.T) {
	service, binding, _ := reportILSourceToolFixture(t, "Alpha fact supports a bounded report.", 64, 64)
	binding.Stage = "il_narrative"
	binding.MaxCallBytes = 0
	binding.MaxReadBytes = 0
	bindSingleSourceReportILEditorialMemory(t, service, &binding)
	server := newReportILSourceToolServer(service, binding)
	readBoundReportILEditorialMemory(t, server)
	start := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentStart, Arguments: mustArgs(t, map[string]any{
		"title": "Bounded report", "language": "en",
	})})
	workspaceID := start.Content.(reportILDocumentStateOutput).WorkspaceID
	appended := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentAppend, Arguments: mustArgs(t, map[string]any{
		"workspace_id": workspaceID, "section_title": "Answer", "kind": "prose",
		"prose": "Alpha fact gives the direct answer. Repeated recap.", "items": []string{}, "code": "", "language": nil, "table": nil,
		"editorial_account_keys": []string{"account_001"},
	})})
	if appended.Error != nil {
		t.Fatalf("append = %#v", appended)
	}
	deleted := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentReplace, Arguments: mustArgs(t, map[string]any{
		"workspace_id": workspaceID, "old_text": " Repeated recap.", "new_text": "",
	})})
	if deleted.Error != nil || deleted.Content.(reportILDocumentStateOutput).Replacements != 1 {
		t.Fatalf("delete = %#v", deleted)
	}
	workspace := server.reportILState.Documents.Workspaces[workspaceID]
	if got := workspace.Document.Sections[0].Blocks[0].Prose; got != "Alpha fact gives the direct answer." {
		t.Fatalf("deleted prose = %q", got)
	}
}

func TestReportILPublicationEditTextTargetsOneProseBlockAndLeavesEquationImmutable(t *testing.T) {
	service, binding, _ := reportILSourceToolFixture(t, "Alpha fact supports a bounded report.", 64, 64)
	bindSingleSourceReportILEditorialMemory(t, service, &binding)
	document := reportilcontract.AuthorDocument{
		SchemaVersion: reportilcontract.LongFormAuthorDocumentSchemaVersion,
		Title:         "Bounded report", Language: "en",
		Parts: []reportilcontract.AuthorPart{
			{
				PartKey: "part_001", Title: "Part 1",
				Sections: []reportilcontract.AuthorSection{{
					SectionKey: "part_001.section_001", Title: "Answer",
					Blocks: []reportilcontract.AuthorBlock{
						{BlockKey: "part_001.section_001.block_001", Kind: "prose", Prose: "The roadmap sentence appears here. Repeated token stays here.", EditorialAccountKeys: []string{"account_001"}, EvidenceSourceKeys: []string{"source_001"}},
						{BlockKey: "part_001.section_001.block_002", Kind: "equation", Equation: &reportilcontract.AuthorEquation{Expression: `C_{task} = O`, Notation: "latex"}, EditorialAccountKeys: []string{"account_001"}, EvidenceSourceKeys: []string{"source_001"}},
					},
				}},
			},
		},
	}
	content := append(mustArgs(t, document), '\n')
	artifact, err := service.CreateRawArtifact(context.Background(), artifactcontract.CreateRequest{
		ArtifactID: "art_reader_structured", MissionID: binding.Catalog.MissionID,
		MediaType: reportilcontract.AuthorDocumentMediaType, Filename: "report-il-author-document.json",
		Producer: ledger.Producer{Type: "test", ID: "fixture"}, Content: content,
	})
	if err != nil {
		t.Fatal(err)
	}
	binding.Stage = "il_reader"
	binding.MaxCallBytes = 0
	binding.MaxReadBytes = 0
	binding.BaseAuthorArtifactID = artifact.ArtifactID
	binding.BaseAuthorSHA256 = artifact.SHA256
	server := newReportILSourceToolServer(service, binding)
	readBoundReportILEditorialMemory(t, server)
	opened := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentOpen, Arguments: json.RawMessage(`{}`)})
	if opened.Error != nil {
		t.Fatalf("open reader document: %#v", opened)
	}
	workspaceID := opened.Content.(reportILDocumentStateOutput).WorkspaceID
	if read := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentRead, Arguments: mustArgs(t, map[string]any{
		"workspace_id": workspaceID, "offset": 0, "max_bytes": reportil.ReportILDocumentMaxReadBytes,
	})}); read.Error != nil {
		t.Fatalf("read reader document: %#v", read)
	}
	edited := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentEditText, Arguments: mustArgs(t, map[string]any{
		"workspace_id": workspaceID, "target_kind": "block_text",
		"target_key": "part_001.section_001.block_001",
		"old_text":   "The roadmap sentence appears here. ", "new_text": "",
	})})
	if edited.Error != nil {
		t.Fatalf("block-scoped publication edit: %#v", edited)
	}
	workspace := server.reportILState.Documents.Workspaces[workspaceID]
	if got := workspace.Document.Parts[0].Sections[0].Blocks[0].Prose; got != "Repeated token stays here." {
		t.Fatalf("edited prose = %q", got)
	}
	if got := workspace.Document.Parts[0].Sections[0].Blocks[1].Equation.Expression; got != `C_{task} = O` {
		t.Fatalf("publication edit changed equation = %q", got)
	}
	for _, target := range []struct {
		kind string
		key  string
	}{
		{kind: "block_text", key: "part_001.section_001.block_002"},
		{kind: "block_text", key: "part_001.section_001.block_001"},
	} {
		oldText := `C_{task} = O`
		if target.key != "part_001.section_001.block_002" {
			oldText = "Repeated token stays here."
		}
		blocked := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentEditText, Arguments: mustArgs(t, map[string]any{
			"workspace_id": workspaceID, "target_kind": target.kind, "target_key": target.key,
			"old_text": oldText, "new_text": "Changed",
		})})
		if target.key == "part_001.section_001.block_001" {
			if blocked.Error != nil {
				t.Fatalf("second prose edit failed: %#v", blocked)
			}
			continue
		}
		if blocked.Error == nil || !strings.Contains(blocked.Error.Message, "editable reader-facing value") {
			t.Fatalf("structured block was editable: %#v", blocked)
		}
	}
}

func TestReportILPublicationEditTextScopesRepeatedTextToNamedBlock(t *testing.T) {
	document := reportilcontract.AuthorDocument{
		Title: "Repeated report", Language: "en",
		Sections: []reportilcontract.AuthorSection{
			{SectionKey: "section_001", Title: "Answer", Blocks: []reportilcontract.AuthorBlock{{BlockKey: "section_001.block_001", Kind: "prose", Prose: "Repeated transition."}}},
			{SectionKey: "section_002", Title: "Judgment", Blocks: []reportilcontract.AuthorBlock{{BlockKey: "section_002.block_001", Kind: "prose", Prose: "Repeated transition."}}},
		},
	}
	if !reportil.EditUniqueReportILDocumentReaderText(&document, "block_text", "section_002.block_001", "Repeated transition.", "Sharper judgment.") {
		t.Fatal("block-scoped edit rejected a unique target-local occurrence")
	}
	if document.Sections[0].Blocks[0].Prose != "Repeated transition." || document.Sections[1].Blocks[0].Prose != "Sharper judgment." {
		t.Fatalf("block-scoped edit changed wrong value: %#v", document.Sections)
	}
}

func TestReportILAuthorDocumentWorkspaceRejectsNoOpReplacement(t *testing.T) {
	service, binding, _ := reportILSourceToolFixture(t, "Alpha fact supports a bounded report.", 64, 64)
	binding.Stage = "il_narrative"
	binding.MaxCallBytes = 0
	binding.MaxReadBytes = 0
	bindSingleSourceReportILEditorialMemory(t, service, &binding)
	server := newReportILSourceToolServer(service, binding)
	readBoundReportILEditorialMemory(t, server)
	start := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentStart, Arguments: mustArgs(t, map[string]any{
		"title": "Bounded report", "language": "en",
	})})
	workspaceID := start.Content.(reportILDocumentStateOutput).WorkspaceID
	appended := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentAppend, Arguments: mustArgs(t, map[string]any{
		"workspace_id": workspaceID, "section_title": "Answer", "kind": "prose",
		"prose": "Alpha fact gives the direct answer.", "items": []string{}, "code": "", "language": nil, "table": nil,
		"editorial_account_keys": []string{"account_001"},
	})})
	if appended.Error != nil {
		t.Fatalf("append = %#v", appended)
	}
	replaced := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentReplace, Arguments: mustArgs(t, map[string]any{
		"workspace_id": workspaceID, "old_text": "direct answer", "new_text": "direct answer",
	})})
	if replaced.Error == nil || replaced.Error.ErrorKind != "validation" {
		t.Fatalf("no-op replace = %#v", replaced)
	}
}

func TestReportILAuthorDocumentWorkspaceRejectsMutationWhileFinalizing(t *testing.T) {
	service, binding, _ := reportILSourceToolFixture(t, "Alpha fact supports a bounded report.", 64, 64)
	binding.Stage = "il_narrative"
	binding.MaxCallBytes = 0
	binding.MaxReadBytes = 0
	bindSingleSourceReportILEditorialMemory(t, service, &binding)
	blockingService := &blockingArtifactMCPService{
		fakeMCPService: service,
		createStarted:  make(chan struct{}),
		releaseCreate:  make(chan struct{}),
	}
	server := newReportILSourceToolServer(blockingService.fakeMCPService, binding)
	server.service = blockingService
	readBoundReportILEditorialMemory(t, server)
	start := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentStart, Arguments: mustArgs(t, map[string]any{
		"title": "Bounded report", "language": "en",
	})})
	if start.Error != nil {
		t.Fatalf("start = %#v", start)
	}
	workspaceID := start.Content.(reportILDocumentStateOutput).WorkspaceID
	for _, block := range []struct {
		section string
		prose   string
	}{
		{section: "Answer", prose: "Alpha fact gives the direct answer."},
		{section: "Judgment", prose: "The evidence supports one bounded judgment."},
	} {
		result := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentAppend, Arguments: mustArgs(t, map[string]any{
			"workspace_id": workspaceID, "section_title": block.section, "kind": "prose",
			"prose": block.prose, "items": []string{}, "code": "", "language": nil, "table": nil,
			"editorial_account_keys": []string{"account_001"},
		})})
		if result.Error != nil {
			t.Fatalf("append = %#v", result)
		}
	}
	if read := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentRead, Arguments: mustArgs(t, map[string]any{
		"workspace_id": workspaceID, "offset": 0, "max_bytes": reportil.ReportILDocumentMaxReadBytes,
	})}); read.Error != nil {
		t.Fatalf("document read = %#v", read)
	}

	finalized := make(chan ToolResult, 1)
	go func() {
		finalized <- server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentFinalize, Arguments: mustArgs(t, map[string]any{"workspace_id": workspaceID})})
	}()
	<-blockingService.createStarted

	appendDuringFinalize := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentAppend, Arguments: mustArgs(t, map[string]any{
		"workspace_id": workspaceID, "section_title": "Judgment", "kind": "prose",
		"prose": "A late mutation must be rejected.", "items": []string{}, "code": "", "language": nil, "table": nil,
		"editorial_account_keys": []string{"account_001"},
	})})
	if appendDuringFinalize.Error == nil || appendDuringFinalize.Error.ErrorKind != "conflict" || !strings.Contains(appendDuringFinalize.Error.Message, "finalizing or finalized") {
		t.Fatalf("append during finalize = %#v", appendDuringFinalize)
	}
	replaceDuringFinalize := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentReplace, Arguments: mustArgs(t, map[string]any{
		"workspace_id": workspaceID, "old_text": "direct answer", "new_text": "changed answer",
	})})
	if replaceDuringFinalize.Error == nil || replaceDuringFinalize.Error.ErrorKind != "conflict" || !strings.Contains(replaceDuringFinalize.Error.Message, "finalizing or finalized") {
		t.Fatalf("replace during finalize = %#v", replaceDuringFinalize)
	}
	select {
	case result := <-finalized:
		t.Fatalf("finalize returned before persistence release: %#v", result)
	default:
	}
	close(blockingService.releaseCreate)
	result := <-finalized
	state := result.Content.(reportILDocumentStateOutput)
	if result.Error != nil || !state.Finalized || state.Revision != 3 || state.Replacements != 0 {
		t.Fatalf("finalize = %#v", result)
	}
	artifacts, err := blockingService.ListRawArtifacts(context.Background(), binding.Catalog.MissionID)
	if err != nil || len(artifacts) != 3 {
		t.Fatalf("final artifacts = %d / %v", len(artifacts), err)
	}
}

func TestReportILAuthorDocumentWorkspaceResetsFinalizingAfterPersistenceFailure(t *testing.T) {
	service, binding, _ := reportILSourceToolFixture(t, "Alpha fact supports a bounded report.", 64, 64)
	binding.Stage = "il_narrative"
	binding.MaxCallBytes = 0
	binding.MaxReadBytes = 0
	bindSingleSourceReportILEditorialMemory(t, service, &binding)
	blockingService := &blockingArtifactMCPService{
		fakeMCPService: service,
		createStarted:  make(chan struct{}),
		releaseCreate:  make(chan struct{}),
		createErr:      errors.New("persistence failed"),
	}
	server := newReportILSourceToolServer(blockingService.fakeMCPService, binding)
	server.service = blockingService
	readBoundReportILEditorialMemory(t, server)
	start := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentStart, Arguments: mustArgs(t, map[string]any{
		"title": "Bounded report", "language": "en",
	})})
	workspaceID := start.Content.(reportILDocumentStateOutput).WorkspaceID
	for _, block := range []struct {
		section string
		prose   string
	}{
		{section: "Answer", prose: "Alpha fact gives the direct answer."},
		{section: "Judgment", prose: "The evidence supports one bounded judgment."},
	} {
		result := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentAppend, Arguments: mustArgs(t, map[string]any{
			"workspace_id": workspaceID, "section_title": block.section, "kind": "prose",
			"prose": block.prose, "items": []string{}, "code": "", "language": nil, "table": nil,
			"editorial_account_keys": []string{"account_001"},
		})})
		if result.Error != nil {
			t.Fatalf("append = %#v", result)
		}
	}
	if read := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentRead, Arguments: mustArgs(t, map[string]any{
		"workspace_id": workspaceID, "offset": 0, "max_bytes": reportil.ReportILDocumentMaxReadBytes,
	})}); read.Error != nil {
		t.Fatalf("document read = %#v", read)
	}

	finalized := make(chan ToolResult, 1)
	go func() {
		finalized <- server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentFinalize, Arguments: mustArgs(t, map[string]any{"workspace_id": workspaceID})})
	}()
	<-blockingService.createStarted
	close(blockingService.releaseCreate)
	failed := <-finalized
	if failed.Error == nil {
		t.Fatalf("persistence failure was accepted: %#v", failed)
	}
	appended := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentAppend, Arguments: mustArgs(t, map[string]any{
		"workspace_id": workspaceID, "section_title": "Judgment", "kind": "prose",
		"prose": "Authoring remains available after persistence failure.", "items": []string{}, "code": "", "language": nil, "table": nil,
		"editorial_account_keys": []string{"account_001"},
	})})
	if appended.Error != nil {
		t.Fatalf("append after persistence failure = %#v", appended)
	}
}

func bindReportILEditorialMemory(
	t *testing.T,
	service *fakeMCPService,
	binding *reportilcontract.SourceAccessBinding,
	accounts []reportilcontract.EditorialAccount,
) reportilcontract.EditorialMemory {
	t.Helper()
	memory := reportilcontract.EditorialMemory{
		SchemaVersion: reportilcontract.EditorialMemorySchemaVersion,
		Language:      "en",
		Accounts:      accounts,
	}
	if err := reportilcontract.ValidateEditorialMemory(memory, binding.Catalog); err != nil {
		t.Fatal(err)
	}
	memoryArtifact := reportILEditorialMemoryArtifactFixture(memory)
	content := append(mustArgs(t, memoryArtifact), '\n')
	artifact, err := service.CreateRawArtifact(context.Background(), artifactcontract.CreateRequest{
		ArtifactID: "art_editorial_memory", MissionID: binding.Catalog.MissionID,
		MediaType: reportilcontract.EditorialMemoryMediaType,
		Filename:  "report-il-editorial-memory.json",
		Producer:  ledger.Producer{Type: "test", ID: "fixture"}, Content: content,
	})
	if err != nil {
		t.Fatal(err)
	}
	binding.EditorialMemoryArtifactID = artifact.ArtifactID
	binding.EditorialMemorySHA256 = artifact.SHA256
	return memory
}

func reportILEditorialMemoryArtifactFixture(memory reportilcontract.EditorialMemory) reportilcontract.EditorialMemoryArtifact {
	artifact := reportilcontract.EditorialMemoryArtifact{
		SchemaVersion: memory.SchemaVersion,
		Language:      memory.Language,
		Accounts:      append([]reportilcontract.EditorialAccount(nil), memory.Accounts...),
	}
	for _, account := range memory.Accounts {
		for _, sourceKey := range account.SourceKeys {
			artifact.Anchors = append(artifact.Anchors, reportilcontract.EditorialAnchor{
				AccountKey: account.AccountKey,
				SourceKey:  sourceKey,
				Excerpt:    "x",
				Offset:     0,
				ByteSize:   1,
				SHA256:     sha256Hex([]byte("x")),
			})
		}
	}
	return artifact
}

func readBoundReportILEditorialMemory(t *testing.T, server *Server) {
	t.Helper()
	result := server.Call(context.Background(), ToolCall{
		Name: ToolReportILEditorialMemoryRead,
		Arguments: mustArgs(t, map[string]any{
			"offset": 0, "max_bytes": reportil.ReportILDocumentMaxReadBytes,
		}),
	})
	if result.Error != nil || result.Content.(reportILEditorialMemoryReadOutput).Truncated {
		t.Fatalf("editorial memory read = %#v", result)
	}
}

func bindSingleSourceReportILEditorialMemory(
	t *testing.T,
	service *fakeMCPService,
	binding *reportilcontract.SourceAccessBinding,
) reportilcontract.EditorialMemory {
	t.Helper()
	return bindReportILEditorialMemory(t, service, binding, []reportilcontract.EditorialAccount{{
		AccountKey: "account_001", Importance: "essential",
		Account:    "Alpha fact and its implications support the report.",
		SourceKeys: []string{"source_001"},
	}})
}

type blockingArtifactMCPService struct {
	*fakeMCPService
	createStarted chan struct{}
	releaseCreate chan struct{}
	createErr     error
}

func (service *blockingArtifactMCPService) CreateRawArtifact(ctx context.Context, request artifactcontract.CreateRequest) (artifactcontract.Raw, error) {
	close(service.createStarted)
	select {
	case <-service.releaseCreate:
	case <-ctx.Done():
		return artifactcontract.Raw{}, ctx.Err()
	}
	if service.createErr != nil {
		return artifactcontract.Raw{}, service.createErr
	}
	return service.fakeMCPService.CreateRawArtifact(ctx, request)
}

func TestReportILContinuityWorkspaceMakesMinimalCausativeSentenceCorrection(t *testing.T) {
	const sourceText = "1433（嘉吉3）年に但馬国守護山名宗全（そうぜん）が13年かけて築かせた城といわれている。"
	service, binding, _ := reportILSourceToolFixture(t, sourceText, 256, 256)
	memory := reportilcontract.EditorialMemory{
		SchemaVersion: reportilcontract.EditorialMemorySchemaVersion,
		Language:      "ko",
		Accounts: []reportilcontract.EditorialAccount{{
			AccountKey: "account_001", Importance: "essential",
			Account:    "1433년 다지마 수호 야마나 소젠이 성을 직접 쌓은 것이 아니라 13년에 걸쳐 쌓게 했다고 전한다.",
			SourceKeys: []string{"source_001"},
		}},
	}
	memoryArtifact := reportilcontract.EditorialMemoryArtifact{
		SchemaVersion: memory.SchemaVersion,
		Language:      memory.Language,
		Accounts:      memory.Accounts,
		Anchors: []reportilcontract.EditorialAnchor{{
			AccountKey: "account_001", SourceKey: "source_001",
			Excerpt:  "山名宗全（そうぜん）が13年かけて築かせた",
			Offset:   strings.Index(sourceText, "山名宗全（そうぜん）が13年かけて築かせた"),
			ByteSize: len([]byte("山名宗全（そうぜん）が13年かけて築かせた")),
			SHA256:   sha256Hex([]byte("山名宗全（そうぜん）が13年かけて築かせた")),
		}},
	}
	memoryContent := append(mustArgs(t, memoryArtifact), '\n')
	storedMemory, err := service.CreateRawArtifact(context.Background(), artifactcontract.CreateRequest{
		ArtifactID: "art_editorial_memory", MissionID: binding.Catalog.MissionID,
		MediaType: reportilcontract.EditorialMemoryMediaType, Filename: "report-il-editorial-memory.json",
		Producer: ledger.Producer{Type: "test", ID: "fixture"}, Content: memoryContent,
	})
	if err != nil {
		t.Fatal(err)
	}
	originalSentence := "다케다성은 다지마 수호 야마나 소젠이 1433년부터 13년에 걸쳐 쌓았다고 전한다."
	correctedSentence := "다케다성은 다지마 수호 야마나 소젠이 1433년부터 13년에 걸쳐 쌓게 했다고 전한다."
	document := reportilcontract.AuthorDocument{
		SchemaVersion: reportilcontract.AuthorDocumentSchemaVersion,
		Title:         "다케다성", Language: "ko",
		Sections: []reportilcontract.AuthorSection{
			{SectionKey: "section_001", Title: "축성 전승", Blocks: []reportilcontract.AuthorBlock{{
				BlockKey: "section_001.block_001", Kind: "prose", Prose: originalSentence,
				EditorialAccountKeys: []string{"account_001"}, EvidenceSourceKeys: []string{"source_001"},
			}}},
			{SectionKey: "section_002", Title: "역사적 판단", Blocks: []reportilcontract.AuthorBlock{{
				BlockKey: "section_002.block_001", Kind: "prose", Prose: "이 전승은 축성 주체와 명령자의 역할을 구분해 읽어야 한다.",
				EditorialAccountKeys: []string{"account_001"}, EvidenceSourceKeys: []string{"source_001"},
			}}},
		},
	}
	documentContent := append(mustArgs(t, document), '\n')
	storedDocument, err := service.CreateRawArtifact(context.Background(), artifactcontract.CreateRequest{
		ArtifactID: "art_publication_document", MissionID: binding.Catalog.MissionID,
		MediaType: reportilcontract.AuthorDocumentMediaType, Filename: "report-il-author-document.json",
		Producer: ledger.Producer{Type: "test", ID: "fixture"}, Content: documentContent,
	})
	if err != nil {
		t.Fatal(err)
	}
	binding.Stage = "il_continuity"
	binding.MaxCallBytes = 0
	binding.MaxReadBytes = 0
	binding.EditorialMemoryArtifactID = storedMemory.ArtifactID
	binding.EditorialMemorySHA256 = storedMemory.SHA256
	binding.BaseAuthorArtifactID = storedDocument.ArtifactID
	binding.BaseAuthorSHA256 = storedDocument.SHA256
	server := newReportILSourceToolServer(service, binding)
	readBoundReportILEditorialMemory(t, server)
	opened := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentOpen, Arguments: json.RawMessage(`{}`)})
	if opened.Error != nil {
		t.Fatalf("open = %#v", opened)
	}
	workspaceID := opened.Content.(reportILDocumentStateOutput).WorkspaceID
	if read := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentRead, Arguments: mustArgs(t, map[string]any{
		"workspace_id": workspaceID, "offset": 0, "max_bytes": reportil.ReportILDocumentMaxReadBytes,
	})}); read.Error != nil {
		t.Fatalf("document read = %#v", read)
	}
	revised := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentReviseBlock, Arguments: mustArgs(t, map[string]any{
		"workspace_id": workspaceID, "block_key": "section_001.block_001",
		"old_text": "쌓았다고", "new_text": "쌓게 했다고",
		"editorial_account_keys": []string{"account_001"},
	})})
	if revised.Error != nil || revised.Content.(reportILDocumentStateOutput).Replacements != 1 {
		t.Fatalf("minimal causative correction = %#v", revised)
	}
	if reread := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentRead, Arguments: mustArgs(t, map[string]any{
		"workspace_id": workspaceID, "offset": 0, "max_bytes": reportil.ReportILDocumentMaxReadBytes,
	})}); reread.Error != nil {
		t.Fatalf("document reread = %#v", reread)
	}
	finalized := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentFinalize, Arguments: mustArgs(t, map[string]any{"workspace_id": workspaceID})})
	if finalized.Error != nil {
		t.Fatalf("finalize = %#v", finalized)
	}
	stored, err := service.GetRawArtifact(context.Background(), finalized.Content.(reportILDocumentStateOutput).ArtifactID)
	if err != nil {
		t.Fatal(err)
	}
	var finalDocument reportilcontract.AuthorDocument
	if err := json.Unmarshal(stored.Content, &finalDocument); err != nil {
		t.Fatal(err)
	}
	if got := finalDocument.Sections[0].Blocks[0].Prose; got != correctedSentence {
		t.Fatalf("corrected sentence = %q, want %q", got, correctedSentence)
	}
	if finalDocument.Sections[1].Blocks[0].Prose != document.Sections[1].Blocks[0].Prose {
		t.Fatal("minimal correction rewrote an unrelated paragraph")
	}
}

func TestReportILPublicationWorkspaceRevisesBlockTextAndSourceBindings(t *testing.T) {
	service, binding, _ := reportILSourceToolFixture(t, "Asago dates the work to 1431-1443. Hyogo says Sōzen had it built from 1433 over thirteen years.", 256, 256)
	secondContent := []byte("Hyogo says Yamana Sōzen had Takeda Castle built from 1433 over thirteen years.")
	secondHash := sha256Hex(secondContent)
	binding.Catalog.Sources = append(binding.Catalog.Sources, reportilcontract.SourceCatalogEntry{
		SourceKey: "source_002", SnapshotID: "src_hyogo",
		SnapshotReceipt: reportilcontract.SourceSnapshotReceipt("src_hyogo", secondHash),
		ContentHash:     secondHash, RetrievalPolicy: sourcecontract.RetrievalPolicySnapshotOnly,
		Artifacts: []reportilcontract.SourceCatalogArtifact{{
			ArtifactID: "art_hyogo", SHA256: secondHash, ByteSize: int64(len(secondContent)), MediaType: "text/plain",
		}},
		ReadableSHA256: secondHash, ReadableBytes: len(secondContent), Extraction: "stored_text",
	})
	var err error
	binding.Catalog, err = reportilcontract.SealSourceCatalog(binding.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	service.sources = append(service.sources, sourcecontract.Snapshot{
		SnapshotID: "src_hyogo", MissionID: binding.Catalog.MissionID, ArtifactIDs: []string{"art_hyogo"},
		ContentHash: sourcecontract.ContentHash{Algorithm: "sha256", Value: secondHash},
		Access:      sourcecontract.Access{RetrievalPolicy: sourcecontract.RetrievalPolicySnapshotOnly},
		State:       sourcecontract.State{State: sourcecontract.StateActive},
	})
	service.artifacts["art_hyogo"] = artifactcontract.Raw{
		ArtifactID: "art_hyogo", MissionID: binding.Catalog.MissionID, MediaType: "text/plain",
		ByteSize: int64(len(secondContent)), SHA256: secondHash, Content: secondContent,
	}
	bindReportILEditorialMemory(t, service, &binding, []reportilcontract.EditorialAccount{
		{AccountKey: "account_001", Importance: "essential", Account: "Asago dates construction from 1431 to 1443 under Yamana Mochitoyo.", SourceKeys: []string{"source_001"}},
		{AccountKey: "account_002", Importance: "essential", Account: "Hyogo says the Tajima governor Yamana Sōzen commissioned the castle in 1433 and had it built over thirteen years.", SourceKeys: []string{"source_002"}},
	})
	document := reportilcontract.AuthorDocument{
		SchemaVersion: reportilcontract.AuthorDocumentSchemaVersion,
		Title:         "Takeda chronology", Language: "en",
		Sections: []reportilcontract.AuthorSection{
			{SectionKey: "section_001", Title: "Chronology", Blocks: []reportilcontract.AuthorBlock{{
				BlockKey: "section_001.block_001", Kind: "prose",
				Prose: "Asago dates construction from 1431 to 1443 under Yamana Mochitoyo.", EditorialAccountKeys: []string{"account_001"},
				EvidenceSourceKeys: []string{"source_001"},
			}}},
			{SectionKey: "section_002", Title: "Judgment", Blocks: []reportilcontract.AuthorBlock{{
				BlockKey: "section_002.block_001", Kind: "prose",
				Prose: "The disagreement should remain visible.", EditorialAccountKeys: []string{"account_001"}, EvidenceSourceKeys: []string{"source_001"},
			}}},
		},
	}
	content := append(mustArgs(t, document), '\n')
	artifact, err := service.CreateRawArtifact(context.Background(), artifactcontract.CreateRequest{
		ArtifactID: "art_author_chronology", MissionID: binding.Catalog.MissionID,
		MediaType: reportilcontract.AuthorDocumentMediaType, Filename: "report-il-author-document.json",
		Producer: ledger.Producer{Type: "test", ID: "fixture"}, Content: content,
	})
	if err != nil {
		t.Fatal(err)
	}
	binding.Stage = "il_continuity"
	binding.MaxCallBytes = 0
	binding.MaxReadBytes = 0
	binding.BaseAuthorArtifactID = artifact.ArtifactID
	binding.BaseAuthorSHA256 = artifact.SHA256
	server := newReportILSourceToolServer(service, binding)
	readBoundReportILEditorialMemory(t, server)
	opened := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentOpen, Arguments: json.RawMessage(`{}`)})
	if opened.Error != nil {
		t.Fatalf("open = %#v", opened)
	}
	workspaceID := opened.Content.(reportILDocumentStateOutput).WorkspaceID
	if read := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentRead, Arguments: mustArgs(t, map[string]any{
		"workspace_id": workspaceID, "offset": 0, "max_bytes": reportil.ReportILDocumentMaxReadBytes,
	})}); read.Error != nil {
		t.Fatalf("document read = %#v", read)
	}
	oldText := "Asago dates construction from 1431 to 1443 under Yamana Mochitoyo."
	newText := oldText + " Hyogo's museum gives a distinct account: in 1433 the Tajima governor Yamana Sōzen commissioned the castle and had it built over thirteen years."
	revised := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentReviseBlock, Arguments: mustArgs(t, map[string]any{
		"workspace_id": workspaceID, "block_key": "section_001.block_001",
		"old_text": oldText, "new_text": newText,
		"editorial_account_keys": []string{"account_001", "account_002"},
	})})
	if revised.Error != nil || revised.Content.(reportILDocumentStateOutput).Replacements != 1 {
		t.Fatalf("revise block = %#v", revised)
	}
	stale := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentFinalize, Arguments: mustArgs(t, map[string]any{"workspace_id": workspaceID})})
	if stale.Error == nil || !strings.Contains(stale.Error.Message, "reread completely") {
		t.Fatalf("finalize without reread = %#v", stale)
	}
	if reread := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentRead, Arguments: mustArgs(t, map[string]any{
		"workspace_id": workspaceID, "offset": 0, "max_bytes": reportil.ReportILDocumentMaxReadBytes,
	})}); reread.Error != nil {
		t.Fatalf("reread = %#v", reread)
	}
	finalized := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentFinalize, Arguments: mustArgs(t, map[string]any{"workspace_id": workspaceID})})
	if finalized.Error != nil || finalized.Content.(reportILDocumentStateOutput).ArtifactID == artifact.ArtifactID {
		t.Fatalf("finalize = %#v", finalized)
	}
	stored, err := service.GetRawArtifact(context.Background(), finalized.Content.(reportILDocumentStateOutput).ArtifactID)
	if err != nil {
		t.Fatal(err)
	}
	var finalDocument reportilcontract.AuthorDocument
	if err := json.Unmarshal(stored.Content, &finalDocument); err != nil {
		t.Fatal(err)
	}
	block := finalDocument.Sections[0].Blocks[0]
	if block.Prose != newText || !reflect.DeepEqual(block.EvidenceSourceKeys, []string{"source_001", "source_002"}) {
		t.Fatalf("final revised block = %#v", block)
	}
	for _, event := range service.events {
		trace := string(event.Payload)
		if !strings.Contains(trace, ToolReportILDocumentReviseBlock) {
			continue
		}
		for _, forbidden := range []string{oldText, newText, "source_001", "source_002"} {
			if strings.Contains(trace, forbidden) {
				t.Fatalf("block revision trace leaked %q: %s", forbidden, trace)
			}
		}
	}
}

func TestReportILPublicationWorkspaceRejectsTamperedBoundArtifactBytes(t *testing.T) {
	service, binding, _ := reportILSourceToolFixture(t, "Alpha fact supports a bounded report.", 64, 64)
	bindSingleSourceReportILEditorialMemory(t, service, &binding)
	document := reportilcontract.AuthorDocument{
		SchemaVersion: reportilcontract.AuthorDocumentSchemaVersion,
		Title:         "Bounded report", Language: "en",
		Sections: []reportilcontract.AuthorSection{
			{SectionKey: "section_001", Title: "Answer", Blocks: []reportilcontract.AuthorBlock{{BlockKey: "section_001.block_001", Kind: "prose", Prose: "Alpha fact gives the direct answer.", EditorialAccountKeys: []string{"account_001"}, EvidenceSourceKeys: []string{"source_001"}}}},
			{SectionKey: "section_002", Title: "Judgment", Blocks: []reportilcontract.AuthorBlock{{BlockKey: "section_002.block_001", Kind: "prose", Prose: "The evidence supports one bounded judgment.", EditorialAccountKeys: []string{"account_001"}, EvidenceSourceKeys: []string{"source_001"}}}},
		},
	}
	content := append(mustArgs(t, document), '\n')
	artifact, err := service.CreateRawArtifact(context.Background(), artifactcontract.CreateRequest{
		ArtifactID: "art_author_tampered", MissionID: binding.Catalog.MissionID,
		MediaType: reportilcontract.AuthorDocumentMediaType, Filename: "report-il-author-document.json",
		Producer: ledger.Producer{Type: "test", ID: "fixture"}, Content: content,
	})
	if err != nil {
		t.Fatal(err)
	}
	binding.Stage = "il_reader"
	binding.MaxCallBytes = 0
	binding.MaxReadBytes = 0
	binding.BaseAuthorArtifactID = artifact.ArtifactID
	binding.BaseAuthorSHA256 = artifact.SHA256
	tampered := service.artifacts[artifact.ArtifactID]
	tampered.Content = append([]byte(nil), tampered.Content...)
	tampered.Content[len(tampered.Content)-2] ^= 1
	service.artifacts[artifact.ArtifactID] = tampered
	server := newReportILSourceToolServer(service, binding)
	readBoundReportILEditorialMemory(t, server)
	opened := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentOpen, Arguments: json.RawMessage(`{}`)})
	if opened.Error == nil || !strings.Contains(opened.Error.Message, "base document binding is invalid") {
		t.Fatalf("tampered base document was accepted = %#v", opened)
	}
}

func TestReportILContinuityWorkspaceCanFinalizeUnchangedByReusingBaseArtifact(t *testing.T) {
	service, binding, _ := reportILSourceToolFixture(t, "Alpha fact supports a bounded report.", 64, 64)
	bindSingleSourceReportILEditorialMemory(t, service, &binding)
	document := reportilcontract.AuthorDocument{
		SchemaVersion: reportilcontract.AuthorDocumentSchemaVersion,
		Title:         "Bounded report", Language: "en",
		Sections: []reportilcontract.AuthorSection{
			{SectionKey: "section_001", Title: "Answer", Blocks: []reportilcontract.AuthorBlock{{BlockKey: "section_001.block_001", Kind: "prose", Prose: "Alpha fact gives the direct answer.", EditorialAccountKeys: []string{"account_001"}, EvidenceSourceKeys: []string{"source_001"}}}},
			{SectionKey: "section_002", Title: "Judgment", Blocks: []reportilcontract.AuthorBlock{{BlockKey: "section_002.block_001", Kind: "prose", Prose: "The evidence supports one bounded judgment.", EditorialAccountKeys: []string{"account_001"}, EvidenceSourceKeys: []string{"source_001"}}}},
		},
	}
	content := append(mustArgs(t, document), '\n')
	artifact, err := service.CreateRawArtifact(context.Background(), artifactcontract.CreateRequest{
		ArtifactID: "art_continuity_base", MissionID: binding.Catalog.MissionID,
		MediaType: reportilcontract.AuthorDocumentMediaType, Filename: "report-il-author-document.json",
		Producer: ledger.Producer{Type: "test", ID: "fixture"}, Content: content,
	})
	if err != nil {
		t.Fatal(err)
	}
	binding.Stage = "il_continuity"
	binding.MaxCallBytes = 0
	binding.MaxReadBytes = 0
	binding.BaseAuthorArtifactID = artifact.ArtifactID
	binding.BaseAuthorSHA256 = artifact.SHA256
	server := newReportILSourceToolServer(service, binding)
	readBoundReportILEditorialMemory(t, server)
	opened := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentOpen, Arguments: json.RawMessage(`{}`)})
	if opened.Error != nil {
		t.Fatalf("open = %#v", opened)
	}
	state := opened.Content.(reportILDocumentStateOutput)
	if read := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentRead, Arguments: mustArgs(t, map[string]any{
		"workspace_id": state.WorkspaceID, "offset": 0, "max_bytes": reportil.ReportILDocumentMaxReadBytes,
	})}); read.Error != nil {
		t.Fatalf("document read = %#v", read)
	}
	finalized := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentFinalize, Arguments: mustArgs(t, map[string]any{"workspace_id": state.WorkspaceID})})
	finalState := finalized.Content.(reportILDocumentStateOutput)
	if finalized.Error != nil || !finalState.Finalized || finalState.ReportILStage != "il_continuity" || finalState.Replacements != 0 || finalState.SHA256 != artifact.SHA256 || finalState.ArtifactID != artifact.ArtifactID {
		t.Fatalf("unchanged continuity finalize = %#v", finalized)
	}
	artifacts, err := service.ListRawArtifacts(context.Background(), binding.Catalog.MissionID)
	if err != nil || len(artifacts) != 3 {
		t.Fatalf("unchanged continuity artifacts = %d / %v", len(artifacts), err)
	}
}

func TestReportILPublicationWorkspaceOpensBoundArtifactAndCanFinalizeUnchanged(t *testing.T) {
	service, binding, _ := reportILSourceToolFixture(t, "Alpha fact supports a bounded report.", 64, 64)
	bindSingleSourceReportILEditorialMemory(t, service, &binding)
	document := reportilcontract.AuthorDocument{
		SchemaVersion: reportilcontract.AuthorDocumentSchemaVersion,
		Title:         "Bounded report", Language: "en",
		Sections: []reportilcontract.AuthorSection{
			{SectionKey: "section_001", Title: "Answer", Blocks: []reportilcontract.AuthorBlock{{BlockKey: "section_001.block_001", Kind: "prose", Prose: "Alpha fact gives the direct answer.", EditorialAccountKeys: []string{"account_001"}, EvidenceSourceKeys: []string{"source_001"}}}},
			{SectionKey: "section_002", Title: "Judgment", Blocks: []reportilcontract.AuthorBlock{{BlockKey: "section_002.block_001", Kind: "prose", Prose: "The evidence supports one bounded judgment.", EditorialAccountKeys: []string{"account_001"}, EvidenceSourceKeys: []string{"source_001"}}}},
		},
	}
	content := append(mustArgs(t, document), '\n')
	artifact, err := service.CreateRawArtifact(context.Background(), artifactcontract.CreateRequest{
		ArtifactID: "art_author_base", MissionID: binding.Catalog.MissionID,
		MediaType: reportilcontract.AuthorDocumentMediaType, Filename: "report-il-author-document.json",
		Producer: ledger.Producer{Type: "test", ID: "fixture"}, Content: content,
	})
	if err != nil {
		t.Fatal(err)
	}
	binding.Stage = "il_reader"
	binding.MaxCallBytes = 0
	binding.MaxReadBytes = 0
	binding.BaseAuthorArtifactID = artifact.ArtifactID
	binding.BaseAuthorSHA256 = artifact.SHA256
	server := newReportILSourceToolServer(service, binding)
	readBoundReportILEditorialMemory(t, server)
	opened := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentOpen, Arguments: json.RawMessage(`{}`)})
	if opened.Error != nil {
		t.Fatalf("open = %#v", opened)
	}
	state := opened.Content.(reportILDocumentStateOutput)
	if state.ReportILStage != "il_reader" || state.Replacements != 0 {
		t.Fatalf("open state = %#v", state)
	}
	if read := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentRead, Arguments: mustArgs(t, map[string]any{
		"workspace_id": state.WorkspaceID, "offset": 0, "max_bytes": reportil.ReportILDocumentMaxReadBytes,
	})}); read.Error != nil {
		t.Fatalf("document read = %#v", read)
	}
	finalized := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentFinalize, Arguments: mustArgs(t, map[string]any{"workspace_id": state.WorkspaceID})})
	finalState := finalized.Content.(reportILDocumentStateOutput)
	if finalized.Error != nil || !finalState.Finalized || finalState.ReportILStage != "il_reader" || finalState.Replacements != 0 || finalState.SHA256 != artifact.SHA256 || finalState.ArtifactID != artifact.ArtifactID {
		t.Fatalf("unchanged finalize = %#v", finalized)
	}
	artifacts, err := service.ListRawArtifacts(context.Background(), binding.Catalog.MissionID)
	if err != nil || len(artifacts) != 3 {
		t.Fatalf("unchanged publication artifacts = %d / %v", len(artifacts), err)
	}
}
