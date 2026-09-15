package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/reportil"
	"github.com/c86j224s/liquid2/plasma/internal/reportilcontract"
	"github.com/c86j224s/liquid2/plasma/internal/reportilsource"
	sourcecontract "github.com/c86j224s/liquid2/plasma/internal/source"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
)

func TestReportILSourceBindingOptionFailsClosedWhenBindingIsInvalid(t *testing.T) {
	service, binding, _ := reportILSourceToolFixture(t, "frozen", 8, 32)
	binding.Stage = "invalid"
	server := NewServer(
		service,
		WithBinding(Binding{MissionID: "mis_report_il", AgentSessionID: "ses_report_il", AgentExecutor: "codex"}),
		WithReportILSourceBinding(binding),
		WithEnabledTools([]string{ToolMissionGet}),
	)
	if got := toolNames(server.ListTools()); len(got) != 0 {
		t.Fatalf("invalid IL binding widened tool surface: %#v", got)
	}
	for _, tool := range []string{ToolReportILSourcesList, ToolReportILSourcesRead} {
		result := server.dispatchCall(context.Background(), ToolCall{Name: tool, Arguments: json.RawMessage(`{}`)})
		if result.Error == nil || !strings.Contains(result.Error.Message, "not enabled") {
			t.Fatalf("invalid IL binding tool %s = %#v", tool, result)
		}
	}
	if result := server.dispatchCall(context.Background(), ToolCall{Name: ToolMissionGet}); result.Error == nil || !strings.Contains(result.Error.Message, "not enabled") {
		t.Fatalf("invalid IL binding exposed ambient tool: %#v", result)
	}
}

func TestReportILReaderToolsUseEditorialMemoryWithoutRawSources(t *testing.T) {
	service, binding, _ := reportILSourceToolFixture(t, "Alpha fact. Beta fact.", 8, 32)
	binding.Stage = "il_reader"
	binding.MaxCallBytes = 0
	binding.MaxReadBytes = 0
	binding.EditorialMemoryArtifactID = "art_editorial_memory"
	binding.EditorialMemorySHA256 = strings.Repeat("d", 64)
	binding.BaseAuthorArtifactID = "art_reader_document"
	binding.BaseAuthorSHA256 = strings.Repeat("a", 64)
	server := newReportILSourceToolServer(service, binding)
	tools := server.ListTools()
	if got := toolNames(tools); !reflect.DeepEqual(got, []string{
		ToolReportILEditorialMemoryRead, ToolReportILDocumentOpen,
		ToolReportILDocumentRead, ToolReportILDocumentEditText, ToolReportILDocumentFinalize,
	}) {
		t.Fatalf("reader tool surface = %#v", got)
	}
	if slices.Contains(toolNames(tools), ToolReportILSourcesRead) || slices.Contains(toolNames(tools), ToolReportILSourcesList) {
		t.Fatal("reader can access raw sources")
	}
}

func TestReportILContinuityToolsSeparateAccountRepairFromStyleEditing(t *testing.T) {
	service, binding, _ := reportILSourceToolFixture(t, "Alpha fact. Beta fact.", 8, 32)
	binding.Stage = "il_continuity"
	binding.MaxCallBytes = 0
	binding.MaxReadBytes = 0
	binding.EditorialMemoryArtifactID = "art_editorial_memory"
	binding.EditorialMemorySHA256 = strings.Repeat("d", 64)
	binding.BaseAuthorArtifactID = "art_continuity_document"
	binding.BaseAuthorSHA256 = strings.Repeat("a", 64)
	server := newReportILSourceToolServer(service, binding)
	tools := server.ListTools()
	if got := toolNames(tools); !reflect.DeepEqual(got, []string{
		ToolReportILEditorialMemoryRead, ToolReportILDocumentOpen,
		ToolReportILDocumentRead, ToolReportILDocumentReviseBlock, ToolReportILDocumentFinalize,
	}) {
		t.Fatalf("continuity tool surface = %#v", got)
	}
	if slices.Contains(toolNames(tools), ToolReportILDocumentReplace) || slices.Contains(toolNames(tools), ToolReportILSourcesRead) {
		t.Fatal("continuity editor has an unavailable style or raw-source tool")
	}
}

func TestReportILSourceToolsExposeOnlyFrozenListAndRead(t *testing.T) {
	service, binding, sourceText := reportILSourceToolFixture(t, "Alpha fact. Beta fact.", 8, 32)
	server := newReportILSourceToolServer(service, binding)
	if got := toolNames(server.ListTools()); !reflect.DeepEqual(got, []string{ToolReportILSourcesList, ToolReportILSourcesRead}) {
		t.Fatalf("report IL tool surface = %#v", got)
	}

	listed := server.Call(context.Background(), ToolCall{Name: ToolReportILSourcesList, Arguments: json.RawMessage(`{}`)})
	if listed.Error != nil || listed.TraceError != "" {
		t.Fatalf("source list = %#v", listed)
	}
	output, ok := listed.Content.(reportILSourcesListOutput)
	if !ok || output.CatalogSHA256 != binding.Catalog.SHA256 || output.Stage != binding.Stage || output.Attempt != binding.Attempt || len(output.Sources) != 1 {
		t.Fatalf("source list output = %#v", listed.Content)
	}
	if output.Sources[0].SourceKey != "source_001" || output.Sources[0].ReadableBytes != len(sourceText) || output.Sources[0].Extraction != "stored_text" {
		t.Fatalf("source list item = %#v", output.Sources[0])
	}
	listedJSON, err := json.Marshal(listed.Content)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{sourceText, "src_report_il", "art_report_il", "filename", "url", "locator"} {
		if strings.Contains(string(listedJSON), forbidden) {
			t.Fatalf("source list leaked %q: %s", forbidden, listedJSON)
		}
	}

	read := server.Call(context.Background(), ToolCall{Name: ToolReportILSourcesRead, Arguments: mustArgs(t, map[string]any{
		"source_key": "source_001", "offset": 0, "max_bytes": 8,
	})})
	if read.Error != nil || read.TraceError != "" || read.TraceEventID == "" {
		t.Fatalf("source read = %#v", read)
	}
	chunk, ok := read.Content.(reportILSourcesReadOutput)
	if !ok || chunk.Content != sourceText[:8] || chunk.Offset != 0 || chunk.NextOffset != 8 || !chunk.Truncated || chunk.AttemptReadBytes != 8 || chunk.AttemptMaxBytes != 32 {
		t.Fatalf("source read output = %#v", read.Content)
	}
	if len(service.events) != 2 {
		t.Fatalf("source tool traces = %d", len(service.events))
	}
	trace := string(service.events[1].Payload)
	if strings.Contains(trace, sourceText) || strings.Contains(trace, chunk.Content) || !strings.Contains(trace, `"source_key":"source_001"`) || !strings.Contains(trace, `"catalog_sha256":"`+binding.Catalog.SHA256+`"`) || !strings.Contains(trace, `"returned_content_bytes":8`) {
		t.Fatalf("source read trace is not content-free and bound: %s", trace)
	}
}

func TestReportILNarrativeExposesMemoryAndDocumentWorkspaceWithoutRawSources(t *testing.T) {
	service, binding, _ := reportILSourceToolFixture(t, "Alpha fact. Beta fact.", 64, 64)
	memory := reportilcontract.EditorialMemory{
		SchemaVersion: reportilcontract.EditorialMemorySchemaVersion, Language: "en",
		Accounts: []reportilcontract.EditorialAccount{{
			AccountKey: "account_001", Importance: "essential", Account: "Alpha and beta facts support the report.",
			SourceKeys: []string{"source_001"},
		}},
	}
	memoryContent := append(mustArgs(t, reportILEditorialMemoryArtifactFixture(memory)), '\n')
	memoryArtifact, err := service.CreateRawArtifact(context.Background(), artifactcontract.CreateRequest{
		ArtifactID: "art_editorial_memory", MissionID: binding.Catalog.MissionID,
		MediaType: reportilcontract.EditorialMemoryMediaType, Filename: "report-il-editorial-memory.json",
		Producer: ledger.Producer{Type: "test", ID: "fixture"}, Content: memoryContent,
	})
	if err != nil {
		t.Fatal(err)
	}
	binding.Stage = "il_narrative"
	binding.MaxCallBytes = 0
	binding.MaxReadBytes = 0
	binding.EditorialMemoryArtifactID = memoryArtifact.ArtifactID
	binding.EditorialMemorySHA256 = memoryArtifact.SHA256
	server := newReportILSourceToolServer(service, binding)

	tools := toolNames(server.ListTools())
	want := []string{
		ToolReportILEditorialMemoryRead,
		ToolReportILDocumentStart, ToolReportILDocumentAppend, ToolReportILDocumentRead,
		ToolReportILDocumentReplace, ToolReportILDocumentFinalize,
	}
	if !reflect.DeepEqual(tools, want) {
		t.Fatalf("narrative workspace tools = %#v", tools)
	}
	if slices.Contains(tools, ToolReportILSourcesQuote) || slices.Contains(tools, ToolReportILSourcesRead) {
		t.Fatal("source tool remains in narrative authoring surface")
	}
	for _, sourceTool := range []string{ToolReportILSourcesList, ToolReportILSourcesRead, ToolReportILSourcesQuote} {
		result := server.Call(context.Background(), ToolCall{Name: sourceTool, Arguments: json.RawMessage(`{}`)})
		if result.Error == nil || !strings.Contains(result.Error.Message, "not enabled") {
			t.Fatalf("narrative stage reached raw source tool %s: %#v", sourceTool, result)
		}
	}
	beforeRead := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentStart, Arguments: mustArgs(t, map[string]any{"title": "Report", "language": "en"})})
	if beforeRead.Error == nil {
		t.Fatal("document workspace started before complete editorial memory read")
	}
	read := server.Call(context.Background(), ToolCall{Name: ToolReportILEditorialMemoryRead, Arguments: mustArgs(t, map[string]any{"offset": 0, "max_bytes": reportil.ReportILDocumentMaxReadBytes})})
	if read.Error != nil || read.Content.(reportILEditorialMemoryReadOutput).Truncated {
		t.Fatalf("complete editorial memory read = %#v", read)
	}
	started := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentStart, Arguments: mustArgs(t, map[string]any{"title": "Report", "language": "en"})})
	if started.Error != nil {
		t.Fatalf("document workspace start = %#v", started)
	}
	state := started.Content.(reportILDocumentStateOutput)
	if !strings.HasPrefix(state.WorkspaceID, "ilw_") || state.Revision != 1 || state.Finalized {
		t.Fatalf("document workspace state = %#v", state)
	}
}

func TestReportILDirectAuthorReadsSourcesAndUsesSourceBoundWorkspace(t *testing.T) {
	service, binding, _ := reportILSourceToolFixture(t, "Alpha fact. Beta fact.", 64, 64)
	binding.Stage = "il_narrative"
	binding.MaxCallBytes = 64
	binding.MaxReadBytes = 64
	server := newReportILSourceToolServer(service, binding)
	tools := toolNames(server.ListTools())
	want := []string{
		ToolReportILSourcesList, ToolReportILSourcesRead,
		ToolReportILDocumentStart, ToolReportILDocumentAppendSource,
		ToolReportILDocumentRead, ToolReportILDocumentReplace, ToolReportILDocumentFinalize,
	}
	if !reflect.DeepEqual(tools, want) {
		t.Fatalf("direct author tools = %#v", tools)
	}
	if slices.Contains(tools, ToolReportILEditorialMemoryRead) || slices.Contains(tools, ToolReportILDocumentAppend) {
		t.Fatal("direct author surface exposed editorial-memory authoring")
	}
	for _, tool := range server.ListTools() {
		if tool.Name != ToolReportILSourcesRead {
			continue
		}
		if !bytes.Equal(tool.InputSchema, schemaReportILSourcesRead) ||
			!strings.Contains(tool.Description, "source_key") ||
			!strings.Contains(tool.Description, "offset 0") ||
			!strings.Contains(tool.Description, "exact returned next_offset") ||
			strings.Contains(tool.Description, "remaining_sources") ||
			strings.Contains(tool.Description, "with {}") {
			t.Fatalf("direct author source read contract = %s / %s", tool.Description, tool.InputSchema)
		}
	}
	if beforeRead := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentStart, Arguments: mustArgs(t, map[string]any{"title": "Report", "language": "en"})}); beforeRead.Error == nil {
		t.Fatal("direct author workspace started before complete source read")
	}
	read := server.Call(context.Background(), ToolCall{Name: ToolReportILSourcesRead, Arguments: mustArgs(t, map[string]any{"source_key": "source_001", "offset": 0, "max_bytes": 64})})
	if read.Error != nil {
		t.Fatalf("source read = %#v error=%+v", read, read.Error)
	}
	started := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentStart, Arguments: mustArgs(t, map[string]any{"title": "Report", "language": "en"})})
	if started.Error != nil {
		t.Fatalf("workspace start = %#v", started)
	}
	workspaceID := started.Content.(reportILDocumentStateOutput).WorkspaceID
	for _, block := range []struct{ section, prose string }{{"Answer", "Alpha fact answers the question."}, {"Judgment", "Beta fact supports the conclusion."}} {
		appended := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentAppendSource, Arguments: mustArgs(t, map[string]any{
			"workspace_id": workspaceID, "section_title": block.section, "kind": "prose", "prose": block.prose,
			"items": []string{}, "code": "", "language": nil, "table": nil, "evidence_source_keys": []string{"source_001"},
		})})
		if appended.Error != nil {
			t.Fatalf("append direct source block = %#v", appended)
		}
	}
	if readDoc := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentRead, Arguments: mustArgs(t, map[string]any{"workspace_id": workspaceID, "offset": 0, "max_bytes": reportil.ReportILDocumentMaxReadBytes})}); readDoc.Error != nil {
		t.Fatalf("document read = %#v", readDoc)
	}
	finalized := server.Call(context.Background(), ToolCall{Name: ToolReportILDocumentFinalize, Arguments: mustArgs(t, map[string]any{"workspace_id": workspaceID})})
	if finalized.Error != nil || !finalized.Content.(reportILDocumentStateOutput).Finalized {
		t.Fatalf("direct author finalize = %#v", finalized)
	}
}

func TestReportILSourceSelectionBatchesExactUTF8AlignedSamples(t *testing.T) {
	service, binding, _ := reportILSourceToolFixture(t, "가나다라마바사", 8, 32)
	binding.Stage = "il_source_selection"
	binding.MaxSourceReadBytes = 8
	server := newReportILSourceToolServer(service, binding)

	wrongArguments := server.Call(context.Background(), ToolCall{Name: ToolReportILSourcesRead, Arguments: mustArgs(t, map[string]any{
		"source_key": "source_001", "offset": 0, "max_bytes": 8,
	})})
	if wrongArguments.Error == nil || !strings.Contains(wrongArguments.Error.Message, "empty object") {
		t.Fatalf("selection accepted caller-controlled sample: %#v", wrongArguments)
	}
	read := server.Call(context.Background(), ToolCall{Name: ToolReportILSourcesRead, Arguments: json.RawMessage(`{}`)})
	if read.Error != nil {
		t.Fatalf("selection sample = %#v", read)
	}
	batch := read.Content.(reportILSourcesBatchReadOutput)
	if len(batch.Sources) != 1 || batch.Sources[0].SourceKey != "source_001" || batch.Sources[0].Content != "가나" || batch.AttemptReadBytes != 6 || batch.RemainingSources != 0 {
		t.Fatalf("UTF-8-aligned selection batch = %#v", batch)
	}
	continued := server.Call(context.Background(), ToolCall{Name: ToolReportILSourcesRead, Arguments: json.RawMessage(`{}`)})
	if continued.Error == nil || !strings.Contains(continued.Error.Message, "already read") {
		t.Fatalf("selection allowed a second source read: %#v", continued)
	}
}

func TestReportILEditorialMemoryBatchContinuesLargeSource(t *testing.T) {
	const maxBytes = reportilcontract.DefaultSourceReadMaxBytes
	service, binding, sourceText := reportILSourceToolFixture(
		t,
		strings.Repeat("x", maxBytes+11),
		maxBytes,
		maxBytes+11,
	)
	binding.Stage = "il_editorial_memory"
	server := newReportILSourceToolServer(service, binding)

	first := server.Call(context.Background(), ToolCall{
		Name: ToolReportILSourcesRead, Arguments: json.RawMessage(`{}`),
	})
	if first.Error != nil || first.TraceError != "" || first.TraceEventID == "" {
		t.Fatalf("first narrative batch = %#v", first)
	}
	firstBatch := first.Content.(reportILSourcesBatchReadOutput)
	if firstBatch.Stage != "il_editorial_memory" || len(firstBatch.Sources) != 1 ||
		firstBatch.Sources[0].Content != sourceText[:maxBytes] ||
		firstBatch.Sources[0].Offset != 0 || firstBatch.Sources[0].NextOffset != maxBytes ||
		!firstBatch.Sources[0].Truncated || firstBatch.RemainingSources != 1 {
		t.Fatalf("first narrative batch output = %#v", firstBatch)
	}

	second := server.Call(context.Background(), ToolCall{
		Name: ToolReportILSourcesRead, Arguments: json.RawMessage(`{}`),
	})
	if second.Error != nil || second.TraceError != "" || second.TraceEventID == "" {
		t.Fatalf("second narrative batch = %#v", second)
	}
	secondBatch := second.Content.(reportILSourcesBatchReadOutput)
	if secondBatch.Stage != "il_editorial_memory" || len(secondBatch.Sources) != 1 ||
		secondBatch.Sources[0].Content != sourceText[maxBytes:] ||
		secondBatch.Sources[0].Offset != maxBytes || secondBatch.Sources[0].NextOffset != 0 ||
		secondBatch.Sources[0].Truncated || secondBatch.RemainingSources != 0 ||
		secondBatch.AttemptReadBytes != len(sourceText) {
		t.Fatalf("second narrative batch output = %#v", secondBatch)
	}
	trace := string(service.events[len(service.events)-1].Payload)
	if strings.Contains(trace, sourceText[maxBytes:]) ||
		!strings.Contains(trace, `"report_il_stage":"il_editorial_memory"`) ||
		!strings.Contains(trace, `"returned_offset":65536`) ||
		!strings.Contains(trace, `"returned_content_bytes":11`) {
		t.Fatalf("narrative continuation trace is invalid: %s", trace)
	}
}

func TestReportILNarrativeBatchDoesNotCrossTruncatedUTF8Source(t *testing.T) {
	const maxBytes = reportilcontract.DefaultSourceReadMaxBytes
	sourceContents := [][]byte{
		[]byte(strings.Repeat("x", maxBytes-1) + "가"),
		[]byte("second source"),
	}
	entries := make([]reportilcontract.SourceCatalogEntry, 0, len(sourceContents))
	service := &fakeMCPService{artifacts: map[string]artifactcontract.Raw{}}
	for index, content := range sourceContents {
		sourceKey := fmt.Sprintf("source_%03d", index+1)
		snapshotID := fmt.Sprintf("src_utf8_%03d", index+1)
		artifactID := fmt.Sprintf("art_utf8_%03d", index+1)
		hash := sha256Hex(content)
		entries = append(entries, reportilcontract.SourceCatalogEntry{
			SourceKey: sourceKey, AcceptedOrdinal: index + 1, SnapshotID: snapshotID,
			SnapshotReceipt: reportilcontract.SourceSnapshotReceipt(snapshotID, hash),
			ContentHash:     hash, RetrievalPolicy: sourcecontract.RetrievalPolicySnapshotOnly,
			Artifacts: []reportilcontract.SourceCatalogArtifact{{
				ArtifactID: artifactID, SHA256: hash, ByteSize: int64(len(content)), MediaType: "text/plain",
			}},
			ReadableSHA256: hash, ReadableBytes: len(content), Extraction: "stored_text",
		})
		service.sources = append(service.sources, sourcecontract.Snapshot{
			SnapshotID: snapshotID, MissionID: "mis_report_il_utf8", ArtifactIDs: []string{artifactID},
			ContentHash: sourcecontract.ContentHash{Algorithm: "sha256", Value: hash},
			Access:      sourcecontract.Access{RetrievalPolicy: sourcecontract.RetrievalPolicySnapshotOnly},
			State:       sourcecontract.State{State: sourcecontract.StateActive},
		})
		service.artifacts[artifactID] = artifactcontract.Raw{
			ArtifactID: artifactID, MissionID: "mis_report_il_utf8", MediaType: "text/plain",
			ByteSize: int64(len(content)), SHA256: hash, Content: content,
		}
	}
	catalog, err := reportilcontract.SealSourceCatalog(reportilcontract.SourceCatalog{
		MissionID: "mis_report_il_utf8", Sources: entries,
	})
	if err != nil {
		t.Fatal(err)
	}
	binding := reportilcontract.SourceAccessBinding{
		PendingEventID: "evt_report_il_utf8", Stage: "il_editorial_memory", Attempt: 1,
		MaxCallBytes: maxBytes, MaxReadBytes: reportilcontract.DefaultSourceAttemptReadBytes,
		Catalog: catalog,
	}
	server := newReportILSourceToolServer(service, binding)

	first := server.Call(context.Background(), ToolCall{
		Name: ToolReportILSourcesRead, Arguments: json.RawMessage(`{}`),
	})
	if first.Error != nil || first.TraceError != "" {
		t.Fatalf("UTF-8 narrative batch = %#v", first)
	}
	batch := first.Content.(reportILSourcesBatchReadOutput)
	if len(batch.Sources) != 1 || batch.Sources[0].SourceKey != "source_001" ||
		len([]byte(batch.Sources[0].Content)) != maxBytes-1 ||
		batch.Sources[0].NextOffset != maxBytes-1 || !batch.Sources[0].Truncated ||
		batch.RemainingSources != 2 {
		t.Fatalf("UTF-8 narrative batch crossed an incomplete source: %#v", batch)
	}
}

func TestReportILFlowReadsOnlyServerBoundCompactSpans(t *testing.T) {
	text := "prefix FIRST evidence middle SECOND evidence suffix"
	service, binding, _ := reportILSourceToolFixture(t, text, 64, 64)
	first := "FIRST evidence"
	second := "SECOND evidence"
	binding.Stage = "il_flow"
	binding.ReadSpans = []reportilcontract.SourceReadSpan{
		{SourceKey: "source_001", Offset: strings.Index(text, first), ByteSize: len(first), SHA256: sha256Hex([]byte(first))},
		{SourceKey: "source_001", Offset: strings.Index(text, second), ByteSize: len(second), SHA256: sha256Hex([]byte(second))},
	}
	binding.MaxReadBytes = len(first) + len(second)
	binding.MaxCallBytes = binding.MaxReadBytes
	if err := reportilcontract.ValidateSourceAccessBinding(binding); err != nil {
		t.Fatal(err)
	}
	server := newReportILSourceToolServer(service, binding)

	firstResult := server.Call(context.Background(), ToolCall{
		Name: ToolReportILSourcesRead, Arguments: json.RawMessage(`{}`),
	})
	if firstResult.Error != nil || firstResult.TraceError != "" {
		t.Fatalf("first compact span read = %#v", firstResult)
	}
	batch := firstResult.Content.(reportILSourcesBatchReadOutput)
	if len(batch.Sources) != 2 || batch.Sources[0].Content != first ||
		batch.Sources[0].Offset != strings.Index(text, first) ||
		batch.Sources[1].Content != second ||
		batch.Sources[1].Offset != strings.Index(text, second) ||
		batch.RemainingSources != 0 || batch.AttemptReadBytes != binding.MaxReadBytes ||
		strings.Contains(batch.Sources[0].Content, "prefix") ||
		strings.Contains(batch.Sources[1].Content, "suffix") {
		t.Fatalf("compact span batch escaped binding: %#v", batch)
	}
}

func TestReportILFlowRejectsTamperedCompactSpanHash(t *testing.T) {
	text := "prefix evidence suffix"
	service, binding, _ := reportILSourceToolFixture(t, text, 64, 64)
	span := "evidence"
	binding.Stage = "il_flow"
	binding.ReadSpans = []reportilcontract.SourceReadSpan{{
		SourceKey: "source_001", Offset: strings.Index(text, span),
		ByteSize: len(span), SHA256: strings.Repeat("0", 64),
	}}
	binding.MaxReadBytes = len(span)
	binding.MaxCallBytes = len(span)
	server := newReportILSourceToolServer(service, binding)
	result := server.Call(context.Background(), ToolCall{
		Name: ToolReportILSourcesRead, Arguments: json.RawMessage(`{}`),
	})
	if result.Error == nil || result.Error.Message != "flow source span validation failed" {
		t.Fatalf("tampered compact span hash was accepted: %#v", result)
	}
}

func TestReportILNarrativeBatchDefersSourceWhenResidualCannotFitFirstRune(t *testing.T) {
	const maxBytes = reportilcontract.DefaultSourceReadMaxBytes
	sourceContents := [][]byte{
		[]byte(strings.Repeat("x", maxBytes-1)),
		[]byte("가 second source"),
	}
	entries := make([]reportilcontract.SourceCatalogEntry, 0, len(sourceContents))
	service := &fakeMCPService{artifacts: map[string]artifactcontract.Raw{}}
	for index, content := range sourceContents {
		sourceKey := fmt.Sprintf("source_%03d", index+1)
		snapshotID := fmt.Sprintf("src_residual_%03d", index+1)
		artifactID := fmt.Sprintf("art_residual_%03d", index+1)
		hash := sha256Hex(content)
		entries = append(entries, reportilcontract.SourceCatalogEntry{
			SourceKey: sourceKey, AcceptedOrdinal: index + 1, SnapshotID: snapshotID,
			SnapshotReceipt: reportilcontract.SourceSnapshotReceipt(snapshotID, hash),
			ContentHash:     hash, RetrievalPolicy: sourcecontract.RetrievalPolicySnapshotOnly,
			Artifacts: []reportilcontract.SourceCatalogArtifact{{
				ArtifactID: artifactID, SHA256: hash, ByteSize: int64(len(content)), MediaType: "text/plain",
			}},
			ReadableSHA256: hash, ReadableBytes: len(content), Extraction: "stored_text",
		})
		service.sources = append(service.sources, sourcecontract.Snapshot{
			SnapshotID: snapshotID, MissionID: "mis_report_il_residual", ArtifactIDs: []string{artifactID},
			ContentHash: sourcecontract.ContentHash{Algorithm: "sha256", Value: hash},
			Access:      sourcecontract.Access{RetrievalPolicy: sourcecontract.RetrievalPolicySnapshotOnly},
			State:       sourcecontract.State{State: sourcecontract.StateActive},
		})
		service.artifacts[artifactID] = artifactcontract.Raw{
			ArtifactID: artifactID, MissionID: "mis_report_il_residual", MediaType: "text/plain",
			ByteSize: int64(len(content)), SHA256: hash, Content: content,
		}
	}
	catalog, err := reportilcontract.SealSourceCatalog(reportilcontract.SourceCatalog{
		MissionID: "mis_report_il_residual", Sources: entries,
	})
	if err != nil {
		t.Fatal(err)
	}
	binding := reportilcontract.SourceAccessBinding{
		PendingEventID: "evt_report_il_residual", Stage: "il_editorial_memory", Attempt: 1,
		MaxCallBytes: maxBytes, MaxReadBytes: reportilcontract.DefaultSourceAttemptReadBytes,
		Catalog: catalog,
	}
	server := newReportILSourceToolServer(service, binding)

	first := server.Call(context.Background(), ToolCall{
		Name: ToolReportILSourcesRead, Arguments: json.RawMessage(`{}`),
	})
	if first.Error != nil || first.TraceError != "" {
		t.Fatalf("residual narrative batch = %#v", first)
	}
	firstBatch := first.Content.(reportILSourcesBatchReadOutput)
	if len(firstBatch.Sources) != 1 || firstBatch.Sources[0].SourceKey != "source_001" ||
		firstBatch.Sources[0].Truncated || firstBatch.RemainingSources != 1 ||
		firstBatch.AttemptReadBytes != maxBytes-1 {
		t.Fatalf("residual narrative batch consumed an incomplete rune: %#v", firstBatch)
	}

	second := server.Call(context.Background(), ToolCall{
		Name: ToolReportILSourcesRead, Arguments: json.RawMessage(`{}`),
	})
	if second.Error != nil || second.TraceError != "" {
		t.Fatalf("deferred narrative source = %#v", second)
	}
	secondBatch := second.Content.(reportILSourcesBatchReadOutput)
	if len(secondBatch.Sources) != 1 || secondBatch.Sources[0].SourceKey != "source_002" ||
		secondBatch.Sources[0].Content != string(sourceContents[1]) ||
		secondBatch.RemainingSources != 0 {
		t.Fatalf("deferred narrative source was not completed: %#v", secondBatch)
	}
}

func TestReportILSourceSelectionBatchesLargeCatalogWithinCallCeiling(t *testing.T) {
	const sourceCount = 18
	const sampleBytes = 8 * 1024
	entries := make([]reportilcontract.SourceCatalogEntry, 0, sourceCount)
	service := &fakeMCPService{artifacts: map[string]artifactcontract.Raw{}}
	for index := 0; index < sourceCount; index++ {
		key := fmt.Sprintf("source_%03d", index+1)
		snapshotID := fmt.Sprintf("src_batch_%03d", index+1)
		artifactID := fmt.Sprintf("art_batch_%03d", index+1)
		content := []byte(strings.Repeat(string(rune('a'+index)), sampleBytes+1))
		hash := sha256Hex(content)
		entries = append(entries, reportilcontract.SourceCatalogEntry{
			SourceKey: key, AcceptedOrdinal: index + 1, SnapshotID: snapshotID,
			SnapshotReceipt: reportilcontract.SourceSnapshotReceipt(snapshotID, hash),
			ContentHash:     hash, RetrievalPolicy: sourcecontract.RetrievalPolicySnapshotOnly,
			Artifacts: []reportilcontract.SourceCatalogArtifact{{
				ArtifactID: artifactID, SHA256: hash, ByteSize: int64(len(content)), MediaType: "text/plain",
			}},
			ReadableSHA256: hash, ReadableBytes: len(content), Extraction: "stored_text",
		})
		service.sources = append(service.sources, sourcecontract.Snapshot{
			SnapshotID: snapshotID, MissionID: "mis_report_il_batch", ArtifactIDs: []string{artifactID},
			ContentHash: sourcecontract.ContentHash{Algorithm: "sha256", Value: hash},
			Access:      sourcecontract.Access{RetrievalPolicy: sourcecontract.RetrievalPolicySnapshotOnly},
			State:       sourcecontract.State{State: sourcecontract.StateActive},
		})
		service.artifacts[artifactID] = artifactcontract.Raw{
			ArtifactID: artifactID, MissionID: "mis_report_il_batch", MediaType: "text/plain",
			ByteSize: int64(len(content)), SHA256: hash, Content: content,
		}
	}
	catalog, err := reportilcontract.SealSourceCatalog(reportilcontract.SourceCatalog{
		MissionID: "mis_report_il_batch", Sources: entries,
	})
	if err != nil {
		t.Fatal(err)
	}
	binding := reportilcontract.SourceAccessBinding{
		PendingEventID: "evt_report_il_batch", Stage: "il_source_selection", Attempt: 1,
		MaxCallBytes:       reportilcontract.DefaultSourceReadMaxBytes,
		MaxReadBytes:       reportilcontract.DefaultSourceAttemptReadBytes,
		MaxSourceReadBytes: sampleBytes,
		Catalog:            catalog,
	}
	server := newReportILSourceToolServer(service, binding)
	tools := server.ListTools()
	var selectionReadSchema string
	for _, tool := range tools {
		if tool.Name == ToolReportILSourcesRead {
			selectionReadSchema = string(tool.InputSchema)
		}
	}
	if selectionReadSchema == "" || strings.Contains(selectionReadSchema, "source_key") {
		t.Fatalf("selection read schema is caller-directed: %s", selectionReadSchema)
	}

	batchCounts := []int{}
	for {
		result := server.Call(context.Background(), ToolCall{
			Name: ToolReportILSourcesRead, Arguments: json.RawMessage(`{}`),
		})
		if result.Error != nil || result.TraceError != "" {
			t.Fatalf("selection batch %d = %#v", len(batchCounts)+1, result)
		}
		batch := result.Content.(reportILSourcesBatchReadOutput)
		batchBytes := 0
		for _, item := range batch.Sources {
			batchBytes += len([]byte(item.Content))
			if len([]byte(item.Content)) != sampleBytes || !item.Truncated {
				t.Fatalf("selection item sample = %#v", item)
			}
		}
		if batchBytes > reportilcontract.DefaultSourceReadMaxBytes {
			t.Fatalf("selection batch returned %d bytes", batchBytes)
		}
		batchCounts = append(batchCounts, len(batch.Sources))
		if batch.RemainingSources == 0 {
			break
		}
	}
	if !reflect.DeepEqual(batchCounts, []int{8, 8, 2}) || server.reportILState.Source.ReadBytes != sourceCount*sampleBytes {
		t.Fatalf("selection batches=%v bytes=%d", batchCounts, server.reportILState.Source.ReadBytes)
	}

	narrativeBinding := binding
	narrativeBinding.Stage = "il_editorial_memory"
	narrativeBinding.MaxSourceReadBytes = 0
	narrativeServer := newReportILSourceToolServer(service, narrativeBinding)
	var narrativeReadSchema string
	for _, tool := range narrativeServer.ListTools() {
		if tool.Name == ToolReportILSourcesRead {
			narrativeReadSchema = string(tool.InputSchema)
		}
	}
	if narrativeReadSchema == "" || strings.Contains(narrativeReadSchema, "source_key") {
		t.Fatalf("narrative read schema is caller-directed: %s", narrativeReadSchema)
	}
	for batchIndex := 0; ; batchIndex++ {
		result := narrativeServer.Call(context.Background(), ToolCall{
			Name: ToolReportILSourcesRead, Arguments: json.RawMessage(`{}`),
		})
		if result.Error != nil || result.TraceError != "" {
			t.Fatalf("narrative batch %d = %#v", batchIndex+1, result)
		}
		batch := result.Content.(reportILSourcesBatchReadOutput)
		if batch.Stage != "il_editorial_memory" || len(batch.Sources) == 0 {
			t.Fatalf("narrative batch %d = %#v", batchIndex+1, batch)
		}
		if batch.RemainingSources == 0 {
			break
		}
	}
	if narrativeServer.reportILState.Source.ReadBytes != sourceCount*(sampleBytes+1) {
		t.Fatalf("narrative complete bytes=%d", narrativeServer.reportILState.Source.ReadBytes)
	}

	flowBinding := narrativeBinding
	flowBinding.Stage = "il_flow"
	flowServer := newReportILSourceToolServer(service, flowBinding)
	var flowReadSchema string
	for _, tool := range flowServer.ListTools() {
		if tool.Name == ToolReportILSourcesRead {
			flowReadSchema = string(tool.InputSchema)
		}
	}
	if flowReadSchema == "" || strings.Contains(flowReadSchema, "source_key") {
		t.Fatalf("flow read schema is caller-directed: %s", flowReadSchema)
	}
	for batchIndex := 0; ; batchIndex++ {
		result := flowServer.Call(context.Background(), ToolCall{
			Name: ToolReportILSourcesRead, Arguments: json.RawMessage(`{}`),
		})
		if result.Error != nil || result.TraceError != "" {
			t.Fatalf("flow batch %d = %#v", batchIndex+1, result)
		}
		batch := result.Content.(reportILSourcesBatchReadOutput)
		if batch.Stage != "il_flow" || len(batch.Sources) == 0 {
			t.Fatalf("flow batch %d = %#v", batchIndex+1, batch)
		}
		if batch.RemainingSources == 0 {
			break
		}
	}
	if flowServer.reportILState.Source.ReadBytes != sourceCount*(sampleBytes+1) {
		t.Fatalf("flow complete bytes=%d", flowServer.reportILState.Source.ReadBytes)
	}
}

func TestReportILSourceReadKeepsOmittedPolicyNormalizedToSnapshotOnly(t *testing.T) {
	service, binding, _ := reportILSourceToolFixture(t, "frozen", 8, 32)
	service.sources[0].Access.RetrievalPolicy = ""
	result := newReportILSourceToolServer(service, binding).Call(context.Background(), ToolCall{
		Name: ToolReportILSourcesRead,
		Arguments: mustArgs(t, map[string]any{
			"source_key": "source_001", "offset": 0, "max_bytes": 6,
		}),
	})
	if result.Error != nil || result.Content.(reportILSourcesReadOutput).Content != "frozen" {
		t.Fatalf("omitted snapshot policy changed at runtime: %#v", result)
	}
}

func TestReportILSourceReadEnforcesUTF8AndAttemptBudgets(t *testing.T) {
	service, binding, _ := reportILSourceToolFixture(t, "가나다라마바사", 4, 6)
	server := newReportILSourceToolServer(service, binding)

	nonzeroFirstOffset := server.Call(context.Background(), ToolCall{Name: ToolReportILSourcesRead, Arguments: mustArgs(t, map[string]any{
		"source_key": "source_001", "offset": 1, "max_bytes": 3,
	})})
	if nonzeroFirstOffset.Error == nil || !strings.Contains(nonzeroFirstOffset.Error.Message, "exact next_offset 0") {
		t.Fatalf("nonzero first source offset = %#v", nonzeroFirstOffset)
	}
	tooLarge := server.Call(context.Background(), ToolCall{Name: ToolReportILSourcesRead, Arguments: mustArgs(t, map[string]any{
		"source_key": "source_001", "offset": 0, "max_bytes": 5,
	})})
	if tooLarge.Error == nil || !strings.Contains(tooLarge.Error.Message, "per-call") {
		t.Fatalf("per-call ceiling = %#v", tooLarge)
	}
	first := server.Call(context.Background(), ToolCall{Name: ToolReportILSourcesRead, Arguments: mustArgs(t, map[string]any{
		"source_key": "source_001", "offset": 0, "max_bytes": 4,
	})})
	if first.Error != nil {
		t.Fatalf("first bounded read = %#v", first)
	}
	firstOutput := first.Content.(reportILSourcesReadOutput)
	if firstOutput.Content != "가" || firstOutput.NextOffset != 3 || firstOutput.AttemptReadBytes != 3 {
		t.Fatalf("UTF-8 bounded chunk = %#v", firstOutput)
	}
	second := server.Call(context.Background(), ToolCall{Name: ToolReportILSourcesRead, Arguments: mustArgs(t, map[string]any{
		"source_key": "source_001", "offset": firstOutput.NextOffset, "max_bytes": 3,
	})})
	if second.Error != nil || second.Content.(reportILSourcesReadOutput).AttemptReadBytes != 6 {
		t.Fatalf("second bounded read = %#v", second)
	}
	exhausted := server.Call(context.Background(), ToolCall{Name: ToolReportILSourcesRead, Arguments: mustArgs(t, map[string]any{
		"source_key": "source_001", "offset": 6, "max_bytes": 1,
	})})
	if exhausted.Error == nil || !strings.Contains(exhausted.Error.Message, "attempt byte ceiling") {
		t.Fatalf("attempt ceiling = %#v", exhausted)
	}
}

func TestReportILSourceReadChargesActualFinalChunkAgainstAttemptBudget(t *testing.T) {
	service, binding, _ := reportILSourceToolFixture(t, strings.Repeat("x", 11), 8, 11)
	server := newReportILSourceToolServer(service, binding)
	first := server.Call(context.Background(), ToolCall{
		Name: ToolReportILSourcesRead,
		Arguments: mustArgs(t, map[string]any{
			"source_key": "source_001", "offset": 0, "max_bytes": 8,
		}),
	})
	if first.Error != nil || first.Content.(reportILSourcesReadOutput).NextOffset != 8 {
		t.Fatalf("first source budget page = %#v", first)
	}
	last := server.Call(context.Background(), ToolCall{
		Name: ToolReportILSourcesRead,
		Arguments: mustArgs(t, map[string]any{
			"source_key": "source_001", "offset": 8, "max_bytes": 8,
		}),
	})
	if last.Error != nil || last.Content.(reportILSourcesReadOutput).Content != "xxx" || last.Content.(reportILSourcesReadOutput).AttemptReadBytes != 11 {
		t.Fatalf("short final source budget page = %#v", last)
	}
}

func TestReportILSourceReadUsesEntire64KiBContractCeiling(t *testing.T) {
	const maxBytes = reportilcontract.DefaultSourceReadMaxBytes
	service, binding, sourceText := reportILSourceToolFixture(
		t,
		strings.Repeat("x", maxBytes+1),
		maxBytes,
		maxBytes+1,
	)
	result := newReportILSourceToolServer(service, binding).Call(context.Background(), ToolCall{
		Name: ToolReportILSourcesRead,
		Arguments: mustArgs(t, map[string]any{
			"source_key": "source_001", "offset": 0, "max_bytes": maxBytes,
		}),
	})
	if result.Error != nil || result.TraceError != "" {
		t.Fatalf("64 KiB report IL source read = %#v", result)
	}
	output := result.Content.(reportILSourcesReadOutput)
	if len(output.Content) != maxBytes || output.Content != sourceText[:maxBytes] || output.NextOffset != maxBytes || !output.Truncated {
		t.Fatalf("64 KiB report IL source chunk = %#v", output)
	}
}

func TestReportILSourceReadRequiresForwardOnlyNonOverlappingContinuation(t *testing.T) {
	service, binding, _ := reportILSourceToolFixture(t, "abcdefghij", 4, 32)
	server := newReportILSourceToolServer(service, binding)
	read := func(offset, maxBytes int) ToolResult {
		return server.Call(context.Background(), ToolCall{
			Name: ToolReportILSourcesRead,
			Arguments: mustArgs(t, map[string]any{
				"source_key": "source_001", "offset": offset, "max_bytes": maxBytes,
			}),
		})
	}

	if result := read(1, 4); result.Error == nil || !strings.Contains(result.Error.Message, "exact next_offset 0") {
		t.Fatalf("nonzero first offset = %#v", result)
	}
	first := read(0, 4)
	if first.Error != nil || first.Content.(reportILSourcesReadOutput).NextOffset != 4 {
		t.Fatalf("first source page = %#v", first)
	}
	for _, offset := range []int{0, 3, 5} {
		if result := read(offset, 4); result.Error == nil || !strings.Contains(result.Error.Message, "exact next_offset 4") {
			t.Fatalf("non-contiguous offset %d = %#v", offset, result)
		}
	}
	second := read(4, 4)
	if second.Error != nil || second.Content.(reportILSourcesReadOutput).NextOffset != 8 {
		t.Fatalf("second source page = %#v", second)
	}
	last := read(8, 4)
	if last.Error != nil || last.Content.(reportILSourcesReadOutput).Truncated || last.Content.(reportILSourcesReadOutput).Content != "ij" {
		t.Fatalf("last source page = %#v", last)
	}
	if result := read(0, 4); result.Error == nil || !strings.Contains(result.Error.Message, "already completely read") {
		t.Fatalf("source restart after EOF = %#v", result)
	}
	if server.reportILState.Source.ReadBytes != len("abcdefghij") {
		t.Fatalf("failed reads changed source byte budget: %d", server.reportILState.Source.ReadBytes)
	}
}

func TestReportILSourceReadExtractsPDFAsCanonicalText(t *testing.T) {
	pdf := testPDFBytes(t, []string{"IL PDF source", "Alpha fact is 42."})
	readable, err := reportilsource.Extract(pdf, "application/pdf")
	if err != nil {
		t.Fatal(err)
	}
	artifactHash := sha256Hex(pdf)
	catalog, err := reportilcontract.SealSourceCatalog(reportilcontract.SourceCatalog{
		MissionID: "mis_report_il",
		Sources: []reportilcontract.SourceCatalogEntry{{
			SourceKey: "source_001", SnapshotID: "src_report_il",
			SnapshotReceipt: reportilcontract.SourceSnapshotReceipt("src_report_il", artifactHash),
			ContentHash:     artifactHash, RetrievalPolicy: sourcecontract.RetrievalPolicySnapshotOnly,
			Artifacts: []reportilcontract.SourceCatalogArtifact{{
				ArtifactID: "art_report_il", SHA256: artifactHash, ByteSize: int64(len(pdf)), MediaType: "application/pdf",
			}},
			ReadableSHA256: readable.SHA256, ReadableBytes: readable.ByteSize, Extraction: readable.Extraction,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	service := &fakeMCPService{
		sources: []sourcecontract.Snapshot{{
			SnapshotID: "src_report_il", MissionID: "mis_report_il", ArtifactIDs: []string{"art_report_il"},
			ContentHash: sourcecontract.ContentHash{Algorithm: "sha256", Value: artifactHash},
			Access:      sourcecontract.Access{RetrievalPolicy: sourcecontract.RetrievalPolicySnapshotOnly},
			State:       sourcecontract.State{State: sourcecontract.StateActive},
		}},
		artifacts: map[string]artifactcontract.Raw{
			"art_report_il": {ArtifactID: "art_report_il", MissionID: "mis_report_il", MediaType: "application/pdf", ByteSize: int64(len(pdf)), SHA256: artifactHash, Content: pdf},
		},
	}
	binding := reportilcontract.SourceAccessBinding{
		PendingEventID: "evt_report_il", Stage: "il_document", Attempt: 1,
		MaxCallBytes: readable.ByteSize, MaxReadBytes: readable.ByteSize, Catalog: catalog,
	}
	result := newReportILSourceToolServer(service, binding).Call(context.Background(), ToolCall{
		Name:      ToolReportILSourcesRead,
		Arguments: mustArgs(t, map[string]any{"source_key": "source_001", "offset": 0, "max_bytes": readable.ByteSize}),
	})
	if result.Error != nil || result.TraceError != "" {
		t.Fatalf("PDF source read = %#v", result)
	}
	output := result.Content.(reportILSourcesReadOutput)
	if output.Content != readable.Text || output.Extraction != "pdf_text" || strings.Contains(output.Content, "%PDF-") || !strings.Contains(output.Content, "Alpha fact is 42") {
		t.Fatalf("PDF canonical source output = %#v", output)
	}
	if trace := string(service.events[0].Payload); strings.Contains(trace, output.Content) || !strings.Contains(trace, `"extraction_type":"pdf_text"`) {
		t.Fatalf("PDF source trace leaked content or missed extraction receipt: %s", trace)
	}
}

func TestReportILSourceReadSerializesConcurrentAttemptBudget(t *testing.T) {
	service, binding, _ := reportILSourceToolFixture(t, strings.Repeat("x", 32), 8, 8)
	server := newReportILSourceToolServer(service, binding)
	const calls = 32
	results := make(chan ToolResult, calls)
	start := make(chan struct{})
	var wait sync.WaitGroup
	for index := 0; index < calls; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			results <- server.dispatchCall(context.Background(), ToolCall{
				Name:      ToolReportILSourcesRead,
				Arguments: mustArgs(t, map[string]any{"source_key": "source_001", "offset": 0, "max_bytes": 8}),
			})
		}()
	}
	close(start)
	wait.Wait()
	close(results)
	successes := 0
	for result := range results {
		if result.Error == nil {
			successes++
			output := result.Content.(reportILSourcesReadOutput)
			if output.AttemptReadBytes > binding.MaxReadBytes {
				t.Fatalf("concurrent read exceeded attempt ceiling: %#v", output)
			}
		} else if !strings.Contains(result.Error.Message, "exact next_offset 8") {
			t.Fatalf("unexpected concurrent read error: %#v", result)
		}
	}
	if successes != 1 || server.reportILState.Source.ReadBytes != binding.MaxReadBytes {
		t.Fatalf("concurrent source budget successes=%d bytes=%d", successes, server.reportILState.Source.ReadBytes)
	}
}

func TestReportILSourceReadFailsClosedForWrongOrChangedFrozenSource(t *testing.T) {
	t.Run("wrong key", func(t *testing.T) {
		service, binding, _ := reportILSourceToolFixture(t, "frozen", 8, 32)
		result := newReportILSourceToolServer(service, binding).Call(context.Background(), ToolCall{Name: ToolReportILSourcesRead, Arguments: mustArgs(t, map[string]any{
			"source_key": "source_999", "offset": 0, "max_bytes": 1,
		})})
		if result.Error == nil || !strings.Contains(result.Error.Message, "not available") {
			t.Fatalf("wrong source key = %#v", result)
		}
	})

	for _, tc := range []struct {
		name   string
		mutate func(*fakeMCPService)
	}{
		{name: "removed", mutate: func(service *fakeMCPService) {
			service.sources[0].State = sourcecontract.State{State: sourcecontract.StateRemoved, Removed: true}
		}},
		{name: "changed snapshot hash", mutate: func(service *fakeMCPService) {
			service.sources[0].ContentHash.Value = strings.Repeat("b", 64)
		}},
		{name: "changed retrieval policy", mutate: func(service *fakeMCPService) {
			service.sources[0].Access.RetrievalPolicy = sourcecontract.RetrievalPolicyLiveReference
		}},
		{name: "changed artifact bytes and metadata", mutate: func(service *fakeMCPService) {
			artifact := service.artifacts["art_report_il"]
			artifact.Content = []byte("changed")
			artifact.ByteSize = int64(len(artifact.Content))
			artifact.SHA256 = sha256Hex(artifact.Content)
			service.artifacts[artifact.ArtifactID] = artifact
		}},
		{name: "changed artifact bytes with stale metadata", mutate: func(service *fakeMCPService) {
			artifact := service.artifacts["art_report_il"]
			artifact.Content = []byte("FROZEN")
			service.artifacts[artifact.ArtifactID] = artifact
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service, binding, _ := reportILSourceToolFixture(t, "frozen", 8, 32)
			tc.mutate(service)
			result := newReportILSourceToolServer(service, binding).Call(context.Background(), ToolCall{Name: ToolReportILSourcesRead, Arguments: mustArgs(t, map[string]any{
				"source_key": "source_001", "offset": 0, "max_bytes": 1,
			})})
			if result.Error == nil || result.Error.Message != "frozen report IL source validation failed" {
				t.Fatalf("changed frozen source did not fail closed: %#v", result)
			}
			trace := string(service.events[len(service.events)-1].Payload)
			for _, forbidden := range []string{"src_report_il", "art_report_il", "identity changed", "receipt changed", "retrieval policy changed", "no longer active"} {
				if strings.Contains(trace, forbidden) {
					t.Fatalf("changed source trace leaked %q: %s", forbidden, trace)
				}
			}
		})
	}
}

func reportILSourceToolFixture(t *testing.T, text string, maxCall, maxRead int) (*fakeMCPService, reportilcontract.SourceAccessBinding, string) {
	t.Helper()
	content := []byte(text)
	hash := sha256Hex(content)
	catalog, err := reportilcontract.SealSourceCatalog(reportilcontract.SourceCatalog{
		MissionID: "mis_report_il",
		Sources: []reportilcontract.SourceCatalogEntry{{
			SourceKey:       "source_001",
			SnapshotID:      "src_report_il",
			SnapshotReceipt: reportilcontract.SourceSnapshotReceipt("src_report_il", hash),
			ContentHash:     hash,
			RetrievalPolicy: sourcecontract.RetrievalPolicySnapshotOnly,
			Artifacts: []reportilcontract.SourceCatalogArtifact{{
				ArtifactID: "art_report_il", SHA256: hash, ByteSize: int64(len(content)), MediaType: "text/plain",
			}},
			ReadableSHA256: hash,
			ReadableBytes:  len(content),
			Extraction:     "stored_text",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	service := &fakeMCPService{
		sources: []sourcecontract.Snapshot{{
			SnapshotID: "src_report_il", MissionID: "mis_report_il", ArtifactIDs: []string{"art_report_il"},
			ContentHash: sourcecontract.ContentHash{Algorithm: "sha256", Value: hash},
			Access:      sourcecontract.Access{RetrievalPolicy: sourcecontract.RetrievalPolicySnapshotOnly},
			State:       sourcecontract.State{State: sourcecontract.StateActive},
		}},
		artifacts: map[string]artifactcontract.Raw{
			"art_report_il": {ArtifactID: "art_report_il", MissionID: "mis_report_il", MediaType: "text/plain", ByteSize: int64(len(content)), SHA256: hash, Content: content},
		},
	}
	binding := reportilcontract.SourceAccessBinding{
		PendingEventID: "evt_report_il", Stage: "il_document", Attempt: 1,
		MaxCallBytes: maxCall, MaxReadBytes: maxRead, Catalog: catalog,
	}
	if err := reportilcontract.ValidateSourceAccessBinding(binding); err != nil {
		t.Fatal(err)
	}
	return service, binding, text
}

func newReportILSourceToolServer(service *fakeMCPService, binding reportilcontract.SourceAccessBinding) *Server {
	return NewServer(
		service,
		WithBinding(Binding{MissionID: binding.Catalog.MissionID, AgentSessionID: "ses_report_il", AgentExecutor: "codex"}),
		WithReportILSourceBinding(binding),
		WithEnabledTools([]string{ToolReportILSourcesList, ToolReportILSourcesRead}),
	)
}
